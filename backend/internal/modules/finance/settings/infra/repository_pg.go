package infra

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"pekan/backend/internal/modules/finance/settings/domain"
	"pekan/backend/internal/platform/db"
)

type RepositoryPG struct {
	conn *sql.DB
}

func NewRepositoryPG(conn *sql.DB) *RepositoryPG {
	return &RepositoryPG{conn: conn}
}

func (r *RepositoryPG) ListNotificationChannels(ctx context.Context, tenantID string) ([]domain.NotificationChannel, error) {
	var items []domain.NotificationChannel
	err := db.WithTenantTx(ctx, r.conn, func(tx *sql.Tx) error {
		const q = `
SELECT id, channel_code, is_enabled, config_json, created_at, updated_at
FROM finance_notification_channels
ORDER BY channel_code ASC`

		rows, err := tx.QueryContext(ctx, q)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var item domain.NotificationChannel
			if err := rows.Scan(
				&item.ID,
				&item.ChannelCode,
				&item.IsEnabled,
				&item.ConfigJSON,
				&item.CreatedAt,
				&item.UpdatedAt,
			); err != nil {
				return err
			}
			item.TenantID = tenantID
			items = append(items, item)
		}
		return rows.Err()
	})
	return items, err
}

func (r *RepositoryPG) UpsertNotificationChannel(ctx context.Context, in domain.NotificationChannel) (domain.NotificationChannel, error) {
	var out domain.NotificationChannel
	err := db.WithTenantTx(ctx, r.conn, func(tx *sql.Tx) error {
		if in.ID == "" {
			in.ID = uuid.NewString()
		}

		const q = `
INSERT INTO finance_notification_channels (
  id, channel_code, is_enabled, config_json, created_by, updated_by, created_at, updated_at
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
ON CONFLICT (channel_code)
DO UPDATE SET
  is_enabled = EXCLUDED.is_enabled,
  config_json = EXCLUDED.config_json,
  updated_by = EXCLUDED.updated_by,
  updated_at = EXCLUDED.updated_at
RETURNING id, channel_code, is_enabled, config_json, created_at, updated_at`

		err := tx.QueryRowContext(ctx, q,
			in.ID, in.ChannelCode, in.IsEnabled, in.ConfigJSON,
			in.CreatedBy, in.UpdatedBy, in.CreatedAt, in.UpdatedAt,
		).Scan(
			&out.ID, &out.ChannelCode, &out.IsEnabled, &out.ConfigJSON, &out.CreatedAt, &out.UpdatedAt,
		)
		if err != nil {
			return err
		}
		out.TenantID = in.TenantID
		return nil
	})
	return out, err
}

func (r *RepositoryPG) ListMessageTemplates(ctx context.Context, tenantID string, templateCode string) ([]domain.MessageTemplate, error) {
	var items []domain.MessageTemplate
	err := db.WithTenantTx(ctx, r.conn, func(tx *sql.Tx) error {
		clauses := []string{"1=1"}
		args := []any{}
		idx := 1

		if strings.TrimSpace(templateCode) != "" {
			clauses = append(clauses, fmt.Sprintf("template_code = $%d", idx))
			args = append(args, strings.TrimSpace(templateCode))
			idx++
		}

		q := fmt.Sprintf(`
SELECT id, template_code, channel_code, language_code, title_template, body_template, is_enabled, created_at, updated_at
FROM finance_message_templates
WHERE %s
ORDER BY template_code ASC, channel_code ASC, language_code ASC`, strings.Join(clauses, " AND "))

		rows, err := tx.QueryContext(ctx, q, args...)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var item domain.MessageTemplate
			if err := rows.Scan(
				&item.ID,
				&item.TemplateCode,
				&item.ChannelCode,
				&item.LanguageCode,
				&item.TitleTemplate,
				&item.BodyTemplate,
				&item.IsEnabled,
				&item.CreatedAt,
				&item.UpdatedAt,
			); err != nil {
				return err
			}
			item.TenantID = tenantID
			items = append(items, item)
		}
		return rows.Err()
	})
	return items, err
}

func (r *RepositoryPG) UpsertMessageTemplate(ctx context.Context, template domain.MessageTemplate) (domain.MessageTemplate, error) {
	var out domain.MessageTemplate
	err := db.WithTenantTx(ctx, r.conn, func(tx *sql.Tx) error {
		if template.ID == "" {
			template.ID = uuid.NewString()
		}

		const q = `
INSERT INTO finance_message_templates (
  id, template_code, channel_code, language_code, title_template, body_template, is_enabled, created_by, updated_by, created_at, updated_at
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
ON CONFLICT (template_code, channel_code, language_code)
DO UPDATE SET
  title_template = EXCLUDED.title_template,
  body_template = EXCLUDED.body_template,
  is_enabled = EXCLUDED.is_enabled,
  updated_by = EXCLUDED.updated_by,
  updated_at = EXCLUDED.updated_at
RETURNING id, template_code, channel_code, language_code, title_template, body_template, is_enabled, created_at, updated_at`

		err := tx.QueryRowContext(ctx, q,
			template.ID,
			template.TemplateCode,
			template.ChannelCode,
			template.LanguageCode,
			template.TitleTemplate,
			template.BodyTemplate,
			template.IsEnabled,
			template.CreatedBy,
			template.UpdatedBy,
			template.CreatedAt,
			template.UpdatedAt,
		).Scan(
			&out.ID,
			&out.TemplateCode,
			&out.ChannelCode,
			&out.LanguageCode,
			&out.TitleTemplate,
			&out.BodyTemplate,
			&out.IsEnabled,
			&out.CreatedAt,
			&out.UpdatedAt,
		)
		if err != nil {
			return err
		}
		out.TenantID = template.TenantID
		return nil
	})
	return out, err
}

