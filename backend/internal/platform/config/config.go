package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	AppEnv                    string
	HTTPPort                  string
	DatabaseURL               string
	DBMaxOpenConns            int
	DBMaxIdleConns            int
	DBConnMaxLifetime         time.Duration
	DBConnMaxIdleTime         time.Duration
	JWTSecret                 string
	ReceiptScanSecret         string
	ReceiptScanLegacySecrets  []string
	JWTIssuer                 string
	AccessTokenTTL            time.Duration
	RefreshTokenTTL           time.Duration
	CORSAllowedOrigins        []string
	RequestBodyMaxBytes       int64
	APIRateLimitPerMinute     int
	APIRateLimitWindowSeconds int
	APIRequestTimeout         time.Duration
	MaxHeaderBytes            int
	RateLimitRedisURL         string
	RateLimitRedisPrefix      string
	FileScanPollInterval      time.Duration
	FileScanMaxAttempts       int
	FileScanRetryDelay        time.Duration
	ReminderPollInterval      time.Duration
	StorageProvider           string
	StorageLocalPath          string
	StorageS3Bucket           string
	StorageS3Region           string
	StorageS3AccessKey        string
	StorageS3SecretKey        string
	StorageS3Endpoint         string
	StorageDriveFolder        string
	StorageGDriveCredentials  string
	AdminSecret               string
}

