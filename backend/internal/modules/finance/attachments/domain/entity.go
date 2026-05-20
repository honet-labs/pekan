package domain

import "time"

type OwnerType string

const (
	OwnerTypeSavings   OwnerType = "savings"
	OwnerTypeBudgets   OwnerType = "budgets"
	OwnerTypeReminders OwnerType = "reminders"
)

type Attachment struct {
	ID               string
	TenantID         string
	OwnerType        OwnerType
	OwnerID          string
	FileID           string
	Provider         string
	ObjectKey        string
	OriginalFilename string
	StoredFilename   string
	MimeType         string
	ScanStatus       string
	SizeBytes        int64
	CreatedAt        time.Time
}