func (r *RepositoryPG) ListRoles(ctx context.Context, tenantID string) ([]domain.Role, error) {
	var items []domain.Role
	err := db.WithTenantTx(ctx, r.conn, func(tx *sql.Tx) error {
		const q = `
SELECT
  r.id,
  r.code,
  r.name,
  r.is_system,
  COALESCE(
    json_agg(rp.permission_id ORDER BY rp.permission_id) FILTER (WHERE rp.permission_id IS NOT NULL),
    '[]'::json
  ) AS permission_ids
FROM roles r
LEFT JOIN role_permissions rp ON rp.role_id = r.id
GROUP BY r.id, r.code, r.name, r.is_system
ORDER BY r.is_system DESC, r.code ASC`

		rows, err := tx.QueryContext(ctx, q)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var item domain.Role
			var permissionIDsRaw []byte
			if err := rows.Scan(&item.ID, &item.Code, &item.Name, &item.IsSystem, &permissionIDsRaw); err != nil {
				return err
			}
			if len(permissionIDsRaw) > 0 {
				if err := json.Unmarshal(permissionIDsRaw, &item.PermissionIDs); err != nil {
					return err
				}
			}
			item.TenantID = tenantID
			items = append(items, item)
		}
		return rows.Err()
	})
	return items, err
}

func (r *RepositoryPG) ListPermissions(ctx context.Context) ([]domain.Permission, error) {
	var items []domain.Permission
	// Permissions are in 'public' schema, WithTenantTx ensures 'public' is in search_path.
	err := db.WithTenantTx(ctx, r.conn, func(tx *sql.Tx) error {
		const q = `
SELECT id, code, name, module_code, action
FROM permissions
ORDER BY module_code ASC, action ASC, code ASC`

		rows, err := tx.QueryContext(ctx, q)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var item domain.Permission
			if err := rows.Scan(&item.ID, &item.Code, &item.Name, &item.ModuleCode, &item.Action); err != nil {
				return err
			}
			items = append(items, item)
		}
		return rows.Err()
	})
	return items, err
}

func (r *RepositoryPG) CreateRole(ctx context.Context, tenantID string, role domain.Role, permissionIDs []string) (domain.Role, error) {
	var created domain.Role
	err := db.WithTenantTx(ctx, r.conn, func(tx *sql.Tx) error {
		cleanPermissions := dedupeRoleIDs(permissionIDs)
		if err := validatePermissionIDsTx(ctx, tx, cleanPermissions); err != nil {
			return err
		}

		roleID := uuid.NewString()
		const insertRoleQuery = `
INSERT INTO roles (id, code, name, is_system, created_at, updated_at)
VALUES ($1, $2, $3, FALSE, $4, $4)`
		now := time.Now().UTC()
		if _, err := tx.ExecContext(ctx, insertRoleQuery, roleID, role.Code, role.Name, now); err != nil {
			if strings.Contains(strings.ToLower(err.Error()), "duplicate") {
				return domain.ErrRoleCodeDuplicate
			}
			return err
		}

		if err := replaceRolePermissionsTx(ctx, tx, roleID, cleanPermissions); err != nil {
			return err
		}

		var err error
		created, err = getRoleByIDTx(ctx, tx, tenantID, roleID)
		return err
	})
	return created, err
}

func (r *RepositoryPG) UpdateRole(ctx context.Context, tenantID string, role domain.Role, permissionIDs []string) (domain.Role, error) {
	var updated domain.Role
	err := db.WithTenantTx(ctx, r.conn, func(tx *sql.Tx) error {
		current, err := getRoleByIDTx(ctx, tx, tenantID, role.ID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return domain.ErrRoleNotFound
			}
			return err
		}
		if current.IsSystem {
			return domain.ErrRoleSystemLocked
		}

		cleanPermissions := dedupeRoleIDs(permissionIDs)
		if err := validatePermissionIDsTx(ctx, tx, cleanPermissions); err != nil {
			return err
		}

		const updateRoleQuery = `
UPDATE roles
SET code = $1, name = $2, updated_at = $3
WHERE id = $4`
		if _, err := tx.ExecContext(ctx, updateRoleQuery, role.Code, role.Name, time.Now().UTC(), role.ID); err != nil {
			if strings.Contains(strings.ToLower(err.Error()), "duplicate") {
				return domain.ErrRoleCodeDuplicate
			}
			return err
		}

		if err := replaceRolePermissionsTx(ctx, tx, role.ID, cleanPermissions); err != nil {
			return err
		}

		updated, err = getRoleByIDTx(ctx, tx, tenantID, role.ID)
		return err
	})
	return updated, err
}

