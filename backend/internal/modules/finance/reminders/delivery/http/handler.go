package http

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"pekan/backend/internal/modules/finance/reminders/domain"
	"pekan/backend/internal/modules/finance/reminders/usecase"
	"pekan/backend/internal/platform/access"
	"pekan/backend/internal/platform/httpx"
	"pekan/backend/internal/platform/middleware"
	"pekan/backend/internal/platform/tenancy"
)

type Service interface {
	Create(ctx context.Context, in usecase.CreateInput) (domain.Reminder, error)
	GetByID(ctx context.Context, tenantID, reminderID string) (domain.Reminder, error)
	List(ctx context.Context, in usecase.ListInput) ([]domain.Reminder, int64, error)
	ListDue(ctx context.Context, tenantID string) ([]domain.Reminder, error)
	Update(ctx context.Context, in usecase.UpdateInput) (domain.Reminder, error)
	MarkStatus(ctx context.Context, tenantID, actorUserID, reminderID, status string) (domain.Reminder, error)
	Delete(ctx context.Context, tenantID, actorUserID, reminderID string) error

	AddPayment(ctx context.Context, in usecase.AddPaymentInput) (domain.ReminderPayment, error)
	UpdatePayment(ctx context.Context, in usecase.UpdatePaymentInput) (domain.ReminderPayment, error)
	DeletePayment(ctx context.Context, tenantID, actorUserID, reminderID, paymentID string) error
	GetPaymentHistory(ctx context.Context, tenantID, reminderID string) ([]domain.ReminderPayment, error)
	GetProofImage(ctx context.Context, tenantID, reminderID, paymentID string) ([]byte, string, error)
}

type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req ReminderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid JSON payload", middleware.GetRequestID(r.Context()))
		return
	}

	dueDate, err := time.Parse("2006-01-02", req.DueDate)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_DATE", "due_date must be YYYY-MM-DD", middleware.GetRequestID(r.Context()))
		return
	}

	tc, err := tenancy.FromContext(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "tenant context missing", middleware.GetRequestID(r.Context()))
		return
	}

	out, err := h.service.Create(r.Context(), usecase.CreateInput{
		TenantID:       tc.TenantID,
		ActorUserID:    tc.UserID,
		Title:          req.Title,
		Description:    req.Description,
		AmountMinor:    req.AmountMinor,
		Currency:       req.Currency,
		DueDate:        dueDate,
		RepeatInterval: req.RepeatInterval,
		Status:         req.Status,
	})
	if err != nil {
		writeUsecaseError(w, r, err)
		return
	}

	httpx.WriteJSON(w, http.StatusCreated, toResponse(out), middleware.GetRequestID(r.Context()))
}

func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	reminderID := chi.URLParam(r, "reminderID")
	tc, err := tenancy.FromContext(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "tenant context missing", middleware.GetRequestID(r.Context()))
		return
	}

	out, err := h.service.GetByID(r.Context(), tc.TenantID, reminderID)
	if err != nil {
		writeUsecaseError(w, r, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, toResponse(out), middleware.GetRequestID(r.Context()))
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
	dateFrom := r.URL.Query().Get("from")
	dateTo := r.URL.Query().Get("to")
	var fromPtr *string
	var toPtr *string
	if dateFrom != "" {
		fromPtr = &dateFrom
	}
	if dateTo != "" {
		toPtr = &dateTo
	}

	items, total, err := h.service.List(r.Context(), usecase.ListInput{
		TenantID: tc.TenantID,
		Status:   statusPtr,
		DateFrom: fromPtr,
		DateTo:   toPtr,
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		writeUsecaseError(w, r, err)
		return
	}

	responseItems := make([]ReminderResponse, 0, len(items))
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

func (h *Handler) ListDue(w http.ResponseWriter, r *http.Request) {
	tc, err := tenancy.FromContext(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "tenant context missing", middleware.GetRequestID(r.Context()))
		return
	}

	items, err := h.service.ListDue(r.Context(), tc.TenantID)
	if err != nil {
		writeUsecaseError(w, r, err)
		return
	}

	responseItems := make([]ReminderResponse, 0, len(items))
	for _, item := range items {
		responseItems = append(responseItems, toResponse(item))
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": responseItems}, middleware.GetRequestID(r.Context()))
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	reminderID := chi.URLParam(r, "reminderID")

	var req ReminderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid JSON payload", middleware.GetRequestID(r.Context()))
		return
	}

	dueDate, err := time.Parse("2006-01-02", req.DueDate)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_DATE", "due_date must be YYYY-MM-DD", middleware.GetRequestID(r.Context()))
		return
	}

	tc, err := tenancy.FromContext(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "tenant context missing", middleware.GetRequestID(r.Context()))
		return
	}

	out, err := h.service.Update(r.Context(), usecase.UpdateInput{
		TenantID:       tc.TenantID,
		ActorUserID:    tc.UserID,
		ReminderID:     reminderID,
		Title:          req.Title,
		Description:    req.Description,
		AmountMinor:    req.AmountMinor,
		Currency:       req.Currency,
		DueDate:        dueDate,
		RepeatInterval: req.RepeatInterval,
		Status:         req.Status,
	})
	if err != nil {
		writeUsecaseError(w, r, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, toResponse(out), middleware.GetRequestID(r.Context()))
}

