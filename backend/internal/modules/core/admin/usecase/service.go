package usecase

import (
	"bytes"
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
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
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


type Service struct {
	repo      Repository
	audit     audit.Logger
	secretKey string
	startTime time.Time
	storage   storage.ObjectStorage
	redis     *redis.Client
	db        *sql.DB
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

	return drv.Send(ctx, destination, "Pesan Uji Coba dari Pekan SaaS Admin Panel. Jika Anda menerima pesan ini, konfigurasi Anda sudah benar.")
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

	// Helper to check if a port is open
	checkTCP := func(addr string) string {
		conn, err := net.DialTimeout("tcp", addr, 1*time.Second)
		if err != nil {
			return "Down"
		}
		conn.Close()
		return "Running"
	}

	pgStatus := "Running"
	if strings.HasPrefix(dbStatus, "Error") {
		pgStatus = "Down"
	}

	return domain.ServerStatus{
		OS:        runtime.GOOS + " " + runtime.GOARCH,
		Uptime:    time.Since(s.startTime).String(),
		IPAddress: hostname,
		Port:      "8080",
		DBStatus:  dbStatus,
		RedisStatus: checkTCP("localhost:6379"),
		Services: []domain.ServiceStatus{
			{Name: "API Server", Status: "Running", Port: 8080},
			{Name: "PostgreSQL", Status: pgStatus, Port: 5432},
			{Name: "Redis", Status: checkTCP("localhost:6379"), Port: 6379},
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

func (s *Service) CreateBackup(ctx context.Context, backupType string, tenantID string) error {
	dbUrl := os.Getenv("DATABASE_URL")
	if dbUrl == "" {
		return errors.New("DATABASE_URL not set")
	}

	backupDir := "data/storage/backups"
	prefix := "global"
	var schemaName string

	if tenantID != "" {
		// Get tenant code to resolve schema and storage path
		const q = `SELECT code FROM public.tenants WHERE id = $1`
		if err := s.db.QueryRowContext(ctx, q, tenantID).Scan(&prefix); err != nil {
			return err
		}
		schemaName = tenancy.GetSchemaName(prefix)
		backupDir = filepath.Join(backupDir, "tenants", prefix)
	}

	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return err
	}

	filename := fmt.Sprintf("backup_%s_%s_%s.dump", prefix, backupType, time.Now().Format("20060102_150405"))
	fp := filepath.Join(backupDir, filename)

	args := []string{"-d", dbUrl, "-F", "c", "-f", fp}
	if schemaName != "" {
		args = append(args, "-n", schemaName)
	}
	if backupType == "schema" {
		args = append(args, "-s")
	} else if backupType == "data" {
		args = append(args, "-a")
	}

	cmd := exec.CommandContext(ctx, "pg_dump", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("pg_dump failed: %v, output: %s", err, string(output))
	}

	log.Printf("[Admin] Backup created at %s", fp)

	// Cloud Backup Integration
	if s.storage != nil {
		data, err := os.ReadFile(fp)
		if err == nil {
			cloudKey := fmt.Sprintf("system/backups/%s", filename)
			if tenantID != "" {
				cloudKey = fmt.Sprintf("tenants/%s/backups/%s", prefix, filename)
			}
			_, err = s.storage.Put(ctx, storage.PutObjectInput{
				TenantID:    "system",
				Module:      "core.admin",
				ObjectKey:   cloudKey,
				ContentType: "application/octet-stream",
				Body:        bytes.NewReader(data),
			})
		}
	}

	_ = s.audit.Write(ctx, "BACKUP_CREATED", "tenant", tenantID, nil, map[string]any{"path": fp, "type": backupType})
	return nil
}

func (s *Service) RestoreBackup(ctx context.Context, filename string, tenantID string) error {
	dbUrl := os.Getenv("DATABASE_URL")
	if dbUrl == "" {
		return errors.New("DATABASE_URL not set")
	}

	backupDir := "data/storage/backups"
	cleanName := filepath.Base(filename)
	fp := filepath.Join(backupDir, cleanName)

	if tenantID != "" {
		var tenantCode string
		const q = `SELECT code FROM public.tenants WHERE id = $1`
		if err := s.db.QueryRowContext(ctx, q, tenantID).Scan(&tenantCode); err != nil {
			return err
		}
		fp = filepath.Join(backupDir, "tenants", tenantCode, cleanName)
	}

	if _, err := os.Stat(fp); os.IsNotExist(err) {
		return errors.New("backup file not found")
	}

	args := []string{"-d", dbUrl, "-1", "-c", fp}
	cmd := exec.CommandContext(ctx, "pg_restore", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("pg_restore failed: %v, output: %s", err, string(output))
	}
	
	_ = s.audit.Write(ctx, "BACKUP_RESTORED", "tenant", tenantID, nil, map[string]any{"filename": cleanName})
	return nil
}

func (s *Service) ListBackups(ctx context.Context, tenantID string) ([]domain.BackupFile, error) {
	backupDir := "data/storage/backups"
	if tenantID != "" {
		var tenantCode string
		const q = `SELECT code FROM public.tenants WHERE id = $1`
		if err := s.db.QueryRowContext(ctx, q, tenantID).Scan(&tenantCode); err != nil {
			return nil, err
		}
		backupDir = filepath.Join(backupDir, "tenants", tenantCode)
	}

	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(backupDir)
	if err != nil {
		return nil, err
	}

	var backups []domain.BackupFile
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".dump") {
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
	
	// reverse sort roughly assuming alphabetical order from time
	for i, j := 0, len(backups)-1; i < j; i, j = i+1, j-1 {
		backups[i], backups[j] = backups[j], backups[i]
	}

	return backups, nil
}

func (s *Service) GetBackupPath(ctx context.Context, tenantID, filename string) (string, error) {
	backupDir := "data/storage/backups"
	cleanName := filepath.Base(filename)
	
	if tenantID != "" {
		var tenantCode string
		const q = `SELECT code FROM public.tenants WHERE id = $1`
		if err := s.db.QueryRowContext(ctx, q, tenantID).Scan(&tenantCode); err != nil {
			return "", err
		}
		backupDir = filepath.Join(backupDir, "tenants", tenantCode)
	}

	return filepath.Join(backupDir, cleanName), nil
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