func (r *RepositoryPG) DeleteRole(ctx context.Context, tenantID, roleID string) error {
	return db.WithTenantTx(ctx, r.conn, func(tx *sql.Tx) error {
		current, err := getRoleByIDTx(ctx, tx, tenantID, roleID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return domain.ErrRoleNotFound
			}
			return err
		}
		if current.IsSystem {
			return domain.ErrRoleSystemLocked
		}

		if _, err := tx.ExecContext(ctx, `DELETE FROM membership_roles WHERE role_id = $1`, roleID); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM role_permissions WHERE role_id = $1`, roleID); err != nil {
			return err
		}
		res, err := tx.ExecContext(ctx, `DELETE FROM roles WHERE id = $1`, roleID)
		if err != nil {
			return err
		}
		affected, err := res.RowsAffected()
		if err != nil {
			return err
		}
		if affected == 0 {
			return domain.ErrRoleNotFound
		}
		return nil
	})
}

func (r *RepositoryPG) ListTenantUsers(ctx context.Context, tenantID string) ([]domain.TenantUser, error) {
	var items []domain.TenantUser
	err := db.WithTenantTx(ctx, r.conn, func(tx *sql.Tx) error {
		const q = `
SELECT
  tm.id,
  tm.user_id,
  u.email,
  u.full_name,
  COALESCE(up.username, '') as username,
  tm.status,
  u.is_active,
  u.last_login_at,
  up.phone as phone_number,
  r.id,
  r.code,
  r.name,
  r.is_system
FROM tenant_memberships tm
JOIN public.users u ON u.id = tm.user_id
LEFT JOIN public.user_profiles up ON up.user_id = u.id
LEFT JOIN membership_roles mr ON mr.membership_id = tm.id
LEFT JOIN roles r ON r.id = mr.role_id
LEFT JOIN whatsapp_sessions ws ON ws.user_id = tm.user_id
ORDER BY u.email ASC, r.code ASC`

		rows, err := tx.QueryContext(ctx, q)
		if err != nil {
			return err
		}
		defer rows.Close()

		indexByMembership := make(map[string]int)
		for rows.Next() {
			var (
				membershipID string
				userID       string
				email        string
				fullName     string
				username     sql.NullString
				status       string
				isActive     bool
				lastLoginAt  sql.NullTime
				waNumber     sql.NullString

				roleID       sql.NullString
				roleCode     sql.NullString
				roleName     sql.NullString
				roleIsSystem sql.NullBool
			)

			if err := rows.Scan(
				&membershipID, &userID, &email, &fullName, &username, &status,
				&isActive, &lastLoginAt, &waNumber,
				&roleID, &roleCode, &roleName, &roleIsSystem,
			); err != nil {
				return err
			}

			idx, exists := indexByMembership[membershipID]
			if !exists {
				entry := domain.TenantUser{
					MembershipID: membershipID,
					UserID:       userID,
					Email:        email,
					FullName:     fullName,
					Username:     nil,
					Status:       status,
					IsActive:     isActive,
					Roles:        make([]domain.Role, 0),
				}
				if lastLoginAt.Valid {
					entry.LastLoginAt = &lastLoginAt.Time
				}
				if waNumber.Valid {
					entry.WhatsAppNumber = &waNumber.String
				}
				if username.Valid {
					entry.Username = &username.String
				}
				items = append(items, entry)
				idx = len(items) - 1
				indexByMembership[membershipID] = idx
			}

			if roleID.Valid {
				items[idx].Roles = append(items[idx].Roles, domain.Role{
					ID:       roleID.String,
					Code:     roleCode.String,
					Name:     roleName.String,
					IsSystem: roleIsSystem.Valid && roleIsSystem.Bool,
				})
			}
		}
		return rows.Err()
	})
	return items, err
}

func (r *RepositoryPG) CreateTenantUser(ctx context.Context, tenantID string, user domain.TenantUser, passwordHash string, roleIDs []string) (domain.TenantUser, error) {
	var created domain.TenantUser
	err := db.WithTenantTx(ctx, r.conn, func(tx *sql.Tx) error {
		cleanRoleIDs := dedupeRoleIDs(roleIDs)
		if err := validateRoleIDsTx(ctx, tx, tenantID, cleanRoleIDs); err != nil {
			return err
		}

		userID := uuid.NewString()
		now := time.Now().UTC()
		const insertUserQuery = `
INSERT INTO public.users (id, email, password_hash, full_name, is_active, created_at, updated_at)
VALUES ($1,$2,$3,$4,$5,$6,$6)`
		if _, err := tx.ExecContext(ctx, insertUserQuery, userID, user.Email, passwordHash, user.FullName, user.IsActive, now); err != nil {
			if strings.Contains(strings.ToLower(err.Error()), "duplicate") {
				return domain.ErrUserEmailDuplicate
			}
			return err
		}

		membershipID := uuid.NewString()
		const insertMembershipQuery = `
INSERT INTO tenant_memberships (id, user_id, status, joined_at, created_at)
VALUES ($1,$2,$3,$4,$4)`
		if _, err := tx.ExecContext(ctx, insertMembershipQuery, membershipID, userID, user.Status, now); err != nil {
			return err
		}

		// Sync with public registry for login access
		const insertPublicMembershipQuery = `
INSERT INTO public.tenant_memberships (id, tenant_id, user_id, status, joined_at, created_at)
VALUES ($1,$2,$3,$4,$5,$5)`
		if _, err := tx.ExecContext(ctx, insertPublicMembershipQuery, membershipID, tenantID, userID, user.Status, now); err != nil {
			return err
		}

		if userID != "" {
			const upsertProfileQuery = `
INSERT INTO public.user_profiles (user_id, username, phone, created_at, updated_at)
VALUES ($1, COALESCE($2, split_part($3, '@', 1)), $4, $5, $5)
ON CONFLICT (user_id)
DO UPDATE SET 
  username = COALESCE(EXCLUDED.username, user_profiles.username),
  phone = EXCLUDED.phone,
  updated_at = EXCLUDED.updated_at`
			if _, err := tx.ExecContext(ctx, upsertProfileQuery, userID, user.Username, user.Email, user.WhatsAppNumber, now); err != nil {
				return err
			}
		}

		if err := replaceMembershipRolesTx(ctx, tx, membershipID, cleanRoleIDs); err != nil {
			return err
		}

		var err error
		created, err = getTenantUserByMembershipTx(ctx, tx, tenantID, membershipID)
		return err
	})
	return created, err
}

func (r *RepositoryPG) UpdateTenantUser(ctx context.Context, tenantID string, user domain.TenantUser, passwordHash *string, roleIDs []string) (domain.TenantUser, error) {
	var updated domain.TenantUser
	err := db.WithTenantTx(ctx, r.conn, func(tx *sql.Tx) error {
		userID, err := getUserIDByMembershipTx(ctx, tx, tenantID, user.MembershipID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return domain.ErrUserNotFound
			}
			return err
		}

		cleanRoleIDs := dedupeRoleIDs(roleIDs)
		if err := validateRoleIDsTx(ctx, tx, tenantID, cleanRoleIDs); err != nil {
			return err
		}

		if passwordHash != nil {
			const updateUserWithPassword = `
UPDATE public.users
SET email = $1, full_name = $2, is_active = $3, password_hash = $4, updated_at = $5
WHERE id = $6`
			if _, err := tx.ExecContext(ctx, updateUserWithPassword, user.Email, user.FullName, user.IsActive, *passwordHash, time.Now().UTC(), userID); err != nil {
				if strings.Contains(strings.ToLower(err.Error()), "duplicate") {
					return domain.ErrUserEmailDuplicate
				}
				return err
			}
		} else {
			const updateUser = `
UPDATE public.users
SET email = $1, full_name = $2, is_active = $3, updated_at = $4
WHERE id = $5`
			if _, err := tx.ExecContext(ctx, updateUser, user.Email, user.FullName, user.IsActive, time.Now().UTC(), userID); err != nil {
				if strings.Contains(strings.ToLower(err.Error()), "duplicate") {
					return domain.ErrUserEmailDuplicate
				}
				return err
			}
		}

		const updateMembershipQuery = `
UPDATE tenant_memberships
SET status = $1
WHERE id = $2`
		res, err := tx.ExecContext(ctx, updateMembershipQuery, user.Status, user.MembershipID)
		if err != nil {
			return err
		}
		affected, err := res.RowsAffected()
		if err != nil {
			return err
		}
		if affected == 0 {
			return domain.ErrUserNotFound
		}

		// Sync status with public registry
		const updatePublicMembershipQuery = `
UPDATE public.tenant_memberships
SET status = $1
WHERE id = $2`
		if _, err := tx.ExecContext(ctx, updatePublicMembershipQuery, user.Status, user.MembershipID); err != nil {
			return err
		}

		if userID != "" {
			const upsertProfileQuery = `
INSERT INTO public.user_profiles (user_id, username, phone, created_at, updated_at)
VALUES ($1, COALESCE($2, split_part($3, '@', 1)), $4, $5, $5)
ON CONFLICT (user_id)
DO UPDATE SET 
  username = COALESCE(EXCLUDED.username, user_profiles.username),
  phone = EXCLUDED.phone,
  updated_at = EXCLUDED.updated_at`
			if _, err := tx.ExecContext(ctx, upsertProfileQuery, userID, user.Username, user.Email, user.WhatsAppNumber, time.Now().UTC()); err != nil {
				return err
			}
		}

		if err := replaceMembershipRolesTx(ctx, tx, user.MembershipID, cleanRoleIDs); err != nil {
			return err
		}

		var err2 error
		updated, err2 = getTenantUserByMembershipTx(ctx, tx, tenantID, user.MembershipID)
		return err2
	})
	return updated, err
}

func (r *RepositoryPG) DeleteTenantUser(ctx context.Context, tenantID, membershipID string) error {
	return db.WithTenantTx(ctx, r.conn, func(tx *sql.Tx) error {
		userID, err := getUserIDByMembershipTx(ctx, tx, tenantID, membershipID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return domain.ErrUserNotFound
			}
			return err
		}

		if _, err := tx.ExecContext(ctx, `DELETE FROM membership_roles WHERE membership_id = $1`, membershipID); err != nil {
			return err
		}
		res, err := tx.ExecContext(ctx, `DELETE FROM tenant_memberships WHERE id = $1`, membershipID)
		if err != nil {
			return err
		}
		affected, err := res.RowsAffected()
		if err != nil {
			return err
		}
		if affected == 0 {
			return domain.ErrUserNotFound
		}

		// Sync removal with public registry
		if _, err := tx.ExecContext(ctx, `DELETE FROM public.tenant_memberships WHERE id = $1`, membershipID); err != nil {
			return err
		}

		var remainingMemberships int
		// Cross-schema check might be needed if we want to know if user is in OTHER tenants
		// For now, this only checks memberships in the CURRENT tenant schema.
		if err := tx.QueryRowContext(ctx, `SELECT COUNT(1) FROM tenant_memberships WHERE user_id = $1`, userID).Scan(&remainingMemberships); err != nil {
			return err
		}
		if remainingMemberships == 0 {
			if _, err := tx.ExecContext(ctx, `UPDATE public.users SET is_active = FALSE, updated_at = $1 WHERE id = $2`, time.Now().UTC(), userID); err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *RepositoryPG) ListMembershipsWithRoles(ctx context.Context, tenantID string) ([]domain.MembershipWithRoles, error) {
	var items []domain.MembershipWithRoles
	err := db.WithTenantTx(ctx, r.conn, func(tx *sql.Tx) error {
		const q = `
SELECT
  tm.id,
  tm.user_id,
  u.email,
  u.full_name,
  COALESCE(up.username, '') as username,
  tm.status,
  up.phone as phone_number,
  r.id,
  r.code,
  r.name,
  r.is_system
FROM tenant_memberships tm
JOIN public.users u ON u.id = tm.user_id
LEFT JOIN public.user_profiles up ON up.user_id = u.id
LEFT JOIN membership_roles mr ON mr.membership_id = tm.id
LEFT JOIN roles r ON r.id = mr.role_id
ORDER BY u.email ASC, r.code ASC`

		rows, err := tx.QueryContext(ctx, q)
		if err != nil {
			return err
		}
		defer rows.Close()

		indexByMembership := make(map[string]int)
		for rows.Next() {
			var (
				membershipID string
				userID       string
				email        string
				fullName     string
				username     sql.NullString
				status       string
				waNumber     sql.NullString

				roleID       sql.NullString
				roleCode     sql.NullString
				roleName     sql.NullString
				roleIsSystem sql.NullBool
			)

			if err := rows.Scan(
				&membershipID, &userID, &email, &fullName, &username, &status, &waNumber,
				&roleID, &roleCode, &roleName, &roleIsSystem,
			); err != nil {
				return err
			}

			idx, exists := indexByMembership[membershipID]
			if !exists {
				items = append(items, domain.MembershipWithRoles{
					MembershipID: membershipID,
					UserID:       userID,
					Email:        email,
					FullName:     fullName,
					Username:     nil,
					Status:       status,
					Roles:        make([]domain.Role, 0),
				})
				if username.Valid {
					items[len(items)-1].Username = &username.String
				}
				if waNumber.Valid {
					items[len(items)-1].WhatsAppNumber = &waNumber.String
				}
				idx = len(items) - 1
				indexByMembership[membershipID] = idx
			}

			if roleID.Valid {
				items[idx].Roles = append(items[idx].Roles, domain.Role{
					ID:       roleID.String,
					Code:     roleCode.String,
					Name:     roleName.String,
					IsSystem: roleIsSystem.Valid && roleIsSystem.Bool,
				})
			}
		}
		return rows.Err()
	})
	return items, err
}

func (r *RepositoryPG) ReplaceMembershipRoles(ctx context.Context, tenantID, membershipID string, roleIDs []string) error {
	return db.WithTenantTx(ctx, r.conn, func(tx *sql.Tx) error {
		var membershipExists int
		const membershipQuery = `SELECT 1 FROM tenant_memberships WHERE id = $1`
		if err := tx.QueryRowContext(ctx, membershipQuery, membershipID).Scan(&membershipExists); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return domain.ErrMembershipNotFound
			}
			return err
		}

		uniqueRoleIDs := dedupeRoleIDs(roleIDs)
		if len(uniqueRoleIDs) > 0 {
			placeholders := make([]string, 0, len(uniqueRoleIDs))
			args := make([]any, 0, len(uniqueRoleIDs))
			for i, roleID := range uniqueRoleIDs {
				placeholders = append(placeholders, fmt.Sprintf("$%d", i+1))
				args = append(args, roleID)
			}

			countQuery := fmt.Sprintf(`SELECT COUNT(1) FROM roles WHERE id IN (%s)`, strings.Join(placeholders, ","))
			var roleCount int
			if err := tx.QueryRowContext(ctx, countQuery, args...).Scan(&roleCount); err != nil {
				return err
			}
			if roleCount != len(uniqueRoleIDs) {
				return domain.ErrRoleNotFound
			}
		}

		const deleteQuery = `DELETE FROM membership_roles WHERE membership_id = $1`
		if _, err := tx.ExecContext(ctx, deleteQuery, membershipID); err != nil {
			return err
		}

		const insertQuery = `INSERT INTO membership_roles (membership_id, role_id, created_at) VALUES ($1,$2,$3)`
		now := time.Now().UTC()
		for _, roleID := range uniqueRoleIDs {
			if _, err := tx.ExecContext(ctx, insertQuery, membershipID, roleID, now); err != nil {
				return err
			}
		}
		return nil
	})
}


func (r *RepositoryPG) ListAuditLogs(ctx context.Context, filter domain.AuditLogFilter) ([]domain.AuditLogItem, int64, error) {
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.PageSize <= 0 || filter.PageSize > 200 {
		filter.PageSize = 50
	}

	var items []domain.AuditLogItem
	var total int64

	err := db.WithTenantTx(ctx, r.conn, func(tx *sql.Tx) error {
		// DIAGNOSTIC LOGGING - START
		fmt.Printf("[DIAGNOSTIC] === AUDIT LOGS DATABASE CHECK ===\n")
		var totalAll int64
		_ = tx.QueryRowContext(ctx, "SELECT COUNT(1) FROM public.audit_logs").Scan(&totalAll)
		fmt.Printf("[DIAGNOSTIC] Total rows in public.audit_logs: %d\n", totalAll)

		var tenantCount int64
		_ = tx.QueryRowContext(ctx, "SELECT COUNT(DISTINCT tenant_id) FROM public.audit_logs").Scan(&tenantCount)
		fmt.Printf("[DIAGNOSTIC] Number of unique tenant_ids in audit_logs: %d\n", tenantCount)

		rowsD, errD := tx.QueryContext(ctx, "SELECT id, COALESCE(tenant_id::text, 'NULL'), action FROM public.audit_logs ORDER BY created_at DESC LIMIT 5")
		if errD == nil {
			fmt.Printf("[DIAGNOSTIC] Last 5 audit logs:\n")
			for rowsD.Next() {
				var id int64
				var tid, action string
				if errS := rowsD.Scan(&id, &tid, &action); errS == nil {
					fmt.Printf("[DIAGNOSTIC]   - ID: %d, TenantID: %s, Action: %s\n", id, tid, action)
				}
			}
			rowsD.Close()
		} else {
			fmt.Printf("[DIAGNOSTIC] Diag query error: %v\n", errD)
		}
		fmt.Printf("[DIAGNOSTIC] Expected TenantID: %s\n", filter.TenantID)
		fmt.Printf("[DIAGNOSTIC] ==================================\n")
		// DIAGNOSTIC LOGGING - END

		clauses := []string{"l.tenant_id = $1"}
		args := []any{filter.TenantID}
		idx := 2

		if filter.ActorUserID != nil && strings.TrimSpace(*filter.ActorUserID) != "" {
			search := strings.TrimSpace(*filter.ActorUserID)
			clauses = append(clauses, fmt.Sprintf(`(
				l.actor_user_id::text ILIKE $%d
				OR COALESCE(NULLIF(up.username, ''), NULLIF(u.full_name, ''), NULLIF(u.email, ''), '') ILIKE $%d
			)`, idx, idx))
			args = append(args, "%"+search+"%")
			idx++
		}
		if filter.Action != nil && strings.TrimSpace(*filter.Action) != "" {
			clauses = append(clauses, fmt.Sprintf("l.action ILIKE $%d", idx))
			args = append(args, "%"+strings.TrimSpace(*filter.Action)+"%")
			idx++
		}
		if filter.ResourceType != nil && strings.TrimSpace(*filter.ResourceType) != "" {
			clauses = append(clauses, fmt.Sprintf("l.resource_type ILIKE $%d", idx))
			args = append(args, "%"+strings.TrimSpace(*filter.ResourceType)+"%")
			idx++
		}
		if filter.DateFrom != nil {
			clauses = append(clauses, fmt.Sprintf("l.created_at >= $%d", idx))
			args = append(args, *filter.DateFrom)
			idx++
		}
		if filter.DateTo != nil {
			clauses = append(clauses, fmt.Sprintf("l.created_at <= $%d", idx))
			args = append(args, *filter.DateTo)
			idx++
		}

		where := strings.Join(clauses, " AND ")
		joins := `
FROM public.audit_logs l
LEFT JOIN public.users u ON u.id = l.actor_user_id
LEFT JOIN public.user_profiles up ON up.user_id = u.id`

		countQuery := "SELECT COUNT(1) " + joins + " WHERE " + where
		if err := tx.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
			return err
		}

		offset := (filter.Page - 1) * filter.PageSize
		dataArgs := append([]any{}, args...)
		dataArgs = append(dataArgs, filter.PageSize, offset)
		dataQuery := fmt.Sprintf(`
SELECT
  l.id, l.actor_user_id,
  COALESCE(NULLIF(up.username, ''), NULLIF(u.full_name, ''), NULLIF(u.email, ''), l.actor_user_id::text) AS actor_user_name,
  l.action, l.resource_type, l.resource_id,
  l.before_json, l.after_json, host(l.ip_address), l.user_agent, l.created_at
%s
WHERE %s
ORDER BY l.created_at DESC
LIMIT $%d OFFSET $%d`, joins, where, idx, idx+1)

		rows, err := tx.QueryContext(ctx, dataQuery, dataArgs...)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var (
				item domain.AuditLogItem
				actorUserID   sql.NullString
				actorUserName sql.NullString
				ipAddress     sql.NullString
				userAgent     sql.NullString
				beforeJSON    sql.NullString
				afterJSON     sql.NullString
			)
			if err := rows.Scan(
				&item.ID, &actorUserID, &actorUserName,
				&item.Action, &item.ResourceType, &item.ResourceID,
				&beforeJSON, &afterJSON, &ipAddress, &userAgent, &item.CreatedAt,
			); err != nil {
				return err
			}
			if beforeJSON.Valid {
				item.BeforeJSON = []byte(beforeJSON.String)
			}
			if afterJSON.Valid {
				item.AfterJSON = []byte(afterJSON.String)
			}
			item.TenantID = &filter.TenantID
			if actorUserID.Valid {
				item.ActorUserID = &actorUserID.String
			}
			if actorUserName.Valid {
				item.ActorUserName = &actorUserName.String
			}
			if ipAddress.Valid {
				item.IPAddress = &ipAddress.String
			}
			if userAgent.Valid {
				item.UserAgent = &userAgent.String
			}
			items = append(items, item)
		}
		return rows.Err()
	})

	return items, total, err
}

func getRoleByIDTx(ctx context.Context, tx *sql.Tx, tenantID, roleID string) (domain.Role, error) {
	const q = `
SELECT
  r.id,
  r.code,
  r.name,
  r.is_system,
  COALESCE(
    json_agg(rp.permission_id ORDER BY rp.permission_id) FILTER (WHERE rp.permission_id IS NOT NULL),
    '[]'::json
  ) AS permission_ids
FROM roles r
LEFT JOIN role_permissions rp ON rp.role_id = r.id
WHERE r.id = $1
GROUP BY r.id, r.code, r.name, r.is_system`
	var out domain.Role
	var permissionIDsRaw []byte
	if err := tx.QueryRowContext(ctx, q, roleID).Scan(&out.ID, &out.Code, &out.Name, &out.IsSystem, &permissionIDsRaw); err != nil {
		return domain.Role{}, err
	}
	if len(permissionIDsRaw) > 0 {
		if err := json.Unmarshal(permissionIDsRaw, &out.PermissionIDs); err != nil {
			return domain.Role{}, err
		}
	}
	out.TenantID = tenantID
	return out, nil
}

func getUserIDByMembershipTx(ctx context.Context, tx *sql.Tx, tenantID, membershipID string) (string, error) {
	const q = `SELECT user_id FROM tenant_memberships WHERE id = $1`
	var userID string
	if err := tx.QueryRowContext(ctx, q, membershipID).Scan(&userID); err != nil {
		return "", err
	}
	return userID, nil
}

func getTenantUserByMembershipTx(ctx context.Context, tx *sql.Tx, tenantID, membershipID string) (domain.TenantUser, error) {
	const q = `
SELECT
  tm.id,
  tm.user_id,
  u.email,
  u.full_name,
  tm.status,
  u.is_active,
  u.last_login_at,
  r.id,
  r.code,
  r.name,
  r.is_system
FROM tenant_memberships tm
JOIN public.users u ON u.id = tm.user_id
LEFT JOIN membership_roles mr ON mr.membership_id = tm.id
LEFT JOIN roles r ON r.id = mr.role_id
WHERE tm.id = $1
ORDER BY r.code ASC`

	rows, err := tx.QueryContext(ctx, q, membershipID)
	if err != nil {
		return domain.TenantUser{}, err
	}
	defer rows.Close()

	var out domain.TenantUser
	firstRow := true
	for rows.Next() {
		var (
			lastLoginAt  sql.NullTime
			roleID       sql.NullString
			roleCode     sql.NullString
			roleName     sql.NullString
			roleIsSystem sql.NullBool
		)
		if err := rows.Scan(
			&out.MembershipID,
			&out.UserID,
			&out.Email,
			&out.FullName,
			&out.Status,
			&out.IsActive,
			&lastLoginAt,
			&roleID,
			&roleCode,
			&roleName,
			&roleIsSystem,
		); err != nil {
			return domain.TenantUser{}, err
		}
		if firstRow {
			out.Roles = make([]domain.Role, 0)
			if lastLoginAt.Valid {
				out.LastLoginAt = &lastLoginAt.Time
			}
			firstRow = false
		}
		if roleID.Valid {
			out.Roles = append(out.Roles, domain.Role{
				ID:       roleID.String,
				Code:     roleCode.String,
				Name:     roleName.String,
				IsSystem: roleIsSystem.Valid && roleIsSystem.Bool,
			})
		}
	}
	if err := rows.Err(); err != nil {
		return domain.TenantUser{}, err
	}
	if firstRow {
		return domain.TenantUser{}, domain.ErrUserNotFound
	}
	return out, nil
}

func validatePermissionIDsTx(ctx context.Context, tx *sql.Tx, permissionIDs []string) error {
	if len(permissionIDs) == 0 {
		return nil
	}
	placeholders := make([]string, 0, len(permissionIDs))
	args := make([]any, 0, len(permissionIDs))
	for i, permissionID := range permissionIDs {
		placeholders = append(placeholders, fmt.Sprintf("$%d", i+1))
		args = append(args, permissionID)
	}
	query := fmt.Sprintf(`SELECT COUNT(1) FROM permissions WHERE id IN (%s)`, strings.Join(placeholders, ","))
	var count int
	if err := tx.QueryRowContext(ctx, query, args...).Scan(&count); err != nil {
		return err
	}
	if count != len(permissionIDs) {
		return domain.ErrRoleNotFound
	}
	return nil
}

func validateRoleIDsTx(ctx context.Context, tx *sql.Tx, tenantID string, roleIDs []string) error {
	if len(roleIDs) == 0 {
		return nil
	}
	placeholders := make([]string, 0, len(roleIDs))
	args := make([]any, 0, len(roleIDs))
	for i, roleID := range roleIDs {
		placeholders = append(placeholders, fmt.Sprintf("$%d", i+1))
		args = append(args, roleID)
	}
	query := fmt.Sprintf(`SELECT COUNT(1) FROM roles WHERE id IN (%s)`, strings.Join(placeholders, ","))
	var count int
	if err := tx.QueryRowContext(ctx, query, args...).Scan(&count); err != nil {
		return err
	}
	if count != len(roleIDs) {
		return domain.ErrRoleNotFound
	}
	return nil
}

func replaceRolePermissionsTx(ctx context.Context, tx *sql.Tx, roleID string, permissionIDs []string) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM role_permissions WHERE role_id = $1`, roleID); err != nil {
		return err
	}
	const insertQuery = `INSERT INTO role_permissions (role_id, permission_id, created_at) VALUES ($1,$2,$3)`
	now := time.Now().UTC()
	for _, permissionID := range permissionIDs {
		if _, err := tx.ExecContext(ctx, insertQuery, roleID, permissionID, now); err != nil {
			return err
		}
	}
	return nil
}