func (h *Handler) MarkStatus(w http.ResponseWriter, r *http.Request) {
	reminderID := chi.URLParam(r, "reminderID")
	var req ReminderStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid JSON payload", middleware.GetRequestID(r.Context()))
		return
	}

	tc, err := tenancy.FromContext(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "tenant context missing", middleware.GetRequestID(r.Context()))
		return
	}

	out, err := h.service.MarkStatus(r.Context(), tc.TenantID, tc.UserID, reminderID, req.Status)
	if err != nil {
		writeUsecaseError(w, r, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, toResponse(out), middleware.GetRequestID(r.Context()))
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	reminderID := chi.URLParam(r, "reminderID")
	tc, err := tenancy.FromContext(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "tenant context missing", middleware.GetRequestID(r.Context()))
		return
	}

	if err := h.service.Delete(r.Context(), tc.TenantID, tc.UserID, reminderID); err != nil {
		writeUsecaseError(w, r, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]any{"deleted": true}, middleware.GetRequestID(r.Context()))
}

func (h *Handler) AddPayment(w http.ResponseWriter, r *http.Request) {
	reminderID := chi.URLParam(r, "reminderID")

	var paidAtStr string
	var amountMinor int64
	var status string
	var notes *string
	var fileContent io.Reader
	var fileName string
	var fileMime string

	if strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			httpx.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid multipart payload", middleware.GetRequestID(r.Context()))
			return
		}
		paidAtStr = r.FormValue("paid_at")
		amountMinor = int64(parseIntOrDefault(r.FormValue("amount_minor"), 0))
		status = r.FormValue("status")
		n := r.FormValue("notes")
		if n != "" {
			notes = &n
		}

		file, header, err := r.FormFile("image")
		if err == nil {
			defer file.Close()
			fileContent = file
			fileName = header.Filename
			fileMime = header.Header.Get("Content-Type")
		}
	} else {
		var req ReminderPaymentRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httpx.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid JSON payload", middleware.GetRequestID(r.Context()))
			return
		}
		paidAtStr = req.PaidAt
		amountMinor = req.AmountMinor
		status = req.Status
		notes = req.Notes
	}

	paidAt, err := time.Parse("2006-01-02", paidAtStr)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_DATE", "paid_at must be YYYY-MM-DD", middleware.GetRequestID(r.Context()))
		return
	}

	tc, err := tenancy.FromContext(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "tenant context missing", middleware.GetRequestID(r.Context()))
		return
	}

	out, err := h.service.AddPayment(r.Context(), usecase.AddPaymentInput{
		TenantID:          tc.TenantID,
		ActorUserID:       tc.UserID,
		ReminderID:        reminderID,
		PaidAt:            paidAt,
		AmountMinor:       amountMinor,
		Status:            status,
		Notes:             notes,
		ProofImageContent: fileContent,
		ProofImageName:    fileName,
		ProofImageMime:    fileMime,
	})
	if err != nil {
		writeUsecaseError(w, r, err)
		return
	}

	httpx.WriteJSON(w, http.StatusCreated, toPaymentResponse(out), middleware.GetRequestID(r.Context()))
}

func (h *Handler) GetPaymentHistory(w http.ResponseWriter, r *http.Request) {
	reminderID := chi.URLParam(r, "reminderID")
	tc, err := tenancy.FromContext(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "tenant context missing", middleware.GetRequestID(r.Context()))
		return
	}

	items, err := h.service.GetPaymentHistory(r.Context(), tc.TenantID, reminderID)
	if err != nil {
		writeUsecaseError(w, r, err)
		return
	}

	responseItems := make([]ReminderPaymentResponse, 0, len(items))
	for _, item := range items {
		responseItems = append(responseItems, toPaymentResponse(item))
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": responseItems}, middleware.GetRequestID(r.Context()))
}

