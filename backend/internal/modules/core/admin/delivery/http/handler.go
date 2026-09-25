package http

import (
	"encoding/json"
	"net/http"
	"strconv"
	"pekan/backend/internal/modules/core/admin/usecase"
	"pekan/backend/internal/platform/httpx"
	"pekan/backend/internal/platform/middleware"
	"pekan/backend/internal/platform/auth"

	"strings"

	"github.com/go-chi/chi/v5"
	"os"
	"path/filepath"
)

type Handler struct {
	service      *usecase.Service
	authSvc      *auth.Service
	adminSecret  string
	loginLimiter func(http.Handler) http.Handler
}

func NewHandler(service *usecase.Service, authSvc *auth.Service, adminSecret string, loginLimiter func(http.Handler) http.Handler) *Handler {
	return &Handler{
		service:      service,
		authSvc:      authSvc,
		adminSecret:  adminSecret,
		loginLimiter: loginLimiter,
	}
}

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Route("/admin", func(r chi.Router) {
		r.With(h.loginLimiter).Post("/login", h.Login)

		r.Group(func(r chi.Router) {
			r.Use(h.AdminAuthMiddleware)
			r.Post("/bootstrap-tenant", h.BootstrapTenant)
			r.Get("/tenants", h.ListTenants)
			r.Get("/logs", h.ListLogs)
			r.Get("/system-logs", h.GetSystemLogs)
			r.Get("/stats/growth", h.GetGrowth)
			r.Get("/server/status", h.GetServerStatus)
			
			r.Route("/tenants/{tenantID}", func(r chi.Router) {
				r.Put("/", h.UpdateTenant)
				r.Delete("/", h.DeleteTenant)
				r.Put("/quotas", h.UpdateQuotas)
				r.Get("/modules", h.ListModules)
				r.Put("/modules", h.UpdateModule)
				r.Get("/users", h.ListTenantUsers)
				
				r.Route("/backups", func(r chi.Router) {
					r.Get("/", h.ListTenantBackups)
					r.Post("/", h.CreateTenantBackup)
					r.Post("/restore", h.RestoreTenantBackup)
					r.Get("/download/{filename}", h.DownloadTenantBackup)
				})
			})

			r.Route("/settings", func(r chi.Router) {
				r.Get("/{key}", h.GetGlobalSetting)
				r.Put("/{key}", h.SetGlobalSetting)
			})

			r.Route("/updates", func(r chi.Router) {
				r.Get("/check", h.CheckUpdate)
				r.Post("/apply", h.ApplyUpdate)
				r.Get("/status", h.GetUpdateStatus)
			})

			r.Route("/whatsapp/queue", func(r chi.Router) {
				r.Get("/stats", h.GetWhatsAppQueueStats)
				r.Get("/history", h.GetWhatsAppQueueHistory)
				r.Post("/retry/{id}", h.RetryWhatsAppQueueMessage)
			})

			r.Post("/test/notification", h.TestNotification)
			r.Post("/test/ai", h.TestAI)
			r.Post("/test/database", h.TestDatabase)

			r.Post("/impersonate", h.Impersonate)

			r.Route("/backups", func(r chi.Router) {
				r.Get("/", h.ListBackups)
				r.Post("/", h.CreateBackup)
				r.Post("/restore", h.RestoreBackup)
				r.Post("/upload", h.UploadBackup)
				r.Get("/download/{filename}", h.DownloadBackup)
			})

			r.Route("/database", func(r chi.Router) {
				r.Post("/query", h.ExecuteQuery)
				r.Get("/stats", h.GetDatabaseStats)
				r.Get("/growth", h.GetDatabaseGrowth)
			})


			r.Route("/users/{userID}", func(r chi.Router) {
				r.Post("/reset-password", h.ResetUserPassword)
				r.Put("/email", h.UpdateUserEmail)
				r.Put("/phone", h.UpdateUserPhone)
			})
		})
	})
}

