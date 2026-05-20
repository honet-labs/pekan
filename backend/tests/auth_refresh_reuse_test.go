package tests

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	authdomain "pekan/backend/internal/modules/core/auth/domain"
	authusecase "pekan/backend/internal/modules/core/auth/usecase"
	platformauth "pekan/backend/internal/platform/auth"
)

func TestRefreshTokenReuseRevokesAllSessions(t *testing.T) {
	t.Parallel()

	passwordHash, err := platformauth.HashPassword("Pekan#123")
	if err != nil {
		t.Fatalf("hash password error: %v", err)
	}

	repo := newInMemoryAuthRepo(passwordHash)
	jwtSvc := platformauth.NewService("pekan-test", "very-strong-test-secret-32chars-min", 15*time.Minute)
	svc := authusecase.NewService(repo, jwtSvc, 24*time.Hour, nil)

	loginOut, err := svc.Login(context.Background(), authusecase.LoginInput{
		Email:    "owner@pekan.local",
		Password: "Pekan#123",
		TenantID: "tenant-1",
	})
	if err != nil {
		t.Fatalf("login error: %v", err)
	}

	refreshOut, err := svc.Refresh(context.Background(), authusecase.RefreshInput{
		RefreshToken: loginOut.Tokens.RefreshToken,
	})
	if err != nil {
		t.Fatalf("first refresh error: %v", err)
	}
	if refreshOut.Tokens.RefreshToken == loginOut.Tokens.RefreshToken {
		t.Fatalf("refresh token was not rotated")
	}

	_, err = svc.Refresh(context.Background(), authusecase.RefreshInput{
		RefreshToken: loginOut.Tokens.RefreshToken,
	})
	if !errors.Is(err, authdomain.ErrRefreshTokenReused) {
		t.Fatalf("expected ErrRefreshTokenReused, got %v", err)
	}
	if !repo.revokeAllCalled {
		t.Fatalf("expected revoke all sessions to be called on token reuse")
	}
}

func TestLogoutAllRevokesSessionsByUser(t *testing.T) {
	t.Parallel()

	passwordHash, err := platformauth.HashPassword("Pekan#123")
	if err != nil {
		t.Fatalf("hash password error: %v", err)
	}

	repo := newInMemoryAuthRepo(passwordHash)
	jwtSvc := platformauth.NewService("pekan-test", "very-strong-test-secret-32chars-min", 15*time.Minute)
	svc := authusecase.NewService(repo, jwtSvc, 24*time.Hour, nil)

	if err := svc.LogoutAll(context.Background(), "user-1"); err != nil {
		t.Fatalf("logout all error: %v", err)
	}
	if !repo.revokeAllCalled {
		t.Fatalf("expected revoke all sessions to be called")
	}
}

type inMemoryAuthRepo struct {
	mu sync.Mutex

	user       authdomain.User
	membership authdomain.Membership
	access     authdomain.AccessProfile

	sessionsByID        map[string]authdomain.Session
	sessionsByTokenHash map[string]authdomain.Session
	revokeAllCalled     bool
}

func newInMemoryAuthRepo(passwordHash string) *inMemoryAuthRepo {
	return &inMemoryAuthRepo{
		user: authdomain.User{
			ID:           "user-1",
			Email:        "owner@pekan.local",
			PasswordHash: passwordHash,
			FullName:     "Owner",
			IsActive:     true,
		},
		membership: authdomain.Membership{
			ID:       "membership-1",
			TenantID: "tenant-1",
			UserID:   "user-1",
			Status:   "active",
		},
		access: authdomain.AccessProfile{
			Permissions: []string{"finance.transactions.read", "finance.transactions.create"},
			Features:    []string{"finance.transactions.read", "finance.transactions.write"},
			Modules:     []string{"finance"},
		},
		sessionsByID:        make(map[string]authdomain.Session),
		sessionsByTokenHash: make(map[string]authdomain.Session),
	}
}

func (r *inMemoryAuthRepo) GetUserByEmail(_ context.Context, email string) (authdomain.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if email != r.user.Email {
		return authdomain.User{}, authdomain.ErrInvalidCredentials
	}
	return r.user, nil
}

func (r *inMemoryAuthRepo) GetUserByID(_ context.Context, userID string) (authdomain.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if userID != r.user.ID {
		return authdomain.User{}, authdomain.ErrInvalidCredentials
	}
	return r.user, nil
}

func (r *inMemoryAuthRepo) GetUserProfile(_ context.Context, userID string) (authdomain.UserProfile, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if userID != r.user.ID {
		return authdomain.UserProfile{}, errors.New("user profile not found")
	}
	return authdomain.UserProfile{
		UserID:   userID,
		Username: "testuser",
	}, nil
}

func (r *inMemoryAuthRepo) GetUserByPhone(_ context.Context, phone string) (authdomain.User, error) {
	// Not implemented in mock as it's not used in these tests
	return authdomain.User{}, authdomain.ErrInvalidCredentials
}

func (r *inMemoryAuthRepo) ListTenantsByEmailOrPhone(_ context.Context, query string) ([]authdomain.TenantInfo, error) {
	return []authdomain.TenantInfo{}, nil
}

func (r *inMemoryAuthRepo) UpdateUserProfile(_ context.Context, profile authdomain.UserProfile) (authdomain.UserProfile, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return profile, nil
}

