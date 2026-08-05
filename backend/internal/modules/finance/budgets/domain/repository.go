package domain

import "context"

type ListFilter struct {
	TenantID string
	Status   *string
	Page     int
	PageSize int
}

type Repository interface {
	Create(ctx context.Context, in Budget) (Budget, error)
	GetByID(ctx context.Context, tenantID, budgetID string) (Budget, error)
	List(ctx context.Context, filter ListFilter) ([]Budget, int64, error)
	Update(ctx context.Context, in Budget) (Budget, error)
	SoftDelete(ctx context.Context, tenantID, budgetID, actorUserID string) error
	ValidateCategory(ctx context.Context, tenantID, categoryID string) error
	ResolveCategoryID(ctx context.Context, tenantID, actorUserID string, categoryID, categoryName *string) (*string, error)
	FindActiveByCategory(ctx context.Context, tenantID, categoryID string) ([]Budget, error)
}