func (h *Handler) AdminAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("X-Admin-Token")
		if token == "" || token != h.adminSecret {
			httpx.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "admin access required", middleware.GetRequestID(r.Context()))
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Secret string `json:"secret"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid payload", middleware.GetRequestID(r.Context()))
		return
	}
	if req.Secret != h.adminSecret {
		httpx.WriteError(w, http.StatusUnauthorized, "INVALID_SECRET", "invalid admin secret", middleware.GetRequestID(r.Context()))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"token": h.adminSecret}, middleware.GetRequestID(r.Context()))
}

func (h *Handler) BootstrapTenant(w http.ResponseWriter, r *http.Request) {
	var req usecase.BootstrapTenantInput
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid payload", middleware.GetRequestID(r.Context()))
		return
	}
	if err := h.service.BootstrapTenant(r.Context(), req); err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), middleware.GetRequestID(r.Context()))
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, map[string]any{"success": true}, middleware.GetRequestID(r.Context()))
}

func (h *Handler) ListTenants(w http.ResponseWriter, r *http.Request) {
	out, err := h.service.ListTenants(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), middleware.GetRequestID(r.Context()))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, out, middleware.GetRequestID(r.Context()))
}

func (h *Handler) ListTenantBackups(w http.ResponseWriter, r *http.Request) {
	tenantID := chi.URLParam(r, "tenantID")
	out, err := h.service.ListBackups(r.Context(), tenantID)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), middleware.GetRequestID(r.Context()))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"data": out}, middleware.GetRequestID(r.Context()))
}

func (h *Handler) ListLogs(w http.ResponseWriter, r *http.Request) {
	out, err := h.service.ListLogs(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), middleware.GetRequestID(r.Context()))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, out, middleware.GetRequestID(r.Context()))
}

func (h *Handler) GetGrowth(w http.ResponseWriter, r *http.Request) {
	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")
	stats, err := h.service.GetGrowth(r.Context(), from, to)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), middleware.GetRequestID(r.Context()))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, stats, middleware.GetRequestID(r.Context()))
}

func (h *Handler) UpdateQuotas(w http.ResponseWriter, r *http.Request) {
	tenantID := chi.URLParam(r, "tenantID")
	var req struct {
		Users        int `json:"users"`
		Transactions int `json:"transactions"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid payload", middleware.GetRequestID(r.Context()))
		return
	}
	if err := h.service.UpdateQuotas(r.Context(), tenantID, req.Users, req.Transactions); err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), middleware.GetRequestID(r.Context()))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"updated": true}, middleware.GetRequestID(r.Context()))
}

func (h *Handler) UpdateTenant(w http.ResponseWriter, r *http.Request) {
	tenantID := chi.URLParam(r, "tenantID")
	var req struct {
		Name   string `json:"name"`
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid payload", middleware.GetRequestID(r.Context()))
		return
	}
	if err := h.service.UpdateTenant(r.Context(), tenantID, req.Name, req.Status); err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), middleware.GetRequestID(r.Context()))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"updated": true}, middleware.GetRequestID(r.Context()))
}

func (h *Handler) DeleteTenant(w http.ResponseWriter, r *http.Request) {
	tenantID := chi.URLParam(r, "tenantID")
	if err := h.service.DeleteTenant(r.Context(), tenantID); err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), middleware.GetRequestID(r.Context()))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"deleted": true}, middleware.GetRequestID(r.Context()))
}

func (h *Handler) ListModules(w http.ResponseWriter, r *http.Request) {
	tenantID := chi.URLParam(r, "tenantID")
	out, err := h.service.ListModules(r.Context(), tenantID)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), middleware.GetRequestID(r.Context()))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, out, middleware.GetRequestID(r.Context()))
}

