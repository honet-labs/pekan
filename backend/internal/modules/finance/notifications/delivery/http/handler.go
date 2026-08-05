package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"pekan/backend/internal/modules/finance/notifications/domain"
	"pekan/backend/internal/modules/finance/notifications/usecase"
	"pekan/backend/internal/platform/access"
	"pekan/backend/internal/platform/httpx"
	"pekan/backend/internal/platform/middleware"
	"pekan/backend/internal/platform/tenancy"
)

type Service interface {
	Create(ctx context.Context, in usecase.CreateInput) (domain.Notification, error)
	List(ctx context.Context, in usecase.ListInput) ([]domain.Notification, int64, error)
	MarkRead(ctx context.Context, tenantID, notificationID string) (domain.Notification, error)
}

type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req NotificationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid JSON payload", middleware.GetRequestID(r.Context()))
		return
	}

	tc, err := tenancy.FromContext(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "tenant context missing", middleware.GetRequestID(r.Context()))
		return
	}

	out, err := h.service.Create(r.Context(), usecase.CreateInput{
		TenantID:         tc.TenantID,
		ActorUserID:      tc.UserID,
		NotificationType: req.NotificationType,
		Title:            req.Title,
		Message:          req.Message,
		Metadata:         req.Metadata,
	})
	if err != nil {
		writeUsecaseError(w, r, err)
		return
	}

	httpx.WriteJSON(w, http.StatusCreated, toResponse(out), middleware.GetRequestID(r.Context()))
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	tc, err := tenancy.FromContext(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "tenant context missing", middleware.GetRequestID(r.Context()))
		return
	}

	page := parseIntOrDefault(r.URL.Query().Get("page"), 1)
	pageSize := parseIntOrDefault(r.URL.Query().Get("page_size"), 20)
	status := r.URL.Query().Get("status")
	var statusPtr *string
	if status != "" {
		statusPtr = &status
	}

	items, total, err := h.service.List(r.Context(), usecase.ListInput{
		TenantID: tc.TenantID,
		Status:   statusPtr,
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		writeUsecaseError(w, r, err)
		return
	}

	responseItems := make([]NotificationResponse, 0, len(items))
	for _, item := range items {
		responseItems = append(responseItems, toResponse(item))
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"items": responseItems,
		"pagination": map[string]any{
			"page":      page,
			"page_size": pageSize,
			"total":     total,
		},
	}, middleware.GetRequestID(r.Context()))
}

func (h *Handler) MarkRead(w http.ResponseWriter, r *http.Request) {
	notificationID := chi.URLParam(r, "notificationID")
	tc, err := tenancy.FromContext(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "tenant context missing", middleware.GetRequestID(r.Context()))
		return
	}

	out, err := h.service.MarkRead(r.Context(), tc.TenantID, notificationID)
	if err != nil {
		writeUsecaseError(w, r, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, toResponse(out), middleware.GetRequestID(r.Context()))
}

func toResponse(item domain.Notification) NotificationResponse {
	var readAt *string
	if item.ReadAt != nil {
		v := item.ReadAt.UTC().Format(time.RFC3339)
		readAt = &v
	}
	return NotificationResponse{
		ID:               item.ID,
		NotificationType: item.NotificationType,
		Title:            item.Title,
		Message:          item.Message,
		Status:           item.Status,
		Metadata:         item.Metadata,
		CreatedAt:        item.CreatedAt.UTC().Format(time.RFC3339),
		ReadAt:           readAt,
	}
}

func parseIntOrDefault(v string, fallback int) int {
	if v == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return parsed
}

func writeUsecaseError(w http.ResponseWriter, r *http.Request, err error) {
	requestID := middleware.GetRequestID(r.Context())
	switch {
	case errors.Is(err, domain.ErrInvalidTitle):
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_TITLE", err.Error(), requestID)
	case errors.Is(err, domain.ErrInvalidMessage):
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_MESSAGE", err.Error(), requestID)
	case errors.Is(err, domain.ErrInvalidType):
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_TYPE", err.Error(), requestID)
	case errors.Is(err, domain.ErrNotificationNotFound):
		httpx.WriteError(w, http.StatusNotFound, "NOT_FOUND", "notification not found", requestID)
	case errors.Is(err, access.ErrModuleDisabled):
		httpx.WriteError(w, http.StatusForbidden, "FORBIDDEN_MODULE_DISABLED", err.Error(), requestID)
	case errors.Is(err, access.ErrFeatureLocked):
		httpx.WriteError(w, http.StatusForbidden, "FORBIDDEN_FEATURE_LOCKED", err.Error(), requestID)
	case errors.Is(err, access.ErrPermissionDenied):
		httpx.WriteError(w, http.StatusForbidden, "FORBIDDEN_PERMISSION", err.Error(), requestID)
	default:
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error", requestID)
	}
}

