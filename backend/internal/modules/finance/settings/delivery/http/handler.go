package http
// trigger sync 2

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"pekan/backend/internal/modules/finance/settings/domain"
	"pekan/backend/internal/modules/finance/settings/usecase"
	"pekan/backend/internal/platform/access"
	"pekan/backend/internal/platform/httpx"
	"pekan/backend/internal/platform/middleware"
	"pekan/backend/internal/platform/tenancy"
)

type Service interface {
	ListNotificationChannels(ctx context.Context, tenantID string) ([]domain.NotificationChannel, error)
	UpsertNotificationChannels(ctx context.Context, in usecase.UpsertNotificationChannelInput) ([]domain.NotificationChannel, error)
	ListTemplates(ctx context.Context, tenantID string, templateCode string) ([]domain.MessageTemplate, error)
	UpsertTemplate(ctx context.Context, in usecase.UpsertTemplateInput) (domain.MessageTemplate, error)
	ListUsersAndRoles(ctx context.Context, tenantID string) (usecase.UsersRolesOutput, error)
	UpdateMembershipRoles(ctx context.Context, in usecase.UpdateMembershipRolesInput) error
	ListRoleCatalog(ctx context.Context, tenantID string) (usecase.RoleCatalogOutput, error)
	CreateRole(ctx context.Context, in usecase.CreateRoleInput) (domain.Role, error)
	UpdateRole(ctx context.Context, in usecase.UpdateRoleInput) (domain.Role, error)
	DeleteRole(ctx context.Context, tenantID, actorUserID, roleID string) error
	ListTenantUsers(ctx context.Context, tenantID string) (usecase.ListTenantUsersOutput, error)
	CreateTenantUser(ctx context.Context, in usecase.CreateTenantUserInput) (domain.TenantUser, error)
	UpdateTenantUser(ctx context.Context, in usecase.UpdateTenantUserInput) (domain.TenantUser, error)
	DeleteTenantUser(ctx context.Context, tenantID, actorUserID, membershipID string) error
	ListAuditLogs(ctx context.Context, in usecase.ListAuditLogsInput) ([]domain.AuditLogItem, int64, error)
}

type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) ListChannels(w http.ResponseWriter, r *http.Request) {
	tc, err := tenancy.FromContext(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "tenant context missing", middleware.GetRequestID(r.Context()))
		return
	}

	items, err := h.service.ListNotificationChannels(r.Context(), tc.TenantID)
	if err != nil {
		writeUsecaseError(w, r, err)
		return
	}

	responseItems := make([]NotificationChannelResponse, 0, len(items))
	for _, item := range items {
		responseItems = append(responseItems, NotificationChannelResponse{
			ID:          item.ID,
			ChannelCode: item.ChannelCode,
			IsEnabled:   item.IsEnabled,
			ConfigJSON:  json.RawMessage(maskConfigJSON(string(item.ConfigJSON))),
			CreatedAt:   item.CreatedAt.UTC().Format(time.RFC3339),
			UpdatedAt:   item.UpdatedAt.UTC().Format(time.RFC3339),
		})
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": responseItems}, middleware.GetRequestID(r.Context()))
}

func (h *Handler) UpdateChannels(w http.ResponseWriter, r *http.Request) {
	var req UpsertNotificationChannelsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid JSON payload", middleware.GetRequestID(r.Context()))
		return
	}

	tc, err := tenancy.FromContext(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "tenant context missing", middleware.GetRequestID(r.Context()))
		return
	}

	channels := make([]domain.NotificationChannel, 0, len(req.Channels))
	for _, channel := range req.Channels {
		channels = append(channels, domain.NotificationChannel{
			ChannelCode: strings.TrimSpace(channel.ChannelCode),
			IsEnabled:   channel.IsEnabled,
			ConfigJSON:  channel.ConfigJSON,
		})
	}

	items, err := h.service.UpsertNotificationChannels(r.Context(), usecase.UpsertNotificationChannelInput{
		TenantID:    tc.TenantID,
		ActorUserID: tc.UserID,
		Channels:    channels,
	})
	if err != nil {
		writeUsecaseError(w, r, err)
		return
	}

	responseItems := make([]NotificationChannelResponse, 0, len(items))
	for _, item := range items {
		responseItems = append(responseItems, NotificationChannelResponse{
			ID:          item.ID,
			ChannelCode: item.ChannelCode,
			IsEnabled:   item.IsEnabled,
			ConfigJSON:  json.RawMessage(maskConfigJSON(string(item.ConfigJSON))),
			CreatedAt:   item.CreatedAt.UTC().Format(time.RFC3339),
			UpdatedAt:   item.UpdatedAt.UTC().Format(time.RFC3339),
		})
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": responseItems}, middleware.GetRequestID(r.Context()))
}

