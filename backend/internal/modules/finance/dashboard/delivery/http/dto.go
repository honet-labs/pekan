package http

type SummaryResponse struct {
	TotalIncomeMinor   int64 `json:"total_income_minor"`
	TotalExpenseMinor  int64 `json:"total_expense_minor"`
	TotalTransferMinor int64 `json:"total_transfer_minor"`
	NetAmountMinor     int64 `json:"net_amount_minor"`
	TotalSavingsMinor  int64 `json:"total_savings_minor"`
	TransactionCount   int64 `json:"transaction_count"`
	IncomeCount        int64 `json:"income_count"`
	ExpenseCount       int64 `json:"expense_count"`
	TransferCount      int64 `json:"transfer_count"`
	SavingsCount       int64 `json:"savings_count"`
}

type SeriesPointResponse struct {
	Date         string `json:"date"`
	IncomeMinor  int64  `json:"income_minor"`
	ExpenseMinor int64  `json:"expense_minor"`
}

type CategoryTotalResponse struct {
	CategoryID      *string `json:"category_id"`
	CategoryName    *string `json:"category_name"`
	TransactionType string  `json:"transaction_type"`
	TotalMinor      int64   `json:"total_minor"`
	Count           int64   `json:"count"`
}
