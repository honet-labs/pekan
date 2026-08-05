package domain

import "time"

type Reminder struct {
	ID             string
	TenantID       string
	Title          string
	Description    *string
	AmountMinor    *int64
	Currency       *string
	DueDate        time.Time
	RepeatInterval string
	Status         string
	TotalTenor     *int
	CurrentTenor   *int
	LastTriggeredAt *time.Time
	CreatedBy      string
	UpdatedBy      string
	CreatedAt      time.Time
	UpdatedAt      time.Time
	DeletedAt      *time.Time
}

type ReminderPayment struct {
	ID            string
	TenantID      string
	ReminderID    string
	PaidAt        time.Time
	AmountMinor   int64
	Status        string
	Notes         *string
	ProofImageURL *string
	CreatedBy     string
	UpdatedBy     string
	CreatedAt     time.Time
	UpdatedAt     time.Time

	TransientProofName string
	TransientProofMime string
}

