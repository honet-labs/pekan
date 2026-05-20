package domain

import "time"

type Summary struct {
	TotalIncomeMinor   int64
	TotalExpenseMinor  int64
	TotalTransferMinor int64
	NetAmountMinor     int64
	TotalSavingsMinor  int64
	TransactionCount   int64
	IncomeCount        int64
	ExpenseCount       int64
	TransferCount      int64
	SavingsCount       int64
}

type SeriesPoint struct {
	Date         time.Time
	IncomeMinor  int64
	ExpenseMinor int64
}

type CategoryTotal struct {
	CategoryID      *string
	CategoryName    *string
	TransactionType string
	TotalMinor      int64
	Count           int64
}
