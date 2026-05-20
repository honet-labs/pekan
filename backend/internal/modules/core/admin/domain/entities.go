package domain

import "errors"

var (
	ErrTenantAlreadyExists = errors.New("tenant already exists")
	ErrUserAlreadyExists   = errors.New("user already exists")
)

type Tenant struct {
	ID       string `json:"id"`
	Code     string `json:"code"`
	Name     string `json:"name"`
	Status   string `json:"status"`
	Timezone string `json:"timezone"`
	QuotaUsers int `json:"quota_users"`
	QuotaTransactions int `json:"quota_transactions"`
}

type User struct {
	ID           string `json:"id"`
	Email        string `json:"email"`
	FullName     string `json:"full_name"`
	IsActive     bool   `json:"is_active"`
	PasswordHash string `json:"-"`
}

type Membership struct {
	ID       string `json:"id"`
	TenantID string `json:"tenant_id"`
	UserID   string `json:"user_id"`
	Status   string `json:"status"`
}

type TenantListItem struct {
	Tenant
	UserCount int `json:"user_count"`
}

type TenantUser struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	FullName  string `json:"full_name"`
	IsActive  bool   `json:"is_active"`
	Status    string `json:"status"` // membership status
	Role      string `json:"role"`   // primary role
	CreatedAt string `json:"created_at"`
}

type AuditLog struct {
	ID            string `json:"id"`
	TenantID      string `json:"tenant_id"`
	TenantName    string `json:"tenant_name"`
	ActorUserID   string `json:"actor_user_id"`
	ActorUserName string `json:"actor_user_name"`
	Action        string `json:"action"`
	Resource      string `json:"resource"`
	ResourceID    string `json:"resource_id"`
	IPAddress     string `json:"ip_address"`
	CreatedAt     string `json:"created_at"`
	Details       string `json:"details"`
}

type TenantModule struct {
	ModuleCode string `json:"module_code"`
	IsEnabled  bool   `json:"is_enabled"`
}

type GrowthPoint struct {
	Date  string `json:"date"`
	Count int    `json:"count"`
}

type PlatformStats struct {
	Tenants           []GrowthPoint `json:"tenants"`
	Users             []GrowthPoint `json:"users"`
	TotalTenants      int           `json:"total_tenants"`
	TotalUsers        int           `json:"total_users"`
	TotalTransactions int           `json:"total_transactions"`
}

type ServerStatus struct {
	OS          string `json:"os"`
	Uptime      string `json:"uptime"`
	IPAddress   string `json:"ip_address"`
	Port        string `json:"port"`
	DBStatus    string `json:"db_status"`
	RedisStatus string `json:"redis_status"`
	Services    []ServiceStatus `json:"services"`
}

type ServiceStatus struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Port   int    `json:"port"`
}

type BackupFile struct {
	Name      string `json:"name"`
	Size      int64  `json:"size"`
	CreatedAt string `json:"created_at"`
}

type DatabaseTable struct {
	Name      string `json:"name"`
	Rows      int64  `json:"rows"`
	DataSize  string `json:"data_size"`
	IndexSize string `json:"index_size"`
	TotalSize string `json:"total_size"`
}

type QueryResult struct {
	Columns []string         `json:"columns"`
	Rows    []map[string]any `json:"rows"`
	Error   string           `json:"error,omitempty"`
}

type DatabaseGrowthPoint struct {
	Date          string `json:"date"`
	SchemaName    string `json:"schema_name"`
	TotalSizeBytes int64  `json:"total_size_bytes"`
}

type DatabaseGrowthStats struct {
	History []DatabaseGrowthPoint `json:"history"`
}

type WhatsAppQueueItem struct {
	ID               string  `json:"id"`
	PhoneNumber      string  `json:"phone_number"`
	Message          string  `json:"message"`
	ReplyMessage     *string `json:"reply_message"`
	Status           string  `json:"status"`
	ErrorMessage     *string `json:"error_message"`
	ProcessingTimeMs *int    `json:"processing_time_ms"`
	TenantID         *string `json:"tenant_id"`
	TenantCode       *string `json:"tenant_code"`
	UserID           *string `json:"user_id"`
	UserEmail        *string `json:"user_email"`
	ReceivedAt       string  `json:"received_at"`
	ProcessedAt      *string `json:"processed_at"`
}

type WhatsAppQueueStats struct {
	TotalProcessed   int64 `json:"total_processed"`
	TotalPending     int64 `json:"total_pending"`
	TotalProcessing  int64 `json:"total_processing"`
	TotalSuccess     int64 `json:"total_success"`
	TotalFailed      int64 `json:"total_failed"`
	AverageLatencyMs int64 `json:"average_latency_ms"`
}

type UpdateStatusInfo struct {
	CurrentCommit   string `json:"current_commit"`
	CurrentDate     string `json:"current_date"`
	LatestCommit    string `json:"latest_commit"`
	LatestDate      string `json:"latest_date"`
	LatestMessage   string `json:"latest_message"`
	UpdateAvailable bool   `json:"update_available"`
	IsGitRepo       bool   `json:"is_git_repo"`
}

