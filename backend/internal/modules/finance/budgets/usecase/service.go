package usecase

import (
	"context"
	"errors"
	"strings"
	"time"

	"pekan/backend/internal/modules/finance/budgets/domain"
	"pekan/backend/internal/platform/access"
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
	repo  domain.Repository
	authz Authorizer
	audit AuditLogger
}

func NewService(repo domain.Repository, authz Authorizer, audit AuditLogger) *Service {
	return &Service{
		repo:  repo,
		authz: authz,
		audit: audit,
	}
}

type CreateInput struct {
	TenantID          string
	ActorUserID       string
	Name              string
	CategoryID        *string
	CategoryName      *string
	AmountLimitMinor  int64
	Currency          string
	Period            string
	StartDate         time.Time
	EndDate           *time.Time
	AlertThresholdPct *int
	Notes             *string
	Status            string
}

func (s *Service) Create(ctx context.Context, in CreateInput) (domain.Budget, error) {
	if err := s.authz.EnsureModule(ctx, "finance.budgets"); err != nil {
		return domain.Budget{}, err
	}
	if err := s.authz.EnsureFeature(ctx, "finance.budgets.write"); err != nil {
		return domain.Budget{}, err
	}
	if err := s.ensureAnyPermission(ctx, "finance.budgets.create", "finance.budgets.update"); err != nil {
		return domain.Budget{}, err
	}
	if err := validateInput(in.Name, in.AmountLimitMinor, in.Currency, in.Period, in.StartDate, in.EndDate, in.AlertThresholdPct, in.Status); err != nil {
		return domain.Budget{}, err
	}
	resolvedCategoryID, err := s.repo.ResolveCategoryID(ctx, in.TenantID, in.ActorUserID, in.CategoryID, in.CategoryName)
	if err != nil {
		return domain.Budget{}, err
	}

	now := time.Now().UTC()
	status := normalizeStatus(in.Status)
	out, err := s.repo.Create(ctx, domain.Budget{
		TenantID:          in.TenantID,
		Name:              strings.TrimSpace(in.Name),
		CategoryID:        resolvedCategoryID,
		AmountLimitMinor:  in.AmountLimitMinor,
		Currency:          strings.ToUpper(strings.TrimSpace(in.Currency)),
		Period:            strings.ToLower(strings.TrimSpace(in.Period)),
		StartDate:         in.StartDate,
		EndDate:           in.EndDate,
		AlertThresholdPct: in.AlertThresholdPct,
		Notes:             in.Notes,
		Status:            status,
		CreatedBy:         in.ActorUserID,
		UpdatedBy:         in.ActorUserID,
		CreatedAt:         now,
		UpdatedAt:         now,
	})
	if err != nil {
		return domain.Budget{}, err
	}

	if s.audit != nil {
		_ = s.audit.Write(ctx, "finance.budget.create", "finance_budget", out.ID, nil, out)
	}
	return out, nil
}

func (s *Service) GetByID(ctx context.Context, tenantID, budgetID string) (domain.Budget, error) {
	if err := s.authz.EnsureModule(ctx, "finance.budgets"); err != nil {
		return domain.Budget{}, err
	}
	if err := s.authz.EnsureFeature(ctx, "finance.budgets.read"); err != nil {
		return domain.Budget{}, err
	}
	if err := s.authz.EnsurePermission(ctx, "finance.budgets.read"); err != nil {
		return domain.Budget{}, err
	}
	return s.repo.GetByID(ctx, tenantID, budgetID)
}

type ListInput struct {
	TenantID string
	Status   *string
	Page     int
	PageSize int
}

func (s *Service) List(ctx context.Context, in ListInput) ([]domain.Budget, int64, error) {
	if err := s.authz.EnsureModule(ctx, "finance.budgets"); err != nil {
		return nil, 0, err
	}
	if err := s.authz.EnsureFeature(ctx, "finance.budgets.read"); err != nil {
		return nil, 0, err
	}
	if err := s.authz.EnsurePermission(ctx, "finance.budgets.read"); err != nil {
		return nil, 0, err
	}
	return s.repo.List(ctx, domain.ListFilter{
		TenantID: in.TenantID,
		Status:   in.Status,
		Page:     in.Page,
		PageSize: in.PageSize,
	})
}

type UpdateInput struct {
	TenantID          string
	ActorUserID       string
	BudgetID          string
	Name              string
	CategoryID        *string
	CategoryName      *string
	AmountLimitMinor  int64
	Currency          string
	Period            string
	StartDate         time.Time
	EndDate           *time.Time
	AlertThresholdPct *int
	Notes             *string
	Status            string
}

