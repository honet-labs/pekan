package httpx

import (
	"encoding/json"
	"net/http"
	"time"
)

type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
	Meta  Meta        `json:"meta"`
}

func WriteError(w http.ResponseWriter, status int, code, message, requestID string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(ErrorResponse{
		Error: ErrorDetail{
			Code:    code,
			Message: message,
		},
		Meta: Meta{
			RequestID: requestID,
			Timestamp: time.Now().UTC(),
		},
	})
}

