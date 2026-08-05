package usecase

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/big"
	"net"
	"net/http"
	"net/smtp"
	"regexp"
	"io"
	"strings"
	"time"

	"github.com/google/uuid"

	authdomain "pekan/backend/internal/modules/core/auth/domain"
	"pekan/backend/internal/platform/audit"
	platformauth "pekan/backend/internal/platform/auth"
	"pekan/backend/internal/platform/session"
)

// GlobalSettingReader allows auth to read platform settings (SMTP/WAHA) without importing admin package.
type GlobalSettingReader interface {
	GetGlobalSettingRaw(ctx context.Context, key string) (string, error)
}

// TenantBootstrapper allows auth to create a new tenant after OTP verification.
type TenantBootstrapper interface {
	BootstrapTenantDirect(ctx context.Context, tenantCode, tenantName, adminEmail, adminName, passwordHash string) error
}

// OTPNotifier allows mocking of OTP delivery (Email/WA) in tests.
type OTPNotifier interface {
	SendOTP(ctx context.Context, method, email, phone, message string) error
}

type Service struct {
	repo            authdomain.Repository
	jwt             *platformauth.Service
	refreshTokenTTL time.Duration
	settings        GlobalSettingReader
	bootstrapper    TenantBootstrapper
	notifier        OTPNotifier
	audit           audit.Logger
	sessionStore    session.Store
}

func NewService(repo authdomain.Repository, jwt *platformauth.Service, refreshTokenTTL time.Duration, logger audit.Logger) *Service {
	return &Service{
		repo:            repo,
		jwt:             jwt,
		refreshTokenTTL: refreshTokenTTL,
		audit:           logger,
	}
}

// WithDependencies attaches optional infrastructure dependencies.
func (s *Service) WithDependencies(settings GlobalSettingReader, bootstrapper TenantBootstrapper, notifier OTPNotifier, sessStore session.Store) {
	s.settings = settings
	s.bootstrapper = bootstrapper
	s.notifier = notifier
	s.sessionStore = sessStore
}

type LoginInput struct {
	Email    string
	Password string
	TenantID string
}

type LoginOutput struct {
	Tokens     TokenOutput
	User       authdomain.User
	Membership authdomain.Membership
	Access     authdomain.AccessProfile
}

type TokenOutput struct {
	AccessToken           string
	RefreshToken          string
	AccessTokenExpiresAt  time.Time
	RefreshTokenExpiresAt time.Time
}

func (s *Service) Login(ctx context.Context, in LoginInput) (LoginOutput, error) {
	email := strings.ToLower(strings.TrimSpace(in.Email))
	tenantID := strings.ToLower(strings.TrimSpace(in.TenantID))

	log.Printf("[Auth] Login attempt for email=%s, tenant=%s", email, tenantID)

	// Resolve Tenant Code to UUID if the input is a Code
	actualTenantID := tenantID
	// If it's not a UUID (36 chars), it MUST be a code that we need to resolve
	if len(tenantID) != 36 {
		resolvedID, err := s.repo.GetTenantIDByCode(ctx, tenantID)
		if err != nil {
			log.Printf("[Auth] Tenant resolution failed for %s: %v", tenantID, err)
			return LoginOutput{}, authdomain.ErrMembershipNotFound
		}
		actualTenantID = resolvedID
	}
	log.Printf("[Auth] Resolved tenant ID: %s", actualTenantID)
	log.Printf("[Auth] TRACE: Checking if account is locked...")

	if s.sessionStore != nil {
		locked, retryAfter, _ := s.sessionStore.IsAccountLocked(ctx, email, actualTenantID)
		if locked {
			log.Printf("[Auth] Account locked: %s", email)
			return LoginOutput{}, fmt.Errorf("account locked due to too many failed attempts, try again in %v", retryAfter.Round(time.Minute))
		}
	}

	log.Printf("[Auth] TRACE: Fetching user by email: %s", email)
	user, err := s.repo.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, authdomain.ErrInvalidCredentials) {
			log.Printf("[Auth] User not found: %s", email)
			s.recordFailedLogin(ctx, email, actualTenantID, "user_not_found")
			return LoginOutput{}, authdomain.ErrInvalidCredentials
		}
		log.Printf("[Auth] GetUserByEmail database error: %v", err)
		return LoginOutput{}, err
	}

	if !user.IsActive {
		log.Printf("[Auth] User inactive: %s", email)
		s.recordFailedLogin(ctx, email, actualTenantID, "user_inactive")
		return LoginOutput{}, authdomain.ErrUserInactive
	}

	log.Printf("[Auth] TRACE: Verifying password for: %s", email)
	if err := platformauth.VerifyPassword(user.PasswordHash, in.Password); err != nil {
		log.Printf("[Auth] Password mismatch for: %s", email)
		s.recordFailedLogin(ctx, email, actualTenantID, "invalid_password")
		return LoginOutput{}, authdomain.ErrInvalidCredentials
	}
	log.Printf("[Auth] TRACE: Password verified successfully")

	// Successful login: clear the failure counter
	log.Printf("[Auth] TRACE: Clearing failed login counter...")
	if s.sessionStore != nil {
		_ = s.sessionStore.ClearFailedLogin(ctx, email, actualTenantID)
	}
	log.Printf("[Auth] TRACE: Fetching membership...")

	membership, err := s.repo.GetMembershipByUserAndTenant(ctx, user.ID, actualTenantID)
	if err != nil {
		log.Printf("[Auth] Membership NOT FOUND for user=%s in tenant=%s: %v", user.ID, actualTenantID, err)
		return LoginOutput{}, authdomain.ErrMembershipNotFound
	}

	if membership.Status != "active" {
		log.Printf("[Auth] Membership suspended for user=%s in tenant=%s", user.ID, actualTenantID)
		return LoginOutput{}, authdomain.ErrMembershipSuspended
	}

	log.Printf("[Auth] TRACE: Fetching access profile...")
	access, err := s.repo.GetAccessProfileByMembership(ctx, membership.ID, membership.TenantID)
	if err != nil {
		log.Printf("[Auth] Access Profile MISSING for membership=%s, tenant=%s: %v", membership.ID, membership.TenantID, err)
		return LoginOutput{}, authdomain.ErrAccessProfileMissing
	}


	log.Printf("[Auth] TRACE: Issuing tokens...")
	tokens, err := s.issueTokens(ctx, user, membership, access)
	if err != nil {
		log.Printf("[Auth] Token issuance failed: %v", err)
		return LoginOutput{}, err
	}
	log.Printf("[Auth] TRACE: Login usecase completed successfully")

	if s.audit != nil {
		loginCtx := audit.WithContext(ctx, audit.AuditContext{
			TenantID:    actualTenantID,
			ActorUserID: user.ID,
		})
		_ = s.audit.Write(loginCtx, "auth.login.success", "user", user.ID, nil, map[string]any{
			"tenant_id": actualTenantID,
			"email":     user.Email,
		})
	}

	return LoginOutput{
		Tokens:     tokens,
		User:       user,
		Membership: membership,
		Access:     access,
	}, nil
}

