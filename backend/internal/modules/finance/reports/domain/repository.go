package domain

import "context"

type ListFilter struct {
	TenantID string
	Page     int
	PageSize int
}

type Repository interface {
	Create(ctx context.Context, in Report) (Report, error)
	UpdateStatus(ctx context.Context, in Report) (Report, error)
	GetByID(ctx context.Context, tenantID, reportID string) (Report, error)
	List(ctx context.Context, filter ListFilter) ([]Report, int64, error)
	Delete(ctx context.Context, tenantID, reportID string) error
}

type ExportRepository interface {
	ListTransactions(ctx context.Context, tenantID string, dateFrom, dateTo, categoryID, transactionType *string) ([]TransactionRow, error)
	ListSavings(ctx context.Context, tenantID string, dateFrom, dateTo, status *string) ([]SavingsRow, error)
	ListBudgets(ctx context.Context, tenantID string, dateFrom, dateTo, status *string) ([]BudgetRow, error)
	ListReminders(ctx context.Context, tenantID string, dateFrom, dateTo, status *string) ([]ReminderRow, error)
}
