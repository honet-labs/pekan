package usecase

import (
	"context"

	authdomain "pekan/backend/internal/modules/core/auth/domain"
)

type MockRepository struct {
	authdomain.Repository
	OnGetUserByEmail                func(ctx context.Context, email string) (authdomain.User, error)
	OnIsTenantCodeTaken            func(ctx context.Context, code string) (bool, error)
	OnIsTenantNameTaken            func(ctx context.Context, name string) (bool, error)
	OnIsEmailRegistered            func(ctx context.Context, email string) (bool, error)
	OnIsPhoneRegistered            func(ctx context.Context, phone string) (bool, error)
	OnSaveRegistrationOTP           func(ctx context.Context, otp authdomain.RegistrationOTP) error
	OnGetRegistrationOTP            func(ctx context.Context, sessionToken string) (authdomain.RegistrationOTP, error)
}

func (m *MockRepository) GetUserByEmail(ctx context.Context, email string) (authdomain.User, error) {
	if m.OnGetUserByEmail != nil {
		return m.OnGetUserByEmail(ctx, email)
	}
	return authdomain.User{}, authdomain.ErrInvalidCredentials
}

func (m *MockRepository) IsTenantCodeTaken(ctx context.Context, code string) (bool, error) {
	if m.OnIsTenantCodeTaken != nil {
		return m.OnIsTenantCodeTaken(ctx, code)
	}
	return false, nil
}

func (m *MockRepository) IsTenantNameTaken(ctx context.Context, name string) (bool, error) {
	if m.OnIsTenantNameTaken != nil {
		return m.OnIsTenantNameTaken(ctx, name)
	}
	return false, nil
}

func (m *MockRepository) IsEmailRegistered(ctx context.Context, email string) (bool, error) {
	if m.OnIsEmailRegistered != nil {
		return m.OnIsEmailRegistered(ctx, email)
	}
	return false, nil
}

func (m *MockRepository) IsPhoneRegistered(ctx context.Context, phone string) (bool, error) {
	if m.OnIsPhoneRegistered != nil {
		return m.OnIsPhoneRegistered(ctx, phone)
	}
	return false, nil
}

func (m *MockRepository) SaveRegistrationOTP(ctx context.Context, otp authdomain.RegistrationOTP) error {
	if m.OnSaveRegistrationOTP != nil {
		return m.OnSaveRegistrationOTP(ctx, otp)
	}
	return nil
}

func (m *MockRepository) GetRegistrationOTP(ctx context.Context, sessionToken string) (authdomain.RegistrationOTP, error) {
	if m.OnGetRegistrationOTP != nil {
		return m.OnGetRegistrationOTP(ctx, sessionToken)
	}
	return authdomain.RegistrationOTP{}, authdomain.ErrSessionNotFound
}
