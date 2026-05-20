package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"pekan/backend/internal/modules/finance/budgets/domain"
	"pekan/backend/internal/modules/finance/budgets/usecase"
	"pekan/backend/internal/platform/access"
	"pekan/backend/internal/platform/httpx"
	"pekan/backend/internal/platform/middleware"
	"pekan/backend/internal/platform/tenancy"
)

type Service interface {
	Create(ctx context.Context, in usecase.CreateInput) (domain.Budget, error)
	GetByID(ctx context.Context, tenantID, budgetID string) (domain.Budget, error)
	List(ctx context.Context, in usecase.ListInput) ([]domain.Budget, int64, error)
	Update(ctx context.Context, in usecase.UpdateInput) (domain.Budget, error)
	Delete(ctx context.Context, tenantID, actorUserID, budgetID string) error
}

type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req BudgetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid JSON payload", middleware.GetRequestID(r.Context()))
		return
	}

	startDate, err := time.Parse("2006-01-02", req.StartDate)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_DATE", "start_date must be YYYY-MM-DD", middleware.GetRequestID(r.Context()))
		return
	}
	endDate, err := parseOptionalDate(req.EndDate)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_DATE", "end_date must be YYYY-MM-DD", middleware.GetRequestID(r.Context()))
		return
	}

	tc, err := tenancy.FromContext(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "tenant context missing", middleware.GetRequestID(r.Context()))
		return
	}

	out, err := h.service.Create(r.Context(), usecase.CreateInput{
		TenantID:          tc.TenantID,
		ActorUserID:       tc.UserID,
		Name:              req.Name,
		CategoryID:        req.CategoryID,
		CategoryName:      req.CategoryName,
		AmountLimitMinor:  req.AmountLimitMinor,
		Currency:          req.Currency,
		Period:            req.Period,
		StartDate:         startDate,
		EndDate:           endDate,
		AlertThresholdPct: req.AlertThresholdPct,
		Notes:             req.Notes,
		Status:            req.Status,
	})
	if err != nil {
		writeUsecaseError(w, r, err)
		return
	}

	httpx.WriteJSON(w, http.StatusCreated, toResponse(out), middleware.GetRequestID(r.Context()))
}

func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	budgetID := chi.URLParam(r, "budgetID")
	tc, err := tenancy.FromContext(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "tenant context missing", middleware.GetRequestID(r.Context()))
		return
	}

	out, err := h.service.GetByID(r.Context(), tc.TenantID, budgetID)
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

	responseItems := make([]BudgetResponse, 0, len(items))
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
	budgetID := chi.URLParam(r, "budgetID")

	var req BudgetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid JSON payload", middleware.GetRequestID(r.Context()))
		return
	}

	startDate, err := time.Parse("2006-01-02", req.StartDate)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_DATE", "start_date must be YYYY-MM-DD", middleware.GetRequestID(r.Context()))
		return
	}
	endDate, err := parseOptionalDate(req.EndDate)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_DATE", "end_date must be YYYY-MM-DD", middleware.GetRequestID(r.Context()))
		return
	}

	tc, err := tenancy.FromContext(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "tenant context missing", middleware.GetRequestID(r.Context()))
		return
	}

	out, err := h.service.Update(r.Context(), usecase.UpdateInput{
		TenantID:          tc.TenantID,
		ActorUserID:       tc.UserID,
		BudgetID:          budgetID,
		Name:              req.Name,
		CategoryID:        req.CategoryID,
		CategoryName:      req.CategoryName,
		AmountLimitMinor:  req.AmountLimitMinor,
		Currency:          req.Currency,
		Period:            req.Period,
		StartDate:         startDate,
		EndDate:           endDate,
		AlertThresholdPct: req.AlertThresholdPct,
		Notes:             req.Notes,
		Status:            req.Status,
	})
	if err != nil {
		writeUsecaseError(w, r, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, toResponse(out), middleware.GetRequestID(r.Context()))
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	budgetID := chi.URLParam(r, "budgetID")

	tc, err := tenancy.FromContext(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "tenant context missing", middleware.GetRequestID(r.Context()))
		return
	}

	if err := h.service.Delete(r.Context(), tc.TenantID, tc.UserID, budgetID); err != nil {
		writeUsecaseError(w, r, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]any{"deleted": true}, middleware.GetRequestID(r.Context()))
}

func toResponse(item domain.Budget) BudgetResponse {
	return BudgetResponse{
		ID:                item.ID,
		IDA:               item.IDA,
		IDAnggaran:        item.IDA,
		Name:              item.Name,
		CategoryID:        item.CategoryID,
		CategoryName:      item.CategoryName,
		AmountLimitMinor:  item.AmountLimitMinor,
		SpentAmountMinor:  item.SpentAmountMinor,
		ProgressPercent:   item.ProgressPercent,
		Currency:          item.Currency,
		Period:            item.Period,
		StartDate:         item.StartDate.UTC().Format("2006-01-02"),
		EndDate:           formatDatePtr(item.EndDate),
		AlertThresholdPct: item.AlertThresholdPct,
		Notes:             item.Notes,
		Status:            item.Status,
		CreatedAt:         item.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:         item.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

func formatDatePtr(t *time.Time) *string {
	if t == nil {
		return nil
	}
	out := t.UTC().Format("2006-01-02")
	return &out
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
	case errors.Is(err, domain.ErrInvalidPeriod):
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_PERIOD", err.Error(), requestID)
	case errors.Is(err, domain.ErrInvalidDateRange):
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_DATE_RANGE", err.Error(), requestID)
	case errors.Is(err, domain.ErrInvalidAlert):
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_ALERT_THRESHOLD", err.Error(), requestID)
	case errors.Is(err, domain.ErrInvalidStatus):
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_STATUS", err.Error(), requestID)
	case errors.Is(err, domain.ErrCategoryNotFound):
		httpx.WriteError(w, http.StatusBadRequest, "CATEGORY_NOT_FOUND", err.Error(), requestID)
	case errors.Is(err, domain.ErrBudgetNotFound):
		httpx.WriteError(w, http.StatusNotFound, "NOT_FOUND", "budget not found", requestID)
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
