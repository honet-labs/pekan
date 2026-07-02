package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"pekan/backend/internal/modules/finance/transactions/domain"
	"pekan/backend/internal/platform/access"
	"pekan/backend/internal/platform/security"
)

type Authorizer interface {
	EnsureModule(ctx context.Context, module string) error
	EnsureFeature(ctx context.Context, feature string) error
	EnsurePermission(ctx context.Context, permission string) error
}

type AuditLogger interface {
	Write(ctx context.Context, action, resourceType, resourceID string, before, after any) error
}

type savingsReconciler interface {
	ReconcileSavingsCurrentAmounts(ctx context.Context, tenantID, actorUserID string, savingsIDs []string) error
}

type budgetAlertChecker interface {
	CheckAlerts(ctx context.Context, tenantID, categoryID string) error
}

type Service struct {
	repo          domain.Repository
	authz         Authorizer
	audit         AuditLogger
	budgetChecker budgetAlertChecker
	redis         *redis.Client
}

func NewService(repo domain.Repository, authz Authorizer, audit AuditLogger) *Service {
	return &Service{
		repo:  repo,
		authz: authz,
		audit: audit,
	}
}

func (s *Service) WithBudgetChecker(bc budgetAlertChecker) {
	s.budgetChecker = bc
}

func (s *Service) WithRedis(rdb *redis.Client) {
	s.redis = rdb
}

func (s *Service) clearCache(ctx context.Context, tenantID string) {
	if s.redis == nil {
		return
	}
	pattern := fmt.Sprintf("finance:dashboard:summary:%s:*", tenantID)
	var cursor uint64
	for {
		keys, nextCursor, err := s.redis.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			break
		}
		if len(keys) > 0 {
			s.redis.Del(ctx, keys...)
		}
		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}
}

type CreateInput struct {
	TenantID             string
	ActorUserID          string
	AccountID            string
	CategoryID           *string
	CategoryName         *string
	SavingsIDs           []string
	Type                 domain.TransactionType
	AmountMinor          int64
	Currency             string
	TransactionDate      time.Time
	Description          *string
	MerchantName         *string
	ReceiptNumber        *string
	PaymentMethod        *string
	SubtotalMinor        int64
	TaxMinor             int64
	ServiceChargeMinor   int64
	ReceiptDiscountMinor int64
	Items                []domain.TransactionItem
	ScanID               *string
}

