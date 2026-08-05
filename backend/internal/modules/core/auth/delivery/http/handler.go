package http

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	authdomain "pekan/backend/internal/modules/core/auth/domain"
	"pekan/backend/internal/modules/core/auth/usecase"
	"pekan/backend/internal/platform/audit"
	"pekan/backend/internal/platform/httpx"
	"pekan/backend/internal/platform/middleware"
	"pekan/backend/internal/platform/tenancy"
)

type Handler struct {
	service        Service
	loginLimiter   func(http.Handler) http.Handler
	refreshLimiter func(http.Handler) http.Handler
	auditLogger    audit.Logger
}

type Option func(*Handler)

type Service interface {
	Login(ctx context.Context, in usecase.LoginInput) (usecase.LoginOutput, error)
	Refresh(ctx context.Context, in usecase.RefreshInput) (usecase.RefreshOutput, error)
	Logout(ctx context.Context, refreshToken string) error
	LogoutAll(ctx context.Context, userID string) error
	GetContext(ctx context.Context, userID, tenantID string) (usecase.ContextOutput, error)
	GetProfile(ctx context.Context, userID string) (authdomain.UserProfile, error)
	UpdateProfile(ctx context.Context, in usecase.UpdateProfileInput) (authdomain.UserProfile, error)
	SwitchTenant(ctx context.Context, in usecase.SwitchTenantInput) (usecase.TokenOutput, error)
	RequestPasswordReset(ctx context.Context, email, tenantID, method string) error
	ForgotTenant(ctx context.Context, emailOrPhone, method string) error
	ResetPassword(ctx context.Context, email, tenantID, newPassword string) error
	RegisterInit(ctx context.Context, in usecase.RegisterInitInput) (usecase.RegisterInitOutput, error)
	RegisterVerify(ctx context.Context, in usecase.RegisterVerifyInput) error
}

func WithRateLimiters(loginLimiter, refreshLimiter func(http.Handler) http.Handler) Option {
	return func(h *Handler) {
		h.loginLimiter = loginLimiter
		h.refreshLimiter = refreshLimiter
	}
}

func WithAuditLogger(logger audit.Logger) Option {
	return func(h *Handler) {
		h.auditLogger = logger
	}
}

func NewHandler(service Service, opts ...Option) *Handler {
	h := &Handler{service: service}
	for _, opt := range opts {
		if opt != nil {
			opt(h)
		}
	}
	return h
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid login payload", middleware.GetRequestID(r.Context()))
		return
	}
	if req.Email == "" || req.Password == "" || strings.TrimSpace(req.TenantID) == "" {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "email, password, and tenant_id are required", middleware.GetRequestID(r.Context()))
		return
	}

	loginCtx := withAuditContext(r.Context(), r, req.TenantID, "")
	out, err := h.service.Login(loginCtx, usecase.LoginInput{
		Email:    req.Email,
		Password: req.Password,
		TenantID: req.TenantID,
	})
	if err != nil {
		if h.auditLogger != nil {
			go func() {
				defer func() { recover() }() // Prevent logging panic from affecting response
				// Use a safe context for failed login where tenant_id might be a code
				safeCtx := withAuditContext(context.Background(), r, "", "") 
				_ = h.auditLogger.Write(safeCtx, "auth.login.failed", "auth_login", strings.ToLower(strings.TrimSpace(req.Email)), nil, map[string]any{
					"input_tenant": req.TenantID,
					"reason":       err.Error(),
				})
			}()
		}
		writeAuthError(w, r, err)
		return
	}
	if h.auditLogger != nil {
		go func() {
			defer func() { recover() }()
			// Now we have the real TenantID (UUID) from the successful login output
			successCtx := withAuditContext(context.Background(), r, out.Membership.TenantID, out.User.ID)
			_ = h.auditLogger.Write(successCtx, "auth.login.success", "auth_login", out.User.ID, nil, map[string]any{
				"tenant_id": out.Membership.TenantID,
			})
		}()
	}
	
	setAuthCookies(w, r, out.Tokens.AccessToken, out.Tokens.RefreshToken, out.Tokens.AccessTokenExpiresAt, out.Tokens.RefreshTokenExpiresAt)

	httpx.WriteJSON(w, http.StatusOK, LoginResponse{
		AccessToken:           out.Tokens.AccessToken,
		RefreshToken:          out.Tokens.RefreshToken,
		TokenType:             "Bearer",
		AccessTokenExpiresAt:  out.Tokens.AccessTokenExpiresAt.Format(time.RFC3339),
		RefreshTokenExpiresAt: out.Tokens.RefreshTokenExpiresAt.Format(time.RFC3339),
		UserID:                out.User.ID,
		Email:                 out.User.Email,
		ActiveTenantID:        out.Membership.TenantID,
		Permissions:           out.Access.Permissions,
		Features:              out.Access.Features,
		Modules:               out.Access.Modules,
		MustChangePassword:    out.User.MustChangePassword,
	}, middleware.GetRequestID(r.Context()))
}

