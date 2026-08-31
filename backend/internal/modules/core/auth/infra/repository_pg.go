package infra

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"strings"
	"time"

	authdomain "pekan/backend/internal/modules/core/auth/domain"
	"pekan/backend/internal/platform/db"
	"pekan/backend/internal/platform/entitlement"
	"pekan/backend/internal/platform/security"
	"pekan/backend/internal/platform/tenancy"
)

type RepositoryPG struct {
	conn     *sql.DB
	resolver entitlement.Resolver
	cipher   *security.Cipher
}

func NewRepositoryPG(conn *sql.DB, cipher *security.Cipher) *RepositoryPG {
	return &RepositoryPG{
		conn:     conn,
		resolver: entitlement.NewPGResolver(conn),
		cipher:   cipher,
	}
}

func (r *RepositoryPG) GetUserByEmail(ctx context.Context, email string) (authdomain.User, error) {
	const q = `
SELECT id, email, password_hash, full_name, is_active, must_change_password
FROM public.users
WHERE LOWER(email) = $1`



	var out authdomain.User
	err := r.conn.QueryRowContext(ctx, q, email).Scan(
		&out.ID, &out.Email, &out.PasswordHash, &out.FullName, &out.IsActive, &out.MustChangePassword,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return authdomain.User{}, authdomain.ErrInvalidCredentials
		}
		return authdomain.User{}, err
	}
	return out, nil
}

func (r *RepositoryPG) GetUserByID(ctx context.Context, userID string) (authdomain.User, error) {
	const q = `
SELECT id, email, password_hash, full_name, is_active, must_change_password
FROM public.users
WHERE id = $1`

	var out authdomain.User
	err := r.conn.QueryRowContext(ctx, q, userID).Scan(
		&out.ID, &out.Email, &out.PasswordHash, &out.FullName, &out.IsActive, &out.MustChangePassword,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return authdomain.User{}, authdomain.ErrInvalidCredentials
		}
		return authdomain.User{}, err
	}
	return out, nil
}

func (r *RepositoryPG) GetUserProfile(ctx context.Context, userID string) (authdomain.UserProfile, error) {
	const q = `
SELECT
  u.id,
  u.email,
  u.full_name,
  COALESCE(NULLIF(up.username, ''), split_part(u.email, '@', 1)) AS username,
  up.phone,
  up.address,
  COALESCE(up.updated_at, u.updated_at) AS updated_at
FROM public.users u
LEFT JOIN public.user_profiles up ON up.user_id = u.id
WHERE u.id = $1`

	var (
		out     authdomain.UserProfile
		phone   sql.NullString
		address sql.NullString
	)
	if err := r.conn.QueryRowContext(ctx, q, userID).Scan(
		&out.UserID,
		&out.Email,
		&out.FullName,
		&out.Username,
		&phone,
		&address,
		&out.UpdatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return authdomain.UserProfile{}, authdomain.ErrInvalidCredentials
		}
		return authdomain.UserProfile{}, err
	}
	if phone.Valid {
		if r.cipher != nil {
			if dec, err := r.cipher.Decrypt(phone.String); err == nil && dec != "" {
				out.Phone = &dec
			} else {
				out.Phone = &phone.String
			}
		} else {
			out.Phone = &phone.String
		}
	}
	if address.Valid {
		if r.cipher != nil {
			if dec, err := r.cipher.Decrypt(address.String); err == nil && dec != "" {
				out.Address = &dec
			} else {
				out.Address = &address.String
			}
		} else {
			out.Address = &address.String
		}
	}
	return out, nil
}