func (h *Handler) ListReminderTemplates(w http.ResponseWriter, r *http.Request) {
	tc, err := tenancy.FromContext(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "tenant context missing", middleware.GetRequestID(r.Context()))
		return
	}

	templateCode := strings.TrimSpace(r.URL.Query().Get("template_code"))
	if templateCode == "" {
		templateCode = "reminder.due"
	}

	items, err := h.service.ListTemplates(r.Context(), tc.TenantID, templateCode)
	if err != nil {
		writeUsecaseError(w, r, err)
		return
	}

	responseItems := make([]MessageTemplateResponse, 0, len(items))
	for _, item := range items {
		responseItems = append(responseItems, MessageTemplateResponse{
			ID:            item.ID,
			TemplateCode:  item.TemplateCode,
			ChannelCode:   item.ChannelCode,
			LanguageCode:  item.LanguageCode,
			TitleTemplate: item.TitleTemplate,
			BodyTemplate:  item.BodyTemplate,
			IsEnabled:     item.IsEnabled,
			CreatedAt:     item.CreatedAt.UTC().Format(time.RFC3339),
			UpdatedAt:     item.UpdatedAt.UTC().Format(time.RFC3339),
		})
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": responseItems}, middleware.GetRequestID(r.Context()))
}

func (h *Handler) UpsertReminderTemplate(w http.ResponseWriter, r *http.Request) {
	var req MessageTemplateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid JSON payload", middleware.GetRequestID(r.Context()))
		return
	}

	tc, err := tenancy.FromContext(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "tenant context missing", middleware.GetRequestID(r.Context()))
		return
	}

	out, err := h.service.UpsertTemplate(r.Context(), usecase.UpsertTemplateInput{
		TenantID:      tc.TenantID,
		ActorUserID:   tc.UserID,
		TemplateCode:  req.TemplateCode,
		ChannelCode:   req.ChannelCode,
		LanguageCode:  req.LanguageCode,
		TitleTemplate: req.TitleTemplate,
		BodyTemplate:  req.BodyTemplate,
		IsEnabled:     req.IsEnabled,
	})
	if err != nil {
		writeUsecaseError(w, r, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, MessageTemplateResponse{
		ID:            out.ID,
		TemplateCode:  out.TemplateCode,
		ChannelCode:   out.ChannelCode,
		LanguageCode:  out.LanguageCode,
		TitleTemplate: out.TitleTemplate,
		BodyTemplate:  out.BodyTemplate,
		IsEnabled:     out.IsEnabled,
		CreatedAt:     out.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:     out.UpdatedAt.UTC().Format(time.RFC3339),
	}, middleware.GetRequestID(r.Context()))
}

func (h *Handler) ListUsersRoles(w http.ResponseWriter, r *http.Request) {
	tc, err := tenancy.FromContext(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "tenant context missing", middleware.GetRequestID(r.Context()))
		return
	}

	out, err := h.service.ListUsersAndRoles(r.Context(), tc.TenantID)
	if err != nil {
		writeUsecaseError(w, r, err)
		return
	}

	users := make([]MembershipRolesResponse, 0, len(out.Users))
	for _, user := range out.Users {
		roles := make([]RoleResponse, 0, len(user.Roles))
		for _, role := range user.Roles {
			roles = append(roles, RoleResponse{
				ID:            role.ID,
				Code:          role.Code,
				Name:          role.Name,
				IsSystem:      role.IsSystem,
				PermissionIDs: role.PermissionIDs,
			})
		}

		users = append(users, MembershipRolesResponse{
			MembershipID: user.MembershipID,
			UserID:       user.UserID,
			Email:        user.Email,
			FullName:     user.FullName,
			Status:       user.Status,
			WhatsAppNumber: func(p *string) *string {
				if p == nil {
					return nil
				}
				masked := maskPhone(*p)
				return &masked
			}(user.WhatsAppNumber),
			Roles:        roles,
		})
	}

	roles := make([]RoleResponse, 0, len(out.Roles))
	for _, role := range out.Roles {
		roles = append(roles, RoleResponse{
			ID:            role.ID,
			Code:          role.Code,
			Name:          role.Name,
			IsSystem:      role.IsSystem,
			PermissionIDs: role.PermissionIDs,
		})
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"users": users,
		"roles": roles,
	}, middleware.GetRequestID(r.Context()))
}

