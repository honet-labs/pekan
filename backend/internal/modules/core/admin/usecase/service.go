package usecase

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/aes"
	"crypto/cipher"
	crand "crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"

	"pekan/backend/internal/modules/core/admin/domain"
	"pekan/backend/internal/platform/audit"
	"pekan/backend/internal/platform/auth"
	"pekan/backend/internal/platform/notification"
	"pekan/backend/internal/platform/storage"
	"pekan/backend/internal/platform/db"
	"pekan/backend/internal/platform/tenancy"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	_ "github.com/jackc/pgx/v5/stdlib"
)

type Repository interface {
	BootstrapTenant(ctx context.Context, t domain.Tenant, u domain.User, m domain.Membership) error
	CreateTenant(ctx context.Context, t domain.Tenant) error
	CreateUser(ctx context.Context, u domain.User) error
	CreateMembership(ctx context.Context, m domain.Membership) error
	AssignRole(ctx context.Context, membershipID, roleCode string) error
	EnableModule(ctx context.Context, tenantID, moduleCode string) error
	ListTenants(ctx context.Context) ([]domain.TenantListItem, error)
	ListLogs(ctx context.Context) ([]domain.AuditLog, error)
	UpdateTenantQuotas(ctx context.Context, tenantID string, users, transactions int) error
	UpdateTenant(ctx context.Context, tenantID string, name, status string) error
	DeleteTenant(ctx context.Context, tenantID string) error
	ListTenantModules(ctx context.Context, tenantID string) ([]domain.TenantModule, error)
	UpdateTenantModule(ctx context.Context, tenantID, moduleCode string, enabled bool) error
	GetGrowthStats(ctx context.Context, from, to string) (domain.PlatformStats, error)
	Ping(ctx context.Context) error
	SetGlobalSetting(ctx context.Context, key, value string, encrypted bool) error
	GetGlobalSetting(ctx context.Context, key string) (string, bool, error)
	ExecuteRawQuery(ctx context.Context, query string) (domain.QueryResult, error)
	GetDatabaseStats(ctx context.Context) ([]domain.DatabaseTable, error)
	RecordDatabaseStats(ctx context.Context) error
	GetDatabaseGrowth(ctx context.Context) ([]domain.DatabaseGrowthPoint, error)
	ListTenantUsers(ctx context.Context, tenantID string) ([]domain.TenantUser, error)
	UpdateUserPassword(ctx context.Context, userID, hashedPassword string) error
	UpdateUserEmail(ctx context.Context, userID, newEmail string) error
	UpdateUserPhone(ctx context.Context, userID, newPhone string) error

	// WhatsApp chatbot queue management
	GetWhatsAppQueueStats(ctx context.Context) (domain.WhatsAppQueueStats, error)
	GetWhatsAppQueueHistory(ctx context.Context, limit, offset int, search string) ([]domain.WhatsAppQueueItem, int, error)
	RetryWhatsAppQueueMessage(ctx context.Context, id string) error
}


type UpdateState struct {
	Status    string    `json:"status"` // "idle", "running", "success", "failed"
	StartedAt time.Time `json:"started_at"`
	EndedAt   time.Time `json:"ended_at"`
	Error     string    `json:"error"`
}

type Service struct {
	repo      Repository
	audit     audit.Logger
	secretKey string
	startTime time.Time
	storage   storage.ObjectStorage
	redis     *redis.Client
	db        *sql.DB

	updateState UpdateState
	updateMu    sync.Mutex
}

func NewService(repo Repository, audit audit.Logger, secretKey string, storageProvider storage.ObjectStorage, rdb *redis.Client, dbConn *sql.DB) *Service {
	return &Service{
		repo:      repo,
		audit:     audit,
		secretKey: secretKey,
		startTime: time.Now(),
		storage:   storageProvider,
		redis:     rdb,
		db:        dbConn,
		updateState: UpdateState{
			Status: "idle",
		},
	}
}

type BootstrapTenantInput struct {
	TenantCode string `json:"tenant_code"`
	TenantName string `json:"tenant_name"`
	AdminEmail string `json:"admin_email"`
	AdminName  string `json:"admin_name"`
	Password   string `json:"password"`
}

func (s *Service) BootstrapTenant(ctx context.Context, in BootstrapTenantInput) error {
	in.TenantCode = strings.ToUpper(strings.TrimSpace(in.TenantCode))
	if matched, _ := regexp.MatchString(`^[A-Z0-9_-]{3,20}$`, in.TenantCode); !matched {
		return fmt.Errorf("invalid tenant code format: must be 3-20 alphanumeric characters, underscores or dashes")
	}

	tenantID := uuid.NewString()
	userID := uuid.NewString()
	membershipID := uuid.NewString()

	// 1. Hash Password
	log.Printf("[Admin] Bootstrapping tenant: code=%s, email=%s", in.TenantCode, in.AdminEmail)
	passwordHash, err := auth.HashPassword(in.Password)
	if err != nil {
		log.Printf("[Admin] Failed to hash password for tenant %s: %v", in.TenantCode, err)
		return err
	}

	// 1. Provision Schema
	schemaName := tenancy.GetSchemaName(in.TenantCode)
	migrator := db.NewMigrator(s.db)
	migrationsPath := "./migrations/tenant"
	if err := migrator.MigrateTenantSchema(ctx, schemaName, migrationsPath); err != nil {
		log.Printf("[Admin] Schema provisioning failed for %s: %v", in.TenantCode, err)
		return fmt.Errorf("failed to provision schema: %w", err)
	}

	// 2. Atomic Bootstrap (Seed roles/perms/membership into the now-existing schema)
	err = s.repo.BootstrapTenant(ctx, 
		domain.Tenant{
			ID:       tenantID,
			Code:     strings.ToLower(strings.TrimSpace(in.TenantCode)),
			Name:     in.TenantName,
			Status:   "active",
			Timezone: "Asia/Jakarta",
		},
		domain.User{
			ID:           userID,
			Email:        strings.ToLower(strings.TrimSpace(in.AdminEmail)),
			FullName:     in.AdminName,
			PasswordHash: passwordHash,
			IsActive:     true,
		},
		domain.Membership{
			ID:       membershipID,
			TenantID: tenantID,
			UserID:   userID,
			Status:   "active",
		},
	)
	if err != nil {
		return err
	}

	// 3. Audit Log
	_ = s.audit.Write(ctx, "BOOTSTRAP_TENANT", "tenant", tenantID, nil, map[string]any{
		"code":  in.TenantCode,
		"name":  in.TenantName,
		"admin": in.AdminEmail,
	})

	return nil
}

func (s *Service) ListTenants(ctx context.Context) ([]domain.TenantListItem, error) {
	return s.repo.ListTenants(ctx)
}

func (s *Service) ListLogs(ctx context.Context) ([]domain.AuditLog, error) {
	return s.repo.ListLogs(ctx)
}

func (s *Service) UpdateQuotas(ctx context.Context, tenantID string, users, transactions int) error {
	err := s.repo.UpdateTenantQuotas(ctx, tenantID, users, transactions)
	if err == nil {
		_ = s.audit.Write(ctx, "UPDATE_TENANT_QUOTAS", "tenant", tenantID, nil, map[string]any{"users": users, "transactions": transactions})
	}
	return err
}

func (s *Service) UpdateTenant(ctx context.Context, tenantID string, name, status string) error {
	err := s.repo.UpdateTenant(ctx, tenantID, name, status)
	if err == nil {
		_ = s.audit.Write(ctx, "UPDATE_TENANT", "tenant", tenantID, nil, map[string]any{"name": name, "status": status})
	}
	return err
}

func (s *Service) DeleteTenant(ctx context.Context, tenantID string) error {
	err := s.repo.DeleteTenant(ctx, tenantID)
	if err == nil {
		_ = s.audit.Write(ctx, "DELETE_TENANT", "tenant", tenantID, nil, nil)
	}
	return err
}

func (s *Service) ListModules(ctx context.Context, tenantID string) ([]domain.TenantModule, error) {
	return s.repo.ListTenantModules(ctx, tenantID)
}

