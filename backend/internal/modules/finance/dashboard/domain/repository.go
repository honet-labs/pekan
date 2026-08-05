package domain

import "context"

type Repository interface {
	GetSummary(ctx context.Context, tenantID string, dateFrom, dateTo *string) (Summary, error)
	GetDailySeries(ctx context.Context, tenantID string, dateFrom, dateTo *string) ([]SeriesPoint, error)
	GetTopCategories(ctx context.Context, tenantID string, dateFrom, dateTo *string, limit int) ([]CategoryTotal, error)
}