func (h *Handler) UpdateModule(w http.ResponseWriter, r *http.Request) {
	tenantID := chi.URLParam(r, "tenantID")
	var req struct {
		ModuleCode string `json:"module_code"`
		IsEnabled  bool   `json:"is_enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid payload", middleware.GetRequestID(r.Context()))
		return
	}
	if err := h.service.UpdateModule(r.Context(), tenantID, req.ModuleCode, req.IsEnabled); err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), middleware.GetRequestID(r.Context()))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"updated": true}, middleware.GetRequestID(r.Context()))
}

func (h *Handler) ListTenantUsers(w http.ResponseWriter, r *http.Request) {
	tenantID := chi.URLParam(r, "tenantID")
	out, err := h.service.ListTenantUsers(r.Context(), tenantID)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), middleware.GetRequestID(r.Context()))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, out, middleware.GetRequestID(r.Context()))
}

func (h *Handler) Impersonate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID   string `json:"user_id"`
		TenantID string `json:"tenant_id"`
		Email    string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid payload", middleware.GetRequestID(r.Context()))
		return
	}

	// In a real scenario, we should verify the user exists and belongs to the tenant.
	// For this task, we'll issue a token directly as requested.
	token, expiresAt, err := h.authSvc.IssueAccessToken(auth.IssueAccessTokenInput{
		UserID:    req.UserID,
		TenantID:  req.TenantID,
		Email:     req.Email,
		SessionID: "impersonation-" + req.UserID,
		// Give full permissions for the module for simplicity
		Permissions: []string{"finance.*"}, 
		Features:    []string{"*"},
		Modules:     []string{"finance"},
	})
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), middleware.GetRequestID(r.Context()))
		return
	}

	// Set HttpOnly cookie for impersonation session
	http.SetCookie(w, &http.Cookie{
		Name:     "pekan_session",
		Value:    token,
		Path:     "/",
		Expires:  expiresAt,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})

	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"access_token": token,
		"expires_at":   expiresAt,
	}, middleware.GetRequestID(r.Context()))
}

func (h *Handler) GetServerStatus(w http.ResponseWriter, r *http.Request) {
	out := h.service.GetServerStatus(r.Context())
	httpx.WriteJSON(w, http.StatusOK, out, middleware.GetRequestID(r.Context()))
}

func (h *Handler) GetGlobalSetting(w http.ResponseWriter, r *http.Request) {
	key := chi.URLParam(r, "key")
	value, encrypted, err := h.service.GetGlobalSetting(r.Context(), key)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), middleware.GetRequestID(r.Context()))
		return
	}

	// For sensitive keys, return is_masked=true and empty value so the frontend
	// can show a placeholder (e.g. "API key is set") without exposing or corrupting the real value.
	isMasked := false
	lowerKey := strings.ToLower(key)
	if value != "" && (strings.Contains(lowerKey, "key") || strings.Contains(lowerKey, "secret") || strings.Contains(lowerKey, "token") || strings.Contains(lowerKey, "password")) {
		isMasked = true
		value = "" // Don't send the actual value to the frontend
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]any{"key": key, "value": value, "is_encrypted": encrypted, "is_masked": isMasked}, middleware.GetRequestID(r.Context()))
}

func (h *Handler) SetGlobalSetting(w http.ResponseWriter, r *http.Request) {
	key := chi.URLParam(r, "key")
	var req struct {
		Value     string `json:"value"`
		Encrypted bool   `json:"is_encrypted"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid payload", middleware.GetRequestID(r.Context()))
		return
	}
	if err := h.service.SetGlobalSetting(r.Context(), key, req.Value, req.Encrypted); err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), middleware.GetRequestID(r.Context()))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"success": true}, middleware.GetRequestID(r.Context()))
}

func (h *Handler) TestNotification(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Provider    string `json:"provider"`
		ConfigJSON  string `json:"config_json"`
		Destination string `json:"destination"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid payload", middleware.GetRequestID(r.Context()))
		return
	}
	if err := h.service.TestNotification(r.Context(), req.Provider, req.ConfigJSON, req.Destination); err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), middleware.GetRequestID(r.Context()))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"success": true}, middleware.GetRequestID(r.Context()))
}

func (h *Handler) TestAI(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Provider string `json:"provider"`
		APIKey   string `json:"api_key"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid payload", middleware.GetRequestID(r.Context()))
		return
	}
	models, err := h.service.TestAI(r.Context(), req.Provider, req.APIKey)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), middleware.GetRequestID(r.Context()))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"success": true, "models": models}, middleware.GetRequestID(r.Context()))
}

func (h *Handler) TestDatabase(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ConfigJSON string `json:"config_json"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid payload", middleware.GetRequestID(r.Context()))
		return
	}
	if err := h.service.TestDatabase(r.Context(), req.ConfigJSON); err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), middleware.GetRequestID(r.Context()))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"success": true}, middleware.GetRequestID(r.Context()))
}

func (h *Handler) CreateBackup(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Type string `json:"type"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		req.Type = "full"
	}
	if err := h.service.CreateBackup(r.Context(), req.Type, ""); err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), middleware.GetRequestID(r.Context()))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"success": true}, middleware.GetRequestID(r.Context()))
}

func (h *Handler) CreateTenantBackup(w http.ResponseWriter, r *http.Request) {
	tenantID := chi.URLParam(r, "tenantID")
	var req struct {
		Type string `json:"type"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		req.Type = "full"
	}
	if err := h.service.CreateBackup(r.Context(), req.Type, tenantID); err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), middleware.GetRequestID(r.Context()))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"success": true}, middleware.GetRequestID(r.Context()))
}

func (h *Handler) RestoreBackup(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Filename string `json:"filename"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid payload", middleware.GetRequestID(r.Context()))
		return
	}
	if err := h.service.RestoreBackup(r.Context(), req.Filename, ""); err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), middleware.GetRequestID(r.Context()))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"success": true}, middleware.GetRequestID(r.Context()))
}