func (h *Handler) UpdateMembershipRoles(w http.ResponseWriter, r *http.Request) {
	membershipID := chi.URLParam(r, "membershipID")
	var req UpdateMembershipRolesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid JSON payload", middleware.GetRequestID(r.Context()))
		return
	}

	tc, err := tenancy.FromContext(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "tenant context missing", middleware.GetRequestID(r.Context()))
		return
	}

	if err := h.service.UpdateMembershipRoles(r.Context(), usecase.UpdateMembershipRolesInput{
		TenantID:     tc.TenantID,
		ActorUserID:  tc.UserID,
		MembershipID: membershipID,
		RoleIDs:      req.RoleIDs,
	}); err != nil {
		writeUsecaseError(w, r, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]any{"updated": true}, middleware.GetRequestID(r.Context()))
}

func (h *Handler) ListRoleCatalog(w http.ResponseWriter, r *http.Request) {
	tc, err := tenancy.FromContext(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "tenant context missing", middleware.GetRequestID(r.Context()))
		return
	}

	out, err := h.service.ListRoleCatalog(r.Context(), tc.TenantID)
	if err != nil {
		writeUsecaseError(w, r, err)
		return
	}

	roleItems := make([]RoleResponse, 0, len(out.Roles))
	for _, item := range out.Roles {
		roleItems = append(roleItems, RoleResponse{
			ID:            item.ID,
			Code:          item.Code,
			Name:          item.Name,
			IsSystem:      item.IsSystem,
			PermissionIDs: item.PermissionIDs,
		})
	}

	permissionItems := make([]PermissionResponse, 0, len(out.Permissions))
	for _, item := range out.Permissions {
		permissionItems = append(permissionItems, PermissionResponse{
			ID:         item.ID,
			Code:       item.Code,
			Name:       item.Name,
			ModuleCode: item.ModuleCode,
			Action:     item.Action,
		})
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"roles":       roleItems,
		"permissions": permissionItems,
	}, middleware.GetRequestID(r.Context()))
}

func (h *Handler) CreateRole(w http.ResponseWriter, r *http.Request) {
	var req CreateRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid JSON payload", middleware.GetRequestID(r.Context()))
		return
	}

	tc, err := tenancy.FromContext(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "tenant context missing", middleware.GetRequestID(r.Context()))
		return
	}

	out, err := h.service.CreateRole(r.Context(), usecase.CreateRoleInput{
		TenantID:      tc.TenantID,
		ActorUserID:   tc.UserID,
		Code:          req.Code,
		Name:          req.Name,
		PermissionIDs: req.PermissionIDs,
	})
	if err != nil {
		writeUsecaseError(w, r, err)
		return
	}

	httpx.WriteJSON(w, http.StatusCreated, RoleResponse{
		ID:            out.ID,
		Code:          out.Code,
		Name:          out.Name,
		IsSystem:      out.IsSystem,
		PermissionIDs: out.PermissionIDs,
	}, middleware.GetRequestID(r.Context()))
}

func (h *Handler) UpdateRole(w http.ResponseWriter, r *http.Request) {
	roleID := chi.URLParam(r, "roleID")
	var req UpdateRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid JSON payload", middleware.GetRequestID(r.Context()))
		return
	}

	tc, err := tenancy.FromContext(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "tenant context missing", middleware.GetRequestID(r.Context()))
		return
	}

	out, err := h.service.UpdateRole(r.Context(), usecase.UpdateRoleInput{
		TenantID:      tc.TenantID,
		ActorUserID:   tc.UserID,
		RoleID:        roleID,
		Code:          req.Code,
		Name:          req.Name,
		PermissionIDs: req.PermissionIDs,
	})
	if err != nil {
		writeUsecaseError(w, r, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, RoleResponse{
		ID:            out.ID,
		Code:          out.Code,
		Name:          out.Name,
		IsSystem:      out.IsSystem,
		PermissionIDs: out.PermissionIDs,
	}, middleware.GetRequestID(r.Context()))
}

