package http

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"pekan/backend/internal/modules/finance/transactions/domain"
	"pekan/backend/internal/modules/finance/transactions/usecase"
	"pekan/backend/internal/platform/access"
	"pekan/backend/internal/platform/httpx"
	"pekan/backend/internal/platform/middleware"
	"pekan/backend/internal/platform/tenancy"
)

const (
	maxAttachmentRequestBytes = 12 * 1024 * 1024
	maxAttachmentMemoryBytes  = 2 * 1024 * 1024
)

type Handler struct {
	service           Service
	attachmentService AttachmentService
}

type Service interface {
	Create(ctx context.Context, in usecase.CreateInput) (domain.Transaction, error)
	GetByID(ctx context.Context, tenantID, transactionID string) (domain.Transaction, error)
	List(ctx context.Context, in usecase.ListInput) ([]domain.Transaction, int64, error)
	Update(ctx context.Context, in usecase.UpdateInput) (domain.Transaction, error)
	Delete(ctx context.Context, tenantID, actorUserID, transactionID string) error
	ListBySavingsID(ctx context.Context, tenantID, savingsID string) ([]domain.Transaction, error)
}

type AttachmentService interface {
	Upload(ctx context.Context, in usecase.UploadAttachmentInput) (domain.Attachment, error)
	AttachFromScan(ctx context.Context, tenantID, actorUserID, transactionID, scanID string) (domain.Attachment, error)
	List(ctx context.Context, tenantID, transactionID string) ([]domain.Attachment, error)
	Download(ctx context.Context, tenantID, transactionID, attachmentID string) (usecase.DownloadAttachmentOutput, error)
	SetScanStatus(ctx context.Context, in usecase.SetAttachmentScanStatusInput) error
}

type SetAttachmentScanStatusRequest struct {
	ScanStatus string  `json:"scan_status"`
	Reason     *string `json:"reason"`
}

func NewHandler(service Service, attachmentService AttachmentService) *Handler {
	return &Handler{
		service:           service,
		attachmentService: attachmentService,
	}
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateTransactionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid JSON payload", middleware.GetRequestID(r.Context()))
		return
	}

	txDate, err := time.Parse("2006-01-02", req.TransactionDate)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_DATE", "transaction_date must be YYYY-MM-DD", middleware.GetRequestID(r.Context()))
		return
	}

	tc, err := tenancy.FromContext(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "tenant context missing", middleware.GetRequestID(r.Context()))
		return
	}

	out, err := h.service.Create(r.Context(), usecase.CreateInput{
		TenantID:             tc.TenantID,
		ActorUserID:          tc.UserID,
		AccountID:            req.AccountID,
		CategoryID:           req.CategoryID,
		CategoryName:         req.CategoryName,
		SavingsIDs:           req.SavingsIDs,
		Type:                 domain.TransactionType(req.Type),
		AmountMinor:          req.AmountMinor,
		Currency:             req.Currency,
		TransactionDate:      txDate,
		Description:          req.Description,
		MerchantName:         req.MerchantName,
		ReceiptNumber:        req.ReceiptNumber,
		PaymentMethod:        req.PaymentMethod,
		SubtotalMinor:        req.SubtotalMinor,
		TaxMinor:             req.TaxMinor,
		ServiceChargeMinor:   req.ServiceChargeMinor,
		ReceiptDiscountMinor: req.ReceiptDiscountMinor,
		Items:                mapTransactionItemRequests(req.Items),
		ScanID:               req.ReceiptScanID,
	})
	if err != nil {
		writeUsecaseError(w, r, err)
		return
	}

	if req.ReceiptScanID != nil && *req.ReceiptScanID != "" {
		_, _ = h.attachmentService.AttachFromScan(r.Context(), tc.TenantID, tc.UserID, out.ID, *req.ReceiptScanID)
	}

	httpx.WriteJSON(w, http.StatusCreated, toResponse(out), middleware.GetRequestID(r.Context()))
}