func Load() Config {
	return Config{
		AppEnv:                    getEnv("APP_ENV", "development"),
		HTTPPort:                  getEnv("HTTP_PORT", "8080"),
		DatabaseURL:               getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/pekan?sslmode=disable"),
		DBMaxOpenConns:            getInt("DB_MAX_OPEN_CONNS", 25),
		DBMaxIdleConns:            getInt("DB_MAX_IDLE_CONNS", 25),
		DBConnMaxLifetime:         time.Duration(getInt("DB_CONN_MAX_LIFETIME_MINUTES", 30)) * time.Minute,
		DBConnMaxIdleTime:         time.Duration(getInt("DB_CONN_MAX_IDLE_MINUTES", 5)) * time.Minute,
		JWTSecret:                 getEnv("JWT_SECRET", "change-me"),
		ReceiptScanSecret:         getEnv("RECEIPT_SCAN_SECRET", getEnv("JWT_SECRET", "change-me")),
		ReceiptScanLegacySecrets:  parseCSV(getEnv("RECEIPT_SCAN_LEGACY_SECRETS", "")),
		JWTIssuer:                 getEnv("JWT_ISSUER", "pekan-api"),
		AccessTokenTTL:            time.Duration(getInt("JWT_ACCESS_TTL_MINUTES", 15)) * time.Minute,
		RefreshTokenTTL:           time.Duration(getInt("JWT_REFRESH_TTL_HOURS", 720)) * time.Hour,
		CORSAllowedOrigins:        parseCSV(getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:5173,http://localhost:3000")),
		RequestBodyMaxBytes:       getInt64("REQUEST_BODY_MAX_BYTES", 1_048_576),
		APIRateLimitPerMinute:     getInt("API_RATE_LIMIT_PER_MINUTE", 2000),
		APIRateLimitWindowSeconds: getInt("API_RATE_LIMIT_WINDOW_SECONDS", 60),
		APIRequestTimeout:         time.Duration(getInt("API_REQUEST_TIMEOUT_SECONDS", 30)) * time.Second,
		MaxHeaderBytes:            getInt("MAX_HEADER_BYTES", 1_048_576),
		RateLimitRedisURL:         getEnv("RATE_LIMIT_REDIS_URL", ""),
		RateLimitRedisPrefix:      getEnv("RATE_LIMIT_REDIS_PREFIX", "pekan:ratelimit"),
		FileScanPollInterval:      time.Duration(getInt("FILE_SCAN_POLL_SECONDS", 5)) * time.Second,
		FileScanMaxAttempts:       getInt("FILE_SCAN_MAX_ATTEMPTS", 3),
		FileScanRetryDelay:        time.Duration(getInt("FILE_SCAN_RETRY_DELAY_SECONDS", 60)) * time.Second,
		ReminderPollInterval:      time.Duration(getInt("REMINDER_POLL_SECONDS", 60)) * time.Second,
		StorageProvider:           getEnv("STORAGE_PROVIDER", "local"),
		StorageLocalPath:          getEnv("STORAGE_LOCAL_PATH", "./data/storage"),
		StorageS3Bucket:           getEnv("STORAGE_S3_BUCKET", ""),
		StorageS3Region:           getEnv("STORAGE_S3_REGION", "us-east-1"),
		StorageS3AccessKey:        getEnv("STORAGE_S3_ACCESS_KEY", ""),
		StorageS3SecretKey:        getEnv("STORAGE_S3_SECRET_KEY", ""),
		StorageS3Endpoint:         getEnv("STORAGE_S3_ENDPOINT", ""),
		StorageDriveFolder:        getEnv("STORAGE_GDRIVE_FOLDER_ID", ""),
		StorageGDriveCredentials:  getEnv("STORAGE_GDRIVE_CREDENTIALS_JSON", ""),
		AdminSecret:               getEnv("ADMIN_SECRET", getEnv("JWT_SECRET", "change-me")),
	}
}

func (c Config) Validate() error {
	appEnv := strings.ToLower(strings.TrimSpace(c.AppEnv))

	if strings.TrimSpace(c.JWTSecret) == "" {
		return errors.New("JWT_SECRET is required")
	}
	if strings.TrimSpace(c.ReceiptScanSecret) == "" {
		c.ReceiptScanSecret = c.JWTSecret
	}

	if appEnv == "production" {
		if len(strings.TrimSpace(c.JWTSecret)) < 32 {
			return errors.New("JWT_SECRET must be at least 32 characters in production")
		}
		if c.JWTSecret == "change-me" || c.JWTSecret == "replace-with-strong-random-secret" {
			return errors.New("JWT_SECRET placeholder is not allowed in production")
		}
		if strings.Contains(strings.ToLower(c.DatabaseURL), "sslmode=disable") {
			return errors.New("DATABASE_URL must not use sslmode=disable in production")
		}
	}

	if strings.TrimSpace(c.JWTIssuer) == "" {
		return errors.New("JWT_ISSUER is required")
	}

	if c.AccessTokenTTL <= 0 {
		return fmt.Errorf("invalid JWT access token TTL: %s", c.AccessTokenTTL)
	}
	if c.RefreshTokenTTL <= 0 {
		return fmt.Errorf("invalid JWT refresh token TTL: %s", c.RefreshTokenTTL)
	}
	if c.RequestBodyMaxBytes <= 0 {
		return errors.New("REQUEST_BODY_MAX_BYTES must be > 0")
	}
	if c.DBMaxOpenConns < 0 || c.DBMaxIdleConns < 0 {
		return errors.New("DB_MAX_OPEN_CONNS and DB_MAX_IDLE_CONNS must be >= 0")
	}
	if c.DBConnMaxLifetime < 0 || c.DBConnMaxIdleTime < 0 {
		return errors.New("DB_CONN_MAX_LIFETIME_MINUTES and DB_CONN_MAX_IDLE_MINUTES must be >= 0")
	}
	if c.APIRateLimitPerMinute < 0 || c.APIRateLimitWindowSeconds < 0 {
		return errors.New("API_RATE_LIMIT_PER_MINUTE and API_RATE_LIMIT_WINDOW_SECONDS must be >= 0")
	}
	if c.APIRequestTimeout < 0 {
		return errors.New("API_REQUEST_TIMEOUT_SECONDS must be >= 0")
	}
	if c.MaxHeaderBytes <= 0 {
		return errors.New("MAX_HEADER_BYTES must be > 0")
	}
	if strings.TrimSpace(c.RateLimitRedisURL) != "" {
		parsedRedisURL, err := url.Parse(c.RateLimitRedisURL)
		if err != nil {
			return errors.New("RATE_LIMIT_REDIS_URL is invalid")
		}
		if parsedRedisURL.Scheme != "redis" && parsedRedisURL.Scheme != "rediss" {
			return errors.New("RATE_LIMIT_REDIS_URL scheme must be redis or rediss")
		}
	}
	if err := validateCORSOrigins(c.CORSAllowedOrigins, appEnv); err != nil {
		return err
	}
	if c.FileScanPollInterval <= 0 {
		return errors.New("FILE_SCAN_POLL_SECONDS must be > 0")
	}
	if c.FileScanMaxAttempts <= 0 {
		return errors.New("FILE_SCAN_MAX_ATTEMPTS must be > 0")
	}
	if c.FileScanRetryDelay <= 0 {
		return errors.New("FILE_SCAN_RETRY_DELAY_SECONDS must be > 0")
	}
	if c.ReminderPollInterval <= 0 {
		return errors.New("REMINDER_POLL_SECONDS must be > 0")
	}

	return nil
}

func getEnv(key, fallback string) string {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	return v
}

func getInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}

	out, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}

	return out
}

func getInt64(key string, fallback int64) int64 {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}

	out, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return fallback
	}

	return out
}

func parseCSV(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		v := strings.TrimSpace(part)
		if v == "" {
			continue
		}
		out = append(out, v)
	}
	return out
}

func validateCORSOrigins(origins []string, appEnv string) error {
	if appEnv == "production" && len(origins) == 0 {
		return errors.New("CORS_ALLOWED_ORIGINS is required in production")
	}

	for _, origin := range origins {
		if origin == "*" {
			if appEnv == "production" {
				return errors.New("CORS wildcard '*' is not allowed in production")
			}
			continue
		}

		parsed, err := url.Parse(origin)
		if err != nil {
			return fmt.Errorf("invalid CORS origin: %s", origin)
		}
		if parsed.Scheme != "http" && parsed.Scheme != "https" {
			return fmt.Errorf("CORS origin must use http/https: %s", origin)
		}
		if parsed.Host == "" {
			return fmt.Errorf("CORS origin host is required: %s", origin)
		}
	}

	return nil
}
