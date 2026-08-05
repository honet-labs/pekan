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

	"pekan/backend/internal/modules/finance/receipts/domain"
	"pekan/backend/internal/modules/finance/receipts/usecase"
	"pekan/backend/internal/platform/httpx"
	"pekan/backend/internal/platform/middleware"
	"pekan/backend/internal/platform/tenancy"
)

type Service interface {
	ListProviderConfigs(ctx context.Context, tenantID string) ([]domain.ProviderConfig, error)
	UpsertProviderConfigs(ctx context.Context, inputs []usecase.UpsertProviderInput) ([]domain.ProviderConfig, error)
	TestProviderConnection(ctx context.Context, in usecase.TestProviderConnectionInput) (usecase.TestProviderConnectionOutput, error)
	GetConfigStatus(ctx context.Context, tenantID string) (usecase.ConfigStatusOutput, error)
	ScanReceipt(ctx context.Context, in usecase.ScanReceiptInput) (usecase.ScanReceiptOutput, error)
	ListReceiptScans(ctx context.Context, tenantID string, limit int) ([]domain.ReceiptScan, error)
	DeleteReceiptScan(ctx context.Context, tenantID, scanID string) error
	ClearReceiptScans(ctx context.Context, tenantID string) error
	GetReceiptScanImage(ctx context.Context, tenantID, scanID string) ([]byte, string, error)
}

type Handler struct{ service Service }

func NewHandler(service Service) *Handler { return &Handler{service: service} }

func (h *Handler) ListProviders(w http.ResponseWriter, r *http.Request) {
	tc, err := tenancy.FromContext(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "tenant context missing", middleware.GetRequestID(r.Context()))
		return
	}
	items, err := h.service.ListProviderConfigs(r.Context(), tc.TenantID)
	if err != nil {
		writeUsecaseError(w, r, err)
		return
	}
	out := make([]ProviderConfigResponse, 0, len(items))
	for _, item := range items {
		out = append(out, ProviderConfigResponse{ProviderCode: item.ProviderCode, DisplayName: item.DisplayName, BaseURL: item.BaseURL, ModelName: item.ModelName, IsEnabled: item.IsEnabled, HasAPIKey: item.HasAPIKey})
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": out}, middleware.GetRequestID(r.Context()))
}

func (h *Handler) UpdateProviders(w http.ResponseWriter, r *http.Request) {
	var req UpsertProviderConfigsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid JSON payload", middleware.GetRequestID(r.Context()))
		return
	}
	tc, err := tenancy.FromContext(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "tenant context missing", middleware.GetRequestID(r.Context()))
		return
	}
	inputs := make([]usecase.UpsertProviderInput, 0, len(req.Items))
	for _, item := range req.Items {
		inputs = append(inputs, usecase.UpsertProviderInput{TenantID: tc.TenantID, ActorUserID: tc.UserID, ProviderCode: item.ProviderCode, DisplayName: item.DisplayName, BaseURL: item.BaseURL, ModelName: item.ModelName, IsEnabled: item.IsEnabled, APIKey: item.APIKey, ClearAPIKey: item.ClearAPIKey})
	}
	updated, err := h.service.UpsertProviderConfigs(r.Context(), inputs)
	if err != nil {
		writeUsecaseError(w, r, err)
		return
	}
	out := make([]ProviderConfigResponse, 0, len(updated))
	for _, item := range updated {
		out = append(out, ProviderConfigResponse{ProviderCode: item.ProviderCode, DisplayName: item.DisplayName, BaseURL: item.BaseURL, ModelName: item.ModelName, IsEnabled: item.IsEnabled, HasAPIKey: item.HasAPIKey})
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": out}, middleware.GetRequestID(r.Context()))
}

