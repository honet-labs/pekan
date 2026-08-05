package tests

import (
	"testing"

	"pekan/backend/internal/platform/middleware"
)

func TestNewRedisRateLimitStoreRequiresURL(t *testing.T) {
	t.Parallel()

	_, err := middleware.NewRedisRateLimitStore("", "pekan:ratelimit", nil)
	if err == nil {
		t.Fatalf("expected error when redis URL is empty")
	}
}

func TestNewRedisRateLimitStoreRejectsInvalidURL(t *testing.T) {
	t.Parallel()

	_, err := middleware.NewRedisRateLimitStore("not-a-redis-url", "pekan:ratelimit", nil)
	if err == nil {
		t.Fatalf("expected error for invalid redis URL")
	}
}
