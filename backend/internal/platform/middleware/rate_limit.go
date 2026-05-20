package middleware

import (
	"context"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"pekan/backend/internal/platform/audit"
	"pekan/backend/internal/platform/httpx"
)

type rateLimitEntry struct {
	Count   int
	ResetAt time.Time
}

type RateLimitStore interface {
	Allow(ctx context.Context, key string, limit int, window time.Duration) (allowed bool, retryAfter time.Duration, err error)
}

type MemoryRateLimitStore struct {
	mu      sync.Mutex
	entries map[string]rateLimitEntry
}

func NewMemoryRateLimitStore() *MemoryRateLimitStore {
	return &MemoryRateLimitStore{
		entries: make(map[string]rateLimitEntry),
	}
}

func (s *MemoryRateLimitStore) Allow(_ context.Context, key string, limit int, window time.Duration) (bool, time.Duration, error) {
	if limit <= 0 {
		limit = 20
	}
	if window <= 0 {
		window = time.Minute
	}

	now := time.Now().UTC()
	s.mu.Lock()
	defer s.mu.Unlock()

	entry, ok := s.entries[key]
	if !ok || now.After(entry.ResetAt) {
		s.entries[key] = rateLimitEntry{
			Count:   1,
			ResetAt: now.Add(window),
		}
		if len(s.entries) > 10000 {
			s.compact(now)
		}
		return true, 0, nil
	}

	if entry.Count >= limit {
		return false, time.Until(entry.ResetAt), nil
	}

	entry.Count++
	s.entries[key] = entry
	return true, 0, nil
}

func (s *MemoryRateLimitStore) compact(now time.Time) {
	for key, entry := range s.entries {
		if now.After(entry.ResetAt) {
			delete(s.entries, key)
		}
	}
}

type IPRateLimiter struct {
	limit  int
	window time.Duration
	store  RateLimitStore
	audit  audit.Logger
}

func NewIPRateLimiter(limit int, window time.Duration, logger audit.Logger) func(http.Handler) http.Handler {
	return NewIPRateLimiterWithStore(limit, window, NewMemoryRateLimitStore(), logger)
}

func NewIPRateLimiterWithStore(limit int, window time.Duration, store RateLimitStore, logger audit.Logger) func(http.Handler) http.Handler {
	if limit <= 0 {
		limit = 20
	}
	if window <= 0 {
		window = time.Minute
	}
	if store == nil {
		store = NewMemoryRateLimitStore()
	}

	limiter := &IPRateLimiter{
		limit:  limit,
		window: window,
		store:  store,
		audit:  logger,
	}
	return limiter.Middleware
}

func (l *IPRateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := ClientIP(r)
		allowed, retryAfter, err := l.store.Allow(r.Context(), key, l.limit, l.window)
		if err != nil {
			w.Header().Set("Retry-After", "1")
			httpx.WriteError(w, http.StatusTooManyRequests, "RATE_LIMIT_UNAVAILABLE", "rate limit service unavailable", GetRequestID(r.Context()))
			return
		}
		if !allowed {
			if retryAfter < time.Second {
				retryAfter = time.Second
			}
			w.Header().Set("Retry-After", strconv.FormatInt(int64(retryAfter.Seconds()), 10))
			
			if l.audit != nil {
				_ = l.audit.Write(r.Context(), "SYSTEM_RATE_LIMITED", "ip", key, nil, map[string]any{
					"limit":       l.limit,
					"window":      l.window.String(),
					"retry_after": retryAfter.String(),
					"path":        r.URL.Path,
				})
			}

			log.Printf("[RateLimit] REJECTED request from IP=%s for path=%s (limit=%d, window=%s)", key, r.URL.Path, l.limit, l.window)
			httpx.WriteError(w, http.StatusTooManyRequests, "RATE_LIMITED", "too many requests", GetRequestID(r.Context()))
			return
		}
		next.ServeHTTP(w, r)
	})
}

func ClientIP(r *http.Request) string {
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		parts := strings.Split(forwarded, ",")
		first := strings.TrimSpace(parts[0])
		if first != "" {
			return first
		}
	}

	if realIP := strings.TrimSpace(r.Header.Get("X-Real-IP")); realIP != "" {
		return realIP
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil && host != "" {
		return host
	}

	if strings.TrimSpace(r.RemoteAddr) == "" {
		return "unknown"
	}

	return r.RemoteAddr
}