func (s *Service) recordFailedLogin(ctx context.Context, email, tenantID, reason string) {
	if s.sessionStore != nil {
		_, _, _ = s.sessionStore.RecordFailedLogin(ctx, email, tenantID)
	}
	if s.audit != nil {
		failCtx := audit.WithContext(ctx, audit.AuditContext{
			TenantID: tenantID,
		})
		_ = s.audit.Write(failCtx, "auth.login.failure", "auth", "", nil, map[string]any{
			"email":     email,
			"tenant_id": tenantID,
			"reason":    reason,
		})
	}
}

type RefreshInput struct {
	RefreshToken string
}

type RefreshOutput struct {
	Tokens     TokenOutput
	User       authdomain.User
	Membership authdomain.Membership
	Access     authdomain.AccessProfile
}

func (s *Service) Refresh(ctx context.Context, in RefreshInput) (RefreshOutput, error) {
	refreshHash := HashToken(in.RefreshToken)
	session, err := s.repo.GetSessionByRefreshHash(ctx, refreshHash)
	if err != nil {
		return RefreshOutput{}, authdomain.ErrSessionNotFound
	}

	if session.TokenRevokedAt != nil || session.TokenConsumedAt != nil {
		_ = s.repo.RevokeAllSessionsByUser(ctx, session.UserID)
		return RefreshOutput{}, authdomain.ErrRefreshTokenReused
	}

	if time.Now().UTC().After(session.ExpiresAt) {
		return RefreshOutput{}, authdomain.ErrSessionExpired
	}

	user, err := s.repo.GetUserByID(ctx, session.UserID)
	if err != nil || !user.IsActive {
		return RefreshOutput{}, authdomain.ErrUserInactive
	}

	membership, err := s.repo.GetMembershipByUserAndTenant(ctx, user.ID, session.TenantID)
	if err != nil || membership.Status != "active" {
		return RefreshOutput{}, authdomain.ErrMembershipNotFound
	}

	access, err := s.repo.GetAccessProfileByMembership(ctx, membership.ID, membership.TenantID)
	if err != nil {
		return RefreshOutput{}, authdomain.ErrAccessProfileMissing
	}

	// Revoke current session (simple rotation)
	if err := s.repo.RevokeSessionByRefreshHash(ctx, refreshHash); err != nil {
		return RefreshOutput{}, err
	}

	tokens, err := s.issueTokens(ctx, user, membership, access)
	if err != nil {
		return RefreshOutput{}, err
	}

	if s.audit != nil {
		_ = s.audit.Write(ctx, "auth.refresh.success", "user", session.UserID, nil, map[string]any{
			"tenant_id": session.TenantID,
			"session":   session.ID,
		})
	}

	return RefreshOutput{
		Tokens:     tokens,
		User:       user,
		Membership: membership,
		Access:     access,
	}, nil
}

func (s *Service) Logout(ctx context.Context, refreshToken string) error {
	refreshHash := HashToken(refreshToken)
	session, err := s.repo.GetSessionByRefreshHash(ctx, refreshHash)
	if err == nil {
		if s.audit != nil {
			_ = s.audit.Write(ctx, "auth.logout.success", "user", session.UserID, nil, map[string]any{
				"tenant_id": session.TenantID,
			})
		}
		if s.sessionStore != nil {
			// Blacklist the SessionID in Redis for 1 hour (plenty of time to cover short-lived Access Tokens)
			_ = s.sessionStore.RevokeToken(ctx, session.ID, time.Hour)
		}
	}
	return s.repo.RevokeSessionByRefreshHash(ctx, refreshHash)
}

