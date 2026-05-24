package usecase

import (
	"context"
	"errors"
	"strings"
	"time"

	"io"
	"path/filepath"
	"github.com/google/uuid"

	"pekan/backend/internal/modules/finance/reminders/domain"
	"pekan/backend/internal/platform/access"
	"pekan/backend/internal/platform/audit"
	"pekan/backend/internal/platform/storage"
)

type Authorizer interface {
	EnsureModule(ctx context.Context, module string) error
	EnsureFeature(ctx context.Context, feature string) error
	EnsurePermission(ctx context.Context, permission string) error
}

type AuditLogger interface {
	Write(ctx context.Context, action, resourceType, resourceID string, before, after any) error
}

type Service struct {
	repo    domain.Repository
	authz   Authorizer
	audit   AuditLogger
	storage storage.ObjectStorage
}

func NewService(repo domain.Repository, authz Authorizer, audit AuditLogger, st storage.ObjectStorage) *Service {
	return &Service{
		repo:    repo,
		authz:   authz,
		audit:   audit,
		storage: st,
	}
}

type CreateInput struct {
	TenantID       string
	ActorUserID    string
	Title          string
	Description    *string
	AmountMinor    *int64
	Currency       *string
	DueDate        time.Time
	RepeatInterval string
	Status         string
	TotalTenor     *int
	CurrentTenor   *int
}

func (s *Service) Create(ctx context.Context, in CreateInput) (domain.Reminder, error) {
	if err := s.authz.EnsureModule(ctx, "finance.reminders"); err != nil {
		return domain.Reminder{}, err
	}
	if err := s.authz.EnsureFeature(ctx, "finance.reminders.write"); err != nil {
		return domain.Reminder{}, err
	}
	if err := s.ensureAnyPermission(ctx, "finance.reminders.create", "finance.reminders.update"); err != nil {
		return domain.Reminder{}, err
	}
	if err := validateInput(in.Title, in.AmountMinor, in.Currency, in.DueDate, in.RepeatInterval, in.Status); err != nil {
		return domain.Reminder{}, err
	}

	now := time.Now().UTC()
	status := normalizeStatus(in.Status)
	repeat := normalizeRepeat(in.RepeatInterval)

	out, err := s.repo.Create(ctx, domain.Reminder{
		TenantID:       in.TenantID,
		Title:          strings.TrimSpace(in.Title),
		Description:    in.Description,
		AmountMinor:    in.AmountMinor,
		Currency:       normalizeCurrencyPtr(in.Currency),
		DueDate:        in.DueDate,
		RepeatInterval: repeat,
		Status:         status,
		TotalTenor:     in.TotalTenor,
		CurrentTenor:   in.CurrentTenor,
		CreatedBy:      in.ActorUserID,
		UpdatedBy:      in.ActorUserID,
		CreatedAt:      now,
		UpdatedAt:      now,
	})
	if err != nil {
		return domain.Reminder{}, err
	}

	if s.audit != nil {
		go func(data domain.Reminder) {
			defer func() { recover() }()
			auditCtx := audit.WithContext(context.Background(), audit.AuditContext{
				TenantID:    data.TenantID,
				ActorUserID: data.CreatedBy,
			})
			_ = s.audit.Write(auditCtx, "finance.reminder.create", "finance_reminder", data.ID, nil, data)
		}(out)
	}
	return out, nil
}

func (s *Service) GetByID(ctx context.Context, tenantID, reminderID string) (domain.Reminder, error) {
	if err := s.authz.EnsureModule(ctx, "finance.reminders"); err != nil {
		return domain.Reminder{}, err
	}
	if err := s.authz.EnsureFeature(ctx, "finance.reminders.read"); err != nil {
		return domain.Reminder{}, err
	}
	if err := s.authz.EnsurePermission(ctx, "finance.reminders.read"); err != nil {
		return domain.Reminder{}, err
	}
	return s.repo.GetByID(ctx, tenantID, reminderID)
}

type ListInput struct {
	TenantID string
	Status   *string
	DateFrom *string
	DateTo   *string
	Page     int
	PageSize int
}