func withAuditContext(ctx context.Context, r *http.Request, tenantID, actorUserID string) context.Context {
	host := middleware.ClientIP(r)

	// Ensure tenantID is a valid UUID before putting it in AuditContext
	// to avoid DB errors in the audit logger
	finalTenantID := ""
	if len(tenantID) == 36 { // Simple UUID length check
		finalTenantID = tenantID
	}

	return audit.WithContext(ctx, audit.AuditContext{
		TenantID:    finalTenantID,
		ActorUserID: actorUserID,
		RequestID:   middleware.GetRequestID(ctx),
		IPAddress:   host,
		UserAgent:   r.UserAgent(),
	})
}

func (h *Handler) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	var req ForgotPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid forgot password payload", middleware.GetRequestID(r.Context()))
		return
	}
	if strings.TrimSpace(req.Email) == "" || strings.TrimSpace(req.TenantID) == "" {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "email and tenant_id are required", middleware.GetRequestID(r.Context()))
		return
	}
	method := strings.ToLower(strings.TrimSpace(req.Method))
	if method == "" {
		method = "email"
	}

	ctx := withAuditContext(r.Context(), r, strings.TrimSpace(req.TenantID), "")
	err := h.service.RequestPasswordReset(ctx, req.Email, req.TenantID, method)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "SEND_ERROR", err.Error(), middleware.GetRequestID(r.Context()))
		return
	}

	if h.auditLogger != nil {
		_ = h.auditLogger.Write(ctx, "auth.password_reset.requested", "auth_password_reset", strings.ToLower(strings.TrimSpace(req.Email)), nil, map[string]any{
			"tenant_id": strings.TrimSpace(req.TenantID),
			"method":    method,
		})
	}
	httpx.WriteJSON(w, http.StatusAccepted, ForgotPasswordResponse{
		Message: "Jika akun tersedia di workspace ini, instruksi pengaturan ulang telah dikirim.",
	}, middleware.GetRequestID(r.Context()))
}


func (h *Handler) ForgotTenant(w http.ResponseWriter, r *http.Request) {
	var req ForgotTenantRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid payload", middleware.GetRequestID(r.Context()))
		return
	}
	if strings.TrimSpace(req.EmailOrPhone) == "" {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "email_or_phone is required", middleware.GetRequestID(r.Context()))
		return
	}
	method := strings.ToLower(strings.TrimSpace(req.Method))
	if method == "" {
		method = "email"
	}

	ctx := withAuditContext(r.Context(), r, "", "")
	err := h.service.ForgotTenant(ctx, req.EmailOrPhone, method)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "SEND_ERROR", err.Error(), middleware.GetRequestID(r.Context()))
		return
	}

	httpx.WriteJSON(w, http.StatusAccepted, ForgotTenantResponse{
		Message: "Jika akun tersedia, informasi telah dikirim ke " + method + " Anda.",
	}, middleware.GetRequestID(r.Context()))
}