func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	transactionID := chi.URLParam(r, "transactionID")
	tc, err := tenancy.FromContext(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "tenant context missing", middleware.GetRequestID(r.Context()))
		return
	}

	out, err := h.service.GetByID(r.Context(), tc.TenantID, transactionID)
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

	var trxType *domain.TransactionType
	if v := r.URL.Query().Get("type"); v != "" {
		t := domain.TransactionType(v)
		trxType = &t
	}

	var dateFrom *time.Time
	if v := r.URL.Query().Get("from"); v != "" {
		parsed, err := time.Parse("2006-01-02", v)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, "INVALID_DATE", "from must be YYYY-MM-DD", middleware.GetRequestID(r.Context()))
			return
		}
		dateFrom = &parsed
	}

	var dateTo *time.Time
	if v := r.URL.Query().Get("to"); v != "" {
		parsed, err := time.Parse("2006-01-02", v)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, "INVALID_DATE", "to must be YYYY-MM-DD", middleware.GetRequestID(r.Context()))
			return
		}
		dateTo = &parsed
	}

	var catID *string
	if v := r.URL.Query().Get("category_id"); v != "" {
		catID = &v
	}

	items, total, err := h.service.List(r.Context(), usecase.ListInput{
		TenantID:   tc.TenantID,
		Type:       trxType,
		DateFrom:   dateFrom,
		DateTo:     dateTo,
		Query:      r.URL.Query().Get("q"),
		CategoryID: catID,
		Page:       page,
		PageSize:   pageSize,
	})
	if err != nil {
		writeUsecaseError(w, r, err)
		return
	}

	responseItems := make([]TransactionResponse, 0, len(items))
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

func (h *Handler) ListBySavingsID(w http.ResponseWriter, r *http.Request) {
	savingsID := chi.URLParam(r, "savingsID")
	tc, err := tenancy.FromContext(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "tenant context missing", middleware.GetRequestID(r.Context()))
		return
	}

	items, err := h.service.ListBySavingsID(r.Context(), tc.TenantID, savingsID)
	if err != nil {
		writeUsecaseError(w, r, err)
		return
	}

	responseItems := make([]TransactionResponse, 0, len(items))
	for _, item := range items {
		responseItems = append(responseItems, toResponse(item))
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"items": responseItems,
	}, middleware.GetRequestID(r.Context()))
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	transactionID := chi.URLParam(r, "transactionID")

	var req UpdateTransactionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid JSON payload", middleware.GetRequestID(r.Context()))
		return
	}

	txDate, err := time.Parse("2006-01-02", req.TransactionDate)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_DATE", "transaction_date must be YYYY-MM-DD", middleware.GetRequestID(r.Context()))
		return
	}

	tc, err := tenancy.FromContext(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "tenant context missing", middleware.GetRequestID(r.Context()))
		return
	}

	out, err := h.service.Update(r.Context(), usecase.UpdateInput{
		TenantID:             tc.TenantID,
		ActorUserID:          tc.UserID,
		TransactionID:        transactionID,
		AccountID:            req.AccountID,
		CategoryID:           req.CategoryID,
		CategoryName:         req.CategoryName,
		SavingsIDs:           req.SavingsIDs,
		Type:                 domain.TransactionType(req.Type),
		AmountMinor:          req.AmountMinor,
		Currency:             req.Currency,
		TransactionDate:      txDate,
		Description:          req.Description,
		MerchantName:         req.MerchantName,
		ReceiptNumber:        req.ReceiptNumber,
		PaymentMethod:        req.PaymentMethod,
		SubtotalMinor:        req.SubtotalMinor,
		TaxMinor:             req.TaxMinor,
		ServiceChargeMinor:   req.ServiceChargeMinor,
		ReceiptDiscountMinor: req.ReceiptDiscountMinor,
		Items:                mapTransactionItemRequests(req.Items),
	})
	if err != nil {
		writeUsecaseError(w, r, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, toResponse(out), middleware.GetRequestID(r.Context()))
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	transactionID := chi.URLParam(r, "transactionID")

	tc, err := tenancy.FromContext(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "tenant context missing", middleware.GetRequestID(r.Context()))
		return
	}

	if err := h.service.Delete(r.Context(), tc.TenantID, tc.UserID, transactionID); err != nil {
		writeUsecaseError(w, r, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]any{"deleted": true}, middleware.GetRequestID(r.Context()))
}

