package domain

import "context"

type ListFilter struct {
	TenantID string
	Status   *string
	DateFrom *string
	DateTo   *string
	Page     int
	PageSize int
}

type Repository interface {
	Create(ctx context.Context, in Reminder) (Reminder, error)
	GetByID(ctx context.Context, tenantID, reminderID string) (Reminder, error)
	List(ctx context.Context, filter ListFilter) ([]Reminder, int64, error)
	ListDue(ctx context.Context, tenantID string) ([]Reminder, error)
	Update(ctx context.Context, in Reminder) (Reminder, error)
	UpdateStatus(ctx context.Context, tenantID, reminderID, status, actorUserID string) (Reminder, error)
	SoftDelete(ctx context.Context, tenantID, reminderID, actorUserID string) error

	CreatePayment(ctx context.Context, in ReminderPayment) (ReminderPayment, error)
	UpdatePayment(ctx context.Context, in ReminderPayment) (ReminderPayment, error)
	DeletePayment(ctx context.Context, tenantID, reminderID, paymentID, actorUserID string) error
	ListPayments(ctx context.Context, tenantID, reminderID string) ([]ReminderPayment, error)
}