func (r *inMemoryAuthRepo) ListMembershipsByUserID(_ context.Context, userID string) ([]authdomain.Membership, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if userID != r.user.ID {
		return []authdomain.Membership{}, nil
	}
	return []authdomain.Membership{r.membership}, nil
}

func (r *inMemoryAuthRepo) GetMembershipByUserAndTenant(_ context.Context, userID, tenantID string) (authdomain.Membership, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if userID != r.membership.UserID || tenantID != r.membership.TenantID {
		return authdomain.Membership{}, authdomain.ErrMembershipNotFound
	}
	return r.membership, nil
}

func (r *inMemoryAuthRepo) GetTenantIDByCode(_ context.Context, code string) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	// In the test, we assume tenant-1 is the only tenant and its code is tenant-1
	if code == "tenant-1" {
		return "tenant-1", nil
	}
	return "", errors.New("tenant not found")
}

func (r *inMemoryAuthRepo) GetAccessProfileByMembership(_ context.Context, membershipID, tenantID string) (authdomain.AccessProfile, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if membershipID != r.membership.ID || tenantID != r.membership.TenantID {
		return authdomain.AccessProfile{}, authdomain.ErrAccessProfileMissing
	}
	return r.access, nil
}

func (r *inMemoryAuthRepo) CreateSession(_ context.Context, session authdomain.Session) (authdomain.Session, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	session.TokenExpiresAt = session.ExpiresAt
	r.sessionsByID[session.ID] = session
	r.sessionsByTokenHash[session.RefreshTokenHash] = session
	return session, nil
}

func (r *inMemoryAuthRepo) GetSessionByRefreshHash(_ context.Context, refreshHash string) (authdomain.Session, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	session, ok := r.sessionsByTokenHash[refreshHash]
	if !ok {
		return authdomain.Session{}, authdomain.ErrSessionNotFound
	}
	return session, nil
}

func (r *inMemoryAuthRepo) RotateSessionRefreshHash(_ context.Context, sessionID, currentRefreshHash, newRefreshHash string, expiresAt time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	session, ok := r.sessionsByTokenHash[currentRefreshHash]
	if !ok || session.ID != sessionID {
		return authdomain.ErrSessionNotFound
	}
	if session.TokenConsumedAt != nil || session.TokenRevokedAt != nil || session.RevokedAt != nil {
		return authdomain.ErrRefreshTokenReused
	}

	now := time.Now().UTC()
	session.TokenConsumedAt = &now
	r.sessionsByTokenHash[currentRefreshHash] = session

	next := session
	next.RefreshTokenHash = newRefreshHash
	next.ExpiresAt = expiresAt
	next.TokenExpiresAt = expiresAt
	next.TokenConsumedAt = nil
	next.TokenRevokedAt = nil

	r.sessionsByID[sessionID] = next
	r.sessionsByTokenHash[newRefreshHash] = next
	return nil
}

func (r *inMemoryAuthRepo) RevokeSessionByRefreshHash(_ context.Context, refreshHash string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	session, ok := r.sessionsByTokenHash[refreshHash]
	if !ok {
		return nil
	}
	now := time.Now().UTC()
	session.RevokedAt = &now
	session.TokenRevokedAt = &now
	r.sessionsByID[session.ID] = session
	r.sessionsByTokenHash[refreshHash] = session
	return nil
}

func (r *inMemoryAuthRepo) RevokeAllSessionsByUser(_ context.Context, userID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if userID != r.user.ID {
		return nil
	}
	r.revokeAllCalled = true
	now := time.Now().UTC()
	for tokenHash, session := range r.sessionsByTokenHash {
		session.RevokedAt = &now
		session.TokenRevokedAt = &now
		r.sessionsByTokenHash[tokenHash] = session
		r.sessionsByID[session.ID] = session
	}
	return nil
}

func (r *inMemoryAuthRepo) SaveRegistrationOTP(_ context.Context, otp authdomain.RegistrationOTP) error {
	return nil
}

func (r *inMemoryAuthRepo) GetRegistrationOTP(_ context.Context, sessionToken string) (authdomain.RegistrationOTP, error) {
	return authdomain.RegistrationOTP{}, errors.New("not implemented")
}

func (r *inMemoryAuthRepo) MarkOTPVerified(_ context.Context, sessionToken string) error {
	return nil
}

func (r *inMemoryAuthRepo) IncrementOTPAttempts(_ context.Context, sessionToken string) error {
	return nil
}

func (r *inMemoryAuthRepo) IsTenantCodeTaken(_ context.Context, code string) (bool, error) {
	return false, nil
}

func (r *inMemoryAuthRepo) IsEmailRegistered(_ context.Context, email string) (bool, error) {
	return false, nil
}

func (r *inMemoryAuthRepo) IsTenantNameTaken(_ context.Context, name string) (bool, error) {
	return false, nil
}

func (r *inMemoryAuthRepo) IsPhoneRegistered(_ context.Context, phone string) (bool, error) {
	return false, nil
}

func (r *inMemoryAuthRepo) IsUsernameTaken(_ context.Context, username string) (bool, error) {
	return false, nil
}

func (r *inMemoryAuthRepo) UpdateUserPassword(_ context.Context, userID, passwordHash string) error {
	return nil
}

func (r *inMemoryAuthRepo) GetTenantCodeByID(_ context.Context, id string) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if id == "tenant-1" {
		return "tenant-1", nil
	}
	return "", errors.New("tenant not found")
}
