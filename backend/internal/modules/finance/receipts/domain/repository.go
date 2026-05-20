package domain

import "context"

type Repository interface {
	ListProviderConfigs(ctx context.Context, tenantID string) ([]ProviderConfig, error)
	GetProviderConfig(ctx context.Context, tenantID, providerCode string) (ProviderConfig, error)
	UpsertProviderConfig(ctx context.Context, item ProviderConfig) (ProviderConfig, error)

	CreateReceiptScan(ctx context.Context, item ReceiptScan) (ReceiptScan, error)
	UpdateReceiptScanResult(ctx context.Context, item ReceiptScan) (ReceiptScan, error)
	ListReceiptScans(ctx context.Context, tenantID string, limit int) ([]ReceiptScan, error)
	GetReceiptScanByID(ctx context.Context, tenantID, scanID string) (ReceiptScan, error)
	DeleteReceiptScan(ctx context.Context, tenantID, scanID string) error
	ClearReceiptScans(ctx context.Context, tenantID string) error
	GetGlobalSetting(ctx context.Context, key string) (string, bool, error)
}