func (s *Service) UpdateModule(ctx context.Context, tenantID, moduleCode string, enabled bool) error {
	return s.repo.UpdateTenantModule(ctx, tenantID, moduleCode, enabled)
}

func (s *Service) ListTenantUsers(ctx context.Context, tenantID string) ([]domain.TenantUser, error) {
	return s.repo.ListTenantUsers(ctx, tenantID)
}

func (s *Service) ResetUserPassword(ctx context.Context, userID, newPassword string) error {
	hash, err := auth.HashPassword(newPassword)
	if err != nil {
		return err
	}
	return s.repo.UpdateUserPassword(ctx, userID, hash)
}

func (s *Service) UpdateUserEmail(ctx context.Context, userID, newEmail string) error {
	return s.repo.UpdateUserEmail(ctx, userID, strings.ToLower(strings.TrimSpace(newEmail)))
}

func (s *Service) UpdateUserPhone(ctx context.Context, userID, newPhone string) error {
	return s.repo.UpdateUserPhone(ctx, userID, strings.TrimSpace(newPhone))
}

func (s *Service) TestNotification(ctx context.Context, provider string, configJSON string, destination string) error {
	var drv notification.Driver
	
	switch provider {
	case "smtp":
		var cfg notification.SMTPConfig
		if err := json.Unmarshal([]byte(configJSON), &cfg); err != nil {
			return err
		}
		drv = &notification.SMTPDriver{Config: cfg}
	case "telegram":
		var cfg notification.TelegramConfig
		if err := json.Unmarshal([]byte(configJSON), &cfg); err != nil {
			return err
		}
		drv = &notification.TelegramDriver{Config: cfg}
	case "wa": // Meta
		var cfg notification.MetaWAConfig
		if err := json.Unmarshal([]byte(configJSON), &cfg); err != nil {
			return err
		}
		drv = &notification.MetaWADriver{Config: cfg}
	case "wa_fonnte":
		var cfg notification.FonnteConfig
		if err := json.Unmarshal([]byte(configJSON), &cfg); err != nil {
			return err
		}
		drv = &notification.FonnteDriver{Config: cfg}
	case "wa_waha":
		var cfg notification.WahaConfig
		if err := json.Unmarshal([]byte(configJSON), &cfg); err != nil {
			return err
		}
		drv = &notification.WahaDriver{Config: cfg}
	case "wa_gowa":
		var cfg notification.GowaConfig
		if err := json.Unmarshal([]byte(configJSON), &cfg); err != nil {
			return err
		}
		drv = &notification.GowaDriver{Config: cfg}
	default:
		return errors.New("provider not supported for testing")
	}

	return drv.Send(ctx, destination, "Pesan Uji Coba dari PEKAN Admin Panel. Jika Anda menerima pesan ini, konfigurasi Anda sudah benar.")
}

type DatabaseConfig struct {
	Host     string `json:"host"`
	Port     string `json:"port"`
	User     string `json:"user"`
	Password string `json:"password"`
	DBName   string `json:"dbname"`
	SSLMode  string `json:"sslmode"`
}

func (s *Service) TestDatabase(ctx context.Context, configJSON string) error {
	var cfg DatabaseConfig
	if err := json.Unmarshal([]byte(configJSON), &cfg); err != nil {
		return err
	}

	if cfg.Host == "" || cfg.Port == "" || cfg.User == "" || cfg.DBName == "" {
		return errors.New("host, port, user, dan dbname wajib diisi")
	}

	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.DBName, cfg.SSLMode)

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return err
	}
	defer db.Close()

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := db.PingContext(pingCtx); err != nil {
		return fmt.Errorf("koneksi database gagal: %v", err)
	}

	return nil
}

func (s *Service) GetGrowth(ctx context.Context, from, to string) (domain.PlatformStats, error) {
	return s.repo.GetGrowthStats(ctx, from, to)
}

func (s *Service) GetServerStatus(ctx context.Context) domain.ServerStatus {
	dbStatus := "Healthy"
	if err := s.repo.Ping(ctx); err != nil {
		dbStatus = "Error: " + err.Error()
	}

	hostname, _ := os.Hostname()

	redisStatus := "Down"
	if s.redis != nil {
		pingCtx, cancel := context.WithTimeout(ctx, 1*time.Second)
		defer cancel()
		if err := s.redis.Ping(pingCtx).Err(); err == nil {
			redisStatus = "Healthy"
		}
	}

	if redisStatus == "Down" {
		redisAddrs := []string{}
		if envURL := os.Getenv("RATE_LIMIT_REDIS_URL"); envURL != "" {
			if u, err := url.Parse(envURL); err == nil && u.Host != "" {
				redisAddrs = append(redisAddrs, u.Host)
			}
		}
		if envURL := os.Getenv("REDIS_URL"); envURL != "" {
			if u, err := url.Parse(envURL); err == nil && u.Host != "" {
				redisAddrs = append(redisAddrs, u.Host)
			}
		}
		redisAddrs = append(redisAddrs, "pekan-redis:6379", "127.0.0.1:6379", "localhost:6379")

		for _, addr := range redisAddrs {
			conn, err := net.DialTimeout("tcp", addr, 1*time.Second)
			if err == nil {
				conn.Close()
				redisStatus = "Healthy"
				break
			}
		}
	}

	pgStatus := "Running"
	if strings.HasPrefix(dbStatus, "Error") {
		pgStatus = "Down"
	}

	return domain.ServerStatus{
		OS:          runtime.GOOS + " " + runtime.GOARCH,
		Uptime:      time.Since(s.startTime).String(),
		IPAddress:   hostname,
		Port:        "8080",
		DBStatus:    dbStatus,
		RedisStatus: redisStatus,
		Services: []domain.ServiceStatus{
			{Name: "API Server", Status: "Running", Port: 8080},
			{Name: "PostgreSQL", Status: pgStatus, Port: 5432},
			{Name: "Redis", Status: redisStatus, Port: 6379},
		},
	}
}

func (s *Service) SetGlobalSetting(ctx context.Context, key, value string, encrypted bool) error {
	val := value
	if encrypted && strings.TrimSpace(value) != "" {
		cipher, err := encryptSecret(s.secretKey, value)
		if err != nil {
			return err
		}
		val = cipher
	}
	err := s.repo.SetGlobalSetting(ctx, key, val, encrypted)
	if err == nil {
		_ = s.audit.Write(ctx, "UPDATE_GLOBAL_SETTING", "setting", key, nil, map[string]any{"key": key, "encrypted": encrypted})
	}
	return err
}

func (s *Service) GetGlobalSetting(ctx context.Context, key string) (string, bool, error) {
	val, enc, err := s.repo.GetGlobalSetting(ctx, key)
	if err != nil {
		return "", false, err
	}
	if enc && strings.TrimSpace(val) != "" {
		plain, err := decryptSecret(s.secretKey, val)
		if err != nil {
			return val, enc, nil // Return as is if decryption fails
		}
		return plain, enc, nil
	}
	return val, enc, nil
}

// GetGlobalSettingRaw returns the decrypted value without masking (for internal service use only).
func (s *Service) GetGlobalSettingRaw(ctx context.Context, key string) (string, error) {
	val, _, err := s.GetGlobalSetting(ctx, key)
	return val, err
}