func (h *Handler) DeleteRole(w http.ResponseWriter, r *http.Request) {
	roleID := chi.URLParam(r, "roleID")
	tc, err := tenancy.FromContext(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "tenant context missing", middleware.GetRequestID(r.Context()))
		return
	}
	if err := h.service.DeleteRole(r.Context(), tc.TenantID, tc.UserID, roleID); err != nil {
		writeUsecaseError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"deleted": true}, middleware.GetRequestID(r.Context()))
}

func (h *Handler) ListTenantUsers(w http.ResponseWriter, r *http.Request) {
	tc, err := tenancy.FromContext(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "tenant context missing", middleware.GetRequestID(r.Context()))
		return
	}

	out, err := h.service.ListTenantUsers(r.Context(), tc.TenantID)
	if err != nil {
		writeUsecaseError(w, r, err)
		return
	}

	users := make([]TenantUserResponse, 0, len(out.Users))
	for _, user := range out.Users {
		roleItems := make([]RoleResponse, 0, len(user.Roles))
		for _, role := range user.Roles {
			roleItems = append(roleItems, RoleResponse{
				ID:       role.ID,
				Code:     role.Code,
				Name:     role.Name,
				IsSystem: role.IsSystem,
			})
		}
		var lastLoginAt *string
		if user.LastLoginAt != nil {
			formatted := user.LastLoginAt.UTC().Format(time.RFC3339)
			lastLoginAt = &formatted
		}
		users = append(users, TenantUserResponse{
			MembershipID:   user.MembershipID,
			UserID:         user.UserID,
			Email:          maskEmail(user.Email),
			FullName:       user.FullName,
			Status:         user.Status,
			IsActive:       user.IsActive,
			LastLoginAt:    lastLoginAt,
			WhatsAppNumber: func(p *string) *string {
				if p == nil {
					return nil
				}
				masked := maskPhone(*p)
				return &masked
			}(user.WhatsAppNumber),
			Roles:          roleItems,
		})
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]any{"items": users}, middleware.GetRequestID(r.Context()))
}

func (h *Handler) CreateTenantUser(w http.ResponseWriter, r *http.Request) {
	var req CreateTenantUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid JSON payload", middleware.GetRequestID(r.Context()))
		return
	}
	tc, err := tenancy.FromContext(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "tenant context missing", middleware.GetRequestID(r.Context()))
		return
	}
	out, err := h.service.CreateTenantUser(r.Context(), usecase.CreateTenantUserInput{
		TenantID:    tc.TenantID,
		ActorUserID: tc.UserID,
		Email:       req.Email,
		FullName:    req.FullName,
		PhoneNumber: req.PhoneNumber,
		Password:    req.Password,
		Status:      req.Status,
		IsActive:    req.IsActive,
		RoleIDs:     req.RoleIDs,
	})
	if err != nil {
		writeUsecaseError(w, r, err)
		return
	}
	roleItems := make([]RoleResponse, 0, len(out.Roles))
	for _, role := range out.Roles {
		roleItems = append(roleItems, RoleResponse{
			ID:       role.ID,
			Code:     role.Code,
			Name:     role.Name,
			IsSystem: role.IsSystem,
		})
	}
	var lastLoginAt *string
	if out.LastLoginAt != nil {
		formatted := out.LastLoginAt.UTC().Format(time.RFC3339)
		lastLoginAt = &formatted
	}
	httpx.WriteJSON(w, http.StatusCreated, TenantUserResponse{
		MembershipID: out.MembershipID,
		UserID:       out.UserID,
		Email:        maskEmail(out.Email),
		FullName:     out.FullName,
		Status:       out.Status,
		IsActive:     out.IsActive,
		LastLoginAt:  lastLoginAt,
		Roles:        roleItems,
	}, middleware.GetRequestID(r.Context()))
}

