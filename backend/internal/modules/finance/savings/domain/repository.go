package domain

import "context"

type ListFilter struct {
	TenantID string
	Status   *string
	Page     int
	PageSize int
}

type Repository interface {
	Create(ctx context.Context, in Savings) (Savings, error)
	GetByID(ctx context.Context, tenantID, savingsID string) (Savings, error)
	List(ctx context.Context, filter ListFilter) ([]Savings, int64, error)
	Update(ctx context.Context, in Savings) (Savings, error)
	SoftDelete(ctx context.Context, tenantID, savingsID, actorUserID string) error
}