func (h *Handler) UploadAttachment(w http.ResponseWriter, r *http.Request) {
	transactionID := chi.URLParam(r, "transactionID")
	r.Body = http.MaxBytesReader(w, r.Body, maxAttachmentRequestBytes)
	if err := r.ParseMultipartForm(maxAttachmentMemoryBytes); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_MULTIPART", "invalid multipart payload", middleware.GetRequestID(r.Context()))
		return
	}

	fileHeaders := collectUploadFileHeaders(r)
	if len(fileHeaders) == 0 {
		httpx.WriteError(w, http.StatusBadRequest, "FILE_REQUIRED", "at least one file is required", middleware.GetRequestID(r.Context()))
		return
	}

	tc, err := tenancy.FromContext(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "tenant context missing", middleware.GetRequestID(r.Context()))
		return
	}

	responseItems := make([]map[string]any, 0, len(fileHeaders))
	for _, fileHeader := range fileHeaders {
		file, openErr := fileHeader.Open()
		if openErr != nil {
			httpx.WriteError(w, http.StatusBadRequest, "FILE_OPEN_FAILED", "failed to open uploaded file", middleware.GetRequestID(r.Context()))
			return
		}

		attachment, uploadErr := h.attachmentService.Upload(r.Context(), usecase.UploadAttachmentInput{
			TenantID:         tc.TenantID,
			ActorUserID:      tc.UserID,
			TransactionID:    transactionID,
			OriginalFilename: fileHeader.Filename,
			MimeType:         fileHeader.Header.Get("Content-Type"),
			SizeBytes:        fileHeader.Size,
			File:             file,
		})
		_ = file.Close()
		if uploadErr != nil {
			writeUsecaseError(w, r, uploadErr)
			return
		}

		responseItems = append(responseItems, map[string]any{
			"id":                attachment.ID,
			"transaction_id":    attachment.TransactionID,
			"original_filename": attachment.OriginalFilename,
			"mime_type":         attachment.MimeType,
			"scan_status":       attachment.ScanStatus,
			"size_bytes":        attachment.SizeBytes,
			"created_at":        attachment.CreatedAt.Format(time.RFC3339),
		})
	}

	httpx.WriteJSON(w, http.StatusCreated, map[string]any{
		"items": responseItems,
	}, middleware.GetRequestID(r.Context()))
}

func collectUploadFileHeaders(r *http.Request) []*multipart.FileHeader {
	keys := []string{"files", "files[]", "file", "file[]", "attachments", "attachments[]", "attachment", "attachment[]"}
	seen := make(map[string]struct{})
	items := make([]*multipart.FileHeader, 0)
	for _, key := range keys {
		headers := r.MultipartForm.File[key]
		for _, header := range headers {
			id := fmt.Sprintf("%s|%d", header.Filename, header.Size)
			if _, exists := seen[id]; exists {
				continue
			}
			seen[id] = struct{}{}
			items = append(items, header)
		}
	}
	return items
}

