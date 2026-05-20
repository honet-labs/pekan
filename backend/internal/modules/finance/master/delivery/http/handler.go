package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	masterdomain "pekan/backend/internal/modules/finance/master/domain"
	"pekan/backend/internal/modules/finance/master/usecase"
	"pekan/backend/internal/platform/access"
	"pekan/backend/internal/platform/httpx"
	"pekan/backend/internal/platform/middleware"
	"pekan/backend/internal/platform/tenancy"
)

type Handler struct {
	service Service
}

type Service interface {
	CreateAccount(ctx context.Context, in usecase.CreateAccountInput) (masterdomain.Account, error)
	ListAccounts(ctx context.Context, tenantID string) ([]masterdomain.Account, error)
	CreateCategory(ctx context.Context, in usecase.CreateCategoryInput) (masterdomain.Category, error)
	ListCategories(ctx context.Context, tenantID string) ([]masterdomain.Category, error)
	UpdateCategory(ctx context.Context, tenantID, categoryID string, in usecase.CreateCategoryInput) (masterdomain.Category, error)
	DeleteCategory(ctx context.Context, tenantID, categoryID string) error
	GetCategory(ctx context.Context, tenantID, categoryID string) (masterdomain.Category, error)
}

func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

type CreateAccountRequest struct {
	Name               string `json:"name"`
	AccountType        string `json:"account_type"`
	Currency           string `json:"currency"`
	OpeningBalanceMinor int64 `json:"opening_balance_minor"`
}

type CreateCategoryRequest struct {
	Name         string  `json:"name"`
	CategoryType string  `json:"category_type"`
	ParentID     *string `json:"parent_id"`
}

func (h *Handler) CreateAccount(w http.ResponseWriter, r *http.Request) {
	var req CreateAccountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid account payload", middleware.GetRequestID(r.Context()))
		return
	}
	tc, err := tenancy.FromContext(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "tenant context missing", middleware.GetRequestID(r.Context()))
		return
	}
	out, err := h.service.CreateAccount(r.Context(), usecase.CreateAccountInput{
		TenantID:            tc.TenantID,
		ActorUserID:         tc.UserID,
		Name:                req.Name,
		AccountType:         req.AccountType,
		Currency:            req.Currency,
		OpeningBalanceMinor: req.OpeningBalanceMinor,
	})
	if err != nil {
		writeError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, toAccountResponse(out), middleware.GetRequestID(r.Context()))
}

func (h *Handler) ListAccounts(w http.ResponseWriter, r *http.Request) {
	tc, err := tenancy.FromContext(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "tenant context missing", middleware.GetRequestID(r.Context()))
		return
	}
	out, err := h.service.ListAccounts(r.Context(), tc.TenantID)
	if err != nil {
		writeError(w, r, err)
		return
	}
	items := make([]map[string]any, 0, len(out))
	for _, row := range out {
		items = append(items, toAccountResponse(row))
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": items}, middleware.GetRequestID(r.Context()))
}

func (h *Handler) CreateCategory(w http.ResponseWriter, r *http.Request) {
	var req CreateCategoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid category payload", middleware.GetRequestID(r.Context()))
		return
	}
	tc, err := tenancy.FromContext(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "tenant context missing", middleware.GetRequestID(r.Context()))
		return
	}
	out, err := h.service.CreateCategory(r.Context(), usecase.CreateCategoryInput{
		TenantID:     tc.TenantID,
		ActorUserID:  tc.UserID,
		Name:         req.Name,
		CategoryType: req.CategoryType,
		ParentID:     req.ParentID,
	})
	if err != nil {
		writeError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, toCategoryResponse(out), middleware.GetRequestID(r.Context()))
}

func (h *Handler) ListCategories(w http.ResponseWriter, r *http.Request) {
	tc, err := tenancy.FromContext(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "tenant context missing", middleware.GetRequestID(r.Context()))
		return
	}
	out, err := h.service.ListCategories(r.Context(), tc.TenantID)
	if err != nil {
		writeError(w, r, err)
		return
	}
	items := make([]map[string]any, 0, len(out))
	for _, row := range out {
		items = append(items, toCategoryResponse(row))
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": items}, middleware.GetRequestID(r.Context()))
}

func (h *Handler) UpdateCategory(w http.ResponseWriter, r *http.Request) {
	var req CreateCategoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid category payload", middleware.GetRequestID(r.Context()))
		return
	}
	tc, err := tenancy.FromContext(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "tenant context missing", middleware.GetRequestID(r.Context()))
		return
	}
	id := chi.URLParam(r, "id")
	out, err := h.service.UpdateCategory(r.Context(), tc.TenantID, id, usecase.CreateCategoryInput{
		TenantID:     tc.TenantID,
		ActorUserID:  tc.UserID,
		Name:         req.Name,
		CategoryType: req.CategoryType,
		ParentID:     req.ParentID,
	})
	if err != nil {
		writeError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toCategoryResponse(out), middleware.GetRequestID(r.Context()))
}

func (h *Handler) DeleteCategory(w http.ResponseWriter, r *http.Request) {
	tc, err := tenancy.FromContext(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "tenant context missing", middleware.GetRequestID(r.Context()))
		return
	}
	id := chi.URLParam(r, "id")
	if err := h.service.DeleteCategory(r.Context(), tc.TenantID, id); err != nil {
		writeError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"success": true}, middleware.GetRequestID(r.Context()))
}

func (h *Handler) GetCategory(w http.ResponseWriter, r *http.Request) {
	tc, err := tenancy.FromContext(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "tenant context missing", middleware.GetRequestID(r.Context()))
		return
	}
	id := chi.URLParam(r, "id")
	out, err := h.service.GetCategory(r.Context(), tc.TenantID, id)
	if err != nil {
		writeError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toCategoryResponse(out), middleware.GetRequestID(r.Context()))
}

func toAccountResponse(v masterdomain.Account) map[string]any {
	return map[string]any{
		"id":                   v.ID,
		"name":                 v.Name,
		"account_type":         v.AccountType,
		"currency":             v.Currency,
		"opening_balance_minor": v.OpeningBalanceMinor,
		"is_active":            v.IsActive,
	}
}

func toCategoryResponse(v masterdomain.Category) map[string]any {
	return map[string]any{
		"id":            v.ID,
		"name":          v.Name,
		"category_type": v.CategoryType,
		"parent_id":     v.ParentID,
		"is_active":     v.IsActive,
	}
}

func writeError(w http.ResponseWriter, r *http.Request, err error) {
	requestID := middleware.GetRequestID(r.Context())
	switch {
	case errors.Is(err, masterdomain.ErrInvalidAccountName):
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_ACCOUNT_NAME", err.Error(), requestID)
	case errors.Is(err, masterdomain.ErrInvalidAccountType):
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_ACCOUNT_TYPE", err.Error(), requestID)
	case errors.Is(err, masterdomain.ErrInvalidCurrency):
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_CURRENCY", err.Error(), requestID)
	case errors.Is(err, masterdomain.ErrInvalidCategoryName):
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_CATEGORY_NAME", err.Error(), requestID)
	case errors.Is(err, masterdomain.ErrInvalidCategoryType):
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_CATEGORY_TYPE", err.Error(), requestID)
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