func (s *Service) Create(ctx context.Context, in CreateInput) (domain.Transaction, error) {
	if err := s.authz.EnsureModule(ctx, "finance"); err != nil {
		return domain.Transaction{}, err
	}
	if err := s.authz.EnsureFeature(ctx, "finance.transactions.write"); err != nil {
		return domain.Transaction{}, err
	}
	if err := s.ensureAnyPermission(ctx, "finance.transactions.create", "finance.transactions.update"); err != nil {
		return domain.Transaction{}, err
	}
	if err := domain.ValidateCreateInput(in.AccountID, in.AmountMinor, in.Currency, in.Type); err != nil {
		return domain.Transaction{}, err
	}
	if err := domain.ValidateStringLengths(in.Description, in.MerchantName, in.ReceiptNumber, in.PaymentMethod); err != nil {
		return domain.Transaction{}, err
	}
	resolvedCategoryID, err := s.repo.ResolveCategoryID(ctx, in.TenantID, in.ActorUserID, in.CategoryID, in.CategoryName, in.Type)
	if err != nil {
		return domain.Transaction{}, err
	}
	if err := s.repo.ValidateReferences(ctx, in.TenantID, in.AccountID, resolvedCategoryID, in.Type); err != nil {
		return domain.Transaction{}, err
	}
	normalizedSavingsIDs := normalizeIDs(in.SavingsIDs)
	if in.Type != domain.TransactionTypeSavings {
		normalizedSavingsIDs = nil
	}
	if in.Type == domain.TransactionTypeSavings && len(normalizedSavingsIDs) == 0 {
		return domain.Transaction{}, domain.ErrInvalidSavingsSelection
	}
	if err := s.repo.ValidateSavingsGoals(ctx, in.TenantID, normalizedSavingsIDs); err != nil {
		return domain.Transaction{}, err
	}

	merchantName, receiptNumber, paymentMethod, subtotalMinor, taxMinor, serviceChargeMinor, receiptDiscountMinor := sanitizeReceiptMeta(
		in.Type, in.MerchantName, in.ReceiptNumber, in.PaymentMethod, in.SubtotalMinor, in.TaxMinor, in.ServiceChargeMinor, in.ReceiptDiscountMinor,
	)

	// targeted sanitization for text fields
	safeDescription := in.Description
	if safeDescription != nil {
		s := security.SanitizeHTML(*safeDescription)
		safeDescription = &s
	}
	if merchantName != nil {
		s := security.SanitizeHTML(*merchantName)
		merchantName = &s
	}

	now := time.Now().UTC()
	trx := domain.Transaction{
		TenantID:    in.TenantID,
		AccountID:   in.AccountID,
		CategoryID:  resolvedCategoryID,
		Type:        in.Type,
		AmountMinor: in.AmountMinor,
		Currency:    in.Currency,
		// InputDate is system-managed and immutable for audit traceability.
		InputDate:            now,
		TransactionDate:      in.TransactionDate,
		Description:          safeDescription,
		MerchantName:         merchantName,
		ReceiptNumber:        receiptNumber,
		PaymentMethod:        paymentMethod,
		SubtotalMinor:        subtotalMinor,
		TaxMinor:             taxMinor,
		ServiceChargeMinor:   serviceChargeMinor,
		ReceiptDiscountMinor: receiptDiscountMinor,
		CreatedBy:            in.ActorUserID,
		UpdatedBy:            in.ActorUserID,
		CreatedAt:            now,
		UpdatedAt:            now,
	}

	created, err := s.repo.Create(ctx, trx)
	if err != nil {
		return domain.Transaction{}, err
	}
	if err := s.repo.ReplaceSavingsLinks(ctx, in.TenantID, created.ID, in.ActorUserID, in.AmountMinor, normalizedSavingsIDs); err != nil {
		return domain.Transaction{}, err
	}
	if in.Type == domain.TransactionTypeSavings {
		allocations, err := s.repo.ListSavingsAllocationsByTransaction(ctx, in.TenantID, created.ID)
		if err != nil {
			return domain.Transaction{}, err
		}
		if err := s.repo.AdjustSavingsCurrentAmounts(ctx, in.TenantID, in.ActorUserID, allocations); err != nil {
			return domain.Transaction{}, err
		}
		if err := s.reconcileSavings(ctx, in.TenantID, in.ActorUserID, normalizedSavingsIDs); err != nil {
			return domain.Transaction{}, err
		}
	}
	created.SavingsIDs = normalizedSavingsIDs
	if in.Type == domain.TransactionTypeSavings {
		_, namesByTrx, err := s.repo.ListSavingsLinks(ctx, in.TenantID, []string{created.ID})
		if err != nil {
			return domain.Transaction{}, err
		}
		created.SavingsNames = namesByTrx[created.ID]
	}
	items := normalizeTransactionItems(in.Items)
	if in.Type != domain.TransactionTypeExpense {
		items = nil
	}
	if err := s.repo.ReplaceItems(ctx, in.TenantID, created.ID, in.ActorUserID, items); err != nil {
		return domain.Transaction{}, err
	}
	created.Items = items

	if created.Type == domain.TransactionTypeExpense && s.budgetChecker != nil && created.CategoryID != nil {
		go func() {
			bgCtx := context.Background()
			_ = s.budgetChecker.CheckAlerts(bgCtx, created.TenantID, *created.CategoryID)
		}()
	}

	if s.audit != nil {
		_ = s.audit.Write(ctx, "finance.transaction.create", "finance_transaction", created.ID, nil, created)
	}
	s.clearCache(ctx, in.TenantID)
	return created, nil
}