func (h *Handler) ListAttachments(w http.ResponseWriter, r *http.Request) {
	transactionID := chi.URLParam(r, "transactionID")

	tc, err := tenancy.FromContext(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "tenant context missing", middleware.GetRequestID(r.Context()))
		return
	}

	items, err := h.attachmentService.List(r.Context(), tc.TenantID, transactionID)
	if err != nil {
		writeUsecaseError(w, r, err)
		return
	}

	responseItems := make([]AttachmentResponse, 0, len(items))
	for _, item := range items {
		responseItems = append(responseItems, AttachmentResponse{
			ID:               item.ID,
			TransactionID:    item.TransactionID,
			OriginalFilename: item.OriginalFilename,
			MimeType:         item.MimeType,
			ScanStatus:       item.ScanStatus,
			SizeBytes:        item.SizeBytes,
			CreatedAt:        item.CreatedAt.UTC().Format(time.RFC3339),
		})
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"items": responseItems,
	}, middleware.GetRequestID(r.Context()))
}

func (h *Handler) DownloadAttachment(w http.ResponseWriter, r *http.Request) {
	transactionID := chi.URLParam(r, "transactionID")
	attachmentID := chi.URLParam(r, "attachmentID")

	tc, err := tenancy.FromContext(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "tenant context missing", middleware.GetRequestID(r.Context()))
		return
	}

	out, err := h.attachmentService.Download(r.Context(), tc.TenantID, transactionID, attachmentID)
	if err != nil {
		writeUsecaseError(w, r, err)
		return
	}
	defer out.Reader.Close()

	w.Header().Set("Content-Type", out.Attachment.MimeType)
	dispositionType := "attachment"
	if r.URL.Query().Get("view") == "1" || r.URL.Query().Get("disposition") == "inline" {
		dispositionType = "inline"
	}
	w.Header().Set("Content-Disposition", fmt.Sprintf("%s; filename=%q", dispositionType, out.Attachment.OriginalFilename))
	w.Header().Set("Cache-Control", "private, max-age=0, no-store")
	w.WriteHeader(http.StatusOK)
	_, _ = io.Copy(w, out.Reader)
}

func (h *Handler) SetAttachmentScanStatus(w http.ResponseWriter, r *http.Request) {
	transactionID := chi.URLParam(r, "transactionID")
	attachmentID := chi.URLParam(r, "attachmentID")

	tc, err := tenancy.FromContext(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "tenant context missing", middleware.GetRequestID(r.Context()))
		return
	}

	var req SetAttachmentScanStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid JSON payload", middleware.GetRequestID(r.Context()))
		return
	}

	if err := h.attachmentService.SetScanStatus(r.Context(), usecase.SetAttachmentScanStatusInput{
		TenantID:      tc.TenantID,
		TransactionID: transactionID,
		AttachmentID:  attachmentID,
		ScanStatus:    req.ScanStatus,
		Reason:        req.Reason,
	}); err != nil {
		writeUsecaseError(w, r, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]any{"updated": true}, middleware.GetRequestID(r.Context()))
}

func toResponse(trx domain.Transaction) TransactionResponse {
	items := make([]TransactionItemResponse, 0, len(trx.Items))
	for _, item := range trx.Items {
		items = append(items, TransactionItemResponse{
			ID:                item.ID,
			ItemName:          item.ItemName,
			Quantity:          item.Quantity,
			PriceMinor: item.PriceMinor,
			DiscountMinor:     item.DiscountMinor,
			TotalMinor:        item.TotalMinor,
			Notes:             item.Notes,
		})
	}
	return TransactionResponse{
		ID:                   trx.ID,
		TID:                  toShortID(trx.ID),
		AccountID:            trx.AccountID,
		AccountName:          trx.AccountName,
		CategoryID:           trx.CategoryID,
		CategoryName:         trx.CategoryName,
		SavingsIDs:           trx.SavingsIDs,
		SavingsNames:         trx.SavingsNames,
		Type:                 string(trx.Type),
		AmountMinor:          trx.AmountMinor,
		Currency:             trx.Currency,
		InputDate:            formatDateOnly(trx.InputDate),
		TransactionDate:      formatDateOnly(trx.TransactionDate),
		Description:          trx.Description,
		MerchantName:         trx.MerchantName,
		ReceiptNumber:        trx.ReceiptNumber,
		PaymentMethod:        trx.PaymentMethod,
		SubtotalMinor:        trx.SubtotalMinor,
		TaxMinor:             trx.TaxMinor,
		ServiceChargeMinor:   trx.ServiceChargeMinor,
		ReceiptDiscountMinor: trx.ReceiptDiscountMinor,
		CreatedBy:            trx.CreatedBy,
		CreatedByName:        trx.CreatedByName,
		CreatedAt:            trx.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:            trx.UpdatedAt.UTC().Format(time.RFC3339),
		Items:                items,
	}
}

