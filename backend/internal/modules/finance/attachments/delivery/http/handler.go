package http

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"pekan/backend/internal/modules/finance/attachments/domain"
	"pekan/backend/internal/modules/finance/attachments/usecase"
	"pekan/backend/internal/platform/access"
	"pekan/backend/internal/platform/httpx"
	"pekan/backend/internal/platform/middleware"
	"pekan/backend/internal/platform/tenancy"
)

const (
	maxAttachmentRequestBytes = 24 * 1024 * 1024
	maxAttachmentMemoryBytes  = 4 * 1024 * 1024
)

type Service interface {
	Upload(ctx context.Context, in usecase.UploadInput) (domain.Attachment, error)
	List(ctx context.Context, in usecase.ListInput) ([]domain.Attachment, error)
	Download(ctx context.Context, tenantID, ownerType, ownerID, attachmentID string) (usecase.DownloadOutput, error)
	Delete(ctx context.Context, tenantID, actorUserID, ownerType, ownerID, attachmentID string) error
}

type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Upload(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxAttachmentRequestBytes)
	if err := r.ParseMultipartForm(maxAttachmentMemoryBytes); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_MULTIPART", "invalid multipart payload", middleware.GetRequestID(r.Context()))
		return
	}

	ownerType := strings.TrimSpace(r.FormValue("owner_type"))
	ownerID := strings.TrimSpace(r.FormValue("owner_id"))
	if ownerType == "" || ownerID == "" {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_OWNER", "owner_type and owner_id are required", middleware.GetRequestID(r.Context()))
		return
	}

	files := collectUploadFileHeaders(r.MultipartForm)
	if len(files) == 0 {
		httpx.WriteError(w, http.StatusBadRequest, "FILE_REQUIRED", "at least one file is required", middleware.GetRequestID(r.Context()))
		return
	}

	tc, err := tenancy.FromContext(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "tenant context missing", middleware.GetRequestID(r.Context()))
		return
	}

	items := make([]AttachmentResponse, 0, len(files))
	for _, fileHeader := range files {
		file, openErr := fileHeader.Open()
		if openErr != nil {
			httpx.WriteError(w, http.StatusBadRequest, "INVALID_FILE", "cannot read uploaded file", middleware.GetRequestID(r.Context()))
			return
		}

		attachment, uploadErr := h.service.Upload(r.Context(), usecase.UploadInput{
			TenantID:         tc.TenantID,
			ActorUserID:      tc.UserID,
			OwnerType:        ownerType,
			OwnerID:          ownerID,
			OriginalFilename: fileHeader.Filename,
			File:             file,
		})
		_ = file.Close()
		if uploadErr != nil {
			writeUsecaseError(w, r, uploadErr)
			return
		}

		items = append(items, toResponse(attachment))
	}

	httpx.WriteJSON(w, http.StatusCreated, map[string]any{
		"items": items,
	}, middleware.GetRequestID(r.Context()))
}

func collectUploadFileHeaders(form *multipart.Form) []*multipart.FileHeader {
	keys := []string{"files", "files[]", "file", "file[]", "attachments", "attachments[]", "attachment", "attachment[]"}
	seen := make(map[string]struct{})
	items := make([]*multipart.FileHeader, 0)
	for _, key := range keys {
		headers := form.File[key]
		for _, header := range headers {
			fingerprint := fmt.Sprintf("%s|%d|%s", header.Filename, header.Size, header.Header.Get("Content-Type"))
			if _, exists := seen[fingerprint]; exists {
				continue
			}
			seen[fingerprint] = struct{}{}
			items = append(items, header)
		}
	}
	return items
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	ownerType := strings.TrimSpace(r.URL.Query().Get("owner_type"))
	ownerID := strings.TrimSpace(r.URL.Query().Get("owner_id"))
	if ownerType == "" || ownerID == "" {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_OWNER", "owner_type and owner_id are required", middleware.GetRequestID(r.Context()))
		return
	}

	tc, err := tenancy.FromContext(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "tenant context missing", middleware.GetRequestID(r.Context()))
		return
	}

	items, err := h.service.List(r.Context(), usecase.ListInput{
		TenantID:  tc.TenantID,
		OwnerType: ownerType,
		OwnerID:   ownerID,
	})
	if err != nil {
		writeUsecaseError(w, r, err)
		return
	}

	responseItems := make([]AttachmentResponse, 0, len(items))
	for _, item := range items {
		responseItems = append(responseItems, toResponse(item))
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": responseItems}, middleware.GetRequestID(r.Context()))
}