func (h *Handler) UpdateTenantUser(w http.ResponseWriter, r *http.Request) {
	membershipID := chi.URLParam(r, "membershipID")
	var req UpdateTenantUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid JSON payload", middleware.GetRequestID(r.Context()))
		return
	}
	tc, err := tenancy.FromContext(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "tenant context missing", middleware.GetRequestID(r.Context()))
		return
	}
	out, err := h.service.UpdateTenantUser(r.Context(), usecase.UpdateTenantUserInput{
		TenantID:     tc.TenantID,
		ActorUserID:  tc.UserID,
		MembershipID: membershipID,
		Email:        req.Email,
		FullName:     req.FullName,
		PhoneNumber:  req.PhoneNumber,
		Password:     req.Password,
		Status:       req.Status,
		IsActive:     req.IsActive,
		RoleIDs:      req.RoleIDs,
	})
	if err != nil {
		writeUsecaseError(w, r, err)
		return
	}
	roleItems := make([]RoleResponse, 0, len(out.Roles))
	for _, role := range out.Roles {
		roleItems = append(roleItems, RoleResponse{
			ID:       role.ID,
			Code:     role.Code,
			Name:     role.Name,
			IsSystem: role.IsSystem,
		})
	}
	var lastLoginAt *string
	if out.LastLoginAt != nil {
		formatted := out.LastLoginAt.UTC().Format(time.RFC3339)
		lastLoginAt = &formatted
	}
	httpx.WriteJSON(w, http.StatusOK, TenantUserResponse{
		MembershipID: out.MembershipID,
		UserID:       out.UserID,
		Email:        maskEmail(out.Email),
		FullName:     out.FullName,
		Status:       out.Status,
		IsActive:     out.IsActive,
		LastLoginAt:  lastLoginAt,
		Roles:        roleItems,
	}, middleware.GetRequestID(r.Context()))
}

func (h *Handler) DeleteTenantUser(w http.ResponseWriter, r *http.Request) {
	membershipID := chi.URLParam(r, "membershipID")
	tc, err := tenancy.FromContext(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "tenant context missing", middleware.GetRequestID(r.Context()))
		return
	}
	if err := h.service.DeleteTenantUser(r.Context(), tc.TenantID, tc.UserID, membershipID); err != nil {
		writeUsecaseError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"deleted": true}, middleware.GetRequestID(r.Context()))
}

