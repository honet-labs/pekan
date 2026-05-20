package tests

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"pekan/backend/internal/platform/middleware"
)

func TestIPRateLimiterBlocksAfterLimit(t *testing.T) {
	t.Parallel()

	limiter := middleware.NewIPRateLimiter(2, time.Minute, nil)
	protected := limiter(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	for i := 1; i <= 3; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", nil)
		req.RemoteAddr = "192.168.10.10:12345"
		rec := httptest.NewRecorder()

		protected.ServeHTTP(rec, req)

		if i < 3 && rec.Code != http.StatusOK {
			t.Fatalf("request #%d expected 200, got %d", i, rec.Code)
		}
		if i == 3 && rec.Code != http.StatusTooManyRequests {
			t.Fatalf("request #%d expected 429, got %d", i, rec.Code)
		}
	}
}

func TestIPRateLimiterStoreUnavailableReturns429(t *testing.T) {
	t.Parallel()

	limiter := middleware.NewIPRateLimiterWithStore(10, time.Minute, failingRateLimitStore{}, nil)
	protected := limiter(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", nil)
	req.RemoteAddr = "192.168.10.11:12345"
	rec := httptest.NewRecorder()
	protected.ServeHTTP(rec, req)

	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429 when store is unavailable, got %d", rec.Code)
	}
}

type failingRateLimitStore struct{}

func (failingRateLimitStore) Allow(_ context.Context, _ string, _ int, _ time.Duration) (bool, time.Duration, error) {
	return false, 0, errors.New("rate limit store down")
}