func mapTransactionItemRequests(items []TransactionItemRequest) []domain.TransactionItem {
	out := make([]domain.TransactionItem, 0, len(items))
	for _, item := range items {
		id := item.ID
		if strings.HasPrefix(id, "temp-") {
			id = ""
		}
		out = append(out, domain.TransactionItem{
			ID:                id,
			ItemName:          item.ItemName,
			Quantity:          item.Quantity,
			PriceMinor: item.PriceMinor,
			DiscountMinor:     item.DiscountMinor,
			TotalMinor:        item.TotalMinor,
			Notes:             item.Notes,
		})
	}
	return out
}

func toShortID(id string) string {
	if len(id) < 8 {
		return strings.ToUpper(id)
	}
	return strings.ToUpper(id[:8])
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
	case errors.Is(err, domain.ErrInvalidAccount):
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_ACCOUNT", err.Error(), requestID)
	case errors.Is(err, domain.ErrInvalidAmount):
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_AMOUNT", err.Error(), requestID)
	case errors.Is(err, domain.ErrInvalidCurrency):
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_CURRENCY", err.Error(), requestID)
	case errors.Is(err, domain.ErrInvalidType):
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_TYPE", err.Error(), requestID)
	case errors.Is(err, domain.ErrInvalidSavingsSelection):
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_SAVINGS_SELECTION", err.Error(), requestID)
	case errors.Is(err, domain.ErrAccountNotFound):
		httpx.WriteError(w, http.StatusBadRequest, "ACCOUNT_NOT_FOUND", err.Error(), requestID)
	case errors.Is(err, domain.ErrCategoryNotFound):
		httpx.WriteError(w, http.StatusBadRequest, "CATEGORY_NOT_FOUND", err.Error(), requestID)
	case errors.Is(err, domain.ErrCategoryTypeMismatch):
		httpx.WriteError(w, http.StatusBadRequest, "CATEGORY_TYPE_MISMATCH", err.Error(), requestID)
	case errors.Is(err, domain.ErrSavingsNotFound):
		httpx.WriteError(w, http.StatusBadRequest, "SAVINGS_NOT_FOUND", err.Error(), requestID)
	case errors.Is(err, domain.ErrTransactionNotFound):
		httpx.WriteError(w, http.StatusNotFound, "NOT_FOUND", "transaction not found", requestID)
	case errors.Is(err, domain.ErrAttachmentNotFound):
		httpx.WriteError(w, http.StatusNotFound, "NOT_FOUND", "attachment not found", requestID)
	case errors.Is(err, domain.ErrInvalidFileType):
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_FILE_TYPE", err.Error(), requestID)
	case errors.Is(err, domain.ErrFileTooLarge):
		httpx.WriteError(w, http.StatusBadRequest, "FILE_TOO_LARGE", err.Error(), requestID)
	case errors.Is(err, domain.ErrInvalidScanStatus):
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_SCAN_STATUS", err.Error(), requestID)
	case errors.Is(err, domain.ErrAttachmentScanPending):
		httpx.WriteError(w, http.StatusLocked, "ATTACHMENT_SCAN_PENDING", err.Error(), requestID)
	case errors.Is(err, domain.ErrAttachmentInfected):
		httpx.WriteError(w, http.StatusForbidden, "ATTACHMENT_INFECTED", err.Error(), requestID)
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
