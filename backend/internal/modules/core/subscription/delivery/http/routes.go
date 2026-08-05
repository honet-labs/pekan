package http

import "github.com/go-chi/chi/v5"

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Get("/tenants/{tenantID}/entitlements/effective", h.GetEffectiveEntitlements)
	r.Post("/tenants/{tenantID}/feature-overrides", h.SetFeatureOverride)
}