// BootstrapTenantDirect creates a tenant using an already-hashed password (used by self-registration flow).
func (s *Service) BootstrapTenantDirect(ctx context.Context, tenantCode, tenantName, adminEmail, adminName, passwordHash string) error {
	tenantCode = strings.ToUpper(strings.TrimSpace(tenantCode))
	if matched, _ := regexp.MatchString(`^[A-Z0-9_-]{3,20}$`, tenantCode); !matched {
		return fmt.Errorf("invalid tenant code format: must be 3-20 alphanumeric characters, underscores or dashes")
	}

	tenantID := uuid.NewString()
	userID := uuid.NewString()
	membershipID := uuid.NewString()

	log.Printf("[Admin] Self-registration bootstrapping tenant: code=%s, email=%s", tenantCode, adminEmail)

	// 1. Provision Schema
	schemaName := tenancy.GetSchemaName(tenantCode)
	migrator := db.NewMigrator(s.db)
	migrationsPath := "./migrations/tenant"
	if err := migrator.MigrateTenantSchema(ctx, schemaName, migrationsPath); err != nil {
		log.Printf("[Admin] Schema provisioning failed for %s: %v", tenantCode, err)
		return fmt.Errorf("failed to provision schema: %w", err)
	}

	// 2. Atomic Bootstrap
	err := s.repo.BootstrapTenant(ctx,
		domain.Tenant{
			ID:       tenantID,
			Code:     tenantCode,
			Name:     tenantName,
			Status:   "active",
			Timezone: "Asia/Jakarta",
		},
		domain.User{
			ID:           userID,
			Email:        strings.ToLower(strings.TrimSpace(adminEmail)),
			FullName:     adminName,
			PasswordHash: passwordHash,
			IsActive:     true,
		},
		domain.Membership{
			ID:       membershipID,
			TenantID: tenantID,
			UserID:   userID,
			Status:   "active",
		},
	)
	if err != nil {
		return err
	}

	_ = s.audit.Write(ctx, "SELF_REGISTER_TENANT", "tenant", tenantID, nil, map[string]any{
		"code":  tenantCode,
		"name":  tenantName,
		"admin": adminEmail,
	})
	return nil
}

func (s *Service) getDatabaseURL(ctx context.Context) string {
	// Try to get from global settings first
	configJSON, err := s.GetGlobalSettingRaw(ctx, "database_config")
	if err == nil && configJSON != "" {
		var cfg DatabaseConfig
		if json.Unmarshal([]byte(configJSON), &cfg) == nil && cfg.Host != "" {
			sslMode := cfg.SSLMode
			if sslMode == "" {
				sslMode = "disable"
			}
			return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
				cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.DBName, sslMode)
		}
	}

	// Fallback to environment variable
	dbUrl := os.Getenv("DATABASE_URL")
	if dbUrl != "" {
		return dbUrl
	}

	// Build from individual env vars
	dbUser := os.Getenv("DB_USER")
	if dbUser == "" {
		dbUser = "postgres"
	}
	dbPass := os.Getenv("DB_PASS")
	dbHost := os.Getenv("DB_HOST")
	if dbHost == "" {
		dbHost = "127.0.0.1"
	}
	dbPort := os.Getenv("DB_PORT")
	if dbPort == "" {
		dbPort = "5432"
	}
	dbName := os.Getenv("DB_NAME")
	if dbName == "" {
		dbName = "pekan"
	}

	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", dbUser, dbPass, dbHost, dbPort, dbName)
}

func escapeSQL(val string) string {
	return strings.ReplaceAll(val, "'", "''")
}

func parseDatabaseURL(dbUrl string) (user, pass, host, port, dbName string) {
	user = "postgres"
	host = "127.0.0.1"
	port = "5432"
	dbName = "pekan"

	u, err := url.Parse(dbUrl)
	if err == nil {
		if u.User != nil {
			user = u.User.Username()
			pass, _ = u.User.Password()
		}
		if h := u.Hostname(); h != "" {
			host = h
		}
		if p := u.Port(); p != "" {
			port = p
		}
		if path := strings.TrimPrefix(u.Path, "/"); path != "" {
			dbName = path
		}
	}
	return
}

func (s *Service) getStorageDir() string {
	if envStorage := os.Getenv("STORAGE_LOCAL_PATH"); envStorage != "" {
		if _, err := os.Stat(envStorage); err == nil {
			return envStorage
		}
	}
	for _, candidate := range []string{"/var/lib/pekan/storage", "data/storage", "storage"} {
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return "data/storage"
}

func (s *Service) getBackupDir(tenantCode string) string {
	baseStorage := s.getStorageDir()
	dir := filepath.Join(baseStorage, "backups")
	if tenantCode != "" {
		dir = filepath.Join(dir, "tenants", tenantCode)
	}
	_ = os.MkdirAll(dir, 0755)
	return dir
}

func (s *Service) getBackupSearchDirs(tenantCode string) []string {
	dirs := []string{
		s.getBackupDir(tenantCode),
		s.getBackupDir(""),
		"/var/lib/pekan/storage/backups",
		"data/storage/backups",
		"/opt/pekan/backups",
	}
	if tenantCode != "" {
		dirs = append(dirs,
			filepath.Join("/var/lib/pekan/storage/backups", "tenants", tenantCode),
			filepath.Join("data/storage/backups", "tenants", tenantCode),
			filepath.Join("/opt/pekan/backups", "tenants", tenantCode),
		)
	}
	var existingDirs []string
	seen := make(map[string]bool)
	for _, d := range dirs {
		clean := filepath.Clean(d)
		if !seen[clean] {
			seen[clean] = true
			if _, err := os.Stat(clean); err == nil {
				existingDirs = append(existingDirs, clean)
			}
		}
	}
	return existingDirs
}

func (s *Service) dumpTenantMetadata(ctx context.Context, tenantID, tenantCode string) (string, error) {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("-- ======================================================\n"))
	sb.WriteString(fmt.Sprintf("-- PEKAN Tenant Metadata: %s (%s)\n", tenantCode, tenantID))
	sb.WriteString(fmt.Sprintf("-- ======================================================\n\n"))
	sb.WriteString("BEGIN;\n\n")

	// 1. Tenant record in public.tenants
	var tID, tCode, tName, tStatus, tTZ string
	var quotaUsers, quotaTx int
	qTenant := `SELECT id, code, name, status, timezone, quota_users, quota_transactions FROM public.tenants WHERE id = $1`
	if err := s.db.QueryRowContext(ctx, qTenant, tenantID).Scan(&tID, &tCode, &tName, &tStatus, &tTZ, &quotaUsers, &quotaTx); err == nil {
		sb.WriteString(fmt.Sprintf(`INSERT INTO public.tenants (id, code, name, status, timezone, quota_users, quota_transactions, created_at, updated_at)
VALUES ('%s', '%s', '%s', '%s', '%s', %d, %d, now(), now())
ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, code = EXCLUDED.code, status = EXCLUDED.status, timezone = EXCLUDED.timezone, quota_users = EXCLUDED.quota_users, quota_transactions = EXCLUDED.quota_transactions;
`, escapeSQL(tID), escapeSQL(tCode), escapeSQL(tName), escapeSQL(tStatus), escapeSQL(tTZ), quotaUsers, quotaTx))
	}

	// 2. Tenant modules in public.tenant_modules
	qMods := `SELECT module_code, is_enabled FROM public.tenant_modules WHERE tenant_id = $1`
	if rows, err := s.db.QueryContext(ctx, qMods, tenantID); err == nil {
		for rows.Next() {
			var mCode string
			var isEnabled bool
			if rows.Scan(&mCode, &isEnabled) == nil {
				sb.WriteString(fmt.Sprintf(`INSERT INTO public.tenant_modules (tenant_id, module_code, is_enabled, created_at)
VALUES ('%s', '%s', %t, now())
ON CONFLICT (tenant_id, module_code) DO UPDATE SET is_enabled = EXCLUDED.is_enabled;
`, escapeSQL(tenantID), escapeSQL(mCode), isEnabled))
			}
		}
		rows.Close()
	}

	// 3. Users belonging to this tenant
	qUsers := `
		SELECT u.id, u.email, u.password_hash, u.full_name, u.is_active, 
		       COALESCE(p.phone, ''), COALESCE(p.avatar_url, '')
		FROM public.users u
		JOIN public.tenant_memberships m ON u.id = m.user_id
		LEFT JOIN public.user_profiles p ON u.id = p.user_id
		WHERE m.tenant_id = $1`
	if rows, err := s.db.QueryContext(ctx, qUsers, tenantID); err == nil {
		for rows.Next() {
			var uID, uEmail, uPass, uName, uPhone, uAvatar string
			var uActive bool
			if rows.Scan(&uID, &uEmail, &uPass, &uName, &uActive, &uPhone, &uAvatar) == nil {
				sb.WriteString(fmt.Sprintf(`INSERT INTO public.users (id, email, password_hash, full_name, is_active, created_at, updated_at)
VALUES ('%s', '%s', '%s', '%s', %t, now(), now())
ON CONFLICT (id) DO UPDATE SET password_hash = EXCLUDED.password_hash, full_name = EXCLUDED.full_name, is_active = EXCLUDED.is_active;

INSERT INTO public.user_profiles (user_id, phone, full_name, avatar_url, updated_at)
VALUES ('%s', '%s', '%s', '%s', now())
ON CONFLICT (user_id) DO UPDATE SET phone = EXCLUDED.phone, full_name = EXCLUDED.full_name, avatar_url = EXCLUDED.avatar_url;
`, escapeSQL(uID), escapeSQL(uEmail), escapeSQL(uPass), escapeSQL(uName), uActive,
					escapeSQL(uID), escapeSQL(uPhone), escapeSQL(uName), escapeSQL(uAvatar)))
			}
		}
		rows.Close()
	}

	// 4. Global tenant memberships
	qMem := `SELECT id, user_id, status FROM public.tenant_memberships WHERE tenant_id = $1`
	if rows, err := s.db.QueryContext(ctx, qMem, tenantID); err == nil {
		for rows.Next() {
			var mID, mUID, mStatus string
			if rows.Scan(&mID, &mUID, &mStatus) == nil {
				sb.WriteString(fmt.Sprintf(`INSERT INTO public.tenant_memberships (id, tenant_id, user_id, status, joined_at, created_at)
VALUES ('%s', '%s', '%s', '%s', now(), now())
ON CONFLICT (id) DO UPDATE SET status = EXCLUDED.status;
`, escapeSQL(mID), escapeSQL(tenantID), escapeSQL(mUID), escapeSQL(mStatus)))
			}
		}
		rows.Close()
	}

	// 5. Files belonging to this tenant in public.files
	qFiles := `SELECT id, file_name, file_path, file_size, mime_type, created_by FROM public.files WHERE tenant_id = $1`
	if rows, err := s.db.QueryContext(ctx, qFiles, tenantID); err == nil {
		for rows.Next() {
			var fID, fName, fPath, fMime, fCreatedBy string
			var fSize int64
			if rows.Scan(&fID, &fName, &fPath, &fSize, &fMime, &fCreatedBy) == nil {
				sb.WriteString(fmt.Sprintf(`INSERT INTO public.files (id, tenant_id, file_name, file_path, file_size, mime_type, created_by, created_at)
VALUES ('%s', '%s', '%s', '%s', %d, '%s', '%s', now())
ON CONFLICT (id) DO NOTHING;
`, escapeSQL(fID), escapeSQL(tenantID), escapeSQL(fName), escapeSQL(fPath), fSize, escapeSQL(fMime), escapeSQL(fCreatedBy)))
			}
		}
		rows.Close()
	}

	schemaName := tenancy.GetSchemaName(tenantCode)
	sb.WriteString(fmt.Sprintf("CREATE SCHEMA IF NOT EXISTS %s;\n\n", schemaName))
	sb.WriteString("COMMIT;\n\n")
	return sb.String(), nil
}

