package domain

import "time"

type Attachment struct {
	ID               string
	TenantID         string
	TransactionID    string
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
