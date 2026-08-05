package http

import (
	"encoding/json"
)

type NotificationChannelRequest struct {
	ChannelCode string          `json:"channel_code"`
	IsEnabled   bool            `json:"is_enabled"`
	ConfigJSON  json.RawMessage `json:"config_json"`
}

type UpsertNotificationChannelsRequest struct {
	Channels []NotificationChannelRequest `json:"channels"`
}

type MessageTemplateRequest struct {
	TemplateCode  string  `json:"template_code"`
	ChannelCode   string  `json:"channel_code"`
	LanguageCode  string  `json:"language_code"`
	TitleTemplate *string `json:"title_template"`
	BodyTemplate  string  `json:"body_template"`
	IsEnabled     bool    `json:"is_enabled"`
}

type UpdateMembershipRolesRequest struct {
	RoleIDs []string `json:"role_ids"`
}

type CreateRoleRequest struct {
	Code          string   `json:"code"`
	Name          string   `json:"name"`
	PermissionIDs []string `json:"permission_ids"`
}

type UpdateRoleRequest struct {
	Code          string   `json:"code"`
	Name          string   `json:"name"`
	PermissionIDs []string `json:"permission_ids"`
}

type CreateTenantUserRequest struct {
	Email       string   `json:"email"`
	FullName    string   `json:"full_name"`
	PhoneNumber string   `json:"phone_number"`
	Password    string   `json:"password"`
	Status      string   `json:"status"`
	IsActive    bool     `json:"is_active"`
	RoleIDs     []string `json:"role_ids"`
}

type UpdateTenantUserRequest struct {
	Email       string   `json:"email"`
	FullName    string   `json:"full_name"`
	PhoneNumber string   `json:"phone_number"`
	Password    *string  `json:"password"`
	Status      string   `json:"status"`
	IsActive    bool     `json:"is_active"`
	RoleIDs     []string `json:"role_ids"`
}

type NotificationChannelResponse struct {
	ID          string          `json:"id,omitempty"`
	ChannelCode string          `json:"channel_code"`
	IsEnabled   bool            `json:"is_enabled"`
	ConfigJSON  json.RawMessage `json:"config_json"`
	CreatedAt   string          `json:"created_at,omitempty"`
	UpdatedAt   string          `json:"updated_at,omitempty"`
}

type MessageTemplateResponse struct {
	ID            string  `json:"id"`
	TemplateCode  string  `json:"template_code"`
	ChannelCode   string  `json:"channel_code"`
	LanguageCode  string  `json:"language_code"`
	TitleTemplate *string `json:"title_template"`
	BodyTemplate  string  `json:"body_template"`
	IsEnabled     bool    `json:"is_enabled"`
	CreatedAt     string  `json:"created_at"`
	UpdatedAt     string  `json:"updated_at"`
}

type RoleResponse struct {
	ID            string   `json:"id"`
	Code          string   `json:"code"`
	Name          string   `json:"name"`
	IsSystem      bool     `json:"is_system"`
	PermissionIDs []string `json:"permission_ids,omitempty"`
}

type PermissionResponse struct {
	ID         string `json:"id"`
	Code       string `json:"code"`
	Name       string `json:"name"`
	ModuleCode string `json:"module_code"`
	Action     string `json:"action"`
}

type TenantUserResponse struct {
	MembershipID   string         `json:"membership_id"`
	UserID         string         `json:"user_id"`
	Email          string         `json:"email"`
	FullName       string         `json:"full_name"`
	Status         string         `json:"status"`
	IsActive       bool           `json:"is_active"`
	LastLoginAt    *string        `json:"last_login_at,omitempty"`
	WhatsAppNumber *string        `json:"whatsapp_number,omitempty"`
	Roles          []RoleResponse `json:"roles"`
}

type MembershipRolesResponse struct {
	MembershipID string         `json:"membership_id"`
	UserID       string         `json:"user_id"`
	Email        string         `json:"email"`
	FullName     string         `json:"full_name"`
	Status       string         `json:"status"`
	WhatsAppNumber *string      `json:"whatsapp_number,omitempty"`
	Roles        []RoleResponse `json:"roles"`
}

type AuditLogResponse struct {
	ID            int64           `json:"id"`
	TenantID      *string         `json:"tenant_id"`
	ActorUserID   *string         `json:"actor_user_id"`
	ActorUserName *string         `json:"actor_user_name"`
	Action        string          `json:"action"`
	ResourceType  string          `json:"resource_type"`
	ResourceID    string          `json:"resource_id"`
	BeforeJSON    json.RawMessage `json:"before_json"`
	AfterJSON     json.RawMessage `json:"after_json"`
	RequestID     *string         `json:"request_id"`
	IPAddress     *string         `json:"ip_address"`
	UserAgent     *string         `json:"user_agent"`
	CreatedAt     string          `json:"created_at"`
}
