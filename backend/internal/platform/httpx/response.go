package httpx

import (
	"encoding/json"
	"net/http"
	"time"
)

type Meta struct {
	RequestID string    `json:"request_id,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

type SuccessResponse struct {
	Data any  `json:"data"`
	Meta Meta `json:"meta"`
}

func WriteJSON(w http.ResponseWriter, status int, data any, requestID string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(SuccessResponse{
		Data: data,
		Meta: Meta{
			RequestID: requestID,
			Timestamp: time.Now().UTC(),
		},
	})
}