func replaceMembershipRolesTx(ctx context.Context, tx *sql.Tx, membershipID string, roleIDs []string) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM membership_roles WHERE membership_id = $1`, membershipID); err != nil {
		return err
	}
	const insertQuery = `INSERT INTO membership_roles (membership_id, role_id, created_at) VALUES ($1,$2,$3)`
	now := time.Now().UTC()
	for _, roleID := range roleIDs {
		if _, err := tx.ExecContext(ctx, insertQuery, membershipID, roleID, now); err != nil {
			return err
		}
	}
	return nil
}

func (r *RepositoryPG) IsEmailTaken(ctx context.Context, email string) (bool, error) {
	const q = `SELECT EXISTS(SELECT 1 FROM public.users WHERE email = $1)`
	var exists bool
	err := r.conn.QueryRowContext(ctx, q, email).Scan(&exists)
	return exists, err
}

func (r *RepositoryPG) IsPhoneTaken(ctx context.Context, phone string) (bool, error) {
	const q = `SELECT EXISTS(SELECT 1 FROM public.user_profiles WHERE phone = $1)`
	var exists bool
	err := r.conn.QueryRowContext(ctx, q, phone).Scan(&exists)
	return exists, err
}

func (r *RepositoryPG) IsUsernameTaken(ctx context.Context, username string) (bool, error) {
	const q = `SELECT EXISTS(SELECT 1 FROM public.user_profiles WHERE username = $1)`
	var exists bool
	err := r.conn.QueryRowContext(ctx, q, username).Scan(&exists)
	return exists, err
}

func dedupeRoleIDs(roleIDs []string) []string {
	out := make([]string, 0, len(roleIDs))
	seen := make(map[string]struct{}, len(roleIDs))
	for _, roleID := range roleIDs {
		id := strings.TrimSpace(roleID)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}