func (s *Service) List(ctx context.Context, in ListInput) ([]domain.Reminder, int64, error) {
	if err := s.authz.EnsureModule(ctx, "finance.reminders"); err != nil {
		return nil, 0, err
	}
	if err := s.authz.EnsureFeature(ctx, "finance.reminders.read"); err != nil {
		return nil, 0, err
	}
	if err := s.authz.EnsurePermission(ctx, "finance.reminders.read"); err != nil {
		return nil, 0, err
	}
	return s.repo.List(ctx, domain.ListFilter{
		TenantID: in.TenantID,
		Status:   in.Status,
		DateFrom: in.DateFrom,
		DateTo:   in.DateTo,
		Page:     in.Page,
		PageSize: in.PageSize,
	})
}

func (s *Service) ListDue(ctx context.Context, tenantID string) ([]domain.Reminder, error) {
	if err := s.authz.EnsureModule(ctx, "finance.reminders"); err != nil {
		return nil, err
	}
	if err := s.authz.EnsureFeature(ctx, "finance.reminders.read"); err != nil {
		return nil, err
	}
	if err := s.authz.EnsurePermission(ctx, "finance.reminders.read"); err != nil {
		return nil, err
	}
	return s.repo.ListDue(ctx, tenantID)
}

type UpdateInput struct {
	TenantID       string
	ActorUserID    string
	ReminderID     string
	Title          string
	Description    *string
	AmountMinor    *int64
	Currency       *string
	DueDate        time.Time
	RepeatInterval string
	Status         string
	TotalTenor     *int
	CurrentTenor   *int
}

func (s *Service) Update(ctx context.Context, in UpdateInput) (domain.Reminder, error) {
	if err := s.authz.EnsureModule(ctx, "finance.reminders"); err != nil {
		return domain.Reminder{}, err
	}
	if err := s.authz.EnsureFeature(ctx, "finance.reminders.write"); err != nil {
		return domain.Reminder{}, err
	}
	if err := s.ensureAnyPermission(ctx, "finance.reminders.update", "finance.reminders.create"); err != nil {
		return domain.Reminder{}, err
	}
	if err := validateInput(in.Title, in.AmountMinor, in.Currency, in.DueDate, in.RepeatInterval, in.Status); err != nil {
		return domain.Reminder{}, err
	}

	before, err := s.repo.GetByID(ctx, in.TenantID, in.ReminderID)
	if err != nil {
		return domain.Reminder{}, err
	}
	snapshot := before

	before.Title = strings.TrimSpace(in.Title)
	before.Description = in.Description
	before.AmountMinor = in.AmountMinor
	before.Currency = normalizeCurrencyPtr(in.Currency)
	before.DueDate = in.DueDate
	before.RepeatInterval = normalizeRepeat(in.RepeatInterval)
	before.Status = normalizeStatus(in.Status)
	before.TotalTenor = in.TotalTenor
	before.CurrentTenor = in.CurrentTenor
	before.UpdatedBy = in.ActorUserID
	before.UpdatedAt = time.Now().UTC()

	updated, err := s.repo.Update(ctx, before)
	if err != nil {
		return domain.Reminder{}, err
	}
	if s.audit != nil {
		go func(beforeData, afterData domain.Reminder) {
			defer func() { recover() }()
			auditCtx := audit.WithContext(context.Background(), audit.AuditContext{
				TenantID:    afterData.TenantID,
				ActorUserID: afterData.UpdatedBy,
			})
			_ = s.audit.Write(auditCtx, "finance.reminder.update", "finance_reminder", afterData.ID, beforeData, afterData)
		}(snapshot, updated)
	}
	return updated, nil
}