func (h *Handler) UpdatePayment(w http.ResponseWriter, r *http.Request) {
	reminderID := chi.URLParam(r, "reminderID")
	paymentID := chi.URLParam(r, "paymentID")

	var paidAtStr string
	var amountMinor int64
	var status string
	var notes *string
	var fileContent io.Reader
	var fileName string
	var fileMime string

	if strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			httpx.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid multipart payload", middleware.GetRequestID(r.Context()))
			return
		}
		paidAtStr = r.FormValue("paid_at")
		amountMinor = int64(parseIntOrDefault(r.FormValue("amount_minor"), 0))
		status = r.FormValue("status")
		n := r.FormValue("notes")
		if n != "" {
			notes = &n
		}

		file, header, err := r.FormFile("image")
		if err == nil {
			defer file.Close()
			fileContent = file
			fileName = header.Filename
			fileMime = header.Header.Get("Content-Type")
		}
	} else {
		var req ReminderPaymentRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httpx.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid JSON payload", middleware.GetRequestID(r.Context()))
			return
		}
		paidAtStr = req.PaidAt
		amountMinor = req.AmountMinor
		status = req.Status
		notes = req.Notes
	}

	paidAt, err := time.Parse("2006-01-02", paidAtStr)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_DATE", "paid_at must be YYYY-MM-DD", middleware.GetRequestID(r.Context()))
		return
	}

	tc, err := tenancy.FromContext(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "tenant context missing", middleware.GetRequestID(r.Context()))
		return
	}

	out, err := h.service.UpdatePayment(r.Context(), usecase.UpdatePaymentInput{
		TenantID:          tc.TenantID,
		ActorUserID:       tc.UserID,
		ReminderID:        reminderID,
		PaymentID:         paymentID,
		PaidAt:            paidAt,
		AmountMinor:       amountMinor,
		Status:            status,
		Notes:             notes,
		ProofImageContent: fileContent,
		ProofImageName:    fileName,
		ProofImageMime:    fileMime,
	})
	if err != nil {
		writeUsecaseError(w, r, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, toPaymentResponse(out), middleware.GetRequestID(r.Context()))
}

func (h *Handler) DeletePayment(w http.ResponseWriter, r *http.Request) {
	reminderID := chi.URLParam(r, "reminderID")
	paymentID := chi.URLParam(r, "paymentID")

	tc, err := tenancy.FromContext(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "tenant context missing", middleware.GetRequestID(r.Context()))
		return
	}

	if err := h.service.DeletePayment(r.Context(), tc.TenantID, tc.UserID, reminderID, paymentID); err != nil {
		writeUsecaseError(w, r, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]any{"deleted": true}, middleware.GetRequestID(r.Context()))
}

func (h *Handler) GetProofImage(w http.ResponseWriter, r *http.Request) {
	reminderID := chi.URLParam(r, "reminderID")
	paymentID := chi.URLParam(r, "paymentID")

	tc, err := tenancy.FromContext(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "tenant context missing", middleware.GetRequestID(r.Context()))
		return
	}

	data, mime, err := h.service.GetProofImage(r.Context(), tc.TenantID, reminderID, paymentID)
	if err != nil {
		writeUsecaseError(w, r, err)
		return
	}

	w.Header().Set("Content-Type", mime)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

func toResponse(item domain.Reminder) ReminderResponse {
	return ReminderResponse{
		ID:             item.ID,
		Title:          item.Title,
		Description:    item.Description,
		AmountMinor:    item.AmountMinor,
		Currency:       item.Currency,
		DueDate:        item.DueDate.UTC().Format("2006-01-02"),
		RepeatInterval: item.RepeatInterval,
		Status:         item.Status,
		CreatedAt:      item.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:      item.UpdatedAt.UTC().Format(time.RFC3339),
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
	case errors.Is(err, domain.ErrInvalidAmount):
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_AMOUNT", err.Error(), requestID)
	case errors.Is(err, domain.ErrInvalidCurrency):
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_CURRENCY", err.Error(), requestID)
	case errors.Is(err, domain.ErrInvalidDate):
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_DATE", err.Error(), requestID)
	case errors.Is(err, domain.ErrInvalidRepeat):
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_REPEAT", err.Error(), requestID)
	case errors.Is(err, domain.ErrInvalidStatus):
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_STATUS", err.Error(), requestID)
	case errors.Is(err, domain.ErrReminderNotFound):
		httpx.WriteError(w, http.StatusNotFound, "NOT_FOUND", "reminder not found", requestID)
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

func toPaymentResponse(item domain.ReminderPayment) ReminderPaymentResponse {
	return ReminderPaymentResponse{
		ID:            item.ID,
		ReminderID:    item.ReminderID,
		PaidAt:        item.PaidAt.UTC().Format("2006-01-02"),
		AmountMinor:   item.AmountMinor,
		Status:        item.Status,
		Notes:         item.Notes,
		ProofImageURL: item.ProofImageURL,
		CreatedAt:     item.CreatedAt.UTC().Format(time.RFC3339),
	}
}

