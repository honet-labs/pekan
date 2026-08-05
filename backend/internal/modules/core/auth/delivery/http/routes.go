package http

import (
	"time"

	"github.com/go-chi/chi/v5"

	"pekan/backend/internal/platform/middleware"
)

const (
	DefaultLoginRateLimitPerMinute   = 100
	DefaultRefreshRateLimitPerMinute = 200
)

func (h *Handler) RegisterPublicRoutes(r chi.Router) {
	loginLimiter := h.loginLimiter
	if loginLimiter == nil {
		loginLimiter = middleware.NewIPRateLimiter(DefaultLoginRateLimitPerMinute, time.Minute, h.auditLogger)
	}
	refreshLimiter := h.refreshLimiter
	if refreshLimiter == nil {
		refreshLimiter = middleware.NewIPRateLimiter(DefaultRefreshRateLimitPerMinute, time.Minute, h.auditLogger)
	}

	r.Route("/auth", func(r chi.Router) {
		r.With(loginLimiter).Post("/login", h.Login)
		r.With(loginLimiter).Post("/forgot-password", h.ForgotPassword)
		r.With(loginLimiter).Post("/forgot-tenant", h.ForgotTenant)
		r.With(loginLimiter).Post("/reset-password", h.ResetPassword)
		r.With(refreshLimiter).Post("/refresh", h.Refresh)

		// Self-registration with OTP
		r.With(loginLimiter).Post("/register/init", h.RegisterInit)
		r.With(loginLimiter).Post("/register/verify", h.RegisterVerify)
	})
}

func (h *Handler) RegisterProtectedRoutes(r chi.Router) {
	// Keep protected auth endpoints as direct routes to avoid mounting /auth twice
	// when public and protected routes are both registered on the same parent mux.
	r.Post("/auth/logout", h.Logout)
	r.Post("/auth/logout-all", h.LogoutAll)
	r.Get("/me/context", h.MeContext)
	r.Get("/me/profile", h.MeProfile)
	r.Put("/me/profile", h.UpdateMeProfile)
	r.Post("/me/change-password", h.ChangePassword)
	r.Post("/tenants/{tenantID}/switch", h.SwitchTenant)
}
