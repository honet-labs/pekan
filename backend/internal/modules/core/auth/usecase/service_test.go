package usecase

import (
	"context"
	"strings"
	"testing"
	"time"

	platformauth "pekan/backend/internal/platform/auth"
)

type mockSettings struct {
	GlobalSettingReader
}

func (m *mockSettings) GetGlobalSettingRaw(ctx context.Context, key string) (string, error) {
	return `{"host":"localhost","port":"25","username":"user","password":"pass"}`, nil
}

type mockNotifier struct{}

func (m *mockNotifier) SendOTP(ctx context.Context, method, email, phone, message string) error {
	return nil
}

func TestRegisterInit(t *testing.T) {
	repo := &MockRepository{}
	jwt := platformauth.NewService("pekan-test", "secret", time.Hour)
	service := NewService(repo, jwt, time.Hour*24, nil)
	service.WithDependencies(&mockSettings{}, nil, &mockNotifier{}, nil)


	ctx := context.Background()

	t.Run("Success Registration Init", func(t *testing.T) {
		repo.OnIsTenantCodeTaken = func(ctx context.Context, code string) (bool, error) { return false, nil }
		repo.OnIsTenantNameTaken = func(ctx context.Context, name string) (bool, error) { return false, nil }
		repo.OnIsEmailRegistered = func(ctx context.Context, email string) (bool, error) { return false, nil }
		repo.OnIsPhoneRegistered = func(ctx context.Context, phone string) (bool, error) { return false, nil }
		
		input := RegisterInitInput{
			TenantCode: "NEWTENANT",
			TenantName: "New Tenant Name",
			AdminEmail: "admin@example.com",
			AdminName:  "Admin User",
			Password:   "Password123!",


			Phone:      "62812345678",
			OTPMethod:  "email",
		}

		out, err := service.RegisterInit(ctx, input)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if out.SessionToken == "" {
			t.Error("expected session token to be generated")
		}
	})

	t.Run("Fail - Tenant Code Taken", func(t *testing.T) {
		repo.OnIsTenantCodeTaken = func(ctx context.Context, code string) (bool, error) { return true, nil }
		
		input := RegisterInitInput{
			TenantCode: "EXISTING",
			TenantName: "New Tenant",
			AdminEmail: "new@example.com",
			AdminName:  "New User",
			Password:   "Password123!",


		}

		_, err := service.RegisterInit(ctx, input)
		if err == nil || !strings.Contains(err.Error(), "sudah digunakan") {
			t.Errorf("expected error 'sudah digunakan', got %v", err)
		}
	})

	t.Run("Fail - Email Taken", func(t *testing.T) {
		repo.OnIsTenantCodeTaken = func(ctx context.Context, code string) (bool, error) { return false, nil }
		repo.OnIsEmailRegistered = func(ctx context.Context, email string) (bool, error) { return true, nil }
		
		input := RegisterInitInput{
			TenantCode: "NEWTENANT",
			TenantName: "New Tenant",
			AdminEmail: "existing@example.com",
			AdminName:  "User",
			Password:   "Password123!",


		}

		_, err := service.RegisterInit(ctx, input)
		if err == nil || !strings.Contains(err.Error(), "sudah terdaftar") {
			t.Errorf("expected error 'sudah terdaftar', got %v", err)
		}
	})

	t.Run("Fail - Phone Taken", func(t *testing.T) {
		repo.OnIsTenantCodeTaken = func(ctx context.Context, code string) (bool, error) { return false, nil }
		repo.OnIsEmailRegistered = func(ctx context.Context, email string) (bool, error) { return false, nil }
		repo.OnIsPhoneRegistered = func(ctx context.Context, phone string) (bool, error) { return true, nil }
		
		input := RegisterInitInput{
			TenantCode: "NEWTENANT",
			TenantName: "New Tenant",
			AdminEmail: "new@example.com",
			AdminName:  "User",
			Password:   "Password123!",


			Phone:      "62812345678",
		}

		_, err := service.RegisterInit(ctx, input)
		if err == nil || !strings.Contains(err.Error(), "nomor telepon") {
			t.Errorf("expected error containing 'nomor telepon', got %v", err)
		}
	})
}