func (h *Handler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email       string `json:"email"`
		TenantID    string `json:"tenant_id"`
		NewPassword string `json:"new_password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid payload", middleware.GetRequestID(r.Context()))
		return
	}
	if strings.TrimSpace(req.Email) == "" || strings.TrimSpace(req.TenantID) == "" || strings.TrimSpace(req.NewPassword) == "" {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "email, tenant_id, and new_password are required", middleware.GetRequestID(r.Context()))
		return
	}

	ctx := withAuditContext(r.Context(), r, strings.TrimSpace(req.TenantID), "")
	err := h.service.ResetPassword(ctx, req.Email, req.TenantID, req.NewPassword)
	if err != nil {
		if errors.Is(err, authdomain.ErrInvalidCredentials) {
			httpx.WriteError(w, http.StatusNotFound, "NOT_FOUND", "Account not found in this workspace", middleware.GetRequestID(r.Context()))
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), middleware.GetRequestID(r.Context()))
		return
	}

	if h.auditLogger != nil {
		_ = h.auditLogger.Write(ctx, "auth.password_reset.success", "auth_password_reset", strings.ToLower(strings.TrimSpace(req.Email)), nil, map[string]any{
			"tenant_id": strings.TrimSpace(req.TenantID),
		})
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"message": "Password has been reset successfully. Please login with your new password.",
	}, middleware.GetRequestID(r.Context()))
}

func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req RefreshRequest
	var refreshToken string

	// Extract from Cookie first
	if cookie, err := r.Cookie("pekan_refresh_token"); err == nil {
		refreshToken = cookie.Value
	}

	// Fallback to JSON body for backward compatibility
	if refreshToken == "" {
		_ = json.NewDecoder(r.Body).Decode(&req)
		refreshToken = req.RefreshToken
	}

	if refreshToken == "" {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "refresh_token is required via cookie or payload", middleware.GetRequestID(r.Context()))
		return
	}

	out, err := h.service.Refresh(r.Context(), usecase.RefreshInput{RefreshToken: refreshToken})
	if err != nil {
		log.Printf("[Refresh] error: %v (RID: %s)", err, middleware.GetRequestID(r.Context()))
		writeAuthError(w, r, err)
		return
	}

	setAuthCookies(w, r, out.Tokens.AccessToken, out.Tokens.RefreshToken, out.Tokens.AccessTokenExpiresAt, out.Tokens.RefreshTokenExpiresAt)

	httpx.WriteJSON(w, http.StatusOK, LoginResponse{
		AccessToken:           out.Tokens.AccessToken,
		RefreshToken:          out.Tokens.RefreshToken,
		TokenType:             "Bearer",
		AccessTokenExpiresAt:  out.Tokens.AccessTokenExpiresAt.Format(time.RFC3339),
		RefreshTokenExpiresAt: out.Tokens.RefreshTokenExpiresAt.Format(time.RFC3339),
		UserID:                out.User.ID,
		Email:                 out.User.Email,
		ActiveTenantID:        out.Membership.TenantID,
		Permissions:           out.Access.Permissions,
		Features:              out.Access.Features,
		Modules:               out.Access.Modules,
		MustChangePassword:    out.User.MustChangePassword,
	}, middleware.GetRequestID(r.Context()))
}

func (h *Handler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	tc, err := tenancy.FromContext(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "unauthorized", middleware.GetRequestID(r.Context()))
		return
	}

	var req ChangePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid payload", middleware.GetRequestID(r.Context()))
		return
	}

	if len(req.NewPassword) < 8 {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "password must be at least 8 characters", middleware.GetRequestID(r.Context()))
		return
	}

	ctx := withAuditContext(r.Context(), r, tc.TenantID, tc.UserID)
	// We reuse ResetPassword usecase logic for simplicity as it handles hashing and updating
	err = h.service.ResetPassword(ctx, tc.Email, tc.TenantID, req.NewPassword)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), middleware.GetRequestID(r.Context()))
		return
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"success": true,
		"message": "Password changed successfully",
	}, middleware.GetRequestID(r.Context()))
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	var req LogoutRequest
	var refreshToken string

	if cookie, err := r.Cookie("pekan_refresh_token"); err == nil {
		refreshToken = cookie.Value
	}

	if refreshToken == "" {
		_ = json.NewDecoder(r.Body).Decode(&req)
		refreshToken = req.RefreshToken
	}

	if refreshToken == "" {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "refresh_token is required via cookie or payload", middleware.GetRequestID(r.Context()))
		return
	}

	if err := h.service.Logout(r.Context(), refreshToken); err != nil {
		writeAuthError(w, r, err)
		return
	}

	clearAuthCookies(w, r)
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"logged_out": true}, middleware.GetRequestID(r.Context()))
}

func (h *Handler) LogoutAll(w http.ResponseWriter, r *http.Request) {
	tc, err := tenancy.FromContext(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "tenant context missing", middleware.GetRequestID(r.Context()))
		return
	}
	if err := h.service.LogoutAll(r.Context(), tc.UserID); err != nil {
		writeAuthError(w, r, err)
		return
	}
	
	clearAuthCookies(w, r)
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"logged_out_all": true}, middleware.GetRequestID(r.Context()))
}

func (h *Handler) MeContext(w http.ResponseWriter, r *http.Request) {
	tc, err := tenancy.FromContext(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "tenant context missing", middleware.GetRequestID(r.Context()))
		return
	}

	out, err := h.service.GetContext(r.Context(), tc.UserID, tc.TenantID)
	if err != nil {
		writeAuthError(w, r, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"user": map[string]any{
			"id":    out.User.ID,
			"email": out.User.Email,
			"name":  out.User.FullName,
		},
		"active_tenant": map[string]any{
			"id":   out.Active.TenantID,
			"code": out.Active.TenantCode,
		},
		"memberships": out.Memberships,
		"permissions": out.Access.Permissions,
		"features":    out.Access.Features,
		"modules":     out.Access.Modules,
	}, middleware.GetRequestID(r.Context()))
}

func (h *Handler) MeProfile(w http.ResponseWriter, r *http.Request) {
	tc, err := tenancy.FromContext(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "tenant context missing", middleware.GetRequestID(r.Context()))
		return
	}

	out, err := h.service.GetProfile(r.Context(), tc.UserID)
	if err != nil {
		writeAuthError(w, r, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, ProfileResponse{
		UserID:    out.UserID,
		FullName:  out.FullName,
		Username:  out.Username,
		Email:     out.Email,
		Phone:     out.Phone,
		Address:   out.Address,
		UpdatedAt: out.UpdatedAt.UTC().Format(time.RFC3339),
	}, middleware.GetRequestID(r.Context()))
}

func (h *Handler) UpdateMeProfile(w http.ResponseWriter, r *http.Request) {
	tc, err := tenancy.FromContext(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "tenant context missing", middleware.GetRequestID(r.Context()))
		return
	}

	var req UpdateProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid profile payload", middleware.GetRequestID(r.Context()))
		return
	}

	out, err := h.service.UpdateProfile(r.Context(), usecase.UpdateProfileInput{
		UserID:   tc.UserID,
		Email:    req.Email,
		FullName: req.FullName,
		Username: req.Username,
		Phone:    req.Phone,
		Address:  req.Address,
	})
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_PROFILE", err.Error(), middleware.GetRequestID(r.Context()))
		return
	}

	httpx.WriteJSON(w, http.StatusOK, ProfileResponse{
		UserID:    out.UserID,
		FullName:  out.FullName,
		Username:  out.Username,
		Email:     out.Email,
		Phone:     out.Phone,
		Address:   out.Address,
		UpdatedAt: out.UpdatedAt.UTC().Format(time.RFC3339),
	}, middleware.GetRequestID(r.Context()))
}

func (h *Handler) SwitchTenant(w http.ResponseWriter, r *http.Request) {
	targetTenantID := chi.URLParam(r, "tenantID")
	tc, err := tenancy.FromContext(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "tenant context missing", middleware.GetRequestID(r.Context()))
		return
	}

	out, err := h.service.SwitchTenant(r.Context(), usecase.SwitchTenantInput{
		UserID:   tc.UserID,
		TenantID: targetTenantID,
		Email:    tc.Email,
	})
	if err != nil {
		writeAuthError(w, r, err)
		return
	}

	setAuthCookies(w, r, out.AccessToken, out.RefreshToken, out.AccessTokenExpiresAt, out.RefreshTokenExpiresAt)

	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"access_token":             out.AccessToken,
		"refresh_token":            out.RefreshToken,
		"token_type":               "Bearer",
		"access_token_expires_at":  out.AccessTokenExpiresAt.Format(time.RFC3339),
		"refresh_token_expires_at": out.RefreshTokenExpiresAt.Format(time.RFC3339),
		"active_tenant_id":         targetTenantID,
	}, middleware.GetRequestID(r.Context()))
}

func writeAuthError(w http.ResponseWriter, r *http.Request, err error) {
	requestID := middleware.GetRequestID(r.Context())
	switch {
	case errors.Is(err, authdomain.ErrInvalidCredentials):
		httpx.WriteError(w, http.StatusUnauthorized, "INVALID_CREDENTIALS", err.Error(), requestID)
	case errors.Is(err, authdomain.ErrUserInactive):
		httpx.WriteError(w, http.StatusForbidden, "USER_INACTIVE", err.Error(), requestID)
	case errors.Is(err, authdomain.ErrMembershipNotFound):
		httpx.WriteError(w, http.StatusForbidden, "MEMBERSHIP_NOT_FOUND", err.Error(), requestID)
	case errors.Is(err, authdomain.ErrMembershipSuspended):
		httpx.WriteError(w, http.StatusForbidden, "MEMBERSHIP_SUSPENDED", err.Error(), requestID)
	case errors.Is(err, authdomain.ErrSessionNotFound):
		httpx.WriteError(w, http.StatusUnauthorized, "INVALID_REFRESH_TOKEN", err.Error(), requestID)
	case errors.Is(err, authdomain.ErrSessionExpired):
		httpx.WriteError(w, http.StatusUnauthorized, "SESSION_EXPIRED", err.Error(), requestID)
	case errors.Is(err, authdomain.ErrSessionRevoked):
		httpx.WriteError(w, http.StatusUnauthorized, "SESSION_REVOKED", err.Error(), requestID)
	case errors.Is(err, authdomain.ErrRefreshTokenReused):
		httpx.WriteError(w, http.StatusUnauthorized, "REFRESH_TOKEN_REUSED", err.Error(), requestID)
	case errors.Is(err, authdomain.ErrAccessProfileMissing):
		httpx.WriteError(w, http.StatusForbidden, "ACCESS_PROFILE_MISSING", err.Error(), requestID)
	default:
		// Map password complexity errors to 400 Bad Request
		errStr := err.Error()
		if strings.Contains(errStr, "password must") {
			httpx.WriteError(w, http.StatusBadRequest, "INVALID_PASSWORD", errStr, requestID)
			return
		}
		// For other errors, include the error message in the response to help debugging
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error: "+errStr, requestID)
	}
}


// RegisterInit handles Step 1 of self-registration: validate form and send OTP
func (h *Handler) RegisterInit(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TenantCode string `json:"tenant_code"`
		TenantName string `json:"tenant_name"`
		AdminEmail string `json:"admin_email"`
		AdminName  string `json:"admin_name"`
		Password   string `json:"password"`
		Phone      string `json:"phone"`
		OTPMethod  string `json:"otp_method"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "payload tidak valid", middleware.GetRequestID(r.Context()))
		return
	}

	out, err := h.service.RegisterInit(r.Context(), usecase.RegisterInitInput{
		TenantCode: req.TenantCode,
		TenantName: req.TenantName,
		AdminEmail: req.AdminEmail,
		AdminName:  req.AdminName,
		Password:   req.Password,
		Phone:      req.Phone,
		OTPMethod:  req.OTPMethod,
	})
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "REGISTRATION_ERROR", err.Error(), middleware.GetRequestID(r.Context()))
		return
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"session_token": out.SessionToken,
		"otp_method":    out.OTPMethod,
		"message":       "Kode OTP telah dikirim. Masukkan kode untuk melanjutkan.",
	}, middleware.GetRequestID(r.Context()))
}