func (s *Service) LogoutAll(ctx context.Context, userID string) error {
	// Ideally we would fetch all active SessionIDs and blacklist them in Redis,
	// but to keep it simple, we just rely on DB revocation for LogoutAll.
	// We can blacklist the user entirely if we modify IsTokenRevoked to also check user blacklists.
	if s.audit != nil {
		_ = s.audit.Write(ctx, "auth.logout_all.success", "user", userID, nil, nil)
	}
	return s.repo.RevokeAllSessionsByUser(ctx, userID)
}

func (s *Service) GetContext(ctx context.Context, userID, tenantID string) (ContextOutput, error) {
	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return ContextOutput{}, err
	}
	memberships, err := s.repo.ListMembershipsByUserID(ctx, userID)
	if err != nil {
		return ContextOutput{}, err
	}
	var active authdomain.Membership
	found := false
	for _, m := range memberships {
		if m.TenantID == tenantID {
			active = m
			found = true
			break
		}
	}
	if !found {
		return ContextOutput{}, authdomain.ErrMembershipNotFound
	}

	access, err := s.repo.GetAccessProfileByMembership(ctx, active.ID, active.TenantID)
	if err != nil {
		return ContextOutput{}, authdomain.ErrAccessProfileMissing
	}

	if s.audit != nil {
		_ = s.audit.Write(ctx, "auth.get_context.success", "user", userID, nil, map[string]any{
			"tenant_id": tenantID,
		})
	}

	return ContextOutput{
		User:        user,
		Active:      active,
		Memberships: memberships,
		Access:      access,
	}, nil
}

type ContextOutput struct {
	User        authdomain.User
	Active      authdomain.Membership
	Memberships []authdomain.Membership
	Access      authdomain.AccessProfile
}

func (s *Service) GetProfile(ctx context.Context, userID string) (authdomain.UserProfile, error) {
	return s.repo.GetUserProfile(ctx, userID)
}

type UpdateProfileInput struct {
	UserID   string
	Email    string
	FullName string
	Username string
	Phone    *string
	Address  *string
}

func (s *Service) UpdateProfile(ctx context.Context, in UpdateProfileInput) (authdomain.UserProfile, error) {
	email := strings.ToLower(strings.TrimSpace(in.Email))
	fullName := strings.TrimSpace(in.FullName)
	username := strings.ToLower(strings.TrimSpace(in.Username))

	if email == "" || fullName == "" {
		return authdomain.UserProfile{}, errors.New("email and full name are required")
	}

	// Check uniqueness
	// Email
	existingUser, err := s.repo.GetUserByEmail(ctx, email)
	if err == nil && existingUser.ID != in.UserID {
		return authdomain.UserProfile{}, fmt.Errorf("email '%s' sudah digunakan oleh pengguna lain", email)
	}

	// Username
	if username != "" {
		taken, err := s.repo.IsUsernameTaken(ctx, username)
		if err != nil {
			return authdomain.UserProfile{}, err
		}
		if taken {
			// Check if it's the current user's username
			current, _ := s.repo.GetUserProfile(ctx, in.UserID)
			if current.Username != username {
				return authdomain.UserProfile{}, fmt.Errorf("username '%s' sudah digunakan", username)
			}
		}
	}

	// Phone
	if in.Phone != nil && *in.Phone != "" {
		cleanedPhone := sanitizePhone(*in.Phone)
		in.Phone = &cleanedPhone
		phone := cleanedPhone
		taken, err := s.repo.IsPhoneRegistered(ctx, phone)
		if err != nil {
			return authdomain.UserProfile{}, err
		}
		if taken {
			current, _ := s.repo.GetUserProfile(ctx, in.UserID)
			if current.Phone == nil || *current.Phone != phone {
				return authdomain.UserProfile{}, fmt.Errorf("nomor telepon '%s' sudah terdaftar", phone)
			}
		}
	}

	profile := authdomain.UserProfile{
		UserID:    in.UserID,
		Email:     email,
		FullName:  fullName,
		Username:  username,
		Phone:     in.Phone,
		Address:   in.Address,
		UpdatedAt: time.Now().UTC(),
	}
	return s.repo.UpdateUserProfile(ctx, profile)
}

type SwitchTenantInput struct {
	UserID   string
	TenantID string
	Email    string
}

func (s *Service) SwitchTenant(ctx context.Context, in SwitchTenantInput) (TokenOutput, error) {
	membership, err := s.repo.GetMembershipByUserAndTenant(ctx, in.UserID, in.TenantID)
	if err != nil {
		return TokenOutput{}, authdomain.ErrMembershipNotFound
	}
	if membership.Status != "active" {
		return TokenOutput{}, authdomain.ErrMembershipSuspended
	}
	access, err := s.repo.GetAccessProfileByMembership(ctx, membership.ID, membership.TenantID)
	if err != nil {
		return TokenOutput{}, err
	}

	tokens, err := s.issueTokens(ctx, authdomain.User{ID: in.UserID, Email: in.Email}, membership, access)
	if err != nil {
		return TokenOutput{}, err
	}

	return tokens, nil
}