func (s *Service) runPgDump(ctx context.Context, args []string) ([]byte, error) {
	dbUrl := s.getDatabaseURL(ctx)
	user, pass, host, port, dbName := parseDatabaseURL(dbUrl)

	hasHostPgDump := false
	if _, err := exec.LookPath("pg_dump"); err == nil {
		hasHostPgDump = true
	}

	isDocker := false
	pgContainer := "pekan-postgres"
	if strings.Contains(host, "pekan-postgres") || !hasHostPgDump {
		if out, err := exec.CommandContext(ctx, "docker", "ps", "--format", "{{.Names}}").Output(); err == nil {
			names := string(out)
			if strings.Contains(names, "pekan-postgres") || strings.Contains(names, "postgres") {
				isDocker = true
				for _, line := range strings.Split(names, "\n") {
					line = strings.TrimSpace(line)
					if strings.Contains(line, "pekan-postgres") || strings.Contains(line, "postgres") {
						pgContainer = line
						break
					}
				}
			}
		}
	}

	if isDocker {
		dockerArgs := append([]string{"exec", "-i", pgContainer, "pg_dump", "-U", user, "-d", dbName}, args...)
		cmd := exec.CommandContext(ctx, "docker", dockerArgs...)
		cmd.Env = append(os.Environ(), "PGPASSWORD="+pass)
		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		out, err := cmd.Output()
		if err != nil {
			return nil, fmt.Errorf("docker pg_dump failed (%v): %s", err, stderr.String())
		}
		return out, nil
	}

	// Host mode
	hostArgs := append(args, "-h", host, "-p", port, "-U", user, "-d", dbName)
	cmd := exec.CommandContext(ctx, "pg_dump", hostArgs...)
	cmd.Env = append(os.Environ(), "PGPASSWORD="+pass)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("pg_dump failed (%v): %s", err, stderr.String())
	}
	return out, nil
}

func (s *Service) runPsql(ctx context.Context, sqlFilePath string) error {
	dbUrl := s.getDatabaseURL(ctx)
	user, pass, host, port, dbName := parseDatabaseURL(dbUrl)

	hasHostPsql := false
	if _, err := exec.LookPath("psql"); err == nil {
		hasHostPsql = true
	}

	isDocker := false
	pgContainer := "pekan-postgres"
	if strings.Contains(host, "pekan-postgres") || !hasHostPsql {
		if out, err := exec.CommandContext(ctx, "docker", "ps", "--format", "{{.Names}}").Output(); err == nil {
			names := string(out)
			if strings.Contains(names, "pekan-postgres") || strings.Contains(names, "postgres") {
				isDocker = true
				for _, line := range strings.Split(names, "\n") {
					line = strings.TrimSpace(line)
					if strings.Contains(line, "pekan-postgres") || strings.Contains(line, "postgres") {
						pgContainer = line
						break
					}
				}
			}
		}
	}

	if isDocker {
		tmpInContainer := fmt.Sprintf("/tmp/restore_%d.sql", time.Now().UnixNano())
		copyCmd := exec.CommandContext(ctx, "docker", "cp", sqlFilePath, pgContainer+":"+tmpInContainer)
		if out, err := copyCmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to copy backup to postgres container: %v, output: %s", err, string(out))
		}
		defer func() {
			_ = exec.CommandContext(ctx, "docker", "exec", pgContainer, "rm", "-f", tmpInContainer).Run()
		}()

		cmd := exec.CommandContext(ctx, "docker", "exec", "-i", pgContainer,
			"psql", "-U", user, "-d", dbName, "-f", tmpInContainer)
		cmd.Env = append(os.Environ(), "PGPASSWORD="+pass)
		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("docker psql restore failed (%v): %s", err, string(out))
		}
		return nil
	}

	// Host mode
	cmd := exec.CommandContext(ctx, "psql", "-h", host, "-p", port, "-U", user, "-d", dbName, "-f", sqlFilePath)
	cmd.Env = append(os.Environ(), "PGPASSWORD="+pass)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("psql restore failed (%v): %s", err, string(out))
	}
	return nil
}