func (s *Service) GetByID(ctx context.Context, tenantID, transactionID string) (domain.Transaction, error) {
	if err := s.authz.EnsureModule(ctx, "finance"); err != nil {
		return domain.Transaction{}, err
	}
	if err := s.authz.EnsureFeature(ctx, "finance.transactions.read"); err != nil {
		return domain.Transaction{}, err
	}
	if err := s.authz.EnsurePermission(ctx, "finance.transactions.read"); err != nil {
		return domain.Transaction{}, err
	}

	trx, err := s.repo.GetByID(ctx, tenantID, transactionID)
	if err != nil {
		return domain.Transaction{}, err
	}
	idMap, nameMap, err := s.repo.ListSavingsLinks(ctx, tenantID, []string{transactionID})
	if err != nil {
		return domain.Transaction{}, err
	}
	trx.SavingsIDs = idMap[transactionID]
	trx.SavingsNames = nameMap[transactionID]
	items, err := s.repo.ListItems(ctx, tenantID, transactionID)
	if err != nil {
		return domain.Transaction{}, err
	}
	trx.Items = items

	attachmentsMap, err := s.repo.ListAttachmentsByTransactionIDs(ctx, tenantID, []string{transactionID})
	if err != nil {
		return domain.Transaction{}, err
	}
	trx.Attachments = attachmentsMap[transactionID]

	return trx, nil
}

type ListInput struct {
	TenantID string
	Type     *domain.TransactionType
	DateFrom *time.Time
	DateTo   *time.Time
	Query    string
	CategoryID *string
	Page       int
	PageSize   int
}

func (s *Service) List(ctx context.Context, in ListInput) ([]domain.Transaction, int64, error) {
	if err := s.authz.EnsureModule(ctx, "finance"); err != nil {
		return nil, 0, err
	}
	if err := s.authz.EnsureFeature(ctx, "finance.transactions.read"); err != nil {
		return nil, 0, err
	}
	if err := s.authz.EnsurePermission(ctx, "finance.transactions.read"); err != nil {
		return nil, 0, err
	}

	items, total, err := s.repo.List(ctx, domain.ListFilter{
		TenantID: in.TenantID,
		Type:     in.Type,
		DateFrom:   in.DateFrom,
		DateTo:     in.DateTo,
		Query:      in.Query,
		CategoryID: in.CategoryID,
		Page:       in.Page,
		PageSize:   in.PageSize,
	})
	if err != nil {
		return nil, 0, err
	}
	transactionIDs := make([]string, 0, len(items))
	for _, item := range items {
		transactionIDs = append(transactionIDs, item.ID)
	}
	idMap, nameMap, err := s.repo.ListSavingsLinks(ctx, in.TenantID, transactionIDs)
	if err != nil {
		return nil, 0, err
	}
	attachmentsMap, err := s.repo.ListAttachmentsByTransactionIDs(ctx, in.TenantID, transactionIDs)
	if err != nil {
		return nil, 0, err
	}
	for idx := range items {
		items[idx].SavingsIDs = idMap[items[idx].ID]
		items[idx].SavingsNames = nameMap[items[idx].ID]
		transactionItems, itemErr := s.repo.ListItems(ctx, in.TenantID, items[idx].ID)
		if itemErr != nil {
			return nil, 0, itemErr
		}
		items[idx].Items = transactionItems
		items[idx].Attachments = attachmentsMap[items[idx].ID]
	}
	return items, total, nil
}

func (s *Service) ListBySavingsID(ctx context.Context, tenantID, savingsID string) ([]domain.Transaction, error) {
	if err := s.authz.EnsureModule(ctx, "finance"); err != nil {
		return nil, err
	}
	if err := s.authz.EnsureFeature(ctx, "finance.transactions.read"); err != nil {
		return nil, err
	}
	if err := s.authz.EnsurePermission(ctx, "finance.transactions.read"); err != nil {
		return nil, err
	}

	items, err := s.repo.ListBySavingsID(ctx, tenantID, savingsID)
	if err != nil {
		return nil, err
	}
	transactionIDs := make([]string, 0, len(items))
	for _, item := range items {
		transactionIDs = append(transactionIDs, item.ID)
	}
	idMap, nameMap, err := s.repo.ListSavingsLinks(ctx, tenantID, transactionIDs)
	if err != nil {
		return nil, err
	}
	attachmentsMap, err := s.repo.ListAttachmentsByTransactionIDs(ctx, tenantID, transactionIDs)
	if err != nil {
		return nil, err
	}
	for idx := range items {
		items[idx].SavingsIDs = idMap[items[idx].ID]
		items[idx].SavingsNames = nameMap[items[idx].ID]
		transactionItems, itemErr := s.repo.ListItems(ctx, tenantID, items[idx].ID)
		if itemErr != nil {
			return nil, itemErr
		}
		items[idx].Items = transactionItems
		items[idx].Attachments = attachmentsMap[items[idx].ID]
	}
	return items, nil
}

