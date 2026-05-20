package domain

import "time"

type Budget struct {
	ID                string
	IDA               string
	TenantID          string
	Name              string
	CategoryID        *string
	CategoryName      *string
	AmountLimitMinor  int64
	SpentAmountMinor  int64
	ProgressPercent   float64
	Currency          string
	Period            string
	StartDate         time.Time
	EndDate           *time.Time
	AlertThresholdPct *int
	Notes             *string
	Status            string
	CreatedBy         string
	UpdatedBy         string
	CreatedAt         time.Time
	UpdatedAt         time.Time
	DeletedAt         *time.Time
}