func (s *Service) RequestPasswordReset(ctx context.Context, email, tenantID, method string) error {
	email = strings.ToLower(strings.TrimSpace(email))
	user, err := s.repo.GetUserByEmail(ctx, email)
	if err != nil {
		// Log internal error but return nil to prevent user enumeration
		log.Printf("[Auth] RequestPasswordReset failed: user %s not found (DB err: %v)", email, err)
		return nil 
	}

	log.Printf("[Auth] RequestPasswordReset: initiating password reset via %s for user %s (%s) in tenant %s", method, email, user.ID, tenantID)

	// In a real app, we'd generate a token and save it to a password_resets table.
	// For now, we'll include email and tenantID in the link so frontend knows who to reset.
	// We use URL encoding for safety.
	resetLink := fmt.Sprintf("https://pekan.honet.web.id/reset-password?t=%s&e=%s", tenantID, email)
	message := fmt.Sprintf("Halo %s, silakan gunakan tautan berikut untuk mengatur ulang kata sandi Anda di Workspace %s: %s", user.FullName, tenantID, resetLink)
	
	var sendErr error
	if method == "whatsapp" {
		profile, err := s.repo.GetUserProfile(ctx, user.ID)
		if err != nil {
			log.Printf("[Auth] RequestPasswordReset failed to get profile for %s: %v", email, err)
			return fmt.Errorf("failed to get user profile: %w", err)
		}
		if profile.Phone != nil && *profile.Phone != "" {
			log.Printf("[Auth] RequestPasswordReset: sending WhatsApp reset link to %s", *profile.Phone)
			if s.notifier != nil {
				sendErr = s.notifier.SendOTP(ctx, "whatsapp", email, *profile.Phone, message)
			} else {
				sendErr = s.sendOTPViaWAHA(ctx, *profile.Phone, message)
			}
		} else {
			log.Printf("[Auth] RequestPasswordReset failed: user %s has no phone number", email)
			return fmt.Errorf("nomor WhatsApp tidak ditemukan di profil Anda")
		}
	} else {
		log.Printf("[Auth] RequestPasswordReset: sending Email reset link to %s", email)
		if s.notifier != nil {
			sendErr = s.notifier.SendOTP(ctx, "email", email, "", message)
		} else {
			sendErr = s.sendEmail(ctx, email, "Atur Ulang Kata Sandi PEKAN", message)
		}
	}

	if sendErr != nil {
		log.Printf("[Auth] Failed to send password reset via %s to %s: %v", method, email, sendErr)
		return fmt.Errorf("gagal mengirim permintaan reset via %s: %w", method, sendErr)
	}

	log.Printf("[Auth] RequestPasswordReset successful: password reset instructions sent via %s to %s", method, email)
	return nil
}


func (s *Service) ForgotTenant(ctx context.Context, emailOrPhone, method string) error {
	query := strings.TrimSpace(emailOrPhone)
	log.Printf("[Auth] ForgotTenant: request received for %s via %s", query, method)

	tenants, err := s.repo.ListTenantsByEmailOrPhone(ctx, query)
	if err != nil {
		log.Printf("[Auth] ForgotTenant failed to list tenants for %s: %v", query, err)
		return nil
	}
	if len(tenants) == 0 {
		log.Printf("[Auth] ForgotTenant completed: no tenants found for query %s", query)
		return nil
	}

	log.Printf("[Auth] ForgotTenant: found %d tenants for query %s", len(tenants), query)

	var sb strings.Builder
	sb.WriteString("Halo, berikut adalah daftar Workspace PEKAN yang terhubung dengan akun Anda:\n\n")
	for _, t := range tenants {
		sb.WriteString(fmt.Sprintf("- %s (ID: %s)\n", t.Name, t.Code))
	}
	sb.WriteString("\nSilakan gunakan ID tersebut untuk login.")
	message := sb.String()

	if method == "whatsapp" {
		phone := query
		if strings.Contains(query, "@") {
			// Find user's phone if query is email
			user, err := s.repo.GetUserByEmail(ctx, query)
			if err == nil {
				profile, _ := s.repo.GetUserProfile(ctx, user.ID)
				if profile.Phone != nil && *profile.Phone != "" {
					phone = *profile.Phone
				} else {
					log.Printf("[Auth] ForgotTenant completed: email %s found but no phone registered", query)
					return nil // No phone found
				}
			} else {
				log.Printf("[Auth] ForgotTenant completed: failed to get user by email %s: %v", query, err)
				return nil
			}
		}
		
		log.Printf("[Auth] ForgotTenant: sending WhatsApp Workspace info to %s", phone)
		errSend := s.sendOTPViaWAHA(ctx, phone, message)
		if errSend != nil {
			log.Printf("[Auth] ForgotTenant WA send failed: %v", errSend)
		} else {
			log.Printf("[Auth] ForgotTenant WA send successful to %s", phone)
		}
	} else {
		email := query
		if !strings.Contains(query, "@") {
			// Find user's email if query is phone
			user, err := s.repo.GetUserByPhone(ctx, query)
			if err == nil {
				email = user.Email
			} else {
				log.Printf("[Auth] ForgotTenant completed: failed to get user by phone %s: %v", query, err)
				return nil
			}
		}
		
		log.Printf("[Auth] ForgotTenant: sending Email Workspace info to %s", email)
		errSend := s.sendEmail(ctx, email, "Informasi Workspace PEKAN Anda", message)
		if errSend != nil {
			log.Printf("[Auth] ForgotTenant Email send failed: %v", errSend)
		} else {
			log.Printf("[Auth] ForgotTenant Email send successful to %s", email)
		}
	}

	return nil
}

