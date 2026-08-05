package domain

import "context"

type ListFilter struct {
	TenantID string
	Status   *string
	Page     int
	PageSize int
}

type Repository interface {
	Create(ctx context.Context, in Notification) (Notification, error)
	List(ctx context.Context, filter ListFilter) ([]Notification, int64, error)
	MarkRead(ctx context.Context, tenantID, notificationID string) (Notification, error)
}