func (h *Handler) TestProviderConnection(w http.ResponseWriter, r *http.Request) {
	var req TestProviderConnectionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid JSON payload", middleware.GetRequestID(r.Context()))
		return
	}
	tc, err := tenancy.FromContext(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "tenant context missing", middleware.GetRequestID(r.Context()))
		return
	}
	baseURL := ""
	if req.BaseURL != nil {
		baseURL = strings.TrimSpace(*req.BaseURL)
	}
	out, err := h.service.TestProviderConnection(r.Context(), usecase.TestProviderConnectionInput{
		TenantID:     tc.TenantID,
		ProviderCode: req.ProviderCode,
		BaseURL:      baseURL,
		APIKey:       req.APIKey,
		ModelName:    req.ModelName,
	})
	if err != nil {
		writeUsecaseError(w, r, err)
		return
	}
	models := make([]ProviderModelOptionResponse, 0, len(out.Models))
	for _, item := range out.Models {
		models = append(models, ProviderModelOptionResponse{ID: item.ID, Label: item.Label})
	}
	httpx.WriteJSON(w, http.StatusOK, TestProviderConnectionResponse{
		ProviderCode:     out.ProviderCode,
		BaseURL:          out.BaseURL,
		UsingSavedAPIKey: out.UsingSavedAPIKey,
		Models:           models,
	}, middleware.GetRequestID(r.Context()))
}

func (h *Handler) GetStatus(w http.ResponseWriter, r *http.Request) {
	tc, err := tenancy.FromContext(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "tenant context missing", middleware.GetRequestID(r.Context()))
		return
	}
	out, err := h.service.GetConfigStatus(r.Context(), tc.TenantID)
	if err != nil {
		writeUsecaseError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, ConfigStatusResponse{HasConfiguredProvider: out.HasConfiguredProvider, ActiveProviders: out.ActiveProviders}, middleware.GetRequestID(r.Context()))
}

func (h *Handler) ListHistory(w http.ResponseWriter, r *http.Request) {
	tc, err := tenancy.FromContext(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "tenant context missing", middleware.GetRequestID(r.Context()))
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	items, err := h.service.ListReceiptScans(r.Context(), tc.TenantID, limit)
	if err != nil {
		writeUsecaseError(w, r, err)
		return
	}
	out := make([]ReceiptScanResponse, 0, len(items))
	for _, item := range items {
		out = append(out, ReceiptScanResponse{ID: item.ID, ProviderCode: item.ProviderCode, ModelName: item.ModelName, Status: item.Status, OriginalFilename: item.OriginalFilename, MimeType: item.MimeType, ExtractedJSON: item.ExtractedJSON, ErrorMessage: item.ErrorMessage, CreatedAt: item.CreatedAt.UTC().Format(time.RFC3339)})
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": out}, middleware.GetRequestID(r.Context()))
}

func (h *Handler) DeleteHistoryItem(w http.ResponseWriter, r *http.Request) {
	tc, err := tenancy.FromContext(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "tenant context missing", middleware.GetRequestID(r.Context()))
		return
	}
	id := chi.URLParam(r, "id")
	if err := h.service.DeleteReceiptScan(r.Context(), tc.TenantID, id); err != nil {
		writeUsecaseError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"success": true}, middleware.GetRequestID(r.Context()))
}

func (h *Handler) ClearHistory(w http.ResponseWriter, r *http.Request) {
	tc, err := tenancy.FromContext(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "tenant context missing", middleware.GetRequestID(r.Context()))
		return
	}
	if err := h.service.ClearReceiptScans(r.Context(), tc.TenantID); err != nil {
		writeUsecaseError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"success": true}, middleware.GetRequestID(r.Context()))
}

func (h *Handler) GetReceiptImage(w http.ResponseWriter, r *http.Request) {
	tc, err := tenancy.FromContext(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "tenant context missing", middleware.GetRequestID(r.Context()))
		return
	}
	id := chi.URLParam(r, "id")
	data, mimeType, err := h.service.GetReceiptScanImage(r.Context(), tc.TenantID, id)
	if err != nil {
		writeUsecaseError(w, r, err)
		return
	}
	w.Header().Set("Content-Type", mimeType)
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