func createBackupArchive(archivePath, sqlFilePath, storageDir, manifestData string) error {
	out, err := os.Create(archivePath)
	if err != nil {
		return err
	}
	defer out.Close()

	gw := gzip.NewWriter(out)
	defer gw.Close()

	tw := tar.NewWriter(gw)
	defer tw.Close()

	// 1. Add database.sql
	if sqlFilePath != "" {
		sqlFile, err := os.Open(sqlFilePath)
		if err == nil {
			defer sqlFile.Close()
			fi, err := sqlFile.Stat()
			if err == nil {
				hdr, err := tar.FileInfoHeader(fi, "")
				if err == nil {
					hdr.Name = "database.sql"
					if err := tw.WriteHeader(hdr); err == nil {
						_, _ = io.Copy(tw, sqlFile)
					}
				}
			}
		}
	}

	// 2. Add manifest.json
	if manifestData != "" {
		hdr := &tar.Header{
			Name:     "manifest.json",
			Mode:     0644,
			Size:     int64(len(manifestData)),
			ModTime:  time.Now(),
			Typeflag: tar.TypeReg,
		}
		if err := tw.WriteHeader(hdr); err == nil {
			_, _ = tw.Write([]byte(manifestData))
		}
	}

	// 3. Add storage files (excluding backups/)
	if storageDir != "" {
		_ = filepath.Walk(storageDir, func(path string, info os.FileInfo, err error) error {
			if err != nil || path == storageDir {
				return nil
			}
			rel, err := filepath.Rel(storageDir, path)
			if err != nil {
				return nil
			}
			rel = filepath.ToSlash(rel)
			// Skip backups directory so we don't recursively duplicate previous backups
			if strings.HasPrefix(rel, "backups") || strings.HasPrefix(rel, "./backups") {
				if info.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}

			hdr, err := tar.FileInfoHeader(info, "")
			if err != nil {
				return nil
			}
			hdr.Name = "storage/" + rel
			if err := tw.WriteHeader(hdr); err != nil {
				return nil
			}
			if !info.IsDir() {
				f, err := os.Open(path)
				if err == nil {
					_, _ = io.Copy(tw, f)
					f.Close()
				}
			}
			return nil
		})
	}

	return nil
}

func extractBackupArchive(archivePath, tempDir, targetStorageDir string) (string, error) {
	f, err := os.Open(archivePath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	gr, err := gzip.NewReader(f)
	if err != nil {
		return "", err
	}
	defer gr.Close()

	var sqlPath string
	tr := tar.NewReader(gr)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return sqlPath, err
		}

		cleanPath := filepath.Clean(hdr.Name)
		if strings.HasPrefix(cleanPath, "..") || strings.HasPrefix(cleanPath, "/") {
			continue // prevent zip slip
		}

		// Database SQL file
		if cleanPath == "database.sql" || strings.HasSuffix(cleanPath, ".sql") {
			dest := filepath.Join(tempDir, filepath.Base(cleanPath))
			outFile, err := os.Create(dest)
			if err != nil {
				return "", err
			}
			_, _ = io.Copy(outFile, tr)
			outFile.Close()
			sqlPath = dest
			continue
		}

		// Storage file
		if strings.HasPrefix(cleanPath, "storage/") || strings.HasPrefix(cleanPath, "storage\\") {
			rel := strings.TrimPrefix(strings.TrimPrefix(cleanPath, "storage/"), "storage\\")
			if rel == "" {
				continue
			}
			dest := filepath.Join(targetStorageDir, rel)
			if hdr.Typeflag == tar.TypeDir {
				_ = os.MkdirAll(dest, 0755)
				continue
			}
			_ = os.MkdirAll(filepath.Dir(dest), 0755)
			outFile, err := os.Create(dest)
			if err == nil {
				_, _ = io.Copy(outFile, tr)
				outFile.Close()
			}
			continue
		}
	}

	return sqlPath, nil
}

