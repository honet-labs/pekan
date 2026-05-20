package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	entitlementdomain "pekan/backend/internal/modules/core/subscription/domain"
	"pekan/backend/internal/modules/core/subscription/usecase"
	"pekan/backend/internal/platform/access"
	"pekan/backend/internal/platform/httpx"
	"pekan/backend/internal/platform/middleware"
	"pekan/backend/internal/platform/tenancy"
)

type Handler struct {
	service Service
}

type Service interface {
	GetEffectiveEntitlements(ctx context.Context, tenantID string) (entitlementdomain.EffectiveEntitlements, error)
	SetFeatureOverride(ctx context.Context, in usecase.SetFeatureOverrideInput) error
}

func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

type SetFeatureOverrideRequest struct {
	FeatureCode string  `json:"feature_code"`
	IsEnabled   bool    `json:"is_enabled"`
	Reason      *string `json:"reason"`
	ExpiresAt   *string `json:"expires_at"`
}

func (h *Handler) GetEffectiveEntitlements(w http.ResponseWriter, r *http.Request) {
	targetTenant := chi.URLParam(r, "tenantID")
	tc, err := tenancy.FromContext(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "tenant context missing", middleware.GetRequestID(r.Context()))
		return
	}
	if targetTenant == "" {
		targetTenant = tc.TenantID
	}
	if targetTenant != tc.TenantID {
		httpx.WriteError(w, http.StatusForbidden, "FORBIDDEN_TENANT_SCOPE", "cannot read entitlements for other tenant", middleware.GetRequestID(r.Context()))
		return
	}

	out, err := h.service.GetEffectiveEntitlements(r.Context(), targetTenant)
	if err != nil {
		writeError(w, r, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"tenant_id": targetTenant,
		"modules":   out.Modules,
		"features":  out.Features,
	}, middleware.GetRequestID(r.Context()))
}

func (h *Handler) SetFeatureOverride(w http.ResponseWriter, r *http.Request) {
	targetTenant := chi.URLParam(r, "tenantID")
	tc, err := tenancy.FromContext(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "tenant context missing", middleware.GetRequestID(r.Context()))
		return
	}
	if targetTenant != tc.TenantID {
		httpx.WriteError(w, http.StatusForbidden, "FORBIDDEN_TENANT_SCOPE", "cannot set feature override for other tenant", middleware.GetRequestID(r.Context()))
		return
	}
	var req SetFeatureOverrideRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid payload", middleware.GetRequestID(r.Context()))
		return
	}

	var expiresAt *time.Time
	if req.ExpiresAt != nil && *req.ExpiresAt != "" {
		t, err := time.Parse(time.RFC3339, *req.ExpiresAt)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, "INVALID_EXPIRES_AT", "expires_at must RFC3339", middleware.GetRequestID(r.Context()))
			return
		}
		expiresAt = &t
	}

	if err := h.service.SetFeatureOverride(r.Context(), usecase.SetFeatureOverrideInput{
		TenantID:   targetTenant,
		FeatureCode: req.FeatureCode,
		IsEnabled:  req.IsEnabled,
		Reason:     req.Reason,
		ExpiresAt:  expiresAt,
	}); err != nil {
		writeError(w, r, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]any{"updated": true}, middleware.GetRequestID(r.Context()))
}

func writeError(w http.ResponseWriter, r *http.Request, err error) {
	requestID := middleware.GetRequestID(r.Context())
	switch {
	case errors.Is(err, entitlementdomain.ErrFeatureNotFound):
		httpx.WriteError(w, http.StatusBadRequest, "FEATURE_NOT_FOUND", err.Error(), requestID)
	case errors.Is(err, access.ErrPermissionDenied):
		httpx.WriteError(w, http.StatusForbidden, "FORBIDDEN_PERMISSION", err.Error(), requestID)
	default:
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error", requestID)
	}
}