func (h *Handler) ScanReceipt(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(12 << 20); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid multipart payload", middleware.GetRequestID(r.Context()))
		return
	}
	tc, err := tenancy.FromContext(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "tenant context missing", middleware.GetRequestID(r.Context()))
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "FILE_REQUIRED", "receipt file is required", middleware.GetRequestID(r.Context()))
		return
	}
	defer file.Close()
	content, err := io.ReadAll(file)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_FILE", "failed to read receipt file", middleware.GetRequestID(r.Context()))
		return
	}
	mimeType := header.Header.Get("Content-Type")
	if mimeType == "" {
		mimeType = http.DetectContentType(content)
	}
	out, err := h.service.ScanReceipt(r.Context(), usecase.ScanReceiptInput{TenantID: tc.TenantID, ActorUserID: tc.UserID, ProviderCode: r.FormValue("provider_code"), OriginalFilename: header.Filename, MimeType: mimeType, Content: content})
	if err != nil {
		writeUsecaseError(w, r, err)
		return
	}
	draftItems := make([]DraftTransactionItemResponse, 0, len(out.Draft.Items))
	for _, item := range out.Draft.Items {
		draftItems = append(draftItems, DraftTransactionItemResponse{
			ItemName:          item.ItemName,
			Quantity:          item.Quantity,
			PriceMinor:        item.PriceMinor,
			DiscountMinor:     item.DiscountMinor,
			TotalMinor:        item.TotalMinor,
			Notes:             item.Notes,
		})
	}
	httpx.WriteJSON(w, http.StatusOK, ScanReceiptOutputResponse{
		Scan:  ReceiptScanResponse{ID: out.Scan.ID, ProviderCode: out.Scan.ProviderCode, ModelName: out.Scan.ModelName, Status: out.Scan.Status, OriginalFilename: out.Scan.OriginalFilename, MimeType: out.Scan.MimeType, ExtractedJSON: out.Scan.ExtractedJSON, ErrorMessage: out.Scan.ErrorMessage, CreatedAt: out.Scan.CreatedAt.UTC().Format(time.RFC3339)},
		Draft: DraftTransactionResponse{Type: out.Draft.Type, CategoryName: out.Draft.CategoryName, AmountMinor: out.Draft.AmountMinor, Currency: out.Draft.Currency, Date: out.Draft.Date, Description: out.Draft.Description, MerchantName: out.Draft.MerchantName, ReceiptNumber: out.Draft.ReceiptNumber, PaymentMethod: out.Draft.PaymentMethod, SubtotalMinor: out.Draft.SubtotalMinor, TaxMinor: out.Draft.TaxMinor, ServiceChargeMinor: out.Draft.ServiceChargeMinor, ReceiptDiscountMinor: out.Draft.ReceiptDiscountMinor, Confidence: out.Draft.Confidence, Items: draftItems},
	}, middleware.GetRequestID(r.Context()))
}

func writeUsecaseError(w http.ResponseWriter, r *http.Request, err error) {
	requestID := middleware.GetRequestID(r.Context())
	switch {
	case errors.Is(err, domain.ErrProviderCredentialInvalid):
		httpx.WriteError(w, http.StatusPreconditionFailed, "RECEIPT_PROVIDER_CREDENTIAL_INVALID", "saved API key cannot be decrypted. please save the API key again", requestID)
	case errors.Is(err, domain.ErrProviderNotConfigured), errors.Is(err, domain.ErrNoConfiguredProvider):
		httpx.WriteError(w, http.StatusPreconditionFailed, "RECEIPT_PROVIDER_NOT_CONFIGURED", err.Error(), requestID)
	case errors.Is(err, domain.ErrInvalidProvider):
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_PROVIDER", err.Error(), requestID)
	case errors.Is(err, domain.ErrInvalidFile):
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_FILE", "receipt file must be JPG, PNG, or WEBP", requestID)
	default:
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), requestID)
	}
}
