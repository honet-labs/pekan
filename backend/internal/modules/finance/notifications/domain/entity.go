package domain

import (
	"encoding/json"
	"time"
)

type Notification struct {
	ID               string
	TenantID         string
	NotificationType string
	Title            string
	Message          string
	Status           string
	Metadata         json.RawMessage
	CreatedBy        string
	CreatedAt        time.Time
	ReadAt           *time.Time
}

