package domain

import "context"

type Repository interface {
	ListNotificationChannels(ctx context.Context, tenantID string) ([]NotificationChannel, error)
	UpsertNotificationChannel(ctx context.Context, channel NotificationChannel) (NotificationChannel, error)

	ListMessageTemplates(ctx context.Context, tenantID, templateCode string) ([]MessageTemplate, error)
	UpsertMessageTemplate(ctx context.Context, template MessageTemplate) (MessageTemplate, error)

	ListRoles(ctx context.Context, tenantID string) ([]Role, error)
	ListPermissions(ctx context.Context) ([]Permission, error)
	CreateRole(ctx context.Context, tenantID string, role Role, permissionIDs []string) (Role, error)
	UpdateRole(ctx context.Context, tenantID string, role Role, permissionIDs []string) (Role, error)
	DeleteRole(ctx context.Context, tenantID, roleID string) error

	ListTenantUsers(ctx context.Context, tenantID string) ([]TenantUser, error)
	CreateTenantUser(ctx context.Context, tenantID string, user TenantUser, passwordHash string, roleIDs []string) (TenantUser, error)
	UpdateTenantUser(ctx context.Context, tenantID string, user TenantUser, passwordHash *string, roleIDs []string) (TenantUser, error)
	DeleteTenantUser(ctx context.Context, tenantID, membershipID string) error

	ListMembershipsWithRoles(ctx context.Context, tenantID string) ([]MembershipWithRoles, error)
	ReplaceMembershipRoles(ctx context.Context, tenantID, membershipID string, roleIDs []string) error

	IsEmailTaken(ctx context.Context, email string) (bool, error)
	IsPhoneTaken(ctx context.Context, phone string) (bool, error)
	IsUsernameTaken(ctx context.Context, username string) (bool, error)

	ListAuditLogs(ctx context.Context, filter AuditLogFilter) ([]AuditLogItem, int64, error)
}