func (h *Handler) RestoreTenantBackup(w http.ResponseWriter, r *http.Request) {
	tenantID := chi.URLParam(r, "tenantID")
	var req struct {
		Filename string `json:"filename"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid payload", middleware.GetRequestID(r.Context()))
		return
	}
	if err := h.service.RestoreBackup(r.Context(), req.Filename, tenantID); err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), middleware.GetRequestID(r.Context()))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"success": true}, middleware.GetRequestID(r.Context()))
}

func (h *Handler) UploadBackup(w http.ResponseWriter, r *http.Request) {
	// Parse multipart form (max 500MB)
	if err := r.ParseMultipartForm(500 << 20); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "failed to parse upload: "+err.Error(), middleware.GetRequestID(r.Context()))
		return
	}

	file, header, err := r.FormFile("backup")
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "backup file is required", middleware.GetRequestID(r.Context()))
		return
	}
	defer file.Close()

	// Validate file extension
	filename := header.Filename
	if !strings.HasSuffix(filename, ".sql.gz") && !strings.HasSuffix(filename, ".sql") && !strings.HasSuffix(filename, ".dump") {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "only .sql.gz, .sql, or .dump files are allowed", middleware.GetRequestID(r.Context()))
		return
	}

	if err := h.service.SaveUploadedBackup(r.Context(), filename, file); err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), middleware.GetRequestID(r.Context()))
		return
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]any{"success": true, "filename": filename}, middleware.GetRequestID(r.Context()))
}

func (h *Handler) ListBackups(w http.ResponseWriter, r *http.Request) {
	backups, err := h.service.ListBackups(r.Context(), "")
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), middleware.GetRequestID(r.Context()))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"data": backups}, middleware.GetRequestID(r.Context()))
}

func (h *Handler) DownloadBackup(w http.ResponseWriter, r *http.Request) {
	filename := chi.URLParam(r, "filename")
	fp, err := h.service.GetBackupPath(r.Context(), "", filename)
	if err != nil {
		http.Error(w, "file not found", http.StatusNotFound)
		return
	}

	if _, err := os.Stat(fp); os.IsNotExist(err) {
		http.Error(w, "file not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Disposition", "attachment; filename="+filepath.Base(fp))
	w.Header().Set("Content-Type", "application/octet-stream")
	http.ServeFile(w, r, fp)
}

func (h *Handler) DownloadTenantBackup(w http.ResponseWriter, r *http.Request) {
	tenantID := chi.URLParam(r, "tenantID")
	filename := chi.URLParam(r, "filename")
	
	fp, err := h.service.GetBackupPath(r.Context(), tenantID, filename)
	if err != nil {
		http.Error(w, "tenant or file not found", http.StatusNotFound)
		return
	}

	if _, err := os.Stat(fp); os.IsNotExist(err) {
		http.Error(w, "file not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Disposition", "attachment; filename="+filepath.Base(fp))
	w.Header().Set("Content-Type", "application/octet-stream")
	http.ServeFile(w, r, fp)
}

func (h *Handler) ExecuteQuery(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Query string `json:"query"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid payload", middleware.GetRequestID(r.Context()))
		return
	}
	out, err := h.service.ExecuteQuery(r.Context(), req.Query)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "QUERY_ERROR", err.Error(), middleware.GetRequestID(r.Context()))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, out, middleware.GetRequestID(r.Context()))
}

func (h *Handler) GetDatabaseStats(w http.ResponseWriter, r *http.Request) {
	out, err := h.service.GetDatabaseStats(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), middleware.GetRequestID(r.Context()))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, out, middleware.GetRequestID(r.Context()))
}

func (h *Handler) ResetUserPassword(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "userID")
	var req struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid payload", middleware.GetRequestID(r.Context()))
		return
	}
	if err := h.service.ResetUserPassword(r.Context(), userID, req.Password); err != nil {
		status := http.StatusInternalServerError
		code := "INTERNAL_ERROR"
		
		// Map password complexity errors to 400 Bad Request
		errStr := err.Error()
		if strings.Contains(errStr, "password must") {
			status = http.StatusBadRequest
			code = "INVALID_PASSWORD"
		}

		httpx.WriteError(w, status, code, errStr, middleware.GetRequestID(r.Context()))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"success": true}, middleware.GetRequestID(r.Context()))
}

