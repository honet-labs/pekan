package usecase

import (
	"context"
	"errors"
	"strings"
	"time"

	"pekan/backend/internal/modules/finance/savings/domain"
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
	TenantID           string
	ActorUserID        string
	Name               string
	TargetAmountMinor  int64
	CurrentAmountMinor int64
	Currency           string
	StartDate          *time.Time
	TargetDate         *time.Time
	Notes              *string
	Status             string
}

func (s *Service) Create(ctx context.Context, in CreateInput) (domain.Savings, error) {
	if err := s.authz.EnsureModule(ctx, "finance.savings"); err != nil {
		return domain.Savings{}, err
	}
	if err := s.authz.EnsureFeature(ctx, "finance.savings.write"); err != nil {
		return domain.Savings{}, err
	}
	if err := s.ensureAnyPermission(ctx, "finance.savings.create", "finance.savings.update"); err != nil {
		return domain.Savings{}, err
	}
	if err := validateInput(in.Name, in.TargetAmountMinor, in.CurrentAmountMinor, in.Currency, in.Status); err != nil {
		return domain.Savings{}, err
	}

	now := time.Now().UTC()
	status := normalizeStatus(in.Status)
	out, err := s.repo.Create(ctx, domain.Savings{
		TenantID:           in.TenantID,
		Name:               strings.TrimSpace(in.Name),
		TargetAmountMinor:  in.TargetAmountMinor,
		CurrentAmountMinor: in.CurrentAmountMinor,
		Currency:           strings.ToUpper(strings.TrimSpace(in.Currency)),
		StartDate:          in.StartDate,
		TargetDate:         in.TargetDate,
		Notes:              in.Notes,
		Status:             status,
		CreatedBy:          in.ActorUserID,
		UpdatedBy:          in.ActorUserID,
		CreatedAt:          now,
		UpdatedAt:          now,
	})
	if err != nil {
		return domain.Savings{}, err
	}

	if s.audit != nil {
		_ = s.audit.Write(ctx, "finance.savings.create", "finance_savings", out.ID, nil, out)
	}
	return out, nil
}

func (s *Service) GetByID(ctx context.Context, tenantID, savingsID string) (domain.Savings, error) {
	if err := s.authz.EnsureModule(ctx, "finance.savings"); err != nil {
		return domain.Savings{}, err
	}
	if err := s.authz.EnsureFeature(ctx, "finance.savings.read"); err != nil {
		return domain.Savings{}, err
	}
	if err := s.authz.EnsurePermission(ctx, "finance.savings.read"); err != nil {
		return domain.Savings{}, err
	}
	return s.repo.GetByID(ctx, tenantID, savingsID)
}

type ListInput struct {
	TenantID string
	Status   *string
	Page     int
	PageSize int
}

func (s *Service) List(ctx context.Context, in ListInput) ([]domain.Savings, int64, error) {
	if err := s.authz.EnsureModule(ctx, "finance.savings"); err != nil {
		return nil, 0, err
	}
	if err := s.authz.EnsureFeature(ctx, "finance.savings.read"); err != nil {
		return nil, 0, err
	}
	if err := s.authz.EnsurePermission(ctx, "finance.savings.read"); err != nil {
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
	TenantID           string
	ActorUserID        string
	SavingsID          string
	Name               string
	TargetAmountMinor  int64
	CurrentAmountMinor int64
	Currency           string
	StartDate          *time.Time
	TargetDate         *time.Time
	Notes              *string
	Status             string
}

func (s *Service) Update(ctx context.Context, in UpdateInput) (domain.Savings, error) {
	if err := s.authz.EnsureModule(ctx, "finance.savings"); err != nil {
		return domain.Savings{}, err
	}
	if err := s.authz.EnsureFeature(ctx, "finance.savings.write"); err != nil {
		return domain.Savings{}, err
	}
	if err := s.ensureAnyPermission(ctx, "finance.savings.update", "finance.savings.create"); err != nil {
		return domain.Savings{}, err
	}
	if err := validateInput(in.Name, in.TargetAmountMinor, in.CurrentAmountMinor, in.Currency, in.Status); err != nil {
		return domain.Savings{}, err
	}

	before, err := s.repo.GetByID(ctx, in.TenantID, in.SavingsID)
	if err != nil {
		return domain.Savings{}, err
	}
	snapshot := before

	before.Name = strings.TrimSpace(in.Name)
	before.TargetAmountMinor = in.TargetAmountMinor
	before.CurrentAmountMinor = in.CurrentAmountMinor
	before.Currency = strings.ToUpper(strings.TrimSpace(in.Currency))
	before.StartDate = in.StartDate
	before.TargetDate = in.TargetDate
	before.Notes = in.Notes
	before.Status = normalizeStatus(in.Status)
	before.UpdatedBy = in.ActorUserID
	before.UpdatedAt = time.Now().UTC()

	updated, err := s.repo.Update(ctx, before)
	if err != nil {
		return domain.Savings{}, err
	}

	if s.audit != nil {
		_ = s.audit.Write(ctx, "finance.savings.update", "finance_savings", updated.ID, snapshot, updated)
	}
	return updated, nil
}

func (s *Service) Delete(ctx context.Context, tenantID, actorUserID, savingsID string) error {
	if err := s.authz.EnsureModule(ctx, "finance.savings"); err != nil {
		return err
	}
	if err := s.authz.EnsureFeature(ctx, "finance.savings.write"); err != nil {
		return err
	}
	if err := s.ensureAnyPermission(ctx, "finance.savings.delete", "finance.savings.update"); err != nil {
		return err
	}
	existing, err := s.repo.GetByID(ctx, tenantID, savingsID)
	if err != nil {
		return err
	}
	if err := s.repo.SoftDelete(ctx, tenantID, savingsID, actorUserID); err != nil {
		return err
	}
	if s.audit != nil {
		_ = s.audit.Write(ctx, "finance.savings.delete", "finance_savings", existing.ID, existing, map[string]any{"deleted": true, "sid": existing.SID, "name": existing.Name})
	}
	return nil
}

func validateInput(name string, targetAmount, currentAmount int64, currency, status string) error {
	if strings.TrimSpace(name) == "" {
		return domain.ErrInvalidName
	}
	if targetAmount < 0 || currentAmount < 0 || (targetAmount > 0 && currentAmount > targetAmount) {
		return domain.ErrInvalidAmount
	}
	if len(strings.TrimSpace(currency)) != 3 {
		return domain.ErrInvalidCurrency
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
	case "active", "completed", "cancelled":
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
