package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"regexp"
	"strings"
	"time"

	"pekan/backend/internal/modules/finance/settings/domain"
	"pekan/backend/internal/platform/access"
	platformauth "pekan/backend/internal/platform/auth"
	"pekan/backend/internal/platform/security"
)

type Authorizer interface {
	EnsureModule(ctx context.Context, module string) error
	EnsureFeature(ctx context.Context, feature string) error
	EnsurePermission(ctx context.Context, permission string) error
}

type AuditLogger interface {
	Write(ctx context.Context, action, resourceType, resourceID string, before, after any) error
}

type Service struct {
	repo  domain.Repository
	authz Authorizer
	audit AuditLogger
}

func NewService(repo domain.Repository, authz Authorizer, audit AuditLogger) *Service {
	return &Service{
		repo:  repo,
		authz: authz,
		audit: audit,
	}
}

var orderedChannelCodes = []string{
	"email",
	"telegram",
	"whatsapp_official",
	"whatsapp_gowa",
	"whatsapp_fonte",
}

var roleCodePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{2,62}$`)

type UpsertNotificationChannelInput struct {
	TenantID    string
	ActorUserID string
	Channels    []domain.NotificationChannel
}

func (s *Service) ListNotificationChannels(ctx context.Context, tenantID string) ([]domain.NotificationChannel, error) {
	if err := s.ensureReadAccess(ctx); err != nil {
		return nil, err
	}

	items, err := s.repo.ListNotificationChannels(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	byCode := make(map[string]domain.NotificationChannel, len(items))
	for _, item := range items {
		byCode[item.ChannelCode] = item
	}

	now := time.Now().UTC()
	result := make([]domain.NotificationChannel, 0, len(orderedChannelCodes))
	for _, channelCode := range orderedChannelCodes {
		if item, ok := byCode[channelCode]; ok {
			result = append(result, item)
			continue
		}
		result = append(result, domain.NotificationChannel{
			ChannelCode: channelCode,
			IsEnabled:   false,
			ConfigJSON:  json.RawMessage(`{}`),
			CreatedAt:   now,
			UpdatedAt:   now,
		})
	}
	return result, nil
}

func (s *Service) UpsertNotificationChannels(ctx context.Context, in UpsertNotificationChannelInput) ([]domain.NotificationChannel, error) {
	if err := s.ensureWriteAccess(ctx); err != nil {
		return nil, err
	}

	if len(in.Channels) == 0 {
		return []domain.NotificationChannel{}, nil
	}

	updated := make([]domain.NotificationChannel, 0, len(in.Channels))
	now := time.Now().UTC()

	for _, channel := range in.Channels {
		code := strings.ToLower(strings.TrimSpace(channel.ChannelCode))
		if !isAllowedChannelCode(code) {
			return nil, domain.ErrInvalidChannelCode
		}
		if len(channel.ConfigJSON) == 0 {
			channel.ConfigJSON = json.RawMessage(`{}`)
		}
		channel.ChannelCode = code
		channel.TenantID = in.TenantID
		channel.CreatedBy = in.ActorUserID
		channel.UpdatedBy = in.ActorUserID
		channel.CreatedAt = now
		channel.UpdatedAt = now

		// Prevent overwriting masked sensitive data
		existingItems, _ := s.repo.ListNotificationChannels(ctx, in.TenantID)
		for _, ex := range existingItems {
			if ex.ChannelCode == code {
				channel.ConfigJSON = mergeMaskedConfig(ex.ConfigJSON, channel.ConfigJSON)
				break
			}
		}

		out, err := s.repo.UpsertNotificationChannel(ctx, channel)
		if err != nil {
			return nil, err
		}
		updated = append(updated, out)
	}

	_ = s.audit.Write(ctx, "finance.settings.notification_channels.update", "finance_notification_channels", in.TenantID, nil, map[string]any{
		"count": len(updated),
	})
	return updated, nil
}

type UpsertTemplateInput struct {
	TenantID      string
	ActorUserID   string
	TemplateCode  string
	ChannelCode   string
	LanguageCode  string
	TitleTemplate *string
	BodyTemplate  string
	IsEnabled     bool
}

func (s *Service) ListTemplates(ctx context.Context, tenantID string, templateCode string) ([]domain.MessageTemplate, error) {
	if err := s.ensureReadAccess(ctx); err != nil {
		return nil, err
	}
	return s.repo.ListMessageTemplates(ctx, tenantID, templateCode)
}

func (s *Service) UpsertTemplate(ctx context.Context, in UpsertTemplateInput) (domain.MessageTemplate, error) {
	if err := s.ensureWriteAccess(ctx); err != nil {
		return domain.MessageTemplate{}, err
	}

	templateCode := strings.TrimSpace(in.TemplateCode)
	channelCode := strings.ToLower(strings.TrimSpace(in.ChannelCode))
	languageCode := strings.ToLower(strings.TrimSpace(in.LanguageCode))
	bodyTemplate := strings.TrimSpace(in.BodyTemplate)
	if templateCode == "" || !isAllowedTemplateChannel(channelCode) || languageCode == "" || bodyTemplate == "" {
		return domain.MessageTemplate{}, domain.ErrInvalidTemplate
	}

	now := time.Now().UTC()
	template, err := s.repo.UpsertMessageTemplate(ctx, domain.MessageTemplate{
		TenantID:      in.TenantID,
		TemplateCode:  templateCode,
		ChannelCode:   channelCode,
		LanguageCode:  languageCode,
		TitleTemplate: in.TitleTemplate,
		BodyTemplate:  bodyTemplate,
		IsEnabled:     in.IsEnabled,
		CreatedBy:     in.ActorUserID,
		UpdatedBy:     in.ActorUserID,
		CreatedAt:     now,
		UpdatedAt:     now,
	})
	if err != nil {
		return domain.MessageTemplate{}, err
	}

	_ = s.audit.Write(ctx, "finance.settings.template.update", "finance_message_template", template.ID, nil, template)
	return template, nil
}

type UsersRolesOutput struct {
	Users []domain.MembershipWithRoles
	Roles []domain.Role
}

type RoleCatalogOutput struct {
	Roles       []domain.Role
	Permissions []domain.Permission
}

func (s *Service) ListUsersAndRoles(ctx context.Context, tenantID string) (UsersRolesOutput, error) {
	if err := s.ensureReadAccess(ctx); err != nil {
		return UsersRolesOutput{}, err
	}

	users, err := s.repo.ListMembershipsWithRoles(ctx, tenantID)
	if err != nil {
		return UsersRolesOutput{}, err
	}
	roles, err := s.repo.ListRoles(ctx, tenantID)
	if err != nil {
		return UsersRolesOutput{}, err
	}

	return UsersRolesOutput{
		Users: users,
		Roles: roles,
	}, nil
}

func (s *Service) ListRoleCatalog(ctx context.Context, tenantID string) (RoleCatalogOutput, error) {
	if err := s.ensureReadAccess(ctx); err != nil {
		return RoleCatalogOutput{}, err
	}

	roles, err := s.repo.ListRoles(ctx, tenantID)
	if err != nil {
		return RoleCatalogOutput{}, err
	}
	permissions, err := s.repo.ListPermissions(ctx)
	if err != nil {
		return RoleCatalogOutput{}, err
	}
	return RoleCatalogOutput{
		Roles:       roles,
		Permissions: permissions,
	}, nil
}

type CreateRoleInput struct {
	TenantID      string
	ActorUserID   string
	Code          string
	Name          string
	PermissionIDs []string
}

func (s *Service) CreateRole(ctx context.Context, in CreateRoleInput) (domain.Role, error) {
	if err := s.ensureRoleManageAccess(ctx); err != nil {
		return domain.Role{}, err
	}
	code := normalizeRoleCode(in.Code)
	name := strings.TrimSpace(in.Name)
	if !roleCodePattern.MatchString(code) {
		return domain.Role{}, domain.ErrInvalidRoleCode
	}
	if name == "" {
		return domain.Role{}, domain.ErrInvalidRoleName
	}
	created, err := s.repo.CreateRole(ctx, in.TenantID, domain.Role{
		Code: code,
		Name: name,
	}, in.PermissionIDs)
	if err != nil {
		return domain.Role{}, err
	}
	_ = s.audit.Write(ctx, "finance.settings.role.create", "role", created.ID, nil, map[string]any{
		"code":          created.Code,
		"name":          created.Name,
		"permission_ids": in.PermissionIDs,
	})
	return created, nil
}

type UpdateRoleInput struct {
	TenantID      string
	ActorUserID   string
	RoleID        string
	Code          string
	Name          string
	PermissionIDs []string
}

func (s *Service) UpdateRole(ctx context.Context, in UpdateRoleInput) (domain.Role, error) {
	if err := s.ensureRoleManageAccess(ctx); err != nil {
		return domain.Role{}, err
	}
	code := normalizeRoleCode(in.Code)
	name := strings.TrimSpace(in.Name)
	if !roleCodePattern.MatchString(code) {
		return domain.Role{}, domain.ErrInvalidRoleCode
	}
	if name == "" {
		return domain.Role{}, domain.ErrInvalidRoleName
	}
	updated, err := s.repo.UpdateRole(ctx, in.TenantID, domain.Role{
		ID:   in.RoleID,
		Code: code,
		Name: name,
	}, in.PermissionIDs)
	if err != nil {
		return domain.Role{}, err
	}
	_ = s.audit.Write(ctx, "finance.settings.role.update", "role", updated.ID, nil, map[string]any{
		"code":          updated.Code,
		"name":          updated.Name,
		"permission_ids": in.PermissionIDs,
	})
	return updated, nil
}

func (s *Service) DeleteRole(ctx context.Context, tenantID, actorUserID, roleID string) error {
	if err := s.ensureRoleManageAccess(ctx); err != nil {
		return err
	}
	if err := s.repo.DeleteRole(ctx, tenantID, roleID); err != nil {
		return err
	}
	_ = s.audit.Write(ctx, "finance.settings.role.delete", "role", roleID, nil, map[string]any{"deleted": true})
	return nil
}

type ListTenantUsersOutput struct {
	Users []domain.TenantUser
}

func (s *Service) ListTenantUsers(ctx context.Context, tenantID string) (ListTenantUsersOutput, error) {
	if err := s.ensureReadAccess(ctx); err != nil {
		return ListTenantUsersOutput{}, err
	}
	items, err := s.repo.ListTenantUsers(ctx, tenantID)
	if err != nil {
		return ListTenantUsersOutput{}, err
	}
	
	// Apply Data Masking
	for i := range items {
		items[i].Email = security.MaskEmail(items[i].Email)
		if items[i].WhatsAppNumber != nil && *items[i].WhatsAppNumber != "" {
			masked := security.MaskPhone(*items[i].WhatsAppNumber)
			items[i].WhatsAppNumber = &masked
		}
	}
	
	return ListTenantUsersOutput{Users: items}, nil
}

type CreateTenantUserInput struct {
	TenantID     string
	ActorUserID  string
	Email        string
	FullName     string
	PhoneNumber  string
	Password     string
	Status       string
	IsActive     bool
	RoleIDs      []string
}

func (s *Service) CreateTenantUser(ctx context.Context, in CreateTenantUserInput) (domain.TenantUser, error) {
	if err := s.ensureRoleManageAccess(ctx); err != nil {
		return domain.TenantUser{}, err
	}
	email, fullName, err := validateUserIdentity(in.Email, in.FullName)
	if err != nil {
		return domain.TenantUser{}, err
	}
	password := strings.TrimSpace(in.Password)
	if len(password) < 8 {
		return domain.TenantUser{}, domain.ErrInvalidPassword
	}
	passwordHash, err := platformauth.HashPassword(password)
	if err != nil {
		return domain.TenantUser{}, err
	}
	status := normalizeMembershipStatus(in.Status)
	created, err := s.repo.CreateTenantUser(ctx, in.TenantID, domain.TenantUser{
		Email:          email,
		FullName:       fullName,
		WhatsAppNumber: &in.PhoneNumber,
		Status:         status,
		IsActive:       in.IsActive,
	}, passwordHash, in.RoleIDs)
	if err != nil {
		return domain.TenantUser{}, err
	}
	_ = s.audit.Write(ctx, "finance.settings.user.create", "tenant_membership", created.MembershipID, nil, map[string]any{
		"user_id":  created.UserID,
		"email":    created.Email,
		"status":   created.Status,
		"is_active": created.IsActive,
		"role_ids": in.RoleIDs,
	})
	return created, nil
}

type UpdateTenantUserInput struct {
	TenantID      string
	ActorUserID   string
	MembershipID  string
	Email         string
	FullName      string
	PhoneNumber   string
	Password      *string
	Status        string
	IsActive      bool
	RoleIDs       []string
}

func (s *Service) UpdateTenantUser(ctx context.Context, in UpdateTenantUserInput) (domain.TenantUser, error) {
	if err := s.ensureRoleManageAccess(ctx); err != nil {
		return domain.TenantUser{}, err
	}
	email, fullName, err := validateUserIdentity(in.Email, in.FullName)
	if err != nil {
		return domain.TenantUser{}, err
	}
	var passwordHash *string
	if in.Password != nil && strings.TrimSpace(*in.Password) != "" {
		rawPassword := strings.TrimSpace(*in.Password)
		if len(rawPassword) < 8 {
			return domain.TenantUser{}, domain.ErrInvalidPassword
		}
		hashed, hashErr := platformauth.HashPassword(rawPassword)
		if hashErr != nil {
			return domain.TenantUser{}, hashErr
		}
		passwordHash = &hashed
	}
	status := normalizeMembershipStatus(in.Status)
	updated, err := s.repo.UpdateTenantUser(ctx, in.TenantID, domain.TenantUser{
		MembershipID:   in.MembershipID,
		Email:          email,
		FullName:       fullName,
		WhatsAppNumber: &in.PhoneNumber,
		Status:         status,
		IsActive:       in.IsActive,
	}, passwordHash, in.RoleIDs)
	if err != nil {
		return domain.TenantUser{}, err
	}
	_ = s.audit.Write(ctx, "finance.settings.user.update", "tenant_membership", updated.MembershipID, nil, map[string]any{
		"user_id":  updated.UserID,
		"email":    updated.Email,
		"status":   updated.Status,
		"is_active": updated.IsActive,
		"role_ids": in.RoleIDs,
	})
	return updated, nil
}

func (s *Service) DeleteTenantUser(ctx context.Context, tenantID, actorUserID, membershipID string) error {
	if err := s.ensureRoleManageAccess(ctx); err != nil {
		return err
	}
	if err := s.repo.DeleteTenantUser(ctx, tenantID, membershipID); err != nil {
		return err
	}
	_ = s.audit.Write(ctx, "finance.settings.user.delete", "tenant_membership", membershipID, nil, map[string]any{"deleted": true})
	return nil
}

type UpdateMembershipRolesInput struct {
	TenantID     string
	ActorUserID  string
	MembershipID string
	RoleIDs      []string
}

func (s *Service) UpdateMembershipRoles(ctx context.Context, in UpdateMembershipRolesInput) error {
	if err := s.ensureRoleManageAccess(ctx); err != nil {
		return err
	}
	if err := s.repo.ReplaceMembershipRoles(ctx, in.TenantID, in.MembershipID, in.RoleIDs); err != nil {
		return err
	}

	_ = s.audit.Write(ctx, "finance.settings.roles.update", "tenant_membership", in.MembershipID, nil, map[string]any{
		"role_ids": in.RoleIDs,
	})
	return nil
}

type ListAuditLogsInput struct {
	TenantID     string
	ActorUserID  *string
	Action       *string
	ResourceType *string
	DateFrom     *time.Time
	DateTo       *time.Time
	Page         int
	PageSize     int
}

func (s *Service) ListAuditLogs(ctx context.Context, in ListAuditLogsInput) ([]domain.AuditLogItem, int64, error) {
	if err := s.ensureAuditAccess(ctx); err != nil {
		return nil, 0, err
	}
	return s.repo.ListAuditLogs(ctx, domain.AuditLogFilter{
		TenantID:     in.TenantID,
		ActorUserID:  in.ActorUserID,
		Action:       in.Action,
		ResourceType: in.ResourceType,
		DateFrom:     in.DateFrom,
		DateTo:       in.DateTo,
		Page:         in.Page,
		PageSize:     in.PageSize,
	})
}

func (s *Service) ensureReadAccess(ctx context.Context) error {
	if err := s.authz.EnsureModule(ctx, "finance.settings"); err != nil {
		return err
	}
	if err := s.authz.EnsureFeature(ctx, "finance.settings.read"); err != nil {
		return err
	}
	return s.ensureAnyPermission(ctx, "finance.settings.read", "finance.settings.roles.manage", "finance.notifications.read")
}

func (s *Service) ensureWriteAccess(ctx context.Context) error {
	if err := s.authz.EnsureModule(ctx, "finance.settings"); err != nil {
		return err
	}
	if err := s.authz.EnsureFeature(ctx, "finance.settings.write"); err != nil {
		return err
	}
	return s.ensureAnyPermission(ctx, "finance.settings.update", "finance.notifications.create", "finance.notifications.read")
}

func (s *Service) ensureRoleManageAccess(ctx context.Context) error {
	if err := s.authz.EnsureModule(ctx, "finance.settings"); err != nil {
		return err
	}
	if err := s.authz.EnsureFeature(ctx, "finance.settings.write"); err != nil {
		return err
	}
	return s.ensureAnyPermission(ctx, "finance.settings.roles.manage", "core.entitlement.manage")
}

func (s *Service) ensureAuditAccess(ctx context.Context) error {
	if err := s.authz.EnsureModule(ctx, "finance.settings"); err != nil {
		return err
	}
	if err := s.authz.EnsureFeature(ctx, "finance.settings.read"); err != nil {
		return err
	}
	return s.ensureAnyPermission(ctx, "finance.settings.audit.read", "finance.settings.read")
}

func (s *Service) ensureAnyPermission(ctx context.Context, permissionCodes ...string) error {
	var deniedErr error
	for _, permissionCode := range permissionCodes {
		if strings.TrimSpace(permissionCode) == "" {
			continue
		}
		err := s.authz.EnsurePermission(ctx, permissionCode)
		if err == nil {
			return nil
		}
		if errors.Is(err, access.ErrPermissionDenied) {
			deniedErr = err
			continue
		}
		return err
	}
	if deniedErr != nil {
		return deniedErr
	}
	return access.ErrPermissionDenied
}

func normalizeRoleCode(code string) string {
	return strings.ToLower(strings.TrimSpace(code))
}

func validateUserIdentity(email, fullName string) (string, string, error) {
	cleanEmail := strings.ToLower(strings.TrimSpace(email))
	cleanName := strings.TrimSpace(fullName)
	if cleanName == "" {
		return "", "", domain.ErrInvalidUserName
	}
	if !strings.Contains(cleanEmail, "@") || strings.HasPrefix(cleanEmail, "@") || strings.HasSuffix(cleanEmail, "@") {
		return "", "", domain.ErrInvalidUserEmail
	}
	return cleanEmail, cleanName, nil
}

func normalizeMembershipStatus(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "invited", "suspended":
		return strings.ToLower(strings.TrimSpace(status))
	default:
		return "active"
	}
}

func isAllowedChannelCode(code string) bool {
	switch code {
	case "email", "telegram", "whatsapp_official", "whatsapp_gowa", "whatsapp_fonte":
		return true
	default:
		return false
	}
}

func isAllowedTemplateChannel(code string) bool {
	switch code {
	case "any", "email", "telegram", "whatsapp_official", "whatsapp_gowa", "whatsapp_fonte":
		return true
	default:
		return false
	}
}

func mergeMaskedConfig(oldJSON, newJSON []byte) []byte {
	if len(oldJSON) == 0 || len(newJSON) == 0 {
		return newJSON
	}
	var oldMap, newMap map[string]any
	if err := json.Unmarshal(oldJSON, &oldMap); err != nil {
		return newJSON
	}
	if err := json.Unmarshal(newJSON, &newMap); err != nil {
		return newJSON
	}

	changed := false
	for k, v := range newMap {
		s, ok := v.(string)
		if ok && strings.Contains(s, "********") {
			if oldVal, exists := oldMap[k]; exists {
				newMap[k] = oldVal
				changed = true
			}
		}
	}

	if !changed {
		return newJSON
	}
	out, err := json.Marshal(newMap)
	if err != nil {
		return newJSON
	}
	return out
}
