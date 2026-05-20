package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"pekan/backend/internal/modules/finance/savings/domain"
	"pekan/backend/internal/modules/finance/savings/usecase"
	transactiondomain "pekan/backend/internal/modules/finance/transactions/domain"
	"pekan/backend/internal/platform/access"
	"pekan/backend/internal/platform/httpx"
	"pekan/backend/internal/platform/middleware"
	"pekan/backend/internal/platform/tenancy"
)

type Service interface {
	Create(ctx context.Context, in usecase.CreateInput) (domain.Savings, error)
	GetByID(ctx context.Context, tenantID, savingsID string) (domain.Savings, error)
	List(ctx context.Context, in usecase.ListInput) ([]domain.Savings, int64, error)
	Update(ctx context.Context, in usecase.UpdateInput) (domain.Savings, error)
	Delete(ctx context.Context, tenantID, actorUserID, savingsID string) error
}

type TransactionRepository interface {
	ListBySavingsID(ctx context.Context, tenantID, savingsID string) ([]transactiondomain.Transaction, error)
}

type Handler struct {
	service               Service
	transactionRepository TransactionRepository
}

func NewHandler(service Service, transactionRepository TransactionRepository) *Handler {
	return &Handler{service: service, transactionRepository: transactionRepository}
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req SavingsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid JSON payload", middleware.GetRequestID(r.Context()))
		return
	}

	startDate, err := parseOptionalDate(req.StartDate)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_DATE", "start_date must be YYYY-MM-DD", middleware.GetRequestID(r.Context()))
		return
	}
	targetDate, err := parseOptionalDate(req.TargetDate)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_DATE", "target_date must be YYYY-MM-DD", middleware.GetRequestID(r.Context()))
		return
	}

	tc, err := tenancy.FromContext(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "tenant context missing", middleware.GetRequestID(r.Context()))
		return
	}

	out, err := h.service.Create(r.Context(), usecase.CreateInput{
		TenantID:           tc.TenantID,
		ActorUserID:        tc.UserID,
		Name:               req.Name,
		TargetAmountMinor:  req.TargetAmountMinor,
		CurrentAmountMinor: req.CurrentAmountMinor,
		Currency:           req.Currency,
		StartDate:          startDate,
		TargetDate:         targetDate,
		Notes:              req.Notes,
		Status:             req.Status,
	})
	if err != nil {
		writeUsecaseError(w, r, err)
		return
	}

	httpx.WriteJSON(w, http.StatusCreated, toResponse(out), middleware.GetRequestID(r.Context()))
}

func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	savingsID := chi.URLParam(r, "savingsID")
	tc, err := tenancy.FromContext(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "tenant context missing", middleware.GetRequestID(r.Context()))
		return
	}

	out, err := h.service.GetByID(r.Context(), tc.TenantID, savingsID)
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

	responseItems := make([]SavingsResponse, 0, len(items))
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

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	savingsID := chi.URLParam(r, "savingsID")

	var req SavingsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid JSON payload", middleware.GetRequestID(r.Context()))
		return
	}

	startDate, err := parseOptionalDate(req.StartDate)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_DATE", "start_date must be YYYY-MM-DD", middleware.GetRequestID(r.Context()))
		return
	}
	targetDate, err := parseOptionalDate(req.TargetDate)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_DATE", "target_date must be YYYY-MM-DD", middleware.GetRequestID(r.Context()))
		return
	}

	tc, err := tenancy.FromContext(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "tenant context missing", middleware.GetRequestID(r.Context()))
		return
	}

	out, err := h.service.Update(r.Context(), usecase.UpdateInput{
		TenantID:           tc.TenantID,
		ActorUserID:        tc.UserID,
		SavingsID:          savingsID,
		Name:               req.Name,
		TargetAmountMinor:  req.TargetAmountMinor,
		CurrentAmountMinor: req.CurrentAmountMinor,
		Currency:           req.Currency,
		StartDate:          startDate,
		TargetDate:         targetDate,
		Notes:              req.Notes,
		Status:             req.Status,
	})
	if err != nil {
		writeUsecaseError(w, r, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, toResponse(out), middleware.GetRequestID(r.Context()))
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	savingsID := chi.URLParam(r, "savingsID")

	tc, err := tenancy.FromContext(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "tenant context missing", middleware.GetRequestID(r.Context()))
		return
	}

	if err := h.service.Delete(r.Context(), tc.TenantID, tc.UserID, savingsID); err != nil {
		writeUsecaseError(w, r, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]any{"deleted": true}, middleware.GetRequestID(r.Context()))
}