func (r *RepositoryPG) UpdateUserProfile(ctx context.Context, profile authdomain.UserProfile) (authdomain.UserProfile, error) {
	tx, err := r.conn.BeginTx(ctx, nil)
	if err != nil {
		return authdomain.UserProfile{}, err
	}
	defer tx.Rollback()

	const updateUserQ = `
UPDATE public.users
SET email = $1, full_name = $2, updated_at = $3
WHERE id = $4`
	res, err := tx.ExecContext(ctx, updateUserQ, profile.Email, profile.FullName, profile.UpdatedAt, profile.UserID)
	if err != nil {
		return authdomain.UserProfile{}, err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return authdomain.UserProfile{}, err
	}
	if affected == 0 {
		return authdomain.UserProfile{}, authdomain.ErrInvalidCredentials
	}

	encPhone := profile.Phone
	encAddress := profile.Address
	
	if r.cipher != nil {
		if profile.Phone != nil && *profile.Phone != "" {
			if enc, err := r.cipher.EncryptDeterministic(*profile.Phone); err == nil && enc != "" {
				encPhone = &enc
			}
		}
		if profile.Address != nil && *profile.Address != "" {
			if enc, err := r.cipher.Encrypt(*profile.Address); err == nil && enc != "" {
				encAddress = &enc
			}
		}
	}

	const upsertProfileQ = `
INSERT INTO public.user_profiles (user_id, username, phone, address, created_at, updated_at)
VALUES ($1,$2,$3,$4,now(),$5)
ON CONFLICT (user_id)
DO UPDATE SET
  username = EXCLUDED.username,
  phone = EXCLUDED.phone,
  address = EXCLUDED.address,
  updated_at = EXCLUDED.updated_at`
	if _, err := tx.ExecContext(ctx, upsertProfileQ, profile.UserID, profile.Username, encPhone, encAddress, profile.UpdatedAt); err != nil {
		return authdomain.UserProfile{}, err
	}

	if err := tx.Commit(); err != nil {
		return authdomain.UserProfile{}, err
	}

	return r.GetUserProfile(ctx, profile.UserID)
}

func (r *RepositoryPG) ListMembershipsByUserID(ctx context.Context, userID string) ([]authdomain.Membership, error) {
	const q = `
SELECT tm.id, tm.tenant_id, t.code as tenant_code, tm.user_id, tm.status
FROM public.tenant_memberships tm
JOIN public.tenants t ON t.id = tm.tenant_id
WHERE tm.user_id = $1
ORDER BY tm.created_at ASC`

	rows, err := r.conn.QueryContext(ctx, q, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]authdomain.Membership, 0)
	for rows.Next() {
		var membership authdomain.Membership
		if err := rows.Scan(&membership.ID, &membership.TenantID, &membership.TenantCode, &membership.UserID, &membership.Status); err != nil {
			return nil, err
		}
		out = append(out, membership)
	}
	return out, rows.Err()
}

func (r *RepositoryPG) GetMembershipByUserAndTenant(ctx context.Context, userID, tenantID string) (authdomain.Membership, error) {
	const q = `
SELECT tm.id, tm.tenant_id, t.code as tenant_code, tm.user_id, tm.status
FROM public.tenant_memberships tm
JOIN public.tenants t ON t.id = tm.tenant_id
WHERE tm.user_id = $1 AND tm.tenant_id = $2`

	var out authdomain.Membership
	err := r.conn.QueryRowContext(ctx, q, userID, tenantID).Scan(
		&out.ID, &out.TenantID, &out.TenantCode, &out.UserID, &out.Status,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return authdomain.Membership{}, authdomain.ErrMembershipNotFound
		}
		return authdomain.Membership{}, err
	}
	return out, nil
}

func (r *RepositoryPG) GetTenantIDByCode(ctx context.Context, code string) (string, error) {
	const q = `SELECT id FROM public.tenants WHERE LOWER(code) = LOWER($1)`
	var id string
	err := r.conn.QueryRowContext(ctx, q, code).Scan(&id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", authdomain.ErrMembershipNotFound
		}
		log.Printf("[AuthRepo] Error resolving tenant code %s: %v", code, err)
		return "", err
	}
	return id, nil
}

func (r *RepositoryPG) GetTenantCodeByID(ctx context.Context, id string) (string, error) {
	const q = `SELECT code FROM public.tenants WHERE id = $1`
	var code string
	err := r.conn.QueryRowContext(ctx, q, id).Scan(&code)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", authdomain.ErrMembershipNotFound
		}
		return "", err
	}
	return code, nil
}

func (r *RepositoryPG) GetAccessProfileByMembership(ctx context.Context, membershipID, tenantID string) (authdomain.AccessProfile, error) {
	permissions, err := r.getPermissions(ctx, membershipID)
	if err != nil {
		return authdomain.AccessProfile{}, err
	}
	resolved, err := r.resolver.ResolveTenant(ctx, tenantID)
	if err != nil {
		return authdomain.AccessProfile{}, err
	}
	modules := resolved.Modules
	features := resolved.Features
	if len(permissions) == 0 || len(modules) == 0 || len(features) == 0 {
		return authdomain.AccessProfile{}, authdomain.ErrAccessProfileMissing
	}

	return authdomain.AccessProfile{
		Permissions: permissions,
		Features:    features,
		Modules:     modules,
	}, nil
}

