package http

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"pekan/backend/internal/modules/finance/whatsapp/domain"
	"pekan/backend/internal/modules/finance/whatsapp/usecase"
	"pekan/backend/internal/platform/tenancy"
)

type AuthInfo interface {
	TenantID() string
	UserID() string
}

type Handler struct {
	service *usecase.Service
}

func NewHandler(service *usecase.Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) GetOTP(w http.ResponseWriter, r *http.Request) {
	tc, err := tenancy.FromContext(r.Context())
	if err != nil {
		http.Error(w, `{"error":{"code":"UNAUTHORIZED","message":"missing or invalid authentication context"}}`, http.StatusUnauthorized)
		return
	}

	token, err := h.service.GenerateOTP(r.Context(), usecase.GenerateOTPInput{
		TenantID: tc.TenantID,
		UserID:   tc.UserID,
	})
	if err != nil {
		log.Printf("[ERROR] failed to generate OTP: %v", err)
		http.Error(w, `{"error":{"code":"INTERNAL_ERROR","message":"failed to generate otp"}}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"data": map[string]string{
			"otp_code": token,
		},
	})
}

func (h *Handler) GetStatus(w http.ResponseWriter, r *http.Request) {
	tc, err := tenancy.FromContext(r.Context())
	if err != nil {
		http.Error(w, `{"error":{"code":"UNAUTHORIZED","message":"missing or invalid authentication context"}}`, http.StatusUnauthorized)
		return
	}

	session, err := h.service.GetConnectionStatus(r.Context(), tc.TenantID, tc.UserID)
	botNumber := h.service.GetWhatsAppBotNumber(r.Context())

	if err != nil {
		if errors.Is(err, domain.ErrSessionNotFound) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"data": map[string]interface{}{
					"connected":        false,
					"bot_phone_number": botNumber,
				},
			})
			return
		}
		http.Error(w, `{"error":{"code":"INTERNAL_ERROR","message":"failed to check status"}}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"data": map[string]interface{}{
			"connected":        true,
			"phone_number":     session.PhoneNumber,
			"last_active":      session.LastActive,
			"bot_phone_number": botNumber,
		},
	})
}

func (h *Handler) Disconnect(w http.ResponseWriter, r *http.Request) {
	tc, err := tenancy.FromContext(r.Context())
	if err != nil {
		http.Error(w, `{"error":{"code":"UNAUTHORIZED","message":"missing or invalid authentication context"}}`, http.StatusUnauthorized)
		return
	}

	err = h.service.Disconnect(r.Context(), tc.TenantID, tc.UserID)
	if err != nil {
		http.Error(w, `{"error":{"code":"INTERNAL_ERROR","message":"failed to disconnect"}}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"data": map[string]interface{}{
			"success": true,
		},
	})
}

func (h *Handler) Connect(w http.ResponseWriter, r *http.Request) {
	tc, err := tenancy.FromContext(r.Context())
	if err != nil {
		http.Error(w, `{"error":{"code":"UNAUTHORIZED","message":"missing or invalid authentication context"}}`, http.StatusUnauthorized)
		return
	}

	var req struct {
		PhoneNumber string `json:"phone_number"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":{"code":"BAD_REQUEST","message":"invalid payload"}}`, http.StatusBadRequest)
		return
	}

	if req.PhoneNumber == "" {
		http.Error(w, `{"error":{"code":"BAD_REQUEST","message":"phone_number is required"}}`, http.StatusBadRequest)
		return
	}

	err = h.service.ConnectDirect(r.Context(), tc.TenantID, tc.UserID, req.PhoneNumber)
	if err != nil {
		if errors.Is(err, domain.ErrAlreadyConnected) {
			http.Error(w, `{"error":{"code":"ALREADY_CONNECTED","message":"whatsapp number already connected to another account"}}`, http.StatusConflict)
			return
		}
		http.Error(w, `{"error":{"code":"INTERNAL_ERROR","message":"failed to connect whatsapp"}}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"data": map[string]interface{}{
			"success": true,
		},
	})
}

func (h *Handler) WebChat(w http.ResponseWriter, r *http.Request) {
	tc, err := tenancy.FromContext(r.Context())
	if err != nil {
		http.Error(w, `{"error":{"code":"UNAUTHORIZED","message":"missing or invalid authentication context"}}`, http.StatusUnauthorized)
		return
	}

	var req struct {
		Message string `json:"message"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":{"code":"BAD_REQUEST","message":"invalid payload"}}`, http.StatusBadRequest)
		return
	}

	if req.Message == "" {
		http.Error(w, `{"error":{"code":"BAD_REQUEST","message":"message is required"}}`, http.StatusBadRequest)
		return
	}

	reply, err := h.service.ProcessWebChat(r.Context(), tc.TenantID, tc.UserID, req.Message)
	if err != nil {
		log.Printf("[ERROR] failed to process web chat: %v", err)
		http.Error(w, `{"error":{"code":"INTERNAL_ERROR","message":"failed to generate ai reply"}}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"data": map[string]interface{}{
			"reply": reply,
		},
	})
}