func (s *Service) ResetPassword(ctx context.Context, email, tenantID, newPassword string) error {
	email = strings.ToLower(strings.TrimSpace(email))
	tenantID = strings.ToLower(strings.TrimSpace(tenantID))

	// Verify account exists in the workspace
	user, err := s.repo.GetUserByEmail(ctx, email)
	if err != nil {
		return authdomain.ErrInvalidCredentials
	}

	// Resolve Tenant Code to UUID if the input is a Code
	actualTenantID := tenantID
	if len(tenantID) != 36 {
		if resolvedID, err := s.repo.GetTenantIDByCode(ctx, tenantID); err == nil {
			actualTenantID = resolvedID
		}
	}

	_, err = s.repo.GetMembershipByUserAndTenant(ctx, user.ID, actualTenantID)
	if err != nil {
		log.Printf("[AuthService] ResetPassword membership check failed: user=%s, tenant=%s, err=%v", user.ID, actualTenantID, err)
		return authdomain.ErrInvalidCredentials // Account doesn't belong to this tenant
	}

	// Update password
	passwordHash, err := platformauth.HashPassword(newPassword)
	if err != nil {
		return err
	}

	err = s.repo.UpdateUserPassword(ctx, user.ID, passwordHash)
	if err != nil {
		return err
	}

	// Optional: Revoke all sessions for security
	_ = s.repo.RevokeAllSessionsByUser(ctx, user.ID)

	return nil
}

func (s *Service) issueTokens(ctx context.Context, user authdomain.User, membership authdomain.Membership, access authdomain.AccessProfile) (TokenOutput, error) {
	refreshToken, refreshHash, refreshExpiresAt, err := s.newRefreshToken()
	if err != nil {
		return TokenOutput{}, err
	}

	sessionID := uuid.NewString()
	_, err = s.repo.CreateSession(ctx, authdomain.Session{
		ID:               sessionID,
		UserID:           user.ID,
		TenantID:         membership.TenantID,
		RefreshTokenHash: refreshHash,
		ExpiresAt:        refreshExpiresAt,
	})
	if err != nil {
		return TokenOutput{}, err
	}

	tenantCode, _ := s.repo.GetTenantCodeByID(ctx, membership.TenantID)

	accessToken, accessExpiresAt, err := s.jwt.IssueAccessToken(platformauth.IssueAccessTokenInput{
		UserID:      user.ID,
		Email:       user.Email,
		TenantID:    membership.TenantID,
		TenantCode:  tenantCode,
		SessionID:   sessionID,
		Permissions: access.Permissions,
		Features:    access.Features,
		Modules:     access.Modules,
	})
	if err != nil {
		return TokenOutput{}, err
	}

	return TokenOutput{
		AccessToken:           accessToken,
		RefreshToken:          refreshToken,
		AccessTokenExpiresAt:  accessExpiresAt,
		RefreshTokenExpiresAt: refreshExpiresAt,
	}, nil
}

func (s *Service) newRefreshToken() (token string, hash string, expiresAt time.Time, err error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", "", time.Time{}, err
	}
	token = base64.URLEncoding.EncodeToString(b)
	hash = HashToken(token)
	expiresAt = time.Now().UTC().Add(s.refreshTokenTTL)
	return token, hash, expiresAt, nil
}

