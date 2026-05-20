package infra

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"pekan/backend/internal/modules/core/admin/domain"
	"pekan/backend/internal/platform/db"
	"pekan/backend/internal/platform/tenancy"
	"strings"
	"time"
)

type RepositoryPG struct {
	conn *sql.DB
}

func NewRepositoryPG(conn *sql.DB) *RepositoryPG {
	return &RepositoryPG{conn: conn}
}

func (r *RepositoryPG) BootstrapTenant(ctx context.Context, t domain.Tenant, u domain.User, m domain.Membership) error {
	// 1. Create Tenant and User in PUBLIC schema first
	tx, err := r.conn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	const qT = `INSERT INTO public.tenants (id, code, name, status, timezone, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, now(), now())`
	if _, err := tx.ExecContext(ctx, qT, t.ID, t.Code, t.Name, t.Status, t.Timezone); err != nil {
		if strings.Contains(err.Error(), "unique constraint") || strings.Contains(err.Error(), "23505") {
			return domain.ErrTenantAlreadyExists
		}
		return err
	}

	const qU = `INSERT INTO public.users (id, email, password_hash, full_name, is_active, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, now(), now())`
	if _, err := tx.ExecContext(ctx, qU, u.ID, u.Email, u.PasswordHash, u.FullName, u.IsActive); err != nil {
		if strings.Contains(err.Error(), "unique constraint") || strings.Contains(err.Error(), "23505") {
			return domain.ErrUserAlreadyExists
		}
		return err
	}

	// 2. Create Global Membership
	const qMG = `INSERT INTO public.tenant_memberships (id, tenant_id, user_id, status, joined_at, created_at) VALUES ($1, $2, $3, $4, now(), now())`
	if _, err := tx.ExecContext(ctx, qMG, m.ID, t.ID, u.ID, m.Status); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	// 3. Setup Tenant-Specific Data in Isolated Schema
	tenantCtx := tenancy.WithContext(ctx, tenancy.Context{
		TenantID:   t.ID,
		SchemaName: tenancy.GetSchemaName(t.Code),
	})

	return db.WithTenantTx(tenantCtx, r.conn, func(tx *sql.Tx) error {
		now := time.Now().UTC()

		// 3a. Create Local Membership
		const qML = `INSERT INTO tenant_memberships (id, user_id, status, joined_at, created_at) VALUES ($1, $2, $3, $4, $4)`
		if _, err := tx.ExecContext(ctx, qML, m.ID, u.ID, m.Status, now); err != nil {
			return err
		}

		// 3b. Create Default Roles
		const qRoles = `
INSERT INTO roles (id, code, name, is_system, created_at, updated_at)
VALUES 
    (gen_random_uuid(), 'owner', 'Owner', TRUE, $1, $1),
    (gen_random_uuid(), 'admin', 'Administrator', TRUE, $1, $1)
ON CONFLICT (code) DO NOTHING`
		if _, err := tx.ExecContext(ctx, qRoles, now); err != nil {
			return err
		}


		// 3c. Assign Roles (Owner)
		const qAssign = `
INSERT INTO membership_roles (membership_id, role_id)
SELECT $1, id FROM roles WHERE code = 'owner' LIMIT 1
ON CONFLICT (membership_id, role_id) DO NOTHING`
		if _, err := tx.ExecContext(ctx, qAssign, m.ID); err != nil {
			return err
		}


		// 3d. Associate all permissions to the 'owner' role
		const qPerms = `
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id 
FROM roles r, public.permissions p 
WHERE r.code = 'owner'
ON CONFLICT (role_id, permission_id) DO NOTHING`
		if _, err := tx.ExecContext(ctx, qPerms); err != nil {
			return err
		}


		// 3e. Enable All Active Modules (in Public)
		const qMod = `
INSERT INTO public.tenant_modules (id, tenant_id, module_code, is_enabled, source, created_at, updated_at)
SELECT gen_random_uuid(), $1, code, TRUE, 'system', $2, $2
FROM public.modules
WHERE is_active = TRUE`
		if _, err := tx.ExecContext(ctx, qMod, t.ID, now); err != nil {
			return err
		}

		// 3f. Enable All Active Features (in Public)
		const qFeat = `
INSERT INTO public.tenant_features (id, tenant_id, feature_code, is_enabled, source, created_at, updated_at)
SELECT gen_random_uuid(), $1, code, TRUE, 'system', $2, $2
FROM public.features
WHERE is_active = TRUE`
		if _, err := tx.ExecContext(ctx, qFeat, t.ID, now); err != nil {
			return err
		}

		// 3g. Create Default Account
		const qAcc = `
INSERT INTO finance_accounts (id, name, account_type, currency, opening_balance_minor, is_active, created_by, created_at, updated_at)
VALUES (gen_random_uuid(), 'Kas Utama', 'cash', 'IDR', 0, TRUE, $1, $2, $2)`
		if _, err := tx.ExecContext(ctx, qAcc, u.ID, now); err != nil {
			return err
		}

		// 3h. Create Default Categories
		const qCat = `
INSERT INTO finance_categories (id, name, category_type, is_active, created_by, created_at, updated_at)
VALUES 
    (gen_random_uuid(), 'Makanan & Minuman', 'expense', TRUE, $1, $2, $2),
    (gen_random_uuid(), 'Transportasi', 'expense', TRUE, $1, $2, $2),
    (gen_random_uuid(), 'Peralatan Rumah', 'expense', TRUE, $1, $2, $2),
    (gen_random_uuid(), 'Belanja', 'expense', TRUE, $1, $2, $2),
    (gen_random_uuid(), 'Gaji', 'income', TRUE, $1, $2, $2),
    (gen_random_uuid(), 'Bonus', 'income', TRUE, $1, $2, $2)`
		if _, err := tx.ExecContext(ctx, qCat, u.ID, now); err != nil {
			return err
		}

		return nil
	})
}