func (s *Service) MarkStatus(ctx context.Context, tenantID, actorUserID, reminderID, status string) (domain.Reminder, error) {
	if err := s.authz.EnsureModule(ctx, "finance.reminders"); err != nil {
		return domain.Reminder{}, err
	}
	if err := s.authz.EnsureFeature(ctx, "finance.reminders.write"); err != nil {
		return domain.Reminder{}, err
	}
	if err := s.ensureAnyPermission(ctx, "finance.reminders.mark", "finance.reminders.update"); err != nil {
		return domain.Reminder{}, err
	}
	if status != "" && !isAllowedStatus(status) {
		return domain.Reminder{}, domain.ErrInvalidStatus
	}

	updated, err := s.repo.UpdateStatus(ctx, tenantID, reminderID, normalizeStatus(status), actorUserID)
	if err != nil {
		return domain.Reminder{}, err
	}
	if s.audit != nil {
		go func(id, stat, tID, aID string) {
			defer func() { recover() }()
			auditCtx := audit.WithContext(context.Background(), audit.AuditContext{
				TenantID:    tID,
				ActorUserID: aID,
			})
			_ = s.audit.Write(auditCtx, "finance.reminder.mark", "finance_reminder", id, nil, map[string]any{"status": stat})
		}(reminderID, updated.Status, tenantID, actorUserID)
	}
	return updated, nil
}

func (s *Service) Delete(ctx context.Context, tenantID, actorUserID, reminderID string) error {
	if err := s.authz.EnsureModule(ctx, "finance.reminders"); err != nil {
		return err
	}
	if err := s.authz.EnsureFeature(ctx, "finance.reminders.write"); err != nil {
		return err
	}
	if err := s.ensureAnyPermission(ctx, "finance.reminders.delete", "finance.reminders.update"); err != nil {
		return err
	}
	if err := s.repo.SoftDelete(ctx, tenantID, reminderID, actorUserID); err != nil {
		return err
	}
	if s.audit != nil {
		go func(id, tID, aID string) {
			defer func() { recover() }()
			auditCtx := audit.WithContext(context.Background(), audit.AuditContext{
				TenantID:    tID,
				ActorUserID: aID,
			})
			_ = s.audit.Write(auditCtx, "finance.reminder.delete", "finance_reminder", id, nil, map[string]any{"deleted": true})
		}(reminderID, tenantID, actorUserID)
	}
	return nil
}

func validateInput(title string, amount *int64, currency *string, dueDate time.Time, repeat, status string) error {
	if strings.TrimSpace(title) == "" {
		return domain.ErrInvalidTitle
	}
	if dueDate.IsZero() {
		return domain.ErrInvalidDate
	}
	if amount != nil && *amount < 0 {
		return domain.ErrInvalidAmount
	}
	if amount != nil && currency == nil {
		return domain.ErrInvalidCurrency
	}
	if currency != nil && len(strings.TrimSpace(*currency)) != 3 {
		return domain.ErrInvalidCurrency
	}
	if repeat != "" && !isAllowedRepeat(repeat) {
		return domain.ErrInvalidRepeat
	}
	if status != "" && !isAllowedStatus(status) {
		return domain.ErrInvalidStatus
	}
	return nil
}

func normalizeRepeat(v string) string {
	if strings.TrimSpace(v) == "" {
		return "none"
	}
	return strings.ToLower(strings.TrimSpace(v))
}

func normalizeStatus(v string) string {
	if strings.TrimSpace(v) == "" {
		return "pending"
	}
	return strings.ToLower(strings.TrimSpace(v))
}

func normalizeCurrencyPtr(v *string) *string {
	if v == nil {
		return nil
	}
	c := strings.ToUpper(strings.TrimSpace(*v))
	return &c
}

func isAllowedRepeat(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "none", "daily", "weekly", "monthly":
		return true
	default:
		return false
	}
}

func isAllowedStatus(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "pending", "paid", "cancelled":
		return true
	default:
		return false
	}
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

type AddPaymentInput struct {
	TenantID          string
	ActorUserID       string
	ReminderID        string
	PaidAt            time.Time
	AmountMinor       int64
	Status            string
	Notes             *string
	ProofImageContent io.Reader
	ProofImageName    string
	ProofImageMime    string
}