// RegisterVerify handles Step 2 of self-registration: verify OTP and create tenant
func (h *Handler) RegisterVerify(w http.ResponseWriter, r *http.Request) {
	var req struct {
		SessionToken string `json:"session_token"`
		OTPCode      string `json:"otp_code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "payload tidak valid", middleware.GetRequestID(r.Context()))
		return
	}

	if strings.TrimSpace(req.SessionToken) == "" || strings.TrimSpace(req.OTPCode) == "" {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "session_token dan otp_code wajib diisi", middleware.GetRequestID(r.Context()))
		return
	}

	if err := h.service.RegisterVerify(r.Context(), usecase.RegisterVerifyInput{
		SessionToken: req.SessionToken,
		OTPCode:      req.OTPCode,
	}); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "REGISTRATION_ERROR", err.Error(), middleware.GetRequestID(r.Context()))
		return
	}

	httpx.WriteJSON(w, http.StatusCreated, map[string]any{
		"success": true,
		"message": "Workspace berhasil dibuat! Silakan login dengan email dan password Anda.",
	}, middleware.GetRequestID(r.Context()))
}

func setAuthCookies(w http.ResponseWriter, r *http.Request, access, refresh string, accessExp, refreshExp time.Time) {
	secure := r.TLS != nil || strings.ToLower(r.Header.Get("X-Forwarded-Proto")) == "https"
	
	http.SetCookie(w, &http.Cookie{
		Name:     "pekan_access_token",
		Value:    access,
		Expires:  accessExp,
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     "pekan_refresh_token",
		Value:    refresh,
		Expires:  refreshExp,
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}

func clearAuthCookies(w http.ResponseWriter, r *http.Request) {
	secure := r.TLS != nil || strings.ToLower(r.Header.Get("X-Forwarded-Proto")) == "https"
	past := time.Unix(0, 0)
	
	http.SetCookie(w, &http.Cookie{
		Name:     "pekan_access_token",
		Value:    "",
		Expires:  past,
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     "pekan_refresh_token",
		Value:    "",
		Expires:  past,
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}
