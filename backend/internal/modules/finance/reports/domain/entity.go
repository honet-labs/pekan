package domain

import (
	"encoding/json"
	"time"
)

type Report struct {
	ID             string
	TenantID       string
	ReportType     string
	Format         string
	Status         string
	Params         json.RawMessage
	StorageProvider *string
	StorageKey     *string
	CreatedBy      string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type TransactionRow struct {
	ID              string
	InputDate       time.Time
	AccountID       string
	AccountName     string
	CategoryID      *string
	CategoryName    *string
	Type            string
	AmountMinor     int64
	Currency        string
	TransactionDate time.Time
	Description     *string
	MerchantName    *string
	PaymentMethod   *string
}

type SavingsRow struct {
	ID                 string
	Name               string
	TargetAmountMinor  int64
	CurrentAmountMinor int64
	ProgressPercent    float64
	Currency           string
	StartDate          *time.Time
	TargetDate         *time.Time
	Status             string
	UpdatedAt          time.Time
}

type BudgetRow struct {
	ID               string
	Name             string
	CategoryID       *string
	CategoryName     *string
	AmountLimitMinor int64
	Currency         string
	Period           string
	StartDate        time.Time
	EndDate          *time.Time
	AlertThresholdPct *int
	Status           string
	UpdatedAt        time.Time
}

type ReminderRow struct {
	ID             string
	Title          string
	Description    *string
	AmountMinor    *int64
	Currency       *string
	DueDate        time.Time
	RepeatInterval string
	Status         string
	LastTriggeredAt *time.Time
	UpdatedAt      time.Time
}
