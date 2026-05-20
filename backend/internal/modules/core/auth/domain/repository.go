package domain

import (
	"context"
	"time"
)

type Repository interface {
	GetUserByEmail(ctx context.Context, email string) (User, error)
	GetUserByID(ctx context.Context, userID string) (User, error)
	GetUserProfile(ctx context.Context, userID string) (UserProfile, error)
	UpdateUserProfile(ctx context.Context, profile UserProfile) (UserProfile, error)
	ListMembershipsByUserID(ctx context.Context, userID string) ([]Membership, error)
	GetMembershipByUserAndTenant(ctx context.Context, userID, tenantID string) (Membership, error)
	GetTenantIDByCode(ctx context.Context, code string) (string, error)
	GetUserByPhone(ctx context.Context, phone string) (User, error)
	ListTenantsByEmailOrPhone(ctx context.Context, query string) ([]TenantInfo, error)
	GetTenantCodeByID(ctx context.Context, id string) (string, error)

	GetAccessProfileByMembership(ctx context.Context, membershipID, tenantID string) (AccessProfile, error)

	CreateSession(ctx context.Context, session Session) (Session, error)
	GetSessionByRefreshHash(ctx context.Context, refreshHash string) (Session, error)
	RotateSessionRefreshHash(ctx context.Context, sessionID, currentRefreshHash, newRefreshHash string, expiresAt time.Time) error
	RevokeSessionByRefreshHash(ctx context.Context, refreshHash string) error
	RevokeAllSessionsByUser(ctx context.Context, userID string) error

	// Registration OTP
	SaveRegistrationOTP(ctx context.Context, otp RegistrationOTP) error
	GetRegistrationOTP(ctx context.Context, sessionToken string) (RegistrationOTP, error)
	MarkOTPVerified(ctx context.Context, sessionToken string) error
	IncrementOTPAttempts(ctx context.Context, sessionToken string) error
	IsTenantCodeTaken(ctx context.Context, code string) (bool, error)
	IsTenantNameTaken(ctx context.Context, name string) (bool, error)
	IsEmailRegistered(ctx context.Context, email string) (bool, error)
	IsPhoneRegistered(ctx context.Context, phone string) (bool, error)
	IsUsernameTaken(ctx context.Context, username string) (bool, error)
	UpdateUserPassword(ctx context.Context, userID, passwordHash string) error
}