func HashToken(token string) string {
	h := sha256.New()
	h.Write([]byte(token))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

func normalizeOptionalText(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	return s
}

func ptr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// --- SELF REGISTRATION FLOW ---

const (
	otpTTL         = 10 * time.Minute
	maxOTPAttempts = 5
)

type RegisterInitInput struct {
	TenantCode string
	TenantName string
	AdminEmail string
	AdminName  string
	Password   string
	Phone      string
	OTPMethod  string // "email" or "whatsapp"
}

type RegisterInitOutput struct {
	SessionToken string
	OTPMethod    string
}

func (s *Service) RegisterInit(ctx context.Context, in RegisterInitInput) (RegisterInitOutput, error) {
	code := strings.ToUpper(strings.TrimSpace(in.TenantCode))
	email := strings.ToLower(strings.TrimSpace(in.AdminEmail))
	method := strings.ToLower(strings.TrimSpace(in.OTPMethod))
	if method == "" {
		method = "email"
	}

	if code == "" || email == "" || in.Password == "" {
		return RegisterInitOutput{}, fmt.Errorf("ID Workspace, Email, dan Password wajib diisi")
	}

	if matched, _ := regexp.MatchString(`^[A-Z0-9_-]{3,20}$`, code); !matched {
		return RegisterInitOutput{}, fmt.Errorf("ID Workspace harus 3-20 karakter (huruf, angka, _ atau -)")
	}

	exists, err := s.repo.IsTenantCodeTaken(ctx, code)
	if err != nil {
		return RegisterInitOutput{}, err
	}
	if exists {
		return RegisterInitOutput{}, fmt.Errorf("ID Workspace '%s' sudah digunakan", code)
	}

	emailTaken, err := s.repo.IsEmailRegistered(ctx, email)
	if err != nil {
		return RegisterInitOutput{}, err
	}
	if emailTaken {
		return RegisterInitOutput{}, fmt.Errorf("email '%s' sudah terdaftar", email)
	}

	// Phone
	if in.Phone != "" {
		phone := strings.TrimSpace(in.Phone)
		phoneTaken, err := s.repo.IsPhoneRegistered(ctx, phone)
		if err != nil {
			return RegisterInitOutput{}, err
		}
		if phoneTaken {
			return RegisterInitOutput{}, fmt.Errorf("nomor telepon '%s' sudah terdaftar", phone)
		}
	}

	// Tenant Name
	name := strings.TrimSpace(in.TenantName)
	nameTaken, err := s.repo.IsTenantNameTaken(ctx, name)
	if err != nil {
		return RegisterInitOutput{}, err
	}
	if nameTaken {
		return RegisterInitOutput{}, fmt.Errorf("nama workspace '%s' sudah digunakan", name)
	}

	passwordHash, err := platformauth.HashPassword(in.Password)
	if err != nil {
		return RegisterInitOutput{}, err
	}

	otpCode, err := generateOTP(6)
	if err != nil {
		return RegisterInitOutput{}, err
	}

	sessionToken, err := generateSessionToken()
	if err != nil {
		return RegisterInitOutput{}, err
	}

	otpRecord := authdomain.RegistrationOTP{
		SessionToken:  sessionToken,
		OTPCode:       otpCode,
		TenantCode:    code,
		TenantName:    strings.TrimSpace(in.TenantName),
		AdminEmail:    email,
		AdminName:     strings.TrimSpace(in.AdminName),
		PasswordHash:  passwordHash,
		PasswordPlain: in.Password,
		Phone:         strings.TrimSpace(in.Phone),
		OTPMethod:     method,
		ExpiresAt:     time.Now().UTC().Add(otpTTL),
	}

	if err := s.repo.SaveRegistrationOTP(ctx, otpRecord); err != nil {
		return RegisterInitOutput{}, err
	}

	message := fmt.Sprintf("Kode OTP pendaftaran workspace PEKAN Anda: *%s*\n\nKode berlaku selama 10 menit.", otpCode)
	if err := s.sendOTP(ctx, method, email, in.Phone, message); err != nil {
		return RegisterInitOutput{}, fmt.Errorf("gagal mengirim OTP: %v", err)
	}

	return RegisterInitOutput{
		SessionToken: sessionToken,
		OTPMethod:    method,
	}, nil
}

type RegisterVerifyInput struct {
	SessionToken string
	OTPCode      string
}

func (s *Service) RegisterVerify(ctx context.Context, in RegisterVerifyInput) error {
	record, err := s.repo.GetRegistrationOTP(ctx, in.SessionToken)
	if err != nil {
		return fmt.Errorf("sesi registrasi tidak ditemukan atau sudah kadaluarsa")
	}

	if record.Verified {
		return fmt.Errorf("sesi registrasi ini sudah digunakan")
	}

	if time.Now().UTC().After(record.ExpiresAt) {
		return fmt.Errorf("kode OTP sudah kadaluarsa")
	}

	if record.Attempts >= maxOTPAttempts {
		return fmt.Errorf("terlalu banyak percobaan yang salah")
	}

	if record.OTPCode != strings.TrimSpace(in.OTPCode) {
		_ = s.repo.IncrementOTPAttempts(ctx, in.SessionToken)
		if s.audit != nil {
			_ = s.audit.Write(ctx, "auth.register.otp_failure", "auth", "", nil, map[string]any{
				"email":       record.AdminEmail,
				"tenant_code": record.TenantCode,
				"attempts":    record.Attempts + 1,
			})
		}
		remaining := maxOTPAttempts - record.Attempts - 1
		return fmt.Errorf("kode OTP salah. Sisa percobaan: %d", remaining)
	}

	if s.bootstrapper == nil {
		return fmt.Errorf("layanan registrasi belum dikonfigurasi")
	}

	if err := s.bootstrapper.BootstrapTenantDirect(ctx,
		record.TenantCode, record.TenantName, record.AdminEmail, record.AdminName, record.PasswordHash,
	); err != nil {
		return fmt.Errorf("gagal membuat workspace: %v", err)
	}

	_ = s.repo.MarkOTPVerified(ctx, in.SessionToken)

	go func() {
		bgCtx := context.Background()
		_ = s.sendWelcomeEmail(bgCtx, record)
	}()

	return nil
}

func (s *Service) sendWelcomeEmail(ctx context.Context, record authdomain.RegistrationOTP) error {
	subject := "Selamat Datang di PEKAN - Workspace Anda Telah Siap!"
	message := fmt.Sprintf(`Halo %s,

Selamat! Workspace keuangan Anda telah berhasil dibuat.

Detail akun:
- Nama Workspace : %s
- ID Workspace   : %s
- Email Admin    : %s
- Password       : %s

Login di: https://pekan.honet.web.id/login

Salam,
Tim PEKAN`, record.AdminName, record.TenantName, record.TenantCode, record.AdminEmail, record.PasswordPlain)

	return s.sendEmail(ctx, record.AdminEmail, subject, message)
}

func (s *Service) sendOTP(ctx context.Context, method, email, phone, message string) error {
	// Use external notifier if provided (e.g. for tests)
	if s.notifier != nil {
		return s.notifier.SendOTP(ctx, method, email, phone, message)
	}

	if s.settings == nil {
		return fmt.Errorf("konfigurasi notifikasi belum diatur")
	}
	if method == "whatsapp" {
		provider, _ := s.settings.GetGlobalSettingRaw(ctx, "notification_wa_active_provider")
		if provider == "" { provider = "wa_waha" } // Default fallback

		switch provider {
		case "wa_fonnte":
			return s.sendOTPViaFonnte(ctx, phone, message)
		case "wa": // Meta/Official
			return s.sendOTPViaMeta(ctx, phone, message)
		default:
			return s.sendOTPViaWAHA(ctx, phone, message)
		}
	}
	return s.sendOTPViaEmail(ctx, email, message)
}

func (s *Service) sendOTPViaEmail(ctx context.Context, toEmail, message string) error {
	return s.sendEmail(ctx, toEmail, "Kode OTP Pendaftaran PEKAN", message)
}

func (s *Service) sendEmail(ctx context.Context, toEmail, subject, message string) error {
	configJSON, err := s.settings.GetGlobalSettingRaw(ctx, "notification_email_smtp")
	if err != nil || configJSON == "" {
		configJSON, _ = s.settings.GetGlobalSettingRaw(ctx, "notification_smtp")
	}
	if configJSON == "" {
		return fmt.Errorf("konfigurasi SMTP belum diatur")
	}
	var cfg struct {
		Host     string `json:"host"`
		Port     string `json:"port"`
		Username string `json:"username"`
		Password string `json:"password"`
		Security string `json:"security"` // none, ssl, starttls
	}
	if err := json.Unmarshal([]byte(configJSON), &cfg); err != nil || cfg.Host == "" {
		return fmt.Errorf("konfigurasi SMTP tidak valid")
	}

	addr := fmt.Sprintf("%s:%s", cfg.Host, cfg.Port)
	subjectLine := fmt.Sprintf("Subject: %s\r\n", subject)
	toLine := fmt.Sprintf("To: %s\r\n", toEmail)
	mime := "MIME-version: 1.0;\nContent-Type: text/plain; charset=\"UTF-8\";\n\n"
	body := []byte(toLine + subjectLine + mime + message)

	auth := smtp.PlainAuth("", cfg.Username, cfg.Password, cfg.Host)

	log.Printf("[SMTP] Attempting connection to %s (Security: %s)", addr, cfg.Security)

	if strings.ToLower(cfg.Security) == "ssl" {
		log.Printf("[SMTP] Dialing TLS to %s...", addr)
		tlsconfig := &tls.Config{
			InsecureSkipVerify: true,
			ServerName:         cfg.Host,
		}
		dialer := &net.Dialer{Timeout: 8 * time.Second}
		conn, err := tls.DialWithDialer(dialer, "tcp", addr, tlsconfig)
		if err != nil {
			log.Printf("[SMTP] SSL Dial failed: %v", err)
			return err
		}
		log.Printf("[SMTP] SSL Connected. Creating client...")
		client, err := smtp.NewClient(conn, cfg.Host)
		if err != nil {
			log.Printf("[SMTP] Failed to create client: %v", err)
			return err
		}
		log.Printf("[SMTP] Authenticating as %s...", cfg.Username)
		if err = client.Auth(auth); err != nil {
			log.Printf("[SMTP] Auth failed: %v", err)
			return err
		}
		log.Printf("[SMTP] Sending data to %s...", toEmail)
		if err = client.Mail(cfg.Username); err != nil {
			return err
		}
		if err = client.Rcpt(toEmail); err != nil {
			return err
		}
		w, err := client.Data()
		if err != nil {
			return err
		}
		_, err = w.Write(body)
		if err != nil {
			return err
		}
		err = w.Close()
		if err != nil {
			return err
		}
		log.Printf("[SMTP] Email sent successfully via SSL")
		return client.Quit()
	}

	if strings.ToLower(cfg.Security) == "starttls" {
		log.Printf("[SMTP] Dialing TCP for STARTTLS to %s...", addr)
		d := &net.Dialer{Timeout: 8 * time.Second}
		conn, err := d.DialContext(ctx, "tcp", addr)
		if err != nil {
			log.Printf("[SMTP] TCP Dial failed: %v", err)
			return err
		}
		c, err := smtp.NewClient(conn, cfg.Host)
		if err != nil {
			log.Printf("[SMTP] Failed to create client: %v", err)
			return err
		}
		tlsconfig := &tls.Config{
			InsecureSkipVerify: true,
			ServerName:         cfg.Host,
		}
		log.Printf("[SMTP] Executing STARTTLS handshake...")
		if err = c.StartTLS(tlsconfig); err != nil {
			log.Printf("[SMTP] STARTTLS failed: %v", err)
			return err
		}
		log.Printf("[SMTP] Authenticating as %s...", cfg.Username)
		if err = c.Auth(auth); err != nil {
			log.Printf("[SMTP] Auth failed: %v", err)
			return err
		}
		log.Printf("[SMTP] Sending data to %s...", toEmail)
		if err = c.Mail(cfg.Username); err != nil {
			return err
		}
		if err = c.Rcpt(toEmail); err != nil {
			return err
		}
		w, err := c.Data()
		if err != nil {
			return err
		}
		_, err = w.Write(body)
		if err != nil {
			return err
		}
		err = w.Close()
		if err != nil {
			return err
		}
		log.Printf("[SMTP] Email sent successfully via STARTTLS")
		return c.Quit()
	}

	// Default / None (Standard SendMail handles opportunistic STARTTLS if possible)
	return smtp.SendMail(addr, auth, cfg.Username, []string{toEmail}, body)
}

func (s *Service) sendOTPViaWAHA(ctx context.Context, phone, message string) error {
	configJSON, err := s.settings.GetGlobalSettingRaw(ctx, "notification_wa_waha")
	if err != nil || configJSON == "" {
		return fmt.Errorf("konfigurasi WAHA belum diatur")
	}
	var cfg struct {
		ApiUrl  string `json:"apiUrl"`
		ApiKey  string `json:"apiKey"`
		Session string `json:"session"`
	}
	if err := json.Unmarshal([]byte(configJSON), &cfg); err != nil || cfg.ApiUrl == "" {
		return fmt.Errorf("konfigurasi WAHA tidak valid")
	}
	
	session := cfg.Session
	if session == "" { session = "default" }
	
	// Sanitize phone and convert to international format
	cleanPhone := sanitizePhone(phone)
	
	// Ensure @c.us suffix
	chatId := cleanPhone
	if !strings.Contains(chatId, "@") {
		chatId = chatId + "@c.us"
	}

	apiUrl := strings.TrimSuffix(cfg.ApiUrl, "/")
	if !strings.Contains(apiUrl, "/api/") {
		apiUrl = apiUrl + "/api/sendText"
	}

	payload, _ := json.Marshal(map[string]any{
		"chatId":  chatId,
		"text":    message,
		"session": session,
	})

	req, _ := http.NewRequestWithContext(ctx, "POST", apiUrl, strings.NewReader(string(payload)))
	req.Header.Set("Content-Type", "application/json")
	if cfg.ApiKey != "" {
		req.Header.Set("Authorization", "Bearer "+cfg.ApiKey)
		req.Header.Set("X-Api-Key", cfg.ApiKey)
	}

	resp, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("WAHA API error (status %d): %s", resp.StatusCode, string(bodyBytes))
	}
	return nil
}