func (s *Service) AddPayment(ctx context.Context, in AddPaymentInput) (domain.ReminderPayment, error) {
	if err := s.authz.EnsureModule(ctx, "finance.reminders"); err != nil {
		return domain.ReminderPayment{}, err
	}
	if err := s.authz.EnsureFeature(ctx, "finance.reminders.write"); err != nil {
		return domain.ReminderPayment{}, err
	}
	if err := s.ensureAnyPermission(ctx, "finance.reminders.mark", "finance.reminders.update"); err != nil {
		return domain.ReminderPayment{}, err
	}

	var proofURL *string
	if in.ProofImageContent != nil && in.ProofImageName != "" {
		ext := filepath.Ext(in.ProofImageName)
		if ext == "" {
			ext = ".jpg"
		}
		key := "reminders/proofs/" + uuid.NewString() + ext
		
		outPut, err := s.storage.Put(ctx, storage.PutObjectInput{
			TenantID:    in.TenantID,
			Module:      "finance.reminders",
			ObjectKey:   key,
			ContentType: in.ProofImageMime,
			Body:        in.ProofImageContent,
		})
		if err != nil {
			return domain.ReminderPayment{}, err
		}
		urlStr := outPut.ObjectKey
		proofURL = &urlStr
	}

	now := time.Now().UTC()
	payment, err := s.repo.CreatePayment(ctx, domain.ReminderPayment{
		TenantID:      in.TenantID,
		ReminderID:    in.ReminderID,
		PaidAt:        in.PaidAt,
		AmountMinor:   in.AmountMinor,
		Status:        normalizeStatus(in.Status),
		Notes:         in.Notes,
		ProofImageURL: proofURL,
		CreatedBy:     in.ActorUserID,
		UpdatedBy:     in.ActorUserID,
		CreatedAt:     now,
		UpdatedAt:     now,
		TransientProofName: in.ProofImageName,
		TransientProofMime: in.ProofImageMime,
	})
	if err != nil {
		return domain.ReminderPayment{}, err
	}

	if s.audit != nil {
		go func(data domain.ReminderPayment) {
			defer func() { recover() }()
			auditCtx := audit.WithContext(context.Background(), audit.AuditContext{
				TenantID:    data.TenantID,
				ActorUserID: data.CreatedBy,
			})
			_ = s.audit.Write(auditCtx, "finance.reminder.payment.add", "finance_reminder_payment", data.ID, nil, data)
		}(payment)
	}
	return payment, nil
}

func (s *Service) GetPaymentHistory(ctx context.Context, tenantID, reminderID string) ([]domain.ReminderPayment, error) {
	if err := s.authz.EnsureModule(ctx, "finance.reminders"); err != nil {
		return nil, err
	}
	if err := s.authz.EnsureFeature(ctx, "finance.reminders.read"); err != nil {
		return nil, err
	}
	if err := s.authz.EnsurePermission(ctx, "finance.reminders.read"); err != nil {
		return nil, err
	}
	return s.repo.ListPayments(ctx, tenantID, reminderID)
}

func (s *Service) GetProofImage(ctx context.Context, tenantID, reminderID, paymentID string) ([]byte, string, error) {
	if err := s.authz.EnsureModule(ctx, "finance.reminders"); err != nil {
		return nil, "", err
	}
	if err := s.authz.EnsureFeature(ctx, "finance.reminders.read"); err != nil {
		return nil, "", err
	}
	
	payments, err := s.repo.ListPayments(ctx, tenantID, reminderID)
	if err != nil {
		return nil, "", err
	}
	
	var payment domain.ReminderPayment
	found := false
	for _, p := range payments {
		if p.ID == paymentID {
			payment = p
			found = true
			break
		}
	}
	
	if !found || payment.ProofImageURL == nil || *payment.ProofImageURL == "" {
		return nil, "", errors.New("proof image not found")
	}
	
	rc, err := s.storage.Open(ctx, storage.GetObjectInput{ObjectKey: *payment.ProofImageURL})
	if err != nil {
		return nil, "", err
	}
	defer rc.Close()
	
	data, err := io.ReadAll(rc)
	if err != nil {
		return nil, "", err
	}
	
	mime := "image/jpeg"
	lowerURL := strings.ToLower(*payment.ProofImageURL)
	if strings.HasSuffix(lowerURL, ".png") {
		mime = "image/png"
	} else if strings.HasSuffix(lowerURL, ".webp") {
		mime = "image/webp"
	}
	
	return data, mime, nil
}

