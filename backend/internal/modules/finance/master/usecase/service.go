package usecase

import (
	"context"
	"strings"
	"time"

	"pekan/backend/internal/modules/finance/master/domain"
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

type CreateAccountInput struct {
	TenantID            string
	ActorUserID         string
	Name                string
	AccountType         string
	Currency            string
	OpeningBalanceMinor int64
}

func (s *Service) CreateAccount(ctx context.Context, in CreateAccountInput) (domain.Account, error) {
	if err := s.authz.EnsureModule(ctx, "finance.masterdata"); err != nil {
		return domain.Account{}, err
	}
	if err := s.authz.EnsureFeature(ctx, "finance.masterdata.write"); err != nil {
		return domain.Account{}, err
	}
	if err := s.authz.EnsurePermission(ctx, "finance.accounts.create"); err != nil {
		return domain.Account{}, err
	}
	if strings.TrimSpace(in.Name) == "" {
		return domain.Account{}, domain.ErrInvalidAccountName
	}
	if !isAllowedAccountType(in.AccountType) {
		return domain.Account{}, domain.ErrInvalidAccountType
	}
	if len(strings.TrimSpace(in.Currency)) != 3 {
		return domain.Account{}, domain.ErrInvalidCurrency
	}

	now := time.Now().UTC()
	out, err := s.repo.CreateAccount(ctx, domain.Account{
		TenantID:            in.TenantID,
		Name:                strings.TrimSpace(in.Name),
		AccountType:         in.AccountType,
		Currency:            strings.ToUpper(strings.TrimSpace(in.Currency)),
		OpeningBalanceMinor: in.OpeningBalanceMinor,
		IsActive:            true,
		CreatedBy:           in.ActorUserID,
		CreatedAt:           now,
		UpdatedAt:           now,
	})
	if err != nil {
		return domain.Account{}, err
	}

	_ = s.audit.Write(ctx, "finance.account.create", "finance_account", out.ID, nil, out)
	return out, nil
}

func (s *Service) ListAccounts(ctx context.Context, tenantID string) ([]domain.Account, error) {
	if err := s.authz.EnsureModule(ctx, "finance.masterdata"); err != nil {
		return nil, err
	}
	if err := s.authz.EnsureFeature(ctx, "finance.masterdata.read"); err != nil {
		return nil, err
	}
	if err := s.authz.EnsurePermission(ctx, "finance.accounts.read"); err != nil {
		return nil, err
	}
	return s.repo.ListAccounts(ctx, tenantID)
}

type CreateCategoryInput struct {
	TenantID     string
	ActorUserID  string
	Name         string
	CategoryType string
	ParentID     *string
}

func (s *Service) CreateCategory(ctx context.Context, in CreateCategoryInput) (domain.Category, error) {
	if err := s.authz.EnsureModule(ctx, "finance.masterdata"); err != nil {
		return domain.Category{}, err
	}
	if err := s.authz.EnsureFeature(ctx, "finance.masterdata.write"); err != nil {
		return domain.Category{}, err
	}
	if err := s.authz.EnsurePermission(ctx, "finance.categories.create"); err != nil {
		return domain.Category{}, err
	}
	if strings.TrimSpace(in.Name) == "" {
		return domain.Category{}, domain.ErrInvalidCategoryName
	}
	if in.CategoryType != "income" && in.CategoryType != "expense" {
		return domain.Category{}, domain.ErrInvalidCategoryType
	}

	now := time.Now().UTC()
	out, err := s.repo.CreateCategory(ctx, domain.Category{
		TenantID:     in.TenantID,
		Name:         strings.TrimSpace(in.Name),
		CategoryType: in.CategoryType,
		ParentID:     in.ParentID,
		IsActive:     true,
		CreatedBy:    in.ActorUserID,
		CreatedAt:    now,
		UpdatedAt:    now,
	})
	if err != nil {
		return domain.Category{}, err
	}

	_ = s.audit.Write(ctx, "finance.category.create", "finance_category", out.ID, nil, out)
	return out, nil
}

func (s *Service) ListCategories(ctx context.Context, tenantID string) ([]domain.Category, error) {
	if err := s.authz.EnsureModule(ctx, "finance.masterdata"); err != nil {
		return nil, err
	}
	if err := s.authz.EnsureFeature(ctx, "finance.masterdata.read"); err != nil {
		return nil, err
	}
	if err := s.authz.EnsurePermission(ctx, "finance.categories.read"); err != nil {
		return nil, err
	}
	return s.repo.ListCategories(ctx, tenantID)
}

func (s *Service) UpdateCategory(ctx context.Context, tenantID, categoryID string, in CreateCategoryInput) (domain.Category, error) {
	if err := s.authz.EnsureModule(ctx, "finance.masterdata"); err != nil {
		return domain.Category{}, err
	}
	if err := s.authz.EnsureFeature(ctx, "finance.masterdata.write"); err != nil {
		return domain.Category{}, err
	}
	if err := s.authz.EnsurePermission(ctx, "finance.categories.update"); err != nil {
		return domain.Category{}, err
	}
	if strings.TrimSpace(in.Name) == "" {
		return domain.Category{}, domain.ErrInvalidCategoryName
	}
	if in.CategoryType != "income" && in.CategoryType != "expense" {
		return domain.Category{}, domain.ErrInvalidCategoryType
	}
	
	out, err := s.repo.UpdateCategory(ctx, tenantID, categoryID, strings.TrimSpace(in.Name), in.CategoryType)
	if err != nil {
		return domain.Category{}, err
	}
	_ = s.audit.Write(ctx, "finance.category.update", "finance_category", out.ID, nil, out)
	return out, nil
}

func (s *Service) DeleteCategory(ctx context.Context, tenantID, categoryID string) error {
	if err := s.authz.EnsureModule(ctx, "finance.masterdata"); err != nil {
		return err
	}
	if err := s.authz.EnsureFeature(ctx, "finance.masterdata.write"); err != nil {
		return err
	}
	if err := s.authz.EnsurePermission(ctx, "finance.categories.delete"); err != nil {
		return err
	}
	err := s.repo.DeleteCategory(ctx, tenantID, categoryID)
	if err == nil {
		_ = s.audit.Write(ctx, "finance.category.delete", "finance_category", categoryID, nil, nil)
	}
	return err
}

func (s *Service) GetCategory(ctx context.Context, tenantID, categoryID string) (domain.Category, error) {
	if err := s.authz.EnsureModule(ctx, "finance.masterdata"); err != nil {
		return domain.Category{}, err
	}
	if err := s.authz.EnsureFeature(ctx, "finance.masterdata.read"); err != nil {
		return domain.Category{}, err
	}
	if err := s.authz.EnsurePermission(ctx, "finance.categories.read"); err != nil {
		return domain.Category{}, err
	}
	return s.repo.GetCategory(ctx, tenantID, categoryID)
}

func isAllowedAccountType(v string) bool {
	switch v {
	case "cash", "bank", "ewallet", "credit":
		return true
	default:
		return false
	}
}

