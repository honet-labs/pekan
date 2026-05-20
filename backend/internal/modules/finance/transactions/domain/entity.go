package domain

import "time"

type TransactionType string

const (
	TransactionTypeIncome   TransactionType = "income"
	TransactionTypeExpense  TransactionType = "expense"
	TransactionTypeTransfer TransactionType = "transfer"
	TransactionTypeSavings  TransactionType = "savings"
)

type Transaction struct {
	ID                   string
	TenantID             string
	AccountID            string
	AccountName          string
	CategoryID           *string
	CategoryName         *string
	SavingsIDs           []string
	SavingsNames         []string
	Type                 TransactionType
	AmountMinor          int64
	Currency             string
	InputDate            time.Time
	TransactionDate      time.Time
	Description          *string
	MerchantName         *string
	ReceiptNumber        *string
	PaymentMethod        *string
	SubtotalMinor        int64
	TaxMinor             int64
	ServiceChargeMinor   int64
	ReceiptDiscountMinor int64
	Notes                *string
	CreatedBy            string
	CreatedByName        string
	UpdatedBy            string
	CreatedAt            time.Time
	UpdatedAt            time.Time
	Items                []TransactionItem
}

type TransactionItem struct {
	ID            string
	TenantID      string
	TransactionID string
	ItemName      string
	Quantity      float64
	PriceMinor    int64
	DiscountMinor int64
	TotalMinor        int64
	Notes             *string
	CreatedBy         string
	UpdatedBy         string
	CreatedAt         time.Time
	UpdatedAt         time.Time
}