func (s *Service) Update(ctx context.Context, in UpdateInput) (domain.Budget, error) {
	if err := s.authz.EnsureModule(ctx, "finance.budgets"); err != nil {
		return domain.Budget{}, err
	}
	if err := s.authz.EnsureFeature(ctx, "finance.budgets.write"); err != nil {
		return domain.Budget{}, err
	}
	if err := s.ensureAnyPermission(ctx, "finance.budgets.update", "finance.budgets.create"); err != nil {
		return domain.Budget{}, err
	}
	if err := validateInput(in.Name, in.AmountLimitMinor, in.Currency, in.Period, in.StartDate, in.EndDate, in.AlertThresholdPct, in.Status); err != nil {
		return domain.Budget{}, err
	}
	resolvedCategoryID, err := s.repo.ResolveCategoryID(ctx, in.TenantID, in.ActorUserID, in.CategoryID, in.CategoryName)
	if err != nil {
		return domain.Budget{}, err
	}

	before, err := s.repo.GetByID(ctx, in.TenantID, in.BudgetID)
	if err != nil {
		return domain.Budget{}, err
	}
	snapshot := before

	before.Name = strings.TrimSpace(in.Name)
	before.CategoryID = resolvedCategoryID
	before.AmountLimitMinor = in.AmountLimitMinor
	before.Currency = strings.ToUpper(strings.TrimSpace(in.Currency))
	before.Period = strings.ToLower(strings.TrimSpace(in.Period))
	before.StartDate = in.StartDate
	before.EndDate = in.EndDate
	before.AlertThresholdPct = in.AlertThresholdPct
	before.Notes = in.Notes
	before.Status = normalizeStatus(in.Status)
	before.UpdatedBy = in.ActorUserID
	before.UpdatedAt = time.Now().UTC()

	updated, err := s.repo.Update(ctx, before)
	if err != nil {
		return domain.Budget{}, err
	}
	if s.audit != nil {
		_ = s.audit.Write(ctx, "finance.budget.update", "finance_budget", updated.ID, snapshot, updated)
	}
	return updated, nil
}

func (s *Service) Delete(ctx context.Context, tenantID, actorUserID, budgetID string) error {
	if err := s.authz.EnsureModule(ctx, "finance.budgets"); err != nil {
		return err
	}
	if err := s.authz.EnsureFeature(ctx, "finance.budgets.write"); err != nil {
		return err
	}
	if err := s.ensureAnyPermission(ctx, "finance.budgets.delete", "finance.budgets.update"); err != nil {
		return err
	}
	existing, err := s.repo.GetByID(ctx, tenantID, budgetID)
	if err != nil {
		return err
	}
	if err := s.repo.SoftDelete(ctx, tenantID, budgetID, actorUserID); err != nil {
		return err
	}
	if s.audit != nil {
		_ = s.audit.Write(ctx, "finance.budget.delete", "finance_budget", existing.ID, existing, map[string]any{"deleted": true, "ida": existing.IDA, "name": existing.Name})
	}
	return nil
}

func (s *Service) CheckAlerts(ctx context.Context, tenantID, categoryID string) error {
	budgets, err := s.repo.FindActiveByCategory(ctx, tenantID, categoryID)
	if err != nil {
		return err
	}

	for _, b := range budgets {
		threshold := 80.0
		if b.AlertThresholdPct != nil {
			threshold = float64(*b.AlertThresholdPct)
		}

		if b.ProgressPercent >= threshold {
			action := "finance.budget.alert"
			if b.ProgressPercent >= 100 {
				action = "finance.budget.exceeded"
			}
			_ = s.audit.Write(ctx, action, "finance_budget", b.ID, nil, map[string]any{
				"budget_name":      b.Name,
				"budget_ida":       b.IDA,
				"limit_minor":      b.AmountLimitMinor,
				"spent_minor":      b.SpentAmountMinor,
				"progress_percent": b.ProgressPercent,
				"threshold":        threshold,
			})
		}
	}
	return nil
}

func validateInput(name string, amount int64, currency, period string, startDate time.Time, endDate *time.Time, alert *int, status string) error {
	if strings.TrimSpace(name) == "" {
		return domain.ErrInvalidName
	}
	if len(name) > 100 {
		return domain.ErrInputTooLong
	}
	if amount <= 0 {
		return domain.ErrInvalidAmount
	}
	if len(strings.TrimSpace(currency)) != 3 {
		return domain.ErrInvalidCurrency
	}
	if !isAllowedPeriod(period) {
		return domain.ErrInvalidPeriod
	}
	if endDate != nil && endDate.Before(startDate) {
		return domain.ErrInvalidDateRange
	}
	if alert != nil && (*alert <= 0 || *alert > 100) {
		return domain.ErrInvalidAlert
	}
	if status != "" && !isAllowedStatus(status) {
		return domain.ErrInvalidStatus
	}
	return nil
}

func normalizeStatus(status string) string {
	if strings.TrimSpace(status) == "" {
		return "active"
	}
	return strings.ToLower(strings.TrimSpace(status))
}

func isAllowedStatus(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "active", "paused", "ended":
		return true
	default:
		return false
	}
}

func isAllowedPeriod(period string) bool {
	switch strings.ToLower(strings.TrimSpace(period)) {
	case "monthly", "weekly", "yearly", "custom":
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
