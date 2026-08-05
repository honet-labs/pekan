package http

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	TenantID string `json:"tenant_id"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type ForgotPasswordRequest struct {
	Email    string `json:"email"`
	TenantID string `json:"tenant_id"`
	Method   string `json:"method"` // "email" or "whatsapp"
}

type ForgotPasswordResponse struct {
	Message string `json:"message"`
}

type ForgotTenantRequest struct {
	EmailOrPhone string `json:"email_or_phone"`
	Method       string `json:"method"` // "email" or "whatsapp"
}

type ForgotTenantResponse struct {
	Message string `json:"message"`
}

type LogoutRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type LoginResponse struct {
	AccessToken           string   `json:"access_token"`
	RefreshToken          string   `json:"refresh_token"`
	TokenType             string   `json:"token_type"`
	AccessTokenExpiresAt  string   `json:"access_token_expires_at"`
	RefreshTokenExpiresAt string   `json:"refresh_token_expires_at"`
	UserID                string   `json:"user_id"`
	Email                 string   `json:"email"`
	ActiveTenantID        string   `json:"active_tenant_id"`
	Permissions           []string `json:"permissions"`
	Features              []string `json:"features"`
	Modules               []string `json:"modules"`
	MustChangePassword    bool     `json:"must_change_password"`
}

type UpdateProfileRequest struct {
	FullName string  `json:"full_name"`
	Username string  `json:"username"`
	Email    string  `json:"email"`
	Phone    *string `json:"phone"`
	Address  *string `json:"address"`
}

type ProfileResponse struct {
	UserID    string  `json:"user_id"`
	FullName  string  `json:"full_name"`
	Username  string  `json:"username"`
	Email     string  `json:"email"`
	Phone     *string `json:"phone"`
	Address   *string `json:"address"`
	UpdatedAt string  `json:"updated_at"`
}

type ChangePasswordRequest struct {
	NewPassword string `json:"new_password"`
}