func (h *Handler) Download(w http.ResponseWriter, r *http.Request) {
	attachmentID := chi.URLParam(r, "attachmentID")
	ownerType := strings.TrimSpace(r.URL.Query().Get("owner_type"))
	ownerID := strings.TrimSpace(r.URL.Query().Get("owner_id"))
	if ownerType == "" || ownerID == "" {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_OWNER", "owner_type and owner_id are required", middleware.GetRequestID(r.Context()))
		return
	}

	tc, err := tenancy.FromContext(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "tenant context missing", middleware.GetRequestID(r.Context()))
		return
	}

	out, err := h.service.Download(r.Context(), tc.TenantID, ownerType, ownerID, attachmentID)
	if err != nil {
		writeUsecaseError(w, r, err)
		return
	}
	defer out.Reader.Close()

	disposition := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("disposition")))
	if r.URL.Query().Get("view") == "1" || disposition == "inline" {
		disposition = "inline"
	} else {
		disposition = "attachment"
	}

	filename := strconv.Quote(out.Attachment.OriginalFilename)
	w.Header().Set("Content-Type", out.Attachment.MimeType)
	w.Header().Set("Content-Disposition", fmt.Sprintf("%s; filename=%s", disposition, filename))
	w.Header().Set("Cache-Control", "private, max-age=0, no-store")
	w.WriteHeader(http.StatusOK)
	_, _ = io.Copy(w, out.Reader)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	attachmentID := chi.URLParam(r, "attachmentID")
	ownerType := strings.TrimSpace(r.URL.Query().Get("owner_type"))
	ownerID := strings.TrimSpace(r.URL.Query().Get("owner_id"))
	if ownerType == "" || ownerID == "" {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_OWNER", "owner_type and owner_id are required", middleware.GetRequestID(r.Context()))
		return
	}

	tc, err := tenancy.FromContext(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "tenant context missing", middleware.GetRequestID(r.Context()))
		return
	}

	if err := h.service.Delete(r.Context(), tc.TenantID, tc.UserID, ownerType, ownerID, attachmentID); err != nil {
		writeUsecaseError(w, r, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]any{"deleted": true}, middleware.GetRequestID(r.Context()))
}

func toResponse(item domain.Attachment) AttachmentResponse {
	return AttachmentResponse{
		ID:               item.ID,
		OwnerType:        string(item.OwnerType),
		OwnerID:          item.OwnerID,
		OriginalFilename: item.OriginalFilename,
		MimeType:         item.MimeType,
		ScanStatus:       item.ScanStatus,
		SizeBytes:        item.SizeBytes,
		CreatedAt:        item.CreatedAt.UTC().Format(time.RFC3339),
	}
}

func writeUsecaseError(w http.ResponseWriter, r *http.Request, err error) {
	requestID := middleware.GetRequestID(r.Context())
	switch {
	case errors.Is(err, domain.ErrInvalidOwnerType):
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_OWNER_TYPE", err.Error(), requestID)
	case errors.Is(err, domain.ErrOwnerNotFound):
		httpx.WriteError(w, http.StatusNotFound, "OWNER_NOT_FOUND", err.Error(), requestID)
	case errors.Is(err, domain.ErrAttachmentNotFound):
		httpx.WriteError(w, http.StatusNotFound, "ATTACHMENT_NOT_FOUND", err.Error(), requestID)
	case errors.Is(err, domain.ErrInvalidFileType):
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_FILE_TYPE", err.Error(), requestID)
	case errors.Is(err, domain.ErrFileTooLarge):
		httpx.WriteError(w, http.StatusBadRequest, "FILE_TOO_LARGE", err.Error(), requestID)
	case errors.Is(err, domain.ErrAttachmentInfected):
		httpx.WriteError(w, http.StatusForbidden, "ATTACHMENT_INFECTED", err.Error(), requestID)
	case errors.Is(err, domain.ErrAttachmentScanPending):
		httpx.WriteError(w, http.StatusConflict, "ATTACHMENT_SCAN_PENDING", err.Error(), requestID)
	case errors.Is(err, access.ErrModuleDisabled):
		httpx.WriteError(w, http.StatusForbidden, "FORBIDDEN_MODULE_DISABLED", err.Error(), requestID)
	case errors.Is(err, access.ErrFeatureLocked):
		httpx.WriteError(w, http.StatusForbidden, "FORBIDDEN_FEATURE_LOCKED", err.Error(), requestID)
	case errors.Is(err, access.ErrPermissionDenied):
		httpx.WriteError(w, http.StatusForbidden, "FORBIDDEN_PERMISSION", err.Error(), requestID)
	default:
		log.Printf("[ERROR] internal server error [request_id=%s]: %v", requestID, err)
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error", requestID)
	}
}
