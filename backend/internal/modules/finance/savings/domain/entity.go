package domain

import "time"

type Savings struct {
	ID                  string
	SID                 string
	TenantID            string
	Name                string
	TargetAmountMinor   int64
	CurrentAmountMinor  int64
	ProgressPercent     float64
	Currency            string
	StartDate           *time.Time
	TargetDate          *time.Time
	Notes               *string
	Status              string
	CreatedBy           string
	UpdatedBy           string
	CreatedAt           time.Time
	UpdatedAt           time.Time
	DeletedAt           *time.Time
}
