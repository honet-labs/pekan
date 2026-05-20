package domain

import (
	"encoding/json"
	"time"
)

type NotificationChannel struct {
	ID          string
	TenantID    string
	ChannelCode string
	IsEnabled   bool
	ConfigJSON  json.RawMessage
	CreatedBy   string
	UpdatedBy   string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type MessageTemplate struct {
	ID            string
	TenantID      string
	TemplateCode  string
	ChannelCode   string
	LanguageCode  string
	TitleTemplate *string
	BodyTemplate  string
	IsEnabled     bool
	CreatedBy     string
	UpdatedBy     string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type Role struct {
	ID            string
	TenantID      string
	Code          string
	Name          string
	IsSystem      bool
	PermissionIDs []string
}

type Permission struct {
	ID         string
	Code       string
	Name       string
	ModuleCode string
	Action     string
}

type TenantUser struct {
	MembershipID   string
	UserID         string
	Email          string
	FullName       string
	Username       *string
	Status         string
	IsActive       bool
	LastLoginAt    *time.Time
	WhatsAppNumber *string
	Roles          []Role
}

type MembershipWithRoles struct {
	MembershipID   string
	UserID         string
	Email          string
	FullName       string
	Username       *string
	Status         string
	WhatsAppNumber *string
	Roles          []Role
}

type AuditLogItem struct {
	ID            int64
	TenantID      *string
	ActorUserID   *string
	ActorUserName *string
	Action        string
	ResourceType  string
	ResourceID    string
	BeforeJSON    json.RawMessage
	AfterJSON     json.RawMessage
	RequestID     *string
	IPAddress     *string
	UserAgent     *string
	CreatedAt     time.Time
}

type AuditLogFilter struct {
	TenantID     string
	ActorUserID  *string
	Action       *string
	ResourceType *string
	DateFrom     *time.Time
	DateTo       *time.Time
	Page         int
	PageSize     int
}
