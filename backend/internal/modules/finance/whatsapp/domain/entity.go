package domain

import (
	"context"
	"errors"
	"time"
)

var (
	ErrTokenNotFound    = errors.New("whatsapp otp token not found or expired")
	ErrSessionNotFound  = errors.New("whatsapp session not found")
	ErrAlreadyConnected = errors.New("whatsapp number already connected to an account")
)

type OTPToken struct {
	Token     string
	TenantID  string
	UserID    string
	ExpiresAt time.Time
	CreatedAt time.Time
}

type Session struct {
	PhoneNumber string
	TenantID    string
	UserID      string
	LastActive  *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type ChatItem struct {
	Name  string  `json:"name"`
	Qty   float64 `json:"qty"`
	Price int64   `json:"price"`
}

type RecentTxItem struct {
	ID           string `json:"id"`
	Amount       int64  `json:"amount"`
	Type         string `json:"type"`
	Description  string `json:"description"`
	CategoryName string `json:"category_name"`
	TxDate       string `json:"tx_date"`
}

type BudgetSummaryItem struct {
	Name             string `json:"name"`
	CategoryName     string `json:"category_name"`
	AmountLimitMinor int64  `json:"amount_limit_minor"`
	SpentAmountMinor int64  `json:"spent_amount_minor"`
}

type FinancialContext struct {
	TotalIncome   int64               `json:"total_income"`
	TotalExpense  int64               `json:"total_expense"`
	RecentTx      []RecentTxItem      `json:"recent_tx"`
	ActiveBudgets []BudgetSummaryItem `json:"active_budgets"`
}

type QueueItem struct {
	ID               string     `json:"id"`
	PhoneNumber      string     `json:"phone_number"`
	Message          string     `json:"message"`
	ReplyMessage     *string    `json:"reply_message"`
	Status           string     `json:"status"`
	ErrorMessage     *string    `json:"error_message"`
	ProcessingTimeMs *int       `json:"processing_time_ms"`
	TenantID         *string    `json:"tenant_id"`
	UserID           *string    `json:"user_id"`
	ReceivedAt       time.Time  `json:"received_at"`
	ProcessedAt      *time.Time `json:"processed_at"`
}

type Repository interface {
	// OTP
	CreateOTPToken(ctx context.Context, in OTPToken) error
	GetOTPToken(ctx context.Context, token string) (OTPToken, error)
	DeleteOTPToken(ctx context.Context, token string) error
	DeleteExpiredTokens(ctx context.Context) error

	// Sessions
	CreateSession(ctx context.Context, in Session) error
	GetSessionByPhone(ctx context.Context, phoneNumber string) (Session, error)
	GetSessionByUser(ctx context.Context, tenantID, userID string) (Session, error)
	UpdateLastActive(ctx context.Context, phoneNumber string) error
	DeleteSessionByUser(ctx context.Context, tenantID, userID string) error
	DeleteSessionByPhone(ctx context.Context, phoneNumber string) error

	// Public Tenant Info
	GetTenantCode(ctx context.Context, tenantID string) (string, error)

	// Transaction creation inside tenant schema
	CreateChatTransaction(ctx context.Context, tenantID, userID, tenantCode string, amount int64, typeStr, description, categoryName string, transactionDate string, items []ChatItem) (string, error)

	// Find transaction by ID (full or prefix) inside tenant schema
	FindChatTransaction(ctx context.Context, tenantID, userID, tenantCode, txID string) (*RecentTxItem, error)

	// Delete transaction by ID (full or prefix) inside tenant schema
	DeleteChatTransaction(ctx context.Context, tenantID, userID, tenantCode, txID string) error

	// Update transaction by ID (full or prefix) inside tenant schema
	UpdateChatTransaction(ctx context.Context, tenantID, userID, tenantCode, txID string, amount int64, typeStr, description, categoryName string, transactionDate string) error

	// Financial Context Query inside tenant schema
	GetFinancialContext(ctx context.Context, tenantID, userID, tenantCode string) (*FinancialContext, error)

	// Asynchronous Queue Operations
	EnqueueMessage(ctx context.Context, phoneNumber, message string, tenantID, userID *string) (string, error)
	GetPendingQueueItems(ctx context.Context, limit int) ([]QueueItem, error)
	UpdateQueueItemStatus(ctx context.Context, id string, status string, replyMessage *string, errorMessage *string, latencyMs *int) error
}