func (r *RepositoryPG) CreateTenant(ctx context.Context, t domain.Tenant) error {
	const q = `
INSERT INTO public.tenants (id, code, name, status, timezone, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, now(), now())`
	_, err := r.conn.ExecContext(ctx, q, t.ID, t.Code, t.Name, t.Status, t.Timezone)
	if err != nil && (strings.Contains(err.Error(), "unique constraint") || strings.Contains(err.Error(), "23505")) {
		return domain.ErrTenantAlreadyExists
	}
	return err
}

func (r *RepositoryPG) CreateUser(ctx context.Context, u domain.User) error {
	const q = `
INSERT INTO public.users (id, email, password_hash, full_name, is_active, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, now(), now())`
	_, err := r.conn.ExecContext(ctx, q, u.ID, u.Email, u.PasswordHash, u.FullName, u.IsActive)
	if err != nil && (strings.Contains(err.Error(), "unique constraint") || strings.Contains(err.Error(), "23505")) {
		return domain.ErrUserAlreadyExists
	}
	return err
}

func (r *RepositoryPG) CreateMembership(ctx context.Context, m domain.Membership) error {
	// 1. Get tenant code to resolve schema
	var tenantCode string
	if err := r.conn.QueryRowContext(ctx, `SELECT code FROM public.tenants WHERE id = $1`, m.TenantID).Scan(&tenantCode); err != nil {
		return err
	}

	// 2. Insert into Public
	const qPublic = `
INSERT INTO public.tenant_memberships (id, tenant_id, user_id, status, joined_at, created_at)
VALUES ($1, $2, $3, $4, now(), now())`
	if _, err := r.conn.ExecContext(ctx, qPublic, m.ID, m.TenantID, m.UserID, m.Status); err != nil {
		return err
	}

	// 3. Insert into Isolated Schema
	tenantCtx := tenancy.WithContext(ctx, tenancy.Context{
		TenantID:   m.TenantID,
		SchemaName: tenancy.GetSchemaName(tenantCode),
	})

	return db.WithTenantTx(tenantCtx, r.conn, func(tx *sql.Tx) error {
		const qLocal = `
INSERT INTO tenant_memberships (id, user_id, status, joined_at, created_at)
VALUES ($1, $2, $3, now(), now())`
		_, err := tx.ExecContext(ctx, qLocal, m.ID, m.UserID, m.Status)
		return err
	})
}

