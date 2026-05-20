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

	"pekan/backend/internal/modules/finance/reports/domain"
	"pekan/backend/internal/modules/finance/reports/usecase"
	"pekan/backend/internal/platform/access"
	"pekan/backend/internal/platform/httpx"
	"pekan/backend/internal/platform/middleware"
	"pekan/backend/internal/platform/tenancy"
)

type Service interface {
	CreateTransactionsReport(ctx context.Context, in usecase.CreateTransactionsReportInput) (domain.Report, error)
	GetByID(ctx context.Context, tenantID, reportID string) (domain.Report, error)
	List(ctx context.Context, in usecase.ListInput) ([]domain.Report, int64, error)
	Download(ctx context.Context, tenantID, reportID string) (domain.Report, io.ReadCloser, error)
	Delete(ctx context.Context, tenantID, actorUserID, reportID string) error
}

type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) CreateTransactionsReport(w http.ResponseWriter, r *http.Request) {
	var req CreateTransactionsReportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid JSON payload", middleware.GetRequestID(r.Context()))
		return
	}

	if err := validateDatePtr(req.DateFrom); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_DATE", "date_from must be YYYY-MM-DD", middleware.GetRequestID(r.Context()))
		return
	}
	if err := validateDatePtr(req.DateTo); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_DATE", "date_to must be YYYY-MM-DD", middleware.GetRequestID(r.Context()))
		return
	}
	if req.DateFrom != nil && req.DateTo != nil && *req.DateFrom != "" && *req.DateTo != "" {
		fromDate, _ := time.Parse("2006-01-02", *req.DateFrom)
		toDate, _ := time.Parse("2006-01-02", *req.DateTo)
		if toDate.Before(fromDate) {
			httpx.WriteError(w, http.StatusBadRequest, "INVALID_DATE_RANGE", "date_to must be after date_from", middleware.GetRequestID(r.Context()))
			return
		}
	}
	reportType := strings.ToLower(strings.TrimSpace(req.ReportType))
	if reportType == "" {
		reportType = "transactions"
	}
	switch reportType {
	case "transactions", "savings", "budgets", "reminders":
	default:
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_REPORT_TYPE", "report_type must be transactions|savings|budgets|reminders", middleware.GetRequestID(r.Context()))
		return
	}

	if reportType == "transactions" && req.Type != nil && *req.Type != "" {
		switch *req.Type {
		case "income", "expense", "transfer", "savings":
		default:
			httpx.WriteError(w, http.StatusBadRequest, "INVALID_TYPE", "type must be income|expense|transfer|savings", middleware.GetRequestID(r.Context()))
			return
		}
	}

	tc, err := tenancy.FromContext(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "tenant context missing", middleware.GetRequestID(r.Context()))
		return
	}

	out, err := h.service.CreateTransactionsReport(r.Context(), usecase.CreateTransactionsReportInput{
		TenantID:    tc.TenantID,
		ActorUserID: tc.UserID,
		ReportType:  reportType,
		DateFrom:    req.DateFrom,
		DateTo:      req.DateTo,
		CategoryID:  req.CategoryID,
		Type:        req.Type,
		Status:      req.Status,
		Format:      req.Format,
	})
	if err != nil {
		writeUsecaseError(w, r, err)
		return
	}

	httpx.WriteJSON(w, http.StatusCreated, toResponse(out), middleware.GetRequestID(r.Context()))
}

func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	reportID := chi.URLParam(r, "reportID")
	tc, err := tenancy.FromContext(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "tenant context missing", middleware.GetRequestID(r.Context()))
		return
	}

	out, err := h.service.GetByID(r.Context(), tc.TenantID, reportID)
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

	items, total, err := h.service.List(r.Context(), usecase.ListInput{
		TenantID: tc.TenantID,
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		writeUsecaseError(w, r, err)
		return
	}

	responseItems := make([]ReportResponse, 0, len(items))
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

func (h *Handler) Download(w http.ResponseWriter, r *http.Request) {
	reportID := chi.URLParam(r, "reportID")
	tc, err := tenancy.FromContext(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "tenant context missing", middleware.GetRequestID(r.Context()))
		return
	}

	report, reader, err := h.service.Download(r.Context(), tc.TenantID, reportID)
	if err != nil {
		writeUsecaseError(w, r, err)
		return
	}
	defer reader.Close()

	contentType := "application/octet-stream"
	switch report.Format {
	case "csv":
		contentType = "text/csv"
	case "pdf":
		contentType = "application/pdf"
	}
	w.Header().Set("Content-Type", contentType)
	filename := "report." + report.Format
	if report.ReportType != "" {
		filename = report.ReportType + "." + report.Format
	}
	w.Header().Set("Content-Disposition", "attachment; filename="+strconv.Quote(filename))
	w.Header().Set("Cache-Control", "private, max-age=0, no-store")
	w.WriteHeader(http.StatusOK)
	_, _ = io.Copy(w, reader)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	reportID := chi.URLParam(r, "reportID")
	tc, err := tenancy.FromContext(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "tenant context missing", middleware.GetRequestID(r.Context()))
		return
	}

	if err := h.service.Delete(r.Context(), tc.TenantID, tc.UserID, reportID); err != nil {
		writeUsecaseError(w, r, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]any{"deleted": true}, middleware.GetRequestID(r.Context()))
}

func toResponse(item domain.Report) ReportResponse {
	return ReportResponse{
		ID:         item.ID,
		ReportType: item.ReportType,
		Format:     item.Format,
		Status:     item.Status,
		StorageKey: item.StorageKey,
		CreatedAt:  item.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:  item.UpdatedAt.UTC().Format(time.RFC3339),
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

func validateDatePtr(raw *string) error {
	if raw == nil || *raw == "" {
		return nil
	}
	_, err := time.Parse("2006-01-02", *raw)
	return err
}

func writeUsecaseError(w http.ResponseWriter, r *http.Request, err error) {
	requestID := middleware.GetRequestID(r.Context())
	switch {
	case errors.Is(err, domain.ErrInvalidFormat):
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_FORMAT", err.Error(), requestID)
	case errors.Is(err, domain.ErrInvalidReportType):
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_REPORT_TYPE", err.Error(), requestID)
	case errors.Is(err, domain.ErrInvalidDateRange):
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_DATE_RANGE", err.Error(), requestID)
	case errors.Is(err, domain.ErrReportNotFound):
		httpx.WriteError(w, http.StatusNotFound, "NOT_FOUND", "report not found", requestID)
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
