package domain

import "time"

type Account struct {
	ID                 string
	TenantID           string
	Name               string
	AccountType        string
	Currency           string
	OpeningBalanceMinor int64
	IsActive           bool
	CreatedBy          string
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

type Category struct {
	ID          string
	TenantID    string
	Name        string
	CategoryType string
	ParentID    *string
	IsActive    bool
	CreatedBy   string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