func (s *Service) sendOTPViaMeta(ctx context.Context, phone, message string) error {
	configJSON, err := s.settings.GetGlobalSettingRaw(ctx, "notification_wa")
	if err != nil || configJSON == "" { return fmt.Errorf("konfigurasi Meta WA belum diatur") }
	var cfg struct {
		ApiToken string `json:"apiToken"`
		PhoneID  string `json:"phoneId"`
	}
	json.Unmarshal([]byte(configJSON), &cfg)
	if cfg.ApiToken == "" || cfg.PhoneID == "" { return fmt.Errorf("konfigurasi Meta WA tidak valid") }

	cleanPhone := sanitizePhone(phone)
	apiUrl := fmt.Sprintf("https://graph.facebook.com/v17.0/%s/messages", cfg.PhoneID)
	
	// Note: Meta Official usually requires templates for business initiated. 
	// This is a simplified version using the free-form text if allowed.
	payload, _ := json.Marshal(map[string]any{
		"messaging_product": "whatsapp",
		"recipient_type":    "individual",
		"to":                cleanPhone,
		"type":              "text",
		"text":              map[string]string{"body": message},
	})

	req, _ := http.NewRequestWithContext(ctx, "POST", apiUrl, strings.NewReader(string(payload)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+cfg.ApiToken)

	resp, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
	if err != nil { return err }
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Meta API error (%d): %s", resp.StatusCode, string(body))
	}
	return nil
}