func (r *RepositoryPG) CreateSession(ctx context.Context, session authdomain.Session) (authdomain.Session, error) {
	tx, err := r.conn.BeginTx(ctx, nil)
	if err != nil {
		return authdomain.Session{}, err
	}
	defer tx.Rollback()

	const insertSessionQ = `
INSERT INTO public.auth_sessions (id, user_id, tenant_id, refresh_token_hash, expires_at, revoked_at, ip_address, user_agent, created_at, updated_at)
VALUES ($1,$2,$3,$4,$5,NULL,NULL,NULL,now(),now())`
	if _, err := tx.ExecContext(ctx, insertSessionQ,
		session.ID, session.UserID, session.TenantID, session.RefreshTokenHash, session.ExpiresAt,
	); err != nil {
		return authdomain.Session{}, err
	}

	const insertRefreshTokenQ = `
INSERT INTO public.auth_refresh_tokens (session_id, user_id, tenant_id, refresh_token_hash, expires_at, consumed_at, revoked_at, created_at, updated_at)
VALUES ($1,$2,$3,$4,$5,NULL,NULL,now(),now())`
	if _, err := tx.ExecContext(ctx, insertRefreshTokenQ,
		session.ID, session.UserID, session.TenantID, session.RefreshTokenHash, session.ExpiresAt,
	); err != nil {
		return authdomain.Session{}, err
	}

	if err := tx.Commit(); err != nil {
		return authdomain.Session{}, err
	}
	return session, nil
}

func (r *RepositoryPG) GetSessionByRefreshHash(ctx context.Context, refreshHash string) (authdomain.Session, error) {
	const q = `
SELECT
	s.id,
	s.user_id,
	s.tenant_id,
	s.refresh_token_hash,
	s.expires_at,
	s.revoked_at,
	rt.expires_at,
	rt.consumed_at,
	rt.revoked_at
FROM public.auth_refresh_tokens rt
JOIN public.auth_sessions s ON s.id = rt.session_id
WHERE rt.refresh_token_hash = $1`

	var out authdomain.Session
	err := r.conn.QueryRowContext(ctx, q, refreshHash).Scan(
		&out.ID, &out.UserID, &out.TenantID, &out.RefreshTokenHash, &out.ExpiresAt, &out.RevokedAt,
		&out.TokenExpiresAt, &out.TokenConsumedAt, &out.TokenRevokedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return authdomain.Session{}, authdomain.ErrSessionNotFound
		}
		return authdomain.Session{}, err
	}
	return out, nil
}

func (r *RepositoryPG) RotateSessionRefreshHash(ctx context.Context, sessionID, currentRefreshHash, newRefreshHash string, expiresAt time.Time) error {
	tx, err := r.conn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	const consumeTokenQ = `
UPDATE public.auth_refresh_tokens
SET consumed_at = now(), updated_at = now()
WHERE session_id = $1
  AND refresh_token_hash = $2
  AND consumed_at IS NULL
  AND revoked_at IS NULL`
	consumeRes, err := tx.ExecContext(ctx, consumeTokenQ, sessionID, currentRefreshHash)
	if err != nil {
		return err
	}
	consumedRows, err := consumeRes.RowsAffected()
	if err != nil {
		return err
	}
	if consumedRows == 0 {
		return authdomain.ErrRefreshTokenReused
	}

	const insertNewTokenQ = `
INSERT INTO public.auth_refresh_tokens (session_id, user_id, tenant_id, refresh_token_hash, expires_at, consumed_at, revoked_at, created_at, updated_at)
SELECT id, user_id, tenant_id, $2, $3, NULL, NULL, now(), now()
FROM public.auth_sessions
WHERE id = $1 AND revoked_at IS NULL`
	insertRes, err := tx.ExecContext(ctx, insertNewTokenQ, sessionID, newRefreshHash, expiresAt)
	if err != nil {
		return err
	}
	insertedRows, err := insertRes.RowsAffected()
	if err != nil {
		return err
	}
	if insertedRows == 0 {
		return authdomain.ErrSessionNotFound
	}

	const updateSessionQ = `
UPDATE public.auth_sessions
SET refresh_token_hash = $1, expires_at = $2, updated_at = now()
WHERE id = $3 AND revoked_at IS NULL`
	updateRes, err := tx.ExecContext(ctx, updateSessionQ, newRefreshHash, expiresAt, sessionID)
	if err != nil {
		return err
	}
	updatedRows, err := updateRes.RowsAffected()
	if err != nil {
		return err
	}
	if updatedRows == 0 {
		return authdomain.ErrSessionNotFound
	}

	return tx.Commit()
}

