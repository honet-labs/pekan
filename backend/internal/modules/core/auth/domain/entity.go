package domain

import "time"

type User struct {
	ID           string `json:"id"`
	Email        string `json:"email"`
	PasswordHash string `json:"-"`
	FullName     string `json:"full_name"`
	IsActive           bool   `json:"is_active"`
	MustChangePassword bool   `json:"must_change_password"`
}

type Membership struct {
	ID         string `json:"id"`
	TenantID   string `json:"tenant_id"`
	TenantCode string `json:"tenant_code"`
	UserID     string `json:"user_id"`
	Status     string `json:"status"`
}

type AccessProfile struct {
	Permissions []string `json:"permissions"`
	Features    []string `json:"features"`
	Modules     []string `json:"modules"`
}

type Session struct {
	ID               string    `json:"id"`
	UserID           string    `json:"user_id"`
	TenantID         string    `json:"tenant_id"`
	RefreshTokenHash string    `json:"-"`
	ExpiresAt        time.Time `json:"expires_at"`
	RevokedAt        *time.Time `json:"revoked_at,omitempty"`
	TokenExpiresAt   time.Time `json:"token_expires_at"`
	TokenConsumedAt  *time.Time `json:"token_consumed_at,omitempty"`
	TokenRevokedAt   *time.Time `json:"token_revoked_at,omitempty"`
}

type UserProfile struct {
	UserID    string    `json:"user_id"`
	Email     string    `json:"email"`
	FullName  string    `json:"full_name"`
	Username  string    `json:"username"`
	Phone     *string   `json:"phone,omitempty"`
	Address   *string   `json:"address,omitempty"`
	UpdatedAt time.Time `json:"updated_at"`
}

// RegistrationOTP holds a pending self-registration request awaiting OTP verification.
type RegistrationOTP struct {
	ID           string    `json:"id"`
	SessionToken string    `json:"session_token"`
	OTPCode      string    `json:"otp_code"`
	TenantCode   string    `json:"tenant_code"`
	TenantName   string    `json:"tenant_name"`
	AdminEmail   string    `json:"admin_email"`
	AdminName    string    `json:"admin_name"`
	PasswordHash string    `json:"-"`
	PasswordPlain string   `json:"-"` // Plain password for welcome email
	Phone        string    `json:"phone"`
	OTPMethod    string    `json:"otp_method"` // "email" or "whatsapp"
	Attempts     int       `json:"attempts"`
	Verified     bool      `json:"verified"`
	ExpiresAt    time.Time `json:"expires_at"`
	CreatedAt    time.Time `json:"created_at"`
}

type TenantInfo struct {
	ID   string `json:"id"`
	Code string `json:"code"`
	Name string `json:"name"`
}