func (s *Service) sendOTPViaFonnte(ctx context.Context, phone, message string) error {
	configJSON, err := s.settings.GetGlobalSettingRaw(ctx, "notification_wa_fonnte")
	if err != nil || configJSON == "" { return fmt.Errorf("konfigurasi Fonnte belum diatur") }
	var cfg struct { ApiKey string `json:"apiKey" `}
	json.Unmarshal([]byte(configJSON), &cfg)
	if cfg.ApiKey == "" { return fmt.Errorf("konfigurasi Fonnte tidak valid") }

	cleanPhone := sanitizePhone(phone)
	
	// Fonnte uses form-data or JSON
	payload, _ := json.Marshal(map[string]any{
		"target":  cleanPhone,
		"message": message,
	})

	req, _ := http.NewRequestWithContext(ctx, "POST", "https://api.fonnte.com/send", strings.NewReader(string(payload)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", cfg.ApiKey)

	resp, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
	if err != nil { return err }
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Fonnte API error (%d): %s", resp.StatusCode, string(body))
	}
	return nil
}

func sanitizePhone(phone string) string {
	clean := ""
	for _, char := range phone {
		if char >= '0' && char <= '9' {
			clean += string(char)
		}
	}
	if strings.HasPrefix(clean, "08") {
		clean = "628" + strings.TrimPrefix(clean, "08")
	}
	return clean
}

func generateOTP(length int) (string, error) {
	const digits = "0123456789"
	result := make([]byte, length)
	for i := range result {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(digits))))
		result[i] = digits[n.Int64()]
	}
	return string(result), nil
}

func generateSessionToken() (string, error) {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b), nil
}