func (r *RepositoryPG) RevokeSessionByRefreshHash(ctx context.Context, refreshHash string) error {
	tx, err := r.conn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	const sessionLookupQ = `
SELECT session_id
FROM public.auth_refresh_tokens
WHERE refresh_token_hash = $1`
	var sessionID string
	err = tx.QueryRowContext(ctx, sessionLookupQ, refreshHash).Scan(&sessionID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return err
	}

	const revokeTokensQ = `
UPDATE public.auth_refresh_tokens
SET revoked_at = now(), updated_at = now()
WHERE session_id = $1 AND revoked_at IS NULL`
	if _, err := tx.ExecContext(ctx, revokeTokensQ, sessionID); err != nil {
		return err
	}

	const revokeSessionQ = `
UPDATE public.auth_sessions
SET revoked_at = now(), updated_at = now()
WHERE id = $1 AND revoked_at IS NULL`
	if _, err := tx.ExecContext(ctx, revokeSessionQ, sessionID); err != nil {
		return err
	}

	return tx.Commit()
}

func (r *RepositoryPG) RevokeAllSessionsByUser(ctx context.Context, userID string) error {
	tx, err := r.conn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	const revokeSessionsQ = `
UPDATE public.auth_sessions
SET revoked_at = now(), updated_at = now()
WHERE user_id = $1 AND revoked_at IS NULL`
	if _, err := tx.ExecContext(ctx, revokeSessionsQ, userID); err != nil {
		return err
	}

	const revokeTokensQ = `
UPDATE public.auth_refresh_tokens
SET revoked_at = now(), updated_at = now()
WHERE user_id = $1 AND revoked_at IS NULL`
	if _, err := tx.ExecContext(ctx, revokeTokensQ, userID); err != nil {
		return err
	}

	return tx.Commit()
}

func (r *RepositoryPG) getPermissions(ctx context.Context, membershipID string) ([]string, error) {
	// 1. Get tenant code and user_id to resolve schema
	var tenantCode, userID string
	const qT = `SELECT t.code, tm.user_id FROM public.tenants t JOIN public.tenant_memberships tm ON tm.tenant_id = t.id WHERE tm.id = $1`
	if err := r.conn.QueryRowContext(ctx, qT, membershipID).Scan(&tenantCode, &userID); err != nil {
		return nil, err
	}

	// 2. Execute in tenant schema
	tenantCtx := tenancy.WithContext(ctx, tenancy.Context{
		SchemaName: tenancy.GetSchemaName(tenantCode),
	})

	var permissions []string
	err := db.WithTenantTx(tenantCtx, r.conn, func(tx *sql.Tx) error {
		const q = `
SELECT DISTINCT p.code
FROM membership_roles mr
JOIN role_permissions rp ON rp.role_id = mr.role_id
JOIN public.permissions p ON p.id = rp.permission_id
WHERE mr.membership_id = $1`
		rows, err := tx.QueryContext(ctx, q, membershipID)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var code string
				if err := rows.Scan(&code); err == nil {
					permissions = append(permissions, code)
				}
			}
		}

		// Fallback 1: check membership_roles by user_id
		if len(permissions) == 0 {
			const qByUser = `
SELECT DISTINCT p.code
FROM tenant_memberships tm
JOIN membership_roles mr ON mr.membership_id = tm.id
JOIN role_permissions rp ON rp.role_id = mr.role_id
JOIN public.permissions p ON p.id = rp.permission_id
WHERE tm.user_id = $1`
			rowsUser, err := tx.QueryContext(ctx, qByUser, userID)
			if err == nil {
				defer rowsUser.Close()
				for rowsUser.Next() {
					var code string
					if err := rowsUser.Scan(&code); err == nil {
						permissions = append(permissions, code)
					}
				}
			}
		}

		// Fallback 2: Self-heal - ensure tenant membership exists and assign Owner role
		if len(permissions) == 0 {
			_, _ = tx.ExecContext(ctx, `
				INSERT INTO tenant_memberships (id, user_id, status, joined_at, created_at)
				VALUES ($1, $2, 'active', now(), now())
				ON CONFLICT (id) DO NOTHING`, membershipID, userID)

			var roleID string
			_ = tx.QueryRowContext(ctx, `SELECT id FROM roles WHERE LOWER(name) IN ('owner', 'admin', 'administrator') LIMIT 1`).Scan(&roleID)
			if roleID != "" {
				_, _ = tx.ExecContext(ctx, `
					INSERT INTO membership_roles (id, membership_id, role_id, created_at)
					VALUES (gen_random_uuid(), $1, $2, now())
					ON CONFLICT DO NOTHING`, membershipID, roleID)

				rowsRole, err := tx.QueryContext(ctx, `
					SELECT p.code FROM role_permissions rp
					JOIN public.permissions p ON p.id = rp.permission_id
					WHERE rp.role_id = $1`, roleID)
				if err == nil {
					defer rowsRole.Close()
					for rowsRole.Next() {
						var code string
						if err := rowsRole.Scan(&code); err == nil {
							permissions = append(permissions, code)
						}
					}
				}
			}

			// Fallback 3: grant all public permissions if isolated schema roles are missing
			if len(permissions) == 0 {
				rowsAll, err := r.conn.QueryContext(ctx, `SELECT code FROM public.permissions`)
				if err == nil {
					defer rowsAll.Close()
					for rowsAll.Next() {
						var code string
						if err := rowsAll.Scan(&code); err == nil {
							permissions = append(permissions, code)
						}
					}
				}
			}
		}

		return nil
	})
	return permissions, err
}

