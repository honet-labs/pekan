package tests

import (
	"testing"
	"time"

	"pekan/backend/internal/platform/config"
)

func TestConfigValidateProductionRejectsWeakSecurity(t *testing.T) {
	t.Parallel()

	cfg := config.Config{
		AppEnv:          "production",
		DatabaseURL:     "postgres://postgres:postgres@localhost:5432/pekan?sslmode=disable",
		JWTSecret:       "change-me",
		JWTIssuer:       "pekan-api",
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: 24 * time.Hour,
		CORSAllowedOrigins: []string{"https://app.pekan.local"},
		RequestBodyMaxBytes: 1_048_576,
		MaxHeaderBytes: 1_048_576,
		FileScanPollInterval: 5 * time.Second,
		FileScanMaxAttempts: 3,
		FileScanRetryDelay: 60 * time.Second,
		ReminderPollInterval: 60 * time.Second,
	}

	if err := cfg.Validate(); err == nil {
		t.Fatalf("expected validation error for insecure production config")
	}
}

func TestConfigValidateProductionAcceptsSecureSetup(t *testing.T) {
	t.Parallel()

	cfg := config.Config{
		AppEnv:          "production",
		DatabaseURL:     "postgres://postgres:postgres@db:5432/pekan?sslmode=require",
		JWTSecret:       "this-is-a-very-strong-jwt-secret-for-production",
		JWTIssuer:       "pekan-api",
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: 24 * time.Hour,
		CORSAllowedOrigins: []string{"https://app.pekan.local"},
		RequestBodyMaxBytes: 1_048_576,
		MaxHeaderBytes: 1_048_576,
		FileScanPollInterval: 5 * time.Second,
		FileScanMaxAttempts: 3,
		FileScanRetryDelay: 60 * time.Second,
		ReminderPollInterval: 60 * time.Second,
	}

	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected secure config to be valid, got=%v", err)
	}
}

func TestConfigValidateProductionRejectsWildcardCORS(t *testing.T) {
	t.Parallel()

	cfg := config.Config{
		AppEnv:             "production",
		DatabaseURL:        "postgres://postgres:postgres@db:5432/pekan?sslmode=require",
		JWTSecret:          "this-is-a-very-strong-jwt-secret-for-production",
		JWTIssuer:          "pekan-api",
		AccessTokenTTL:     15 * time.Minute,
		RefreshTokenTTL:    24 * time.Hour,
		CORSAllowedOrigins: []string{"*"},
		RequestBodyMaxBytes: 1_048_576,
		MaxHeaderBytes: 1_048_576,
		FileScanPollInterval: 5 * time.Second,
		FileScanMaxAttempts: 3,
		FileScanRetryDelay: 60 * time.Second,
		ReminderPollInterval: 60 * time.Second,
	}

	if err := cfg.Validate(); err == nil {
		t.Fatalf("expected validation error for wildcard CORS in production")
	}
}

func TestConfigValidateRejectsInvalidBodyLimit(t *testing.T) {
	t.Parallel()

	cfg := config.Config{
		AppEnv:             "development",
		DatabaseURL:        "postgres://postgres:postgres@localhost:5432/pekan?sslmode=disable",
		JWTSecret:          "local-secret",
		JWTIssuer:          "pekan-api",
		AccessTokenTTL:     15 * time.Minute,
		RefreshTokenTTL:    24 * time.Hour,
		CORSAllowedOrigins: []string{"http://localhost:5173"},
		RequestBodyMaxBytes: 0,
		MaxHeaderBytes: 1_048_576,
		FileScanPollInterval: 5 * time.Second,
		FileScanMaxAttempts: 3,
		FileScanRetryDelay: 60 * time.Second,
		ReminderPollInterval: 60 * time.Second,
	}

	if err := cfg.Validate(); err == nil {
		t.Fatalf("expected validation error for invalid body limit")
	}
}

func TestConfigValidateRejectsInvalidRateLimitRedisURL(t *testing.T) {
	t.Parallel()

	cfg := config.Config{
		AppEnv:               "development",
		DatabaseURL:          "postgres://postgres:postgres@localhost:5432/pekan?sslmode=disable",
		JWTSecret:            "local-secret",
		JWTIssuer:            "pekan-api",
		AccessTokenTTL:       15 * time.Minute,
		RefreshTokenTTL:      24 * time.Hour,
		CORSAllowedOrigins:   []string{"http://localhost:5173"},
		RequestBodyMaxBytes:  1_048_576,
		MaxHeaderBytes: 1_048_576,
		RateLimitRedisURL:    "http://localhost:6379",
		FileScanPollInterval: 5 * time.Second,
		FileScanMaxAttempts:  3,
		FileScanRetryDelay:   60 * time.Second,
		ReminderPollInterval: 60 * time.Second,
	}

	if err := cfg.Validate(); err == nil {
		t.Fatalf("expected validation error for invalid RATE_LIMIT_REDIS_URL scheme")
	}
}