type UpdatePaymentInput struct {
	TenantID          string
	ActorUserID       string
	ReminderID        string
	PaymentID         string
	PaidAt            time.Time
	AmountMinor       int64
	Status            string
	Notes             *string
	ProofImageContent io.Reader
	ProofImageName    string
	ProofImageMime    string
}

func (s *Service) UpdatePayment(ctx context.Context, in UpdatePaymentInput) (domain.ReminderPayment, error) {
	if err := s.authz.EnsureModule(ctx, "finance.reminders"); err != nil {
		return domain.ReminderPayment{}, err
	}
	if err := s.authz.EnsureFeature(ctx, "finance.reminders.write"); err != nil {
		return domain.ReminderPayment{}, err
	}
	if err := s.ensureAnyPermission(ctx, "finance.reminders.mark", "finance.reminders.update"); err != nil {
		return domain.ReminderPayment{}, err
	}

	var proofURL *string
	if in.ProofImageContent != nil && in.ProofImageName != "" {
		ext := filepath.Ext(in.ProofImageName)
		if ext == "" {
			ext = ".jpg"
		}
		key := "reminders/proofs/" + uuid.NewString() + ext
		
		outPut, err := s.storage.Put(ctx, storage.PutObjectInput{
			TenantID:    in.TenantID,
			Module:      "finance.reminders",
			ObjectKey:   key,
			ContentType: in.ProofImageMime,
			Body:        in.ProofImageContent,
		})
		if err != nil {
			return domain.ReminderPayment{}, err
		}
		
		urlStr := outPut.ObjectKey
		proofURL = &urlStr
	}

	updateData := domain.ReminderPayment{
		ID:            in.PaymentID,
		TenantID:      in.TenantID,
		ReminderID:    in.ReminderID,
		PaidAt:        in.PaidAt,
		AmountMinor:   in.AmountMinor,
		Status:        normalizeStatus(in.Status),
		Notes:         in.Notes,
		UpdatedBy:     in.ActorUserID,
	}

	// Only update the proof URL if a new image was actually uploaded
	if proofURL != nil {
		updateData.ProofImageURL = proofURL
	}

	payment, err := s.repo.UpdatePayment(ctx, updateData)
	if err != nil {
		return domain.ReminderPayment{}, err
	}

	if s.audit != nil {
		go func(data domain.ReminderPayment) {
			defer func() { recover() }()
			auditCtx := audit.WithContext(context.Background(), audit.AuditContext{
				TenantID:    data.TenantID,
				ActorUserID: data.UpdatedBy,
			})
			_ = s.audit.Write(auditCtx, "finance.reminder.payment.update", "finance_reminder_payment", data.ID, nil, data)
		}(payment)
	}
	return payment, nil
}

func (s *Service) DeletePayment(ctx context.Context, tenantID, actorUserID, reminderID, paymentID string) error {
	if err := s.authz.EnsureModule(ctx, "finance.reminders"); err != nil {
		return err
	}
	if err := s.authz.EnsureFeature(ctx, "finance.reminders.write"); err != nil {
		return err
	}
	if err := s.ensureAnyPermission(ctx, "finance.reminders.mark", "finance.reminders.update"); err != nil {
		return err
	}

	if err := s.repo.DeletePayment(ctx, tenantID, reminderID, paymentID, actorUserID); err != nil {
		return err
	}

	if s.audit != nil {
		go func(id, tID, aID string) {
			defer func() { recover() }()
			auditCtx := audit.WithContext(context.Background(), audit.AuditContext{
				TenantID:    tID,
				ActorUserID: aID,
			})
			_ = s.audit.Write(auditCtx, "finance.reminder.payment.delete", "finance_reminder_payment", id, nil, map[string]any{"deleted": true})
		}(paymentID, tenantID, actorUserID)
	}
	return nil
}