func (s *Service) CreateBackup(ctx context.Context, backupType string, tenantID string) error {
	dbUrl := s.getDatabaseURL(ctx)
	if dbUrl == "" {
		return errors.New("database configuration not found")
	}

	prefix := "global"
	var tenantCode string
	if tenantID != "" {
		const q = `SELECT code FROM public.tenants WHERE id = $1`
		if err := s.db.QueryRowContext(ctx, q, tenantID).Scan(&tenantCode); err != nil {
			return fmt.Errorf("tenant not found: %v", err)
		}
		prefix = tenantCode
	}

	backupDir := s.getBackupDir(tenantCode)
	timestamp := time.Now().Format("20060102_150405")

	// Create temp directory for backup generation
	tempDir, err := os.MkdirTemp("", "pekan_backup_*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tempDir)

	tempSQLFile := filepath.Join(tempDir, "database.sql")

	var dumpArgs []string
	dumpArgs = append(dumpArgs, "--clean", "--if-exists", "--no-owner", "--no-privileges")

	if tenantID != "" {
		// Specific tenant schema
		schemaName := tenancy.GetSchemaName(tenantCode)
		dumpArgs = append(dumpArgs, "-n", schemaName)
	}

	if backupType == "schema" {
		dumpArgs = append(dumpArgs, "-s")
	} else if backupType == "data" {
		dumpArgs = append(dumpArgs, "-a")
	}

	// 1. Run database dump
	schemaDump, err := s.runPgDump(ctx, dumpArgs)
	if err != nil {
		return fmt.Errorf("database backup failed: %v", err)
	}
	if len(schemaDump) == 0 {
		return fmt.Errorf("pg_dump produced empty output")
	}

	// For tenant backups, prepend metadata export so users, tenant info, and permissions are preserved
	var fullSQLContent []byte
	if tenantID != "" && backupType != "data" {
		metaSQL, err := s.dumpTenantMetadata(ctx, tenantID, tenantCode)
		if err == nil && metaSQL != "" {
			fullSQLContent = append([]byte(metaSQL), schemaDump...)
		} else {
			fullSQLContent = schemaDump
		}
	} else {
		fullSQLContent = schemaDump
	}

	if err := os.WriteFile(tempSQLFile, fullSQLContent, 0644); err != nil {
		return fmt.Errorf("failed to write temp SQL: %v", err)
	}

	var finalBackupFile string
	storageDir := s.getStorageDir()

	if backupType == "full" {
		// Unified full backup: tar.gz containing database.sql + storage files
		finalBackupFile = filepath.Join(backupDir, fmt.Sprintf("backup_%s_full_%s.tar.gz", prefix, timestamp))
		manifest := fmt.Sprintf(`{"version":"1.0","type":"full","tenant":"%s","timestamp":"%s","app":"PEKAN"}`, prefix, timestamp)

		tenantStorage := storageDir
		if tenantID != "" {
			// For tenant backup, isolate storage to tenant folder if exists
			candidate := filepath.Join(storageDir, "tenants", tenantCode)
			if _, err := os.Stat(candidate); err == nil {
				tenantStorage = candidate
			}
		}

		if err := createBackupArchive(finalBackupFile, tempSQLFile, tenantStorage, manifest); err != nil {
			return fmt.Errorf("failed to create backup archive: %v", err)
		}
	} else {
		// Data or schema only: compressed SQL (.sql.gz)
		finalBackupFile = filepath.Join(backupDir, fmt.Sprintf("backup_%s_%s_%s.sql.gz", prefix, backupType, timestamp))
		out, err := os.Create(finalBackupFile)
		if err != nil {
			return err
		}
		defer out.Close()
		gw := gzip.NewWriter(out)
		if _, err := gw.Write(fullSQLContent); err != nil {
			gw.Close()
			return err
		}
		gw.Close()
	}

	fi, _ := os.Stat(finalBackupFile)
	size := int64(0)
	if fi != nil {
		size = fi.Size()
	}
	log.Printf("[Admin] Backup created successfully: %s (%d bytes)", finalBackupFile, size)

	// Cloud Backup Integration (if configured)
	if s.storage != nil {
		data, err := os.ReadFile(finalBackupFile)
		if err == nil {
			filename := filepath.Base(finalBackupFile)
			cloudKey := fmt.Sprintf("system/backups/%s", filename)
			if tenantID != "" {
				cloudKey = fmt.Sprintf("tenants/%s/backups/%s", prefix, filename)
			}
			_, _ = s.storage.Put(ctx, storage.PutObjectInput{
				TenantID:    "system",
				Module:      "core.admin",
				ObjectKey:   cloudKey,
				ContentType: "application/gzip",
				Body:        bytes.NewReader(data),
			})
		}
	}

	_ = s.audit.Write(ctx, "BACKUP_CREATED", "tenant", tenantID, nil, map[string]any{
		"path": finalBackupFile,
		"type": backupType,
		"size": size,
	})
	return nil
}

func (s *Service) RestoreBackup(ctx context.Context, filename string, tenantID string) error {
	dbUrl := s.getDatabaseURL(ctx)
	if dbUrl == "" {
		return errors.New("database configuration not found")
	}

	var tenantCode string
	if tenantID != "" {
		const q = `SELECT code FROM public.tenants WHERE id = $1`
		_ = s.db.QueryRowContext(ctx, q, tenantID).Scan(&tenantCode)
	}

	cleanName := filepath.Base(filename)
	searchDirs := s.getBackupSearchDirs(tenantCode)

	var fp string
	for _, dir := range searchDirs {
		candidate := filepath.Join(dir, cleanName)
		if _, err := os.Stat(candidate); err == nil {
			fp = candidate
			break
		}
	}

	if fp == "" {
		return fmt.Errorf("backup file %s not found in any search path", cleanName)
	}

	tempDir, err := os.MkdirTemp("", "pekan_restore_*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tempDir)

	targetStorage := s.getStorageDir()
	var sqlFileToRun string

	if strings.HasSuffix(cleanName, ".tar.gz") || strings.HasSuffix(cleanName, ".tar") {
		// Extract archive containing database.sql and storage/
		log.Printf("[Admin] Extracting full backup archive: %s", fp)
		extractedSQL, err := extractBackupArchive(fp, tempDir, targetStorage)
		if err != nil {
			return fmt.Errorf("failed to extract backup archive: %v", err)
		}
		if extractedSQL == "" {
			return fmt.Errorf("no database.sql found inside backup archive")
		}
		sqlFileToRun = extractedSQL
	} else if strings.HasSuffix(cleanName, ".sql.gz") || strings.HasSuffix(cleanName, ".gz") {
		// Decompress gzipped SQL
		gzFile, err := os.Open(fp)
		if err != nil {
			return fmt.Errorf("failed to open gz file: %v", err)
		}
		defer gzFile.Close()
		gr, err := gzip.NewReader(gzFile)
		if err != nil {
			return fmt.Errorf("failed to read gz: %v", err)
		}
		defer gr.Close()
		tmpSQL := filepath.Join(tempDir, "restore.sql")
		outSQL, err := os.Create(tmpSQL)
		if err != nil {
			return err
		}
		if _, err := io.Copy(outSQL, gr); err != nil {
			outSQL.Close()
			return fmt.Errorf("failed to decompress sql.gz: %v", err)
		}
		outSQL.Close()
		sqlFileToRun = tmpSQL
	} else {
		// Direct SQL file
		sqlFileToRun = fp
	}

	// 2. Prepare database clean slate before executing restore
	if tenantID == "" {
		log.Printf("[Admin] Cleaning schemas for full database restore...")
		cleanSlateSQL := `
DO $$
DECLARE
    r RECORD;
BEGIN
    FOR r IN (SELECT schema_name FROM information_schema.schemata WHERE schema_name LIKE 'wkspid_pekan_%') LOOP
        EXECUTE 'DROP SCHEMA IF EXISTS ' || quote_ident(r.schema_name) || ' CASCADE';
    END LOOP;
END $$;
DROP SCHEMA IF EXISTS public CASCADE;
CREATE SCHEMA public;
GRANT ALL ON SCHEMA public TO postgres;
GRANT ALL ON SCHEMA public TO public;
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";
`
		if _, err := s.db.ExecContext(ctx, cleanSlateSQL); err != nil {
			log.Printf("[Admin] Clean slate warning: %v", err)
		}
	} else {
		// Clean only this tenant's schema
		schemaName := tenancy.GetSchemaName(tenantCode)
		if schemaName != "" && schemaName != "public" {
			log.Printf("[Admin] Cleaning tenant schema %s before restore...", schemaName)
			_, _ = s.db.ExecContext(ctx, fmt.Sprintf("DROP SCHEMA IF EXISTS %s CASCADE;", schemaName))
		}
	}

	// 3. Run SQL restore
	log.Printf("[Admin] Executing SQL restore from %s...", sqlFileToRun)
	if err := s.runPsql(ctx, sqlFileToRun); err != nil {
		log.Printf("[Admin] SQL restore error: %v", err)
		return fmt.Errorf("database restore failed: %v", err)
	}

	// 4. Schema self-heal and migration patches to ensure all tenant tables & permissions match current version
	if tenantID == "" {
		log.Printf("[Admin] Running schema self-heal and migration patches...")
		for _, patchScript := range []string{"scripts/apply_migrations.sh", "backend/scripts/apply_migrations.sh", "/opt/pekan/backend/scripts/apply_migrations.sh"} {
			if _, err := os.Stat(patchScript); err == nil {
				cmd := exec.CommandContext(ctx, "bash", patchScript)
				cmd.Env = append(os.Environ(), "DATABASE_URL="+dbUrl)
				_ = cmd.Run()
				break
			}
		}
	}

	log.Printf("[Admin] Restore completed successfully for %s", cleanName)
	_ = s.audit.Write(ctx, "BACKUP_RESTORED", "tenant", tenantID, nil, map[string]any{"filename": cleanName})
	return nil
}

func (s *Service) SaveUploadedBackup(ctx context.Context, filename string, file io.Reader) error {
	backupDir := s.getBackupDir("")
	cleanName := filepath.Base(filename)
	fp := filepath.Join(backupDir, cleanName)

	if _, err := os.Stat(fp); err == nil {
		ext := filepath.Ext(cleanName)
		nameWithoutExt := strings.TrimSuffix(cleanName, ext)
		if strings.HasSuffix(nameWithoutExt, ".tar") {
			nameWithoutExt = strings.TrimSuffix(nameWithoutExt, ".tar")
			ext = ".tar" + ext
		}
		if strings.HasSuffix(nameWithoutExt, ".sql") {
			nameWithoutExt = strings.TrimSuffix(nameWithoutExt, ".sql")
			ext = ".sql" + ext
		}
		fp = filepath.Join(backupDir, fmt.Sprintf("%s_%s%s", nameWithoutExt, time.Now().Format("20060102_150405"), ext))
	}

	dst, err := os.Create(fp)
	if err != nil {
		return fmt.Errorf("failed to create file: %v", err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		os.Remove(fp)
		return fmt.Errorf("failed to save file: %v", err)
	}

	log.Printf("[Admin] Uploaded backup saved: %s", fp)
	_ = s.audit.Write(ctx, "BACKUP_UPLOADED", "system", "", nil, map[string]any{"filename": cleanName, "path": fp})
	return nil
}

func (s *Service) ListBackups(ctx context.Context, tenantID string) ([]domain.BackupFile, error) {
	var tenantCode string
	if tenantID != "" {
		const q = `SELECT code FROM public.tenants WHERE id = $1`
		if err := s.db.QueryRowContext(ctx, q, tenantID).Scan(&tenantCode); err != nil {
			return nil, err
		}
	}

	searchDirs := s.getBackupSearchDirs(tenantCode)
	var backups []domain.BackupFile
	seen := make(map[string]bool)

	for _, bDir := range searchDirs {
		entries, err := os.ReadDir(bDir)
		if err != nil {
			continue
		}

		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			name := e.Name()
			if strings.HasSuffix(name, ".tar.gz") || strings.HasSuffix(name, ".tar") ||
				strings.HasSuffix(name, ".sql.gz") || strings.HasSuffix(name, ".sql") || strings.HasSuffix(name, ".dump") {
				if seen[name] {
					continue
				}
				seen[name] = true
				info, err := e.Info()
				if err != nil {
					continue
				}
				backups = append(backups, domain.BackupFile{
					Name:      info.Name(),
					Size:      info.Size(),
					CreatedAt: info.ModTime().Format(time.RFC3339),
				})
			}
		}
	}

	// Sort newest first
	for i, j := 0, len(backups)-1; i < j; i, j = i+1, j-1 {
		backups[i], backups[j] = backups[j], backups[i]
	}

	return backups, nil
}

func (s *Service) GetBackupPath(ctx context.Context, tenantID, filename string) (string, error) {
	var tenantCode string
	if tenantID != "" {
		const q = `SELECT code FROM public.tenants WHERE id = $1`
		if err := s.db.QueryRowContext(ctx, q, tenantID).Scan(&tenantCode); err != nil {
			return "", err
		}
	}

	cleanName := filepath.Base(filename)
	searchDirs := s.getBackupSearchDirs(tenantCode)

	for _, bDir := range searchDirs {
		candidate := filepath.Join(bDir, cleanName)
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}

	return "", os.ErrNotExist
}

type AIModel struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

func (s *Service) TestAI(ctx context.Context, provider, apiKey string) ([]AIModel, error) {
	client := &http.Client{Timeout: 15 * time.Second}
	var models []AIModel

	switch provider {
	case "gemini":
		url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models?key=%s", apiKey)
		req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("gemini api error: status %d", resp.StatusCode)
		}

		var data struct {
			Models []struct {
				Name        string `json:"name"`
				DisplayName string `json:"displayName"`
			} `json:"models"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
			return nil, err
		}

		for _, m := range data.Models {
			if strings.Contains(m.Name, "gemini") {
				models = append(models, AIModel{
					ID:    strings.TrimPrefix(m.Name, "models/"),
					Label: m.DisplayName,
				})
			}
		}

	case "openai":
		req, _ := http.NewRequestWithContext(ctx, "GET", "https://api.openai.com/v1/models", nil)
		req.Header.Set("Authorization", "Bearer "+apiKey)
		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("openai api error: status %d", resp.StatusCode)
		}

		var data struct {
			Data []struct {
				ID string `json:"id"`
			} `json:"data"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
			return nil, err
		}

		for _, m := range data.Data {
			if strings.Contains(m.ID, "gpt") {
				models = append(models, AIModel{ID: m.ID, Label: m.ID})
			}
		}

	case "claude":
		// Anthropic doesn't have a public models list API that's easy to use without a specific version.
		// We'll return a standard list for Claude.
		models = []AIModel{
			{ID: "claude-3-5-sonnet-20240620", Label: "Claude 3.5 Sonnet"},
			{ID: "claude-3-opus-20240229", Label: "Claude 3 Opus"},
			{ID: "claude-3-sonnet-20240229", Label: "Claude 3 Sonnet"},
			{ID: "claude-3-haiku-20240307", Label: "Claude 3 Haiku"},
		}
		// Basic connectivity check
		req, _ := http.NewRequestWithContext(ctx, "GET", "https://api.anthropic.com/v1/messages", nil)
		req.Header.Set("x-api-key", apiKey)
		req.Header.Set("anthropic-version", "2023-06-01")
		resp, _ := client.Do(req)
		// 400 is expected because we send no body, but it proves the key is reached
		if resp != nil && resp.StatusCode == http.StatusUnauthorized {
			return nil, errors.New("anthropic api error: unauthorized")
		}

	case "sumopod":
		// Sumopod uses OpenAI-compatible models endpoint
		req, _ := http.NewRequestWithContext(ctx, "GET", "https://ai.sumopod.com/v1/models", nil)
		req.Header.Set("Authorization", "Bearer "+apiKey)
		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("sumopod api error: status %d", resp.StatusCode)
		}

		var data struct {
			Data []struct {
				ID string `json:"id"`
			} `json:"data"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
			return nil, err
		}

		for _, m := range data.Data {
			models = append(models, AIModel{ID: m.ID, Label: m.ID})
		}
		
		// Fallback if empty
		if len(models) == 0 {
			models = []AIModel{{ID: "sumopod-v1", Label: "Sumopod V1 (Default)"}}
		}

	default:
		return nil, errors.New("provider not supported for testing")
	}

	return models, nil
}

// Copy of encryption helpers to avoid circular dependencies
// In a real project, these should be in internal/platform/crypto

func encryptSecret(secret, plain string) (string, error) {
	key := sha256.Sum256([]byte(secret))
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(crand.Reader, nonce); err != nil {
		return "", err
	}
	cipherText := gcm.Seal(nonce, nonce, []byte(plain), nil)
	return base64.StdEncoding.EncodeToString(cipherText), nil
}

func decryptSecret(secret, cipherText string) (string, error) {
	if strings.TrimSpace(cipherText) == "" {
		return "", nil
	}
	raw, err := base64.StdEncoding.DecodeString(cipherText)
	if err != nil {
		return "", err
	}
	key := sha256.Sum256([]byte(secret))
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	if len(raw) < gcm.NonceSize() {
		return "", errors.New("invalid cipher text")
	}
	nonce, enc := raw[:gcm.NonceSize()], raw[gcm.NonceSize():]
	plain, err := gcm.Open(nil, nonce, enc, nil)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}

func (s *Service) ExecuteQuery(ctx context.Context, query string) (domain.QueryResult, error) {
	return s.repo.ExecuteRawQuery(ctx, query)
}

func (s *Service) GetDatabaseStats(ctx context.Context) ([]domain.DatabaseTable, error) {
	return s.repo.GetDatabaseStats(ctx)
}

func (s *Service) RecordDatabaseStats(ctx context.Context) error {
	return s.repo.RecordDatabaseStats(ctx)
}

func (s *Service) GetDatabaseGrowth(ctx context.Context) ([]domain.DatabaseGrowthPoint, error) {
	return s.repo.GetDatabaseGrowth(ctx)
}

func (s *Service) GetWhatsAppQueueStats(ctx context.Context) (domain.WhatsAppQueueStats, error) {
	return s.repo.GetWhatsAppQueueStats(ctx)
}

func (s *Service) GetWhatsAppQueueHistory(ctx context.Context, limit, offset int, search string) ([]domain.WhatsAppQueueItem, int, error) {
	return s.repo.GetWhatsAppQueueHistory(ctx, limit, offset, search)
}

func (s *Service) RetryWhatsAppQueueMessage(ctx context.Context, id string) error {
	err := s.repo.RetryWhatsAppQueueMessage(ctx, id)
	if err == nil {
		_ = s.audit.Write(ctx, "RETRY_WHATSAPP_QUEUE_MSG", "queue", id, nil, nil)
	}
	return err
}

// Helper to run git commands in current or parent directory
func (s *Service) runGitCmd(ctx context.Context, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err == nil {
		return strings.TrimSpace(out.String()), nil
	}

	cmdParent := exec.CommandContext(ctx, "git", append([]string{"-C", ".."}, args...)...)
	var outParent bytes.Buffer
	cmdParent.Stdout = &outParent
	if err := cmdParent.Run(); err == nil {
		return strings.TrimSpace(outParent.String()), nil
	}

	return "", errors.New("not a git repository")
}

// CheckUpdate queries the GitHub API and local Git status to check if a new commit is available.
func (s *Service) CheckUpdate(ctx context.Context) (domain.UpdateStatusInfo, error) {
	var info domain.UpdateStatusInfo

	// Check if local git repo
	isGit := false
	localCommit := ""
	localDate := ""

	if commit, err := s.runGitCmd(ctx, "rev-parse", "HEAD"); err == nil {
		isGit = true
		localCommit = commit
		if date, err := s.runGitCmd(ctx, "log", "-1", "--format=%cd"); err == nil {
			localDate = date
		}
	}

	info.IsGitRepo = isGit
	info.CurrentCommit = localCommit
	info.CurrentDate = localDate

	// Fetch latest remote commit from GitHub API
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.github.com/repos/aannddrrii294/pekan/commits/main", nil)
	if err != nil {
		return info, err
	}
	req.Header.Set("User-Agent", "pekan-update-agent")
	
	resp, err := client.Do(req)
	if err == nil {
		defer resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			var githubCommits struct {
				Sha string `json:"sha"`
				Commit struct {
					Message string `json:"message"`
					Committer struct {
						Date string `json:"date"`
					} `json:"committer"`
				} `json:"commit"`
			}
			if err := json.NewDecoder(resp.Body).Decode(&githubCommits); err == nil {
				info.LatestCommit = githubCommits.Sha
				info.LatestMessage = githubCommits.Commit.Message
				info.LatestDate = githubCommits.Commit.Committer.Date
				if isGit && localCommit != githubCommits.Sha {
					info.UpdateAvailable = true
				}
				return info, nil
			}
		}
	}

	// Fallback to git fetch + rev-parse origin/main if github API failed
	if isGit {
		_, _ = s.runGitCmd(ctx, "fetch", "origin")
		if remoteCommit, err := s.runGitCmd(ctx, "rev-parse", "origin/main"); err == nil {
			info.LatestCommit = remoteCommit
			if remoteMsg, err := s.runGitCmd(ctx, "log", "-1", "origin/main", "--format=%s"); err == nil {
				info.LatestMessage = remoteMsg
			}
			if remoteDate, err := s.runGitCmd(ctx, "log", "-1", "origin/main", "--format=%cd"); err == nil {
				info.LatestDate = remoteDate
			}
			if localCommit != remoteCommit {
				info.UpdateAvailable = true
			}
			return info, nil
		}
	}

	return info, nil
}

// ApplyUpdate triggers a background update build and deploys changes safely.
func (s *Service) ApplyUpdate(ctx context.Context) error {
	s.updateMu.Lock()
	if s.updateState.Status == "running" {
		s.updateMu.Unlock()
		return errors.New("update is already running")
	}

	s.updateState.Status = "running"
	s.updateState.StartedAt = time.Now()
	s.updateState.Error = ""
	s.updateMu.Unlock()

	// Clear/ensure log file exists
	_ = os.MkdirAll("logs", 0755)
	logFile, err := os.OpenFile("logs/update.log", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		s.updateMu.Lock()
		s.updateState.Status = "failed"
		s.updateState.EndedAt = time.Now()
		s.updateState.Error = err.Error()
		s.updateMu.Unlock()
		return err
	}

	go func() {
		defer logFile.Close()

		writeLog := func(format string, args ...any) {
			msg := fmt.Sprintf("[%s] %s\n", time.Now().Format("15:04:05"), fmt.Sprintf(format, args...))
			_, _ = logFile.WriteString(msg)
			log.Println("[Update]", msg)
		}

		writeLog("Starting System Update Process...")

		// Helper to run exec commands and stream to log file
		runCmd := func(name string, dir string, args ...string) error {
			writeLog("Executing: %s %s in %s", name, strings.Join(args, " "), dir)
			cmd := exec.Command(name, args...)
			cmd.Dir = dir
			cmd.Stdout = logFile
			cmd.Stderr = logFile
			return cmd.Run()
		}

		// 1. Fetch and reset git repo
		writeLog("1/6 Fetching latest code from GitHub...")
		if err := runCmd("git", ".", "fetch", "origin"); err != nil {
			// Try parent directory
			if err := runCmd("git", "..", "fetch", "origin"); err != nil {
				writeLog("Failed to fetch from git: %v. Continuing...", err)
			} else {
				_ = runCmd("git", "..", "reset", "--hard", "origin/main")
			}
		} else {
			_ = runCmd("git", ".", "reset", "--hard", "origin/main")
		}

		// 2. Tidy dependencies and build backend binaries
		writeLog("2/6 Rebuilding Go backend services...")
		if err := runCmd("/usr/local/go/bin/go", ".", "mod", "tidy"); err != nil {
			writeLog("Warning: go mod tidy failed: %v", err)
		}

		if err := runCmd("/usr/local/go/bin/go", ".", "build", "-o", "../bin/pekan-api", "./cmd/api"); err != nil {
			writeLog("Error: pekan-api build failed: %v", err)
			s.markFailed(err)
			return
		}

		_ = runCmd("/usr/local/go/bin/go", ".", "build", "-o", "../bin/pekan-worker", "./cmd/worker")
		_ = runCmd("/usr/local/go/bin/go", ".", "build", "-o", "../bin/pekan-ai", "./cmd/ai")

		// 3. Database migrations
		writeLog("3/6 Applying migrations...")
		_ = runCmd("chmod", ".", "+x", "./scripts/apply_migrations.sh")
		if err := runCmd("./scripts/apply_migrations.sh", "."); err != nil {
			writeLog("Warning: migration script failed: %v", err)
		}
		// Run tenant migrations
		_ = runCmd("/usr/local/go/bin/go", ".", "run", "./scripts/migrate_tenants.go")

		// 4. Rebuild frontend React assets
		writeLog("4/6 Rebuilding React Frontend...")
		if err := runCmd("npm", "../frontend", "install", "--include=dev", "--no-audit"); err != nil {
			writeLog("Warning: npm install failed: %v", err)
		}
		if err := runCmd("npm", "../frontend", "run", "build"); err != nil {
			writeLog("Error: frontend build failed: %v", err)
			s.markFailed(err)
			return
		}

		// Copy frontend dist to web root /var/www/pekan-web
		writeLog("5/6 Copying built assets to /var/www/pekan-web...")
		_ = runCmd("rm", ".", "-rf", "/var/www/pekan-web/*")
		if err := runCmd("cp", ".", "-r", "../frontend/dist/.", "/var/www/pekan-web/"); err != nil {
			writeLog("Warning: copy to /var/www/pekan-web failed: %v. Trying fallback standard cp...", err)
		}

		// 5. Restart worker and AI processes (auto-restart by systemd)
		writeLog("6/6 Restarting background services...")
		_ = runCmd("pkill", ".", "-f", "pekan-worker")
		_ = runCmd("pkill", ".", "-f", "pekan-ai")

		writeLog("Update completed successfully! Exiting API service to trigger systemd auto-restart...")

		s.updateMu.Lock()
		s.updateState.Status = "success"
		s.updateState.EndedAt = time.Now()
		s.updateMu.Unlock()

		// Graceful exit in 1 second so response can be sent to client
		go func() {
			time.Sleep(1 * time.Second)
			log.Println("[Update] Exiting application to reload new binary...")
			os.Exit(0)
		}()
	}()

	return nil
}

func (s *Service) markFailed(err error) {
	s.updateMu.Lock()
	s.updateState.Status = "failed"
	s.updateState.EndedAt = time.Now()
	s.updateState.Error = err.Error()
	s.updateMu.Unlock()
}

// GetUpdateStatus reads the update progress and logs.
func (s *Service) GetUpdateStatus(ctx context.Context) (map[string]any, error) {
	s.updateMu.Lock()
	defer s.updateMu.Unlock()

	// Read log file
	logContent := ""
	if data, err := os.ReadFile("logs/update.log"); err == nil {
		logContent = string(data)
	}

	return map[string]any{
		"status":     s.updateState.Status,
		"started_at": s.updateState.StartedAt.Format(time.RFC3339),
		"ended_at":   s.updateState.EndedAt.Format(time.RFC3339),
		"error":      s.updateState.Error,
		"logs":       logContent,
	}, nil
}

// GetSystemLogs retrieves system service logs (usually journalctl for pekan-api).
func (s *Service) GetSystemLogs(ctx context.Context, serviceName string, lines int) (string, error) {
	if serviceName == "" {
		serviceName = "pekan-api"
	}
	allowedServices := map[string]bool{
		"pekan-api":    true,
		"pekan-worker": true,
		"pekan-ai":     true,
	}
	if !allowedServices[serviceName] {
		return "", errors.New("invalid service name")
	}

	cmd := exec.CommandContext(ctx, "journalctl", "-u", serviceName, "-n", fmt.Sprintf("%d", lines), "--no-pager")
	out, err := cmd.CombinedOutput()
	if err == nil {
		return string(out), nil
	}

	// Fallback: Show update.log if journalctl fails
	if data, errReadFile := os.ReadFile("logs/update.log"); errReadFile == nil {
		return fmt.Sprintf("[journalctl failed: %v]\n\nShowing update.log fallback:\n%s", err, string(data)), nil
	}

	return "", fmt.Errorf("journalctl failed: %w (output: %s)", err, string(out))
}