func (h *Handler) ListAuditLogs(w http.ResponseWriter, r *http.Request) {
	tc, err := tenancy.FromContext(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "tenant context missing", middleware.GetRequestID(r.Context()))
		return
	}

	page := parseIntOrDefault(r.URL.Query().Get("page"), 1)
	pageSize := parseIntOrDefault(r.URL.Query().Get("page_size"), 50)
	action := optionalStringPtr(r.URL.Query().Get("action"))
	resourceType := optionalStringPtr(r.URL.Query().Get("resource_type"))
	actorUserID := optionalStringPtr(r.URL.Query().Get("actor_user_id"))
	dateFrom, err := parseOptionalDateTime(r.URL.Query().Get("from"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_DATE", "from must be RFC3339 datetime", middleware.GetRequestID(r.Context()))
		return
	}
	dateTo, err := parseOptionalDateTime(r.URL.Query().Get("to"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_DATE", "to must be RFC3339 datetime", middleware.GetRequestID(r.Context()))
		return
	}

	items, total, err := h.service.ListAuditLogs(r.Context(), usecase.ListAuditLogsInput{
		TenantID:     tc.TenantID,
		ActorUserID:  actorUserID,
		Action:       action,
		ResourceType: resourceType,
		DateFrom:     dateFrom,
		DateTo:       dateTo,
		Page:         page,
		PageSize:     pageSize,
	})
	if err != nil {
		writeUsecaseError(w, r, err)
		return
	}

	responseItems := make([]AuditLogResponse, 0, len(items))
	for _, item := range items {
		responseItems = append(responseItems, AuditLogResponse{
			ID:            item.ID,
			TenantID:      item.TenantID,
			ActorUserID:   item.ActorUserID,
			ActorUserName: item.ActorUserName,
			Action:        item.Action,
			ResourceType:  item.ResourceType,
			ResourceID:    item.ResourceID,
			BeforeJSON:    json.RawMessage(maskConfigJSON(string(item.BeforeJSON))),
			AfterJSON:     json.RawMessage(maskConfigJSON(string(item.AfterJSON))),
			RequestID:     item.RequestID,
			IPAddress:     item.IPAddress,
			UserAgent:     item.UserAgent,
			CreatedAt:     item.CreatedAt.UTC().Format(time.RFC3339),
		})
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

func optionalStringPtr(raw string) *string {
	val := strings.TrimSpace(raw)
	if val == "" {
		return nil
	}
	return &val
}

func parseOptionalDateTime(raw string) (*time.Time, error) {
	val := strings.TrimSpace(raw)
	if val == "" {
		return nil, nil
	}
	out, err := time.Parse(time.RFC3339, val)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func parseIntOrDefault(raw string, fallback int) int {
	if strings.TrimSpace(raw) == "" {
		return fallback
	}
	out, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil {
		return fallback
	}
	return out
}

func writeUsecaseError(w http.ResponseWriter, r *http.Request, err error) {
	requestID := middleware.GetRequestID(r.Context())
	switch {
	case errors.Is(err, domain.ErrInvalidChannelCode):
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_CHANNEL_CODE", err.Error(), requestID)
	case errors.Is(err, domain.ErrInvalidTemplate):
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_TEMPLATE", err.Error(), requestID)
	case errors.Is(err, domain.ErrInvalidRoleCode):
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_ROLE_CODE", err.Error(), requestID)
	case errors.Is(err, domain.ErrInvalidRoleName):
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_ROLE_NAME", err.Error(), requestID)
	case errors.Is(err, domain.ErrRoleCodeDuplicate):
		httpx.WriteError(w, http.StatusBadRequest, "ROLE_CODE_DUPLICATE", err.Error(), requestID)
	case errors.Is(err, domain.ErrRoleSystemLocked):
		httpx.WriteError(w, http.StatusBadRequest, "ROLE_SYSTEM_LOCKED", err.Error(), requestID)
	case errors.Is(err, domain.ErrMembershipNotFound):
		httpx.WriteError(w, http.StatusNotFound, "MEMBERSHIP_NOT_FOUND", err.Error(), requestID)
	case errors.Is(err, domain.ErrRoleNotFound):
		httpx.WriteError(w, http.StatusBadRequest, "ROLE_NOT_FOUND", err.Error(), requestID)
	case errors.Is(err, domain.ErrUserNotFound):
		httpx.WriteError(w, http.StatusNotFound, "USER_NOT_FOUND", err.Error(), requestID)
	case errors.Is(err, domain.ErrInvalidUserEmail):
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_USER_EMAIL", err.Error(), requestID)
	case errors.Is(err, domain.ErrInvalidUserName):
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_USER_NAME", err.Error(), requestID)
	case errors.Is(err, domain.ErrInvalidPassword):
		httpx.WriteError(w, http.StatusBadRequest, "INVALID_PASSWORD", err.Error(), requestID)
	case errors.Is(err, domain.ErrUserEmailDuplicate):
		httpx.WriteError(w, http.StatusBadRequest, "USER_EMAIL_DUPLICATE", err.Error(), requestID)
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

func maskConfigJSON(rawJSON string) string {
	if strings.TrimSpace(rawJSON) == "" {
		return rawJSON
	}

	var data map[string]any
	if err := json.Unmarshal([]byte(rawJSON), &data); err != nil {
		return rawJSON
	}

	sensitiveKeys := []string{"password", "api_key", "apiKey", "token", "secret", "client_secret", "bot_token", "email", "phone", "wa", "whatsapp"}
	
	mask := func(val any) any {
		s, ok := val.(string)
		if !ok || s == "" {
			return val
		}
		if strings.Contains(s, "@") {
			return maskEmail(s)
		}
		if len(s) > 8 {
			return s[:4] + "********" + s[len(s)-4:]
		}
		return "********"
	}

	changed := false
	for k, v := range data {
		lowerK := strings.ToLower(k)
		for _, sk := range sensitiveKeys {
			if strings.Contains(lowerK, strings.ToLower(sk)) {
				data[k] = mask(v)
				changed = true
				break
			}
		}
	}

	if !changed {
		return rawJSON
	}

	out, err := json.Marshal(data)
	if err != nil {
		return rawJSON
	}
	return string(out)
}

func maskEmail(email string) string {
	if email == "" {
		return ""
	}
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return "****"
	}
	name := parts[0]
	if len(name) <= 2 {
		return name + "***@" + parts[1]
	}
	return name[:2] + "******@" + parts[1]
}

func maskPhone(phone string) string {
	p := strings.TrimSpace(phone)
	if p == "" {
		return ""
	}
	if len(p) <= 6 {
		return "****"
	}
	return p[:3] + "******" + p[len(p)-3:]
}