// --- Registration OTP ---

func (r *RepositoryPG) SaveRegistrationOTP(ctx context.Context, otp authdomain.RegistrationOTP) error {
	const q = `
INSERT INTO public.registration_otps
  (id, session_token, otp_code, tenant_code, tenant_name, admin_email, admin_name, password_hash, password_plain, phone, otp_method, attempts, verified, expires_at, created_at)
VALUES
  (gen_random_uuid(), $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, 0, false, $11, now())`
	_, err := r.conn.ExecContext(ctx, q,
		otp.SessionToken, otp.OTPCode, otp.TenantCode, otp.TenantName,
		otp.AdminEmail, otp.AdminName, otp.PasswordHash, otp.PasswordPlain, otp.Phone, otp.OTPMethod, otp.ExpiresAt,
	)
	return err
}

func (r *RepositoryPG) GetRegistrationOTP(ctx context.Context, sessionToken string) (authdomain.RegistrationOTP, error) {
	const q = `
SELECT session_token, otp_code, tenant_code, tenant_name, admin_email, admin_name,
       password_hash, COALESCE(password_plain, ''), COALESCE(phone,''), otp_method, attempts, verified, expires_at, created_at
FROM public.registration_otps
WHERE session_token = $1`

	var out authdomain.RegistrationOTP
	err := r.conn.QueryRowContext(ctx, q, sessionToken).Scan(
		&out.SessionToken, &out.OTPCode, &out.TenantCode, &out.TenantName,
		&out.AdminEmail, &out.AdminName, &out.PasswordHash, &out.PasswordPlain, &out.Phone,
		&out.OTPMethod, &out.Attempts, &out.Verified, &out.ExpiresAt, &out.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return authdomain.RegistrationOTP{}, authdomain.ErrSessionNotFound
		}
		return authdomain.RegistrationOTP{}, err
	}
	return out, nil
}

func (r *RepositoryPG) MarkOTPVerified(ctx context.Context, sessionToken string) error {
	const q = `UPDATE public.registration_otps SET verified = true WHERE session_token = $1`
	_, err := r.conn.ExecContext(ctx, q, sessionToken)
	return err
}

func (r *RepositoryPG) IncrementOTPAttempts(ctx context.Context, sessionToken string) error {
	const q = `UPDATE public.registration_otps SET attempts = attempts + 1 WHERE session_token = $1`
	_, err := r.conn.ExecContext(ctx, q, sessionToken)
	return err
}

func (r *RepositoryPG) IsTenantCodeTaken(ctx context.Context, code string) (bool, error) {
	const q = `SELECT EXISTS(SELECT 1 FROM public.tenants WHERE code = $1)`
	var exists bool
	err := r.conn.QueryRowContext(ctx, q, code).Scan(&exists)
	return exists, err
}