type UpdateInput struct {
	TenantID             string
	ActorUserID          string
	TransactionID        string
	AccountID            string
	CategoryID           *string
	CategoryName         *string
	SavingsIDs           []string
	Type                 domain.TransactionType
	AmountMinor          int64
	Currency             string
	TransactionDate      time.Time
	Description          *string
	MerchantName         *string
	ReceiptNumber        *string
	PaymentMethod        *string
	SubtotalMinor        int64
	TaxMinor             int64
	ServiceChargeMinor   int64
	ReceiptDiscountMinor int64
	Items                []domain.TransactionItem
}

func (s *Service) Update(ctx context.Context, in UpdateInput) (domain.Transaction, error) {
	if err := s.authz.EnsureModule(ctx, "finance"); err != nil {
		return domain.Transaction{}, err
	}
	if err := s.authz.EnsureFeature(ctx, "finance.transactions.write"); err != nil {
		return domain.Transaction{}, err
	}
	if err := s.ensureAnyPermission(ctx, "finance.transactions.update", "finance.transactions.create"); err != nil {
		return domain.Transaction{}, err
	}
	if err := domain.ValidateCreateInput(in.AccountID, in.AmountMinor, in.Currency, in.Type); err != nil {
		return domain.Transaction{}, err
	}
	resolvedCategoryID, err := s.repo.ResolveCategoryID(ctx, in.TenantID, in.ActorUserID, in.CategoryID, in.CategoryName, in.Type)
	if err != nil {
		return domain.Transaction{}, err
	}
	if err := s.repo.ValidateReferences(ctx, in.TenantID, in.AccountID, resolvedCategoryID, in.Type); err != nil {
		return domain.Transaction{}, err
	}
	normalizedSavingsIDs := normalizeIDs(in.SavingsIDs)
	if in.Type != domain.TransactionTypeSavings {
		normalizedSavingsIDs = nil
	}
	if in.Type == domain.TransactionTypeSavings && len(normalizedSavingsIDs) == 0 {
		return domain.Transaction{}, domain.ErrInvalidSavingsSelection
	}
	if err := s.repo.ValidateSavingsGoals(ctx, in.TenantID, normalizedSavingsIDs); err != nil {
		return domain.Transaction{}, err
	}

	before, err := s.repo.GetByID(ctx, in.TenantID, in.TransactionID)
	if err != nil {
		return domain.Transaction{}, err
	}
	beforeItems, err := s.repo.ListItems(ctx, in.TenantID, in.TransactionID)
	if err != nil {
		return domain.Transaction{}, err
	}
	before.Items = beforeItems
	beforeSnapshot := before
	oldAllocations := map[string]int64{}
	if before.Type == domain.TransactionTypeSavings {
		oldAllocations, err = s.repo.ListSavingsAllocationsByTransaction(ctx, in.TenantID, in.TransactionID)
		if err != nil {
			return domain.Transaction{}, err
		}
	}

	before.AccountID = in.AccountID
	before.CategoryID = resolvedCategoryID
	before.Type = in.Type
	before.AmountMinor = in.AmountMinor
	before.Currency = in.Currency
	before.TransactionDate = in.TransactionDate
	merchantName, receiptNumber, paymentMethod, subtotalMinor, taxMinor, serviceChargeMinor, receiptDiscountMinor := sanitizeReceiptMeta(
		in.Type, in.MerchantName, in.ReceiptNumber, in.PaymentMethod, in.SubtotalMinor, in.TaxMinor, in.ServiceChargeMinor, in.ReceiptDiscountMinor,
	)
	// targeted sanitization for text fields
	safeDescription := in.Description
	if safeDescription != nil {
		s := security.SanitizeHTML(*safeDescription)
		safeDescription = &s
	}
	if merchantName != nil {
		s := security.SanitizeHTML(*merchantName)
		merchantName = &s
	}

	before.Description = safeDescription
	before.MerchantName = merchantName
	before.ReceiptNumber = receiptNumber
	before.PaymentMethod = paymentMethod
	before.SubtotalMinor = subtotalMinor
	before.TaxMinor = taxMinor
	before.ServiceChargeMinor = serviceChargeMinor
	before.ReceiptDiscountMinor = receiptDiscountMinor
	before.UpdatedBy = in.ActorUserID
	before.UpdatedAt = time.Now().UTC()

	updated, err := s.repo.Update(ctx, before)
	if err != nil {
		return domain.Transaction{}, err
	}
	if err := s.repo.ReplaceSavingsLinks(ctx, in.TenantID, in.TransactionID, in.ActorUserID, in.AmountMinor, normalizedSavingsIDs); err != nil {
		return domain.Transaction{}, err
	}
	newAllocations := map[string]int64{}
	if in.Type == domain.TransactionTypeSavings {
		newAllocations, err = s.repo.ListSavingsAllocationsByTransaction(ctx, in.TenantID, in.TransactionID)
		if err != nil {
			return domain.Transaction{}, err
		}
	}
	if err := s.repo.AdjustSavingsCurrentAmounts(ctx, in.TenantID, in.ActorUserID, diffSavingsAllocations(oldAllocations, newAllocations)); err != nil {
		return domain.Transaction{}, err
	}
	if err := s.reconcileSavings(
		ctx,
		in.TenantID,
		in.ActorUserID,
		unionSavingsIDs(normalizedSavingsIDs, mapKeys(oldAllocations), mapKeys(newAllocations)),
	); err != nil {
		return domain.Transaction{}, err
	}
	updated.SavingsIDs = normalizedSavingsIDs
	if in.Type == domain.TransactionTypeSavings {
		_, namesByTrx, err := s.repo.ListSavingsLinks(ctx, in.TenantID, []string{updated.ID})
		if err != nil {
			return domain.Transaction{}, err
		}
		updated.SavingsNames = namesByTrx[updated.ID]
	}
	items := normalizeTransactionItems(in.Items)
	if in.Type != domain.TransactionTypeExpense {
		items = nil
	}
	if err := s.repo.ReplaceItems(ctx, in.TenantID, updated.ID, in.ActorUserID, items); err != nil {
		return domain.Transaction{}, err
	}
	updated.Items = items

	if s.audit != nil {
		_ = s.audit.Write(ctx, "finance.transaction.update", "finance_transaction", updated.ID, beforeSnapshot, updated)
	}
	s.clearCache(ctx, in.TenantID)
	return updated, nil
}

