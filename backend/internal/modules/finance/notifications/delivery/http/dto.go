package http

import "encoding/json"

type NotificationRequest struct {
	NotificationType string          `json:"notification_type"`
	Title            string          `json:"title"`
	Message          string          `json:"message"`
	Metadata         json.RawMessage `json:"metadata"`
}

type NotificationResponse struct {
	ID               string          `json:"id"`
	NotificationType string          `json:"notification_type"`
	Title            string          `json:"title"`
	Message          string          `json:"message"`
	Status           string          `json:"status"`
	Metadata         json.RawMessage `json:"metadata"`
	CreatedAt        string          `json:"created_at"`
	ReadAt           *string         `json:"read_at"`
}