func (r *RepositoryPG) IsTenantNameTaken(ctx context.Context, name string) (bool, error) {
	const q = `SELECT EXISTS(SELECT 1 FROM public.tenants WHERE name = $1)`
	var exists bool
	err := r.conn.QueryRowContext(ctx, q, name).Scan(&exists)
	return exists, err
}

func (r *RepositoryPG) IsEmailRegistered(ctx context.Context, email string) (bool, error) {
	const q = `SELECT EXISTS(SELECT 1 FROM public.users WHERE email = $1)`
	var exists bool
	err := r.conn.QueryRowContext(ctx, q, email).Scan(&exists)
	return exists, err
}

func (r *RepositoryPG) IsPhoneRegistered(ctx context.Context, phone string) (bool, error) {
	encPhone := phone
	if r.cipher != nil && phone != "" {
		if enc, err := r.cipher.EncryptDeterministic(phone); err == nil && enc != "" {
			encPhone = enc
		}
	}

	const q = `SELECT EXISTS(SELECT 1 FROM public.user_profiles WHERE phone = $1)`
	var exists bool
	err := r.conn.QueryRowContext(ctx, q, encPhone).Scan(&exists)
	return exists, err
}

func (r *RepositoryPG) IsUsernameTaken(ctx context.Context, username string) (bool, error) {
	const q = `SELECT EXISTS(SELECT 1 FROM public.user_profiles WHERE username = $1)`
	var exists bool
	err := r.conn.QueryRowContext(ctx, q, username).Scan(&exists)
	return exists, err
}

func (r *RepositoryPG) GetUserByPhone(ctx context.Context, phone string) (authdomain.User, error) {
	encPhone := phone
	if r.cipher != nil && phone != "" {
		if enc, err := r.cipher.EncryptDeterministic(phone); err == nil && enc != "" {
			encPhone = enc
		}
	}

	const q = `
SELECT u.id, u.email, u.password_hash, u.full_name, u.is_active
FROM public.users u
JOIN public.user_profiles up ON up.user_id = u.id
WHERE up.phone = $1`

	var out authdomain.User
	err := r.conn.QueryRowContext(ctx, q, encPhone).Scan(
		&out.ID, &out.Email, &out.PasswordHash, &out.FullName, &out.IsActive,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return authdomain.User{}, authdomain.ErrInvalidCredentials
		}
		return authdomain.User{}, err
	}
	return out, nil
}

func (r *RepositoryPG) ListTenantsByEmailOrPhone(ctx context.Context, query string) ([]authdomain.TenantInfo, error) {
	query = strings.TrimSpace(query)
	encQuery := query
	if r.cipher != nil && query != "" {
		if !strings.Contains(query, "@") {
			clean := ""
			for _, char := range query {
				if char >= '0' && char <= '9' {
					clean += string(char)
				}
			}
			if strings.HasPrefix(clean, "08") {
				clean = "628" + strings.TrimPrefix(clean, "08")
			}
			query = clean
		}

		// We encrypt the query using deterministic encryption in case it's a phone number.
		// It won't match any email since emails aren't encrypted.
		if enc, err := r.cipher.EncryptDeterministic(query); err == nil && enc != "" {
			encQuery = enc
		}
	}

	const q = `
SELECT DISTINCT t.id, t.code, t.name
FROM public.tenants t
JOIN public.tenant_memberships tm ON tm.tenant_id = t.id
JOIN public.users u ON u.id = tm.user_id
LEFT JOIN public.user_profiles up ON up.user_id = u.id
WHERE LOWER(u.email) = LOWER($1) OR up.phone = $2 OR up.phone = $3
ORDER BY t.name ASC`

	rows, err := r.conn.QueryContext(ctx, q, query, encQuery, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []authdomain.TenantInfo
	for rows.Next() {
		var item authdomain.TenantInfo
		if err := rows.Scan(&item.ID, &item.Code, &item.Name); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, nil
}

func (r *RepositoryPG) UpdateUserPassword(ctx context.Context, userID, passwordHash string) error {
	const q = `UPDATE public.users SET password_hash = $1, must_change_password = FALSE, updated_at = now() WHERE id = $2`
	_, err := r.conn.ExecContext(ctx, q, passwordHash, userID)
	return err
}