func (s *Service) Delete(ctx context.Context, tenantID, actorUserID, transactionID string) error {
	if err := s.authz.EnsureModule(ctx, "finance"); err != nil {
		return err
	}
	if err := s.authz.EnsureFeature(ctx, "finance.transactions.write"); err != nil {
		return err
	}
	if err := s.ensureAnyPermission(ctx, "finance.transactions.delete", "finance.transactions.update"); err != nil {
		return err
	}

	existing, err := s.repo.GetByID(ctx, tenantID, transactionID)
	if err != nil {
		return err
	}
	oldAllocations := map[string]int64{}
	if existing.Type == domain.TransactionTypeSavings {
		oldAllocations, err = s.repo.ListSavingsAllocationsByTransaction(ctx, tenantID, transactionID)
		if err != nil {
			return err
		}
	}

	if err := s.repo.SoftDelete(ctx, tenantID, transactionID, actorUserID); err != nil {
		return err
	}
	if len(oldAllocations) > 0 {
		reversal := make(map[string]int64, len(oldAllocations))
		for savingsID, amount := range oldAllocations {
			reversal[savingsID] = -amount
		}
		if err := s.repo.AdjustSavingsCurrentAmounts(ctx, tenantID, actorUserID, reversal); err != nil {
			return err
		}
	}
	if err := s.repo.ReplaceSavingsLinks(ctx, tenantID, transactionID, actorUserID, 0, nil); err != nil {
		return err
	}
	if err := s.reconcileSavings(ctx, tenantID, actorUserID, mapKeys(oldAllocations)); err != nil {
		return err
	}
	if s.audit != nil {
		_ = s.audit.Write(ctx, "finance.transaction.delete", "finance_transaction", transactionID, nil, map[string]any{"deleted": true})
	}
	s.clearCache(ctx, tenantID)
	return nil
}

