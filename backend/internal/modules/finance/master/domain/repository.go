package domain

import "context"

type Repository interface {
	CreateAccount(ctx context.Context, account Account) (Account, error)
	ListAccounts(ctx context.Context, tenantID string) ([]Account, error)
	CreateCategory(ctx context.Context, category Category) (Category, error)
	ListCategories(ctx context.Context, tenantID string) ([]Category, error)
	UpdateCategory(ctx context.Context, tenantID, categoryID, name, categoryType string) (Category, error)
	DeleteCategory(ctx context.Context, tenantID, categoryID string) error
	GetCategory(ctx context.Context, tenantID, categoryID string) (Category, error)
	EnsureDefaultData(ctx context.Context, tenantID string) error
}


