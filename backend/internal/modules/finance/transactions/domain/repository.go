package domain

import (
	"context"
	"time"
)

type ListFilter struct {
	TenantID string
	Type     *TransactionType
	DateFrom *time.Time
	DateTo   *time.Time
	Query    string
	CategoryID *string
	Page       int
	PageSize   int
}

type Repository interface {
	Create(ctx context.Context, trx Transaction) (Transaction, error)
	GetByID(ctx context.Context, tenantID, transactionID string) (Transaction, error)
	List(ctx context.Context, filter ListFilter) ([]Transaction, int64, error)
	ListBySavingsID(ctx context.Context, tenantID, savingsID string) ([]Transaction, error)
	Update(ctx context.Context, trx Transaction) (Transaction, error)
	SoftDelete(ctx context.Context, tenantID, transactionID, deletedBy string) error
	ResolveCategoryID(ctx context.Context, tenantID, actorUserID string, categoryID, categoryName *string, trxType TransactionType) (*string, error)
	ValidateReferences(ctx context.Context, tenantID, accountID string, categoryID *string, trxType TransactionType) error
	ValidateSavingsGoals(ctx context.Context, tenantID string, savingsIDs []string) error
	ReplaceSavingsLinks(ctx context.Context, tenantID, transactionID, actorUserID string, amountMinor int64, savingsIDs []string) error
	ListSavingsLinks(ctx context.Context, tenantID string, transactionIDs []string) (map[string][]string, map[string][]string, error)
	ListSavingsAllocationsByTransaction(ctx context.Context, tenantID, transactionID string) (map[string]int64, error)
	AdjustSavingsCurrentAmounts(ctx context.Context, tenantID, actorUserID string, deltas map[string]int64) error
	ReplaceItems(ctx context.Context, tenantID, transactionID, actorUserID string, items []TransactionItem) error
	ListItems(ctx context.Context, tenantID, transactionID string) ([]TransactionItem, error)
	ListItemsByTransactionIDs(ctx context.Context, tenantID string, transactionIDs []string) (map[string][]TransactionItem, error)
}