func parseOptionalDate(raw *string) (*time.Time, error) {
	if raw == nil || *raw == "" {
		return nil, nil
	}
	parsed, err := time.Parse("2006-01-02", *raw)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}

func toResponse(item domain.Savings) SavingsResponse {
	return SavingsResponse{
		ID:                 item.ID,
		SID:                item.SID,
		Name:               item.Name,
		TargetAmountMinor:  item.TargetAmountMinor,
		CurrentAmountMinor: item.CurrentAmountMinor,
		ProgressPercent:    item.ProgressPercent,
		Currency:           item.Currency,
		StartDate:          formatDatePtr(item.StartDate),
		TargetDate:         formatDatePtr(item.TargetDate),
		Notes:              item.Notes,
		Status:             item.Status,
		CreatedAt:          item.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:          item.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

func formatDatePtr(t *time.Time) *string {
	if t == nil {
		return nil
	}
	out := t.UTC().Format("2006-01-02")
	return &out
}

func (h *Handler) ListRelatedTransactions(w http.ResponseWriter, r *http.Request) {
	savingsID := chi.URLParam(r, "savingsID")
	tc, err := tenancy.FromContext(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "tenant context missing", middleware.GetRequestID(r.Context()))
		return
	}

	if err := access.NewAuthorizer().EnsureModule(r.Context(), "finance"); err != nil {
		httpx.WriteError(w, http.StatusForbidden, "FORBIDDEN", "access denied", middleware.GetRequestID(r.Context()))
		return
	}

	transactions, err := h.transactionRepository.ListBySavingsID(r.Context(), tc.TenantID, savingsID)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "ERROR", err.Error(), middleware.GetRequestID(r.Context()))
		return
	}

	responseItems := make([]RelatedTransactionResponse, 0, len(transactions))
	for _, item := range transactions {
		responseItems = append(responseItems, toRelatedTransactionResponse(item))
	}

	httpx.WriteJSON(w, http.StatusOK, responseItems, middleware.GetRequestID(r.Context()))
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
	case errors.Is(err, domain.ErrInvalidName):
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_NAME", err.Error(), requestID)
	case errors.Is(err, domain.ErrInvalidAmount):
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_AMOUNT", err.Error(), requestID)
	case errors.Is(err, domain.ErrInvalidCurrency):
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_CURRENCY", err.Error(), requestID)
	case errors.Is(err, domain.ErrInvalidStatus):
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_STATUS", err.Error(), requestID)
	case errors.Is(err, domain.ErrSavingsNotFound):
		httpx.WriteError(w, http.StatusNotFound, "NOT_FOUND", "savings goal not found", requestID)
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

func toRelatedTransactionResponse(trx transactiondomain.Transaction) RelatedTransactionResponse {
	return RelatedTransactionResponse{
		ID:              trx.ID,
		TID:             toShortID(trx.ID),
		CategoryName:    trx.CategoryName,
		AmountMinor:     trx.AmountMinor,
		Currency:        trx.Currency,
		TransactionDate: trx.TransactionDate.UTC().Format("2006-01-02"),
		Description:     trx.Description,
	}
}

func toShortID(id string) string {
	if len(id) < 8 {
		return strings.ToUpper(id)
	}
	return strings.ToUpper(id[:8])
}