func (r *RepositoryPG) ListTenants(ctx context.Context) ([]domain.TenantListItem, error) {
	const q = `
SELECT 
    t.id, t.code, t.name, t.status, t.timezone,
    COALESCE(t.quota_users, 10), COALESCE(t.quota_transactions, 1000),
    COUNT(tm.id) as user_count
FROM public.tenants t
LEFT JOIN public.tenant_memberships tm ON tm.tenant_id = t.id AND tm.status = 'active'
GROUP BY t.id
ORDER BY t.created_at DESC`

	rows, err := r.conn.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.TenantListItem
	for rows.Next() {
		var item domain.TenantListItem
		if err := rows.Scan(&item.ID, &item.Code, &item.Name, &item.Status, &item.Timezone, &item.QuotaUsers, &item.QuotaTransactions, &item.UserCount); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, nil
}

func (r *RepositoryPG) UpdateTenantQuotas(ctx context.Context, tenantID string, users, transactions int) error {
	const q = `UPDATE public.tenants SET quota_users = $1, quota_transactions = $2, updated_at = now() WHERE id = $3`
	_, err := r.conn.ExecContext(ctx, q, users, transactions, tenantID)
	return err
}

func (r *RepositoryPG) UpdateTenant(ctx context.Context, tenantID string, name, status string) error {
	const q = `UPDATE public.tenants SET name = $1, status = $2, updated_at = now() WHERE id = $3`
	_, err := r.conn.ExecContext(ctx, q, name, status, tenantID)
	return err
}


func (r *RepositoryPG) ListTenantModules(ctx context.Context, tenantID string) ([]domain.TenantModule, error) {
	const q = `SELECT module_code, is_enabled FROM public.tenant_modules WHERE tenant_id = $1`
	rows, err := r.conn.QueryContext(ctx, q, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.TenantModule
	for rows.Next() {
		var m domain.TenantModule
		if err := rows.Scan(&m.ModuleCode, &m.IsEnabled); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, nil
}

func (r *RepositoryPG) UpdateTenantModule(ctx context.Context, tenantID, moduleCode string, enabled bool) error {
	const q = `
INSERT INTO public.tenant_modules (id, tenant_id, module_code, is_enabled, source, created_at, updated_at)
VALUES (gen_random_uuid(), $1, $2, $3, 'manual', now(), now())
ON CONFLICT (tenant_id, module_code) DO UPDATE SET is_enabled = $3, updated_at = now()`
	_, err := r.conn.ExecContext(ctx, q, tenantID, moduleCode, enabled)
	return err
}

func (r *RepositoryPG) GetGrowthStats(ctx context.Context, from, to string) (domain.PlatformStats, error) {
	var stats domain.PlatformStats

	// Default to last 30 days if not provided
	whereClause := "created_at > now() - interval '30 days'"
	if from != "" && to != "" {
		whereClause = "created_at >= '" + from + "' AND created_at <= '" + to + " 23:59:59'"
	} else if from != "" {
		whereClause = "created_at >= '" + from + "'"
	}

	qTenants := `SELECT to_char(created_at, 'YYYY-MM-DD') as d, COUNT(*) FROM public.tenants WHERE ` + whereClause + ` GROUP BY d ORDER BY d ASC`
	qUsers := `SELECT to_char(created_at, 'YYYY-MM-DD') as d, COUNT(*) FROM public.users WHERE ` + whereClause + ` GROUP BY d ORDER BY d ASC`

	const qTotals = `
SELECT 
  (SELECT COUNT(*) FROM public.tenants) as tt,
  (SELECT COUNT(*) FROM public.users) as tu,
  (SELECT COUNT(*) FROM finance_transactions) as tx` // Note: finance_transactions is NOT prefixed because it's per-tenant, but this global stats query might be broken if search_path isn't set.
	// Actually, GetGrowthStats in ADMIN module should probably query all tenant schemas or a global aggregation.
	// For now, I'll prefix tenants/users and leave transactions as is (might need a better solution for platform-wide stats).

	rowsT, err := r.conn.QueryContext(ctx, qTenants)
	if err == nil {
		for rowsT.Next() {
			var p domain.GrowthPoint
			if err := rowsT.Scan(&p.Date, &p.Count); err == nil { stats.Tenants = append(stats.Tenants, p) }
		}
		rowsT.Close()
	}

	rowsU, err := r.conn.QueryContext(ctx, qUsers)
	if err == nil {
		for rowsU.Next() {
			var p domain.GrowthPoint
			if err := rowsU.Scan(&p.Date, &p.Count); err == nil { stats.Users = append(stats.Users, p) }
		}
		rowsU.Close()
	}

	_ = r.conn.QueryRowContext(ctx, qTotals).Scan(&stats.TotalTenants, &stats.TotalUsers, &stats.TotalTransactions)

	return stats, nil
}

func (r *RepositoryPG) ListLogs(ctx context.Context) ([]domain.AuditLog, error) {
	const q = `
SELECT 
    l.id, l.tenant_id, t.name as tenant_name, 
    l.actor_user_id, u.full_name as actor_name, 
    l.action, l.resource_type, l.resource_id, l.ip_address, l.created_at
FROM public.audit_logs l
LEFT JOIN public.tenants t ON t.id = l.tenant_id
LEFT JOIN public.users u ON u.id = l.actor_user_id
ORDER BY l.created_at DESC
LIMIT 100`

	rows, err := r.conn.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.AuditLog
	for rows.Next() {
		var item domain.AuditLog
		var tID, tName, aUID, aName, rID, ip sql.NullString
		if err := rows.Scan(&item.ID, &tID, &tName, &aUID, &aName, &item.Action, &item.Resource, &rID, &ip, &item.CreatedAt); err != nil {
			return nil, err
		}
		item.TenantID = tID.String
		item.TenantName = tName.String
		item.ActorUserID = aUID.String
		item.ActorUserName = aName.String
		item.ResourceID = rID.String
		item.IPAddress = ip.String
		
		// Optional: Construct details string if resource_id is present
		if item.ResourceID != "" {
			item.Details = fmt.Sprintf("ID: %s", item.ResourceID)
		}

		out = append(out, item)
	}
	return out, nil
}

func (r *RepositoryPG) AssignRole(ctx context.Context, membershipID, roleCode string) error {
	// 1. Get tenant info
	var tID, tCode string
	const qT = `SELECT t.id, t.code FROM public.tenants t JOIN public.tenant_memberships tm ON tm.tenant_id = t.id WHERE tm.id = $1`
	if err := r.conn.QueryRowContext(ctx, qT, membershipID).Scan(&tID, &tCode); err != nil {
		return err
	}

	// 2. Execute in tenant schema
	tenantCtx := tenancy.WithContext(ctx, tenancy.Context{
		TenantID:   tID,
		SchemaName: tenancy.GetSchemaName(tCode),
	})

	return db.WithTenantTx(tenantCtx, r.conn, func(tx *sql.Tx) error {
		const q = `
INSERT INTO membership_roles (membership_id, role_id)
SELECT $1, id FROM roles WHERE code = $2
LIMIT 1`
		_, err := tx.ExecContext(ctx, q, membershipID, roleCode)
		return err
	})
}

func (r *RepositoryPG) EnableModule(ctx context.Context, tenantID, moduleCode string) error {
	const q = `
INSERT INTO public.tenant_modules (id, tenant_id, module_code, is_enabled, source, created_at, updated_at)
VALUES (gen_random_uuid(), $1, $2, TRUE, 'manual', now(), now())
ON CONFLICT (tenant_id, module_code) DO UPDATE SET is_enabled = TRUE, updated_at = now()`
	_, err := r.conn.ExecContext(ctx, q, tenantID, moduleCode)
	return err
}

func (r *RepositoryPG) Ping(ctx context.Context) error {
	return r.conn.PingContext(ctx)
}

func (r *RepositoryPG) SetGlobalSetting(ctx context.Context, key, value string, encrypted bool) error {
	const q = `
INSERT INTO public.global_settings (key, value, is_encrypted, updated_at)
VALUES ($1, $2, $3, now())
ON CONFLICT (key) DO UPDATE SET value = $2, is_encrypted = $3, updated_at = now()`
	_, err := r.conn.ExecContext(ctx, q, key, value, encrypted)
	return err
}

func (r *RepositoryPG) DeleteTenant(ctx context.Context, tenantID string) error {
	tx, err := r.conn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 1. Get tenant code to identify schema
	var tenantCode string
	if err := tx.QueryRowContext(ctx, `SELECT code FROM tenants WHERE id = $1`, tenantID).Scan(&tenantCode); err != nil {
		if err == sql.ErrNoRows {
			return nil // Already deleted
		}
		return err
	}

	// 2. Get user IDs for cleanup later
	const qUserIDs = `SELECT user_id FROM public.tenant_memberships WHERE tenant_id = $1`
	rows, err := tx.QueryContext(ctx, qUserIDs, tenantID)
	if err != nil {
		return err
	}
	var userIDs []string
	for rows.Next() {
		var uid string
		if err := rows.Scan(&uid); err != nil {
			rows.Close()
			return err
		}
		userIDs = append(userIDs, uid)
	}
	rows.Close()

	// 3. Delete from global tables explicitly to ensure no FK violations
	// Even with ON DELETE CASCADE in migrations, explicit delete is safer
	tx.ExecContext(ctx, `DELETE FROM tenant_modules WHERE tenant_id = $1`, tenantID)
	tx.ExecContext(ctx, `DELETE FROM tenant_features WHERE tenant_id = $1`, tenantID)
	tx.ExecContext(ctx, `DELETE FROM roles WHERE tenant_id = $1`, tenantID)
	tx.ExecContext(ctx, `DELETE FROM tenant_memberships WHERE tenant_id = $1`, tenantID)

	// 4. Delete from tenants table
	if _, err := tx.ExecContext(ctx, `DELETE FROM tenants WHERE id = $1`, tenantID); err != nil {
		return err
	}


	// 4. Drop the physical schema
	schemaName := tenancy.GetSchemaName(tenantCode)
	if schemaName != "public" {
		// Use fmt.Sprintf for schema name because it cannot be parameterized, 
		// but we trust tenantCode from DB.
		dropQ := fmt.Sprintf("DROP SCHEMA IF EXISTS %s CASCADE", schemaName)
		if _, err := tx.ExecContext(ctx, dropQ); err != nil {
			return fmt.Errorf("failed to drop schema %s: %w", schemaName, err)
		}
	}

	// 5. Clean up orphaned users
	if len(userIDs) > 0 {
		for _, uid := range userIDs {
			const qCleanUsers = `
DELETE FROM users 
WHERE id = $1 
AND NOT EXISTS (SELECT 1 FROM tenant_memberships WHERE user_id = $1)`
			if _, err := tx.ExecContext(ctx, qCleanUsers, uid); err != nil {
				return err
			}
		}
	}

	return tx.Commit()
}


func (r *RepositoryPG) GetGlobalSetting(ctx context.Context, key string) (string, bool, error) {
	const q = `SELECT value, is_encrypted FROM global_settings WHERE key = $1`
	var val string
	var enc bool
	err := r.conn.QueryRowContext(ctx, q, key).Scan(&val, &enc)
	if err == sql.ErrNoRows {
		return "", false, nil
	}
	return val, enc, err
}

func (r *RepositoryPG) ExecuteRawQuery(ctx context.Context, query string) (domain.QueryResult, error) {
	cleanQuery := strings.TrimSpace(query)
	if !strings.HasPrefix(strings.ToUpper(cleanQuery), "SELECT") {
		return domain.QueryResult{}, errors.New("hanya perintah SELECT yang diperbolehkan demi keamanan")
	}

	rows, err := r.conn.QueryContext(ctx, cleanQuery)
	if err != nil {
		return domain.QueryResult{}, err
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return domain.QueryResult{}, err
	}

	var result domain.QueryResult
	result.Columns = cols

	for rows.Next() {
		// Create a slice of interface{} to hold the row data
		values := make([]any, len(cols))
		valuePtrs := make([]any, len(cols))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return result, err
		}

		rowMap := make(map[string]any)
		for i, col := range cols {
			val := values[i]
			// Handle byte slices (often used by driver for various types)
			if b, ok := val.([]byte); ok {
				rowMap[col] = string(b)
			} else {
				rowMap[col] = val
			}
		}
		result.Rows = append(result.Rows, rowMap)
	}

	return result, nil
}

func (r *RepositoryPG) GetDatabaseStats(ctx context.Context) ([]domain.DatabaseTable, error) {
	const q = `
SELECT 
    relname as table_name,
    reltuples::bigint as row_count,
    pg_size_pretty(pg_table_size(c.oid)) as data_size,
    pg_size_pretty(pg_indexes_size(c.oid)) as index_size,
    pg_size_pretty(pg_total_relation_size(c.oid)) as total_size
FROM pg_class c
JOIN pg_namespace n ON n.oid = relnamespace
WHERE nspname = 'public' 
  AND relkind = 'r'
ORDER BY pg_total_relation_size(c.oid) DESC`

	rows, err := r.conn.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []domain.DatabaseTable
	for rows.Next() {
		var s domain.DatabaseTable
		if err := rows.Scan(&s.Name, &s.Rows, &s.DataSize, &s.IndexSize, &s.TotalSize); err != nil {
			return nil, err
		}
		stats = append(stats, s)
	}
	return stats, nil
}
func (r *RepositoryPG) ListTenantUsers(ctx context.Context, tenantID string) ([]domain.TenantUser, error) {
	// 1. Get tenant code to resolve schema
	var tenantCode string
	if err := r.conn.QueryRowContext(ctx, `SELECT code FROM public.tenants WHERE id = $1`, tenantID).Scan(&tenantCode); err != nil {
		return nil, err
	}

	schemaName := tenancy.GetSchemaName(tenantCode)
	
	// 2. Query from public registry as master list, then left join local roles
	q := fmt.Sprintf(`
		SELECT DISTINCT ON (u.id)
			u.id, u.email, u.full_name, u.is_active,
			ptm.status, 
			COALESCE(r.name, 'Member') as role_name,
			ptm.joined_at
		FROM public.tenant_memberships ptm
		JOIN public.users u ON u.id = ptm.user_id
		LEFT JOIN %s.tenant_memberships tm ON tm.user_id = u.id
		LEFT JOIN %s.membership_roles mr ON mr.membership_id = tm.id
		LEFT JOIN %s.roles r ON r.id = mr.role_id
		WHERE ptm.tenant_id = $1
		ORDER BY u.id, ptm.joined_at ASC`, schemaName, schemaName, schemaName)

	rows, err := r.conn.QueryContext(ctx, q, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []domain.TenantUser
	for rows.Next() {
		var u domain.TenantUser
		var joinedAt time.Time
		if err := rows.Scan(&u.ID, &u.Email, &u.FullName, &u.IsActive, &u.Status, &u.Role, &joinedAt); err != nil {
			return nil, err
		}
		u.CreatedAt = joinedAt.Format(time.RFC3339)
		users = append(users, u)
	}
	return users, nil
}

func (r *RepositoryPG) UpdateUserPassword(ctx context.Context, userID, hashedPassword string) error {
	const q = `UPDATE public.users SET password_hash = $1, must_change_password = true, updated_at = now() WHERE id = $2`
	_, err := r.conn.ExecContext(ctx, q, hashedPassword, userID)
	return err
}

func (r *RepositoryPG) UpdateUserEmail(ctx context.Context, userID, newEmail string) error {
	const q = `UPDATE public.users SET email = $1, updated_at = now() WHERE id = $2`
	_, err := r.conn.ExecContext(ctx, q, newEmail, userID)
	return err
}

func (r *RepositoryPG) UpdateUserPhone(ctx context.Context, userID, newPhone string) error {
	const q = `
		INSERT INTO public.user_profiles (user_id, phone, created_at, updated_at)
		VALUES ($1, $2, now(), now())
		ON CONFLICT (user_id) DO UPDATE SET phone = $2, updated_at = now()`
	_, err := r.conn.ExecContext(ctx, q, userID, newPhone)
	return err
}

func (r *RepositoryPG) RecordDatabaseStats(ctx context.Context) error {
	// Ensure table exists
	const createTableQ = `
		CREATE TABLE IF NOT EXISTS public.database_stats_history (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			recorded_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			schema_name TEXT NOT NULL,
			total_size_bytes BIGINT NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_db_stats_history_date ON public.database_stats_history (recorded_at DESC);`
	
	if _, err := r.conn.ExecContext(ctx, createTableQ); err != nil {
		return err
	}

	const q = `
		INSERT INTO public.database_stats_history (schema_name, total_size_bytes)
		SELECT 
			schemaname as schema_name, 
			SUM(pg_total_relation_size(quote_ident(schemaname) || '.' || quote_ident(relname)))::bigint as total_size_bytes
		FROM pg_stat_user_tables 
		WHERE schemaname LIKE 'wkspid_pekan_%' OR schemaname = 'public'
		GROUP BY schemaname`
	
	_, err := r.conn.ExecContext(ctx, q)
	return err
}


func (r *RepositoryPG) GetDatabaseGrowth(ctx context.Context) ([]domain.DatabaseGrowthPoint, error) {
	// Ensure table exists
	const createTableQ = `
		CREATE TABLE IF NOT EXISTS public.database_stats_history (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			recorded_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			schema_name TEXT NOT NULL,
			total_size_bytes BIGINT NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_db_stats_history_date ON public.database_stats_history (recorded_at DESC);`
	
	if _, err := r.conn.ExecContext(ctx, createTableQ); err != nil {
		return nil, err
	}

	const q = `
		SELECT recorded_at, schema_name, total_size_bytes
		FROM public.database_stats_history
		WHERE recorded_at >= now() - INTERVAL '30 days'
		ORDER BY recorded_at ASC, schema_name ASC`
	
	rows, err := r.conn.QueryContext(ctx, q)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var history []domain.DatabaseGrowthPoint
	for rows.Next() {
		var p domain.DatabaseGrowthPoint
		var t time.Time
		if err := rows.Scan(&t, &p.SchemaName, &p.TotalSizeBytes); err != nil {
			return nil, err
		}
		p.Date = t.Format(time.RFC3339)
		history = append(history, p)
	}
	return history, nil
}

func (r *RepositoryPG) GetWhatsAppQueueStats(ctx context.Context) (domain.WhatsAppQueueStats, error) {
	var stats domain.WhatsAppQueueStats
	const q = `
SELECT 
    COUNT(*) as total_processed,
    COUNT(CASE WHEN status = 'pending' THEN 1 END) as total_pending,
    COUNT(CASE WHEN status = 'processing' THEN 1 END) as total_processing,
    COUNT(CASE WHEN status = 'success' THEN 1 END) as total_success,
    COUNT(CASE WHEN status = 'failed' THEN 1 END) as total_failed,
    COALESCE(AVG(CASE WHEN status IN ('success', 'failed') THEN processing_time_ms END), 0)::bigint as average_latency_ms
FROM public.whatsapp_bot_queue`
	err := r.conn.QueryRowContext(ctx, q).Scan(
		&stats.TotalProcessed,
		&stats.TotalPending,
		&stats.TotalProcessing,
		&stats.TotalSuccess,
		&stats.TotalFailed,
		&stats.AverageLatencyMs,
	)
	return stats, err
}

func (r *RepositoryPG) GetWhatsAppQueueHistory(ctx context.Context, limit, offset int, search string) ([]domain.WhatsAppQueueItem, int, error) {
	var items []domain.WhatsAppQueueItem
	var total int

	// First count total matching records
	countQ := `
SELECT COUNT(*) 
FROM public.whatsapp_bot_queue q
LEFT JOIN public.tenants t ON q.tenant_id = t.id
LEFT JOIN public.users u ON q.user_id = u.id`

	var args []any
	whereClause := ""
	if search != "" {
		whereClause = " WHERE q.phone_number ILIKE $1 OR q.message ILIKE $1 OR q.reply_message ILIKE $1 OR q.error_message ILIKE $1 OR t.code ILIKE $1 OR u.email ILIKE $1"
		args = append(args, "%"+search+"%")
	}

	err := r.conn.QueryRowContext(ctx, countQ+whereClause, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// Fetch page of items
	dataQ := `
SELECT 
    q.id, q.phone_number, q.message, q.reply_message, q.status, q.error_message, q.processing_time_ms,
    q.tenant_id, t.code as tenant_code, q.user_id, u.email as user_email,
    q.received_at, q.processed_at
FROM public.whatsapp_bot_queue q
LEFT JOIN public.tenants t ON q.tenant_id = t.id
LEFT JOIN public.users u ON q.user_id = u.id` + whereClause + `
ORDER BY q.received_at DESC
LIMIT $%d OFFSET $%d`

	limitIdx := len(args) + 1
	offsetIdx := len(args) + 2
	dataQFormatted := fmt.Sprintf(dataQ, limitIdx, offsetIdx)

	args = append(args, limit, offset)
	rows, err := r.conn.QueryContext(ctx, dataQFormatted, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	for rows.Next() {
		var item domain.WhatsAppQueueItem
		var reply, errMsg, tID, tCode, uID, uEmail sql.NullString
		var lat sql.NullInt32
		var recvTime time.Time
		var procTime sql.NullTime

		err := rows.Scan(
			&item.ID, &item.PhoneNumber, &item.Message, &reply, &item.Status, &errMsg, &lat,
			&tID, &tCode, &uID, &uEmail, &recvTime, &procTime,
		)
		if err != nil {
			return nil, 0, err
		}

		item.ReceivedAt = recvTime.Format(time.RFC3339)
		if reply.Valid {
			s := reply.String
			item.ReplyMessage = &s
		}
		if errMsg.Valid {
			s := errMsg.String
			item.ErrorMessage = &s
		}
		if lat.Valid {
			i := int(lat.Int32)
			item.ProcessingTimeMs = &i
		}
		if tID.Valid {
			s := tID.String
			item.TenantID = &s
		}
		if tCode.Valid {
			s := tCode.String
			item.TenantCode = &s
		}
		if uID.Valid {
			s := uID.String
			item.UserID = &s
		}
		if uEmail.Valid {
			s := uEmail.String
			item.UserEmail = &s
		}
		if procTime.Valid {
			s := procTime.Time.Format(time.RFC3339)
			item.ProcessedAt = &s
		}

		items = append(items, item)
	}

	return items, total, nil
}

func (r *RepositoryPG) RetryWhatsAppQueueMessage(ctx context.Context, id string) error {
	const q = `
UPDATE public.whatsapp_bot_queue
SET status = 'pending', reply_message = NULL, error_message = NULL, processing_time_ms = NULL, processed_at = NULL
WHERE id = $1`
	_, err := r.conn.ExecContext(ctx, q, id)
	return err
}