func normalizeIDs(raw []string) []string {
	unique := make(map[string]struct{})
	out := make([]string, 0, len(raw))
	for _, item := range raw {
		clean := strings.TrimSpace(item)
		if clean == "" {
			continue
		}
		if _, exists := unique[clean]; exists {
			continue
		}
		unique[clean] = struct{}{}
		out = append(out, clean)
	}
	return out
}

func diffSavingsAllocations(before map[string]int64, after map[string]int64) map[string]int64 {
	deltas := make(map[string]int64)
	for savingsID, amount := range before {
		deltas[savingsID] -= amount
	}
	for savingsID, amount := range after {
		deltas[savingsID] += amount
	}
	return deltas
}

func mapKeys(values map[string]int64) []string {
	out := make([]string, 0, len(values))
	for key := range values {
		out = append(out, key)
	}
	return out
}

func unionSavingsIDs(groups ...[]string) []string {
	unique := make(map[string]struct{})
	out := make([]string, 0)
	for _, ids := range groups {
		for _, id := range ids {
			clean := strings.TrimSpace(id)
			if clean == "" {
				continue
			}
			if _, exists := unique[clean]; exists {
				continue
			}
			unique[clean] = struct{}{}
			out = append(out, clean)
		}
	}
	return out
}

func normalizeTransactionItems(items []domain.TransactionItem) []domain.TransactionItem {
	out := make([]domain.TransactionItem, 0, len(items))
	for _, item := range items {
		name := strings.TrimSpace(item.ItemName)
		if name == "" {
			continue
		}
		qty := item.Quantity
		if qty <= 0 {
			qty = 1
		}
		price := item.PriceMinor
		if price <= 0 {
			price = 1
		}
		discount := item.DiscountMinor
		if discount < 0 {
			discount = 0
		}
		total := item.TotalMinor
		if total <= 0 {
			total = int64(qty*float64(price)) - discount
		}
		if total < 0 {
			total = 0
		}
		if total == 0 {
			total = 1
		}
		out = append(out, domain.TransactionItem{
			ID:            item.ID,
			ItemName:      name,
			Quantity:      qty,
			PriceMinor:    price,
			DiscountMinor: discount,
			TotalMinor:    total,
			Notes:         item.Notes,
		})
	}
	return out
}

func (s *Service) reconcileSavings(ctx context.Context, tenantID, actorUserID string, savingsIDs []string) error {
	reconciler, ok := s.repo.(savingsReconciler)
	if !ok {
		return nil
	}
	normalized := normalizeIDs(savingsIDs)
	if len(normalized) == 0 {
		return nil
	}
	return reconciler.ReconcileSavingsCurrentAmounts(ctx, tenantID, actorUserID, normalized)
}

func (s *Service) ensureAnyPermission(ctx context.Context, permissionCodes ...string) error {
	var deniedErr error
	for _, permissionCode := range permissionCodes {
		if strings.TrimSpace(permissionCode) == "" {
			continue
		}
		err := s.authz.EnsurePermission(ctx, permissionCode)
		if err == nil {
			return nil
		}
		if errors.Is(err, access.ErrPermissionDenied) {
			deniedErr = err
			continue
		}
		return err
	}
	if deniedErr != nil {
		return deniedErr
	}
	return access.ErrPermissionDenied
}

func sanitizeReceiptMeta(trxType domain.TransactionType, merchantName, receiptNumber, paymentMethod *string, subtotalMinor, taxMinor, serviceChargeMinor, receiptDiscountMinor int64) (*string, *string, *string, int64, int64, int64, int64) {
	if trxType != domain.TransactionTypeExpense {
		return nil, nil, nil, 0, 0, 0, 0
	}
	return trimOptionalString(merchantName), trimOptionalString(receiptNumber), trimOptionalString(paymentMethod), maxInt64(subtotalMinor, 0), maxInt64(taxMinor, 0), maxInt64(serviceChargeMinor, 0), maxInt64(receiptDiscountMinor, 0)
}

func trimOptionalString(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func maxInt64(value, min int64) int64 {
	if value < min {
		return min
	}
	return value
}