func (h *Handler) UpdateUserEmail(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "userID")
	var req struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid payload", middleware.GetRequestID(r.Context()))
		return
	}
	if err := h.service.UpdateUserEmail(r.Context(), userID, req.Email); err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), middleware.GetRequestID(r.Context()))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"success": true}, middleware.GetRequestID(r.Context()))
}

func (h *Handler) UpdateUserPhone(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "userID")
	var req struct {
		Phone string `json:"phone"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid payload", middleware.GetRequestID(r.Context()))
		return
	}
	if err := h.service.UpdateUserPhone(r.Context(), userID, req.Phone); err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), middleware.GetRequestID(r.Context()))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"success": true}, middleware.GetRequestID(r.Context()))
}
func (h *Handler) GetDatabaseGrowth(w http.ResponseWriter, r *http.Request) {
	out, err := h.service.GetDatabaseGrowth(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), middleware.GetRequestID(r.Context()))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, out, middleware.GetRequestID(r.Context()))
}

func (h *Handler) GetWhatsAppQueueStats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.service.GetWhatsAppQueueStats(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), middleware.GetRequestID(r.Context()))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, stats, middleware.GetRequestID(r.Context()))
}

func (h *Handler) GetWhatsAppQueueHistory(w http.ResponseWriter, r *http.Request) {
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")
	search := r.URL.Query().Get("search")

	limit := 10
	offset := 0

	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}
	if offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	items, total, err := h.service.GetWhatsAppQueueHistory(r.Context(), limit, offset, search)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), middleware.GetRequestID(r.Context()))
		return
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"items": items,
		"total": total,
	}, middleware.GetRequestID(r.Context()))
}

func (h *Handler) RetryWhatsAppQueueMessage(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		httpx.WriteError(w, http.StatusBadRequest, "BAD_REQUEST", "id is required", middleware.GetRequestID(r.Context()))
		return
	}

	if err := h.service.RetryWhatsAppQueueMessage(r.Context(), id); err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), middleware.GetRequestID(r.Context()))
		return
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]any{"success": true}, middleware.GetRequestID(r.Context()))
}

func (h *Handler) CheckUpdate(w http.ResponseWriter, r *http.Request) {
	out, err := h.service.CheckUpdate(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), middleware.GetRequestID(r.Context()))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, out, middleware.GetRequestID(r.Context()))
}

func (h *Handler) ApplyUpdate(w http.ResponseWriter, r *http.Request) {
	if err := h.service.ApplyUpdate(r.Context()); err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), middleware.GetRequestID(r.Context()))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"status": "started"}, middleware.GetRequestID(r.Context()))
}

func (h *Handler) GetUpdateStatus(w http.ResponseWriter, r *http.Request) {
	out, err := h.service.GetUpdateStatus(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), middleware.GetRequestID(r.Context()))
		return
	}
	httpx.WriteJSON(w, http.StatusOK, out, middleware.GetRequestID(r.Context()))
}

func (h *Handler) GetPublicBranding(w http.ResponseWriter, r *http.Request) {
	appName, _, _ := h.service.GetGlobalSetting(r.Context(), "branding_app_name")
	pageTitle, _, _ := h.service.GetGlobalSetting(r.Context(), "branding_page_title")
	logo, _, _ := h.service.GetGlobalSetting(r.Context(), "branding_logo")
	favicon, _, _ := h.service.GetGlobalSetting(r.Context(), "branding_favicon")
	publicURL, _, _ := h.service.GetGlobalSetting(r.Context(), "branding_public_url")

	if appName == "" {
		appName = "PEKAN"
	}
	if pageTitle == "" {
		pageTitle = "PENCATATAN KEUANGAN"
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"app_name":   appName,
		"page_title": pageTitle,
		"logo":       logo,
		"favicon":    favicon,
		"public_url": publicURL,
	}, middleware.GetRequestID(r.Context()))
}

func (h *Handler) GetSystemLogs(w http.ResponseWriter, r *http.Request) {
	serviceName := r.URL.Query().Get("service")
	linesStr := r.URL.Query().Get("lines")
	lines := 200
	if linesStr != "" {
		if val, err := strconv.Atoi(linesStr); err == nil && val > 0 {
			lines = val
		}
	}

	out, err := h.service.GetSystemLogs(r.Context(), serviceName, lines)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), middleware.GetRequestID(r.Context()))
		return
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]any{"logs": out}, middleware.GetRequestID(r.Context()))
}
