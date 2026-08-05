package http

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	"pekan/backend/internal/modules/finance/dashboard/domain"
	"pekan/backend/internal/modules/finance/dashboard/usecase"
	"pekan/backend/internal/platform/access"
	"pekan/backend/internal/platform/httpx"
	"pekan/backend/internal/platform/middleware"
	"pekan/backend/internal/platform/tenancy"
)

type Service interface {
	GetSummary(ctx context.Context, in usecase.SummaryInput) (domain.Summary, error)
	GetDailySeries(ctx context.Context, in usecase.SeriesInput) ([]domain.SeriesPoint, error)
	GetTopCategories(ctx context.Context, in usecase.TopCategoriesInput) ([]domain.CategoryTotal, error)
}

type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Summary(w http.ResponseWriter, r *http.Request) {
	tc, err := tenancy.FromContext(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "tenant context missing", middleware.GetRequestID(r.Context()))
		return
	}

	dateFrom := r.URL.Query().Get("from")
	dateTo := r.URL.Query().Get("to")
	if err := validateDate(dateFrom); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_DATE", "from must be YYYY-MM-DD", middleware.GetRequestID(r.Context()))
		return
	}
	if err := validateDate(dateTo); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_DATE", "to must be YYYY-MM-DD", middleware.GetRequestID(r.Context()))
		return
	}

	var fromPtr *string
	var toPtr *string
	if dateFrom != "" {
		fromPtr = &dateFrom
	}
	if dateTo != "" {
		toPtr = &dateTo
	}

	out, err := h.service.GetSummary(r.Context(), usecase.SummaryInput{
		TenantID: tc.TenantID,
		DateFrom: fromPtr,
		DateTo:   toPtr,
	})
	if err != nil {
		writeUsecaseError(w, r, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, SummaryResponse{
		TotalIncomeMinor:   out.TotalIncomeMinor,
		TotalExpenseMinor:  out.TotalExpenseMinor,
		TotalTransferMinor: out.TotalTransferMinor,
		NetAmountMinor:     out.NetAmountMinor,
		TotalSavingsMinor:  out.TotalSavingsMinor,
		TransactionCount:   out.TransactionCount,
		IncomeCount:        out.IncomeCount,
		ExpenseCount:       out.ExpenseCount,
		TransferCount:      out.TransferCount,
		SavingsCount:       out.SavingsCount,
	}, middleware.GetRequestID(r.Context()))
}

func (h *Handler) Series(w http.ResponseWriter, r *http.Request) {
	tc, err := tenancy.FromContext(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "tenant context missing", middleware.GetRequestID(r.Context()))
		return
	}

	dateFrom := r.URL.Query().Get("from")
	dateTo := r.URL.Query().Get("to")
	if err := validateDate(dateFrom); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_DATE", "from must be YYYY-MM-DD", middleware.GetRequestID(r.Context()))
		return
	}
	if err := validateDate(dateTo); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_DATE", "to must be YYYY-MM-DD", middleware.GetRequestID(r.Context()))
		return
	}

	var fromPtr *string
	var toPtr *string
	if dateFrom != "" {
		fromPtr = &dateFrom
	}
	if dateTo != "" {
		toPtr = &dateTo
	}

	points, err := h.service.GetDailySeries(r.Context(), usecase.SeriesInput{
		TenantID: tc.TenantID,
		DateFrom: fromPtr,
		DateTo:   toPtr,
	})
	if err != nil {
		writeUsecaseError(w, r, err)
		return
	}

	response := make([]SeriesPointResponse, 0, len(points))
	for _, point := range points {
		response = append(response, SeriesPointResponse{
			Date:         point.Date.UTC().Format("2006-01-02"),
			IncomeMinor:  point.IncomeMinor,
			ExpenseMinor: point.ExpenseMinor,
		})
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": response}, middleware.GetRequestID(r.Context()))
}

func (h *Handler) TopCategories(w http.ResponseWriter, r *http.Request) {
	tc, err := tenancy.FromContext(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "tenant context missing", middleware.GetRequestID(r.Context()))
		return
	}

	dateFrom := r.URL.Query().Get("from")
	dateTo := r.URL.Query().Get("to")
	if err := validateDate(dateFrom); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_DATE", "from must be YYYY-MM-DD", middleware.GetRequestID(r.Context()))
		return
	}
	if err := validateDate(dateTo); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_DATE", "to must be YYYY-MM-DD", middleware.GetRequestID(r.Context()))
		return
	}

	limit := parseIntOrDefault(r.URL.Query().Get("limit"), 10)
	var fromPtr *string
	var toPtr *string
	if dateFrom != "" {
		fromPtr = &dateFrom
	}
	if dateTo != "" {
		toPtr = &dateTo
	}

	items, err := h.service.GetTopCategories(r.Context(), usecase.TopCategoriesInput{
		TenantID: tc.TenantID,
		DateFrom: fromPtr,
		DateTo:   toPtr,
		Limit:    limit,
	})
	if err != nil {
		writeUsecaseError(w, r, err)
		return
	}

	response := make([]CategoryTotalResponse, 0, len(items))
	for _, item := range items {
		response = append(response, CategoryTotalResponse{
			CategoryID:      item.CategoryID,
			CategoryName:    item.CategoryName,
			TransactionType: item.TransactionType,
			TotalMinor:      item.TotalMinor,
			Count:           item.Count,
		})
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": response}, middleware.GetRequestID(r.Context()))
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

func validateDate(raw string) error {
	if raw == "" {
		return nil
	}
	_, err := time.Parse("2006-01-02", raw)
	return err
}

func writeUsecaseError(w http.ResponseWriter, r *http.Request, err error) {
	requestID := middleware.GetRequestID(r.Context())
	switch {
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
