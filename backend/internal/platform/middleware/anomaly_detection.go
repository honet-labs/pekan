package middleware

import (
	"net/http"
	"strings"
	"sync"
	"time"

	"pekan/backend/internal/platform/audit"
)

type anomalyTracker struct {
	mu           sync.Mutex
	failureCount map[string]int // IP -> count
	lastReset    time.Time
}

var tracker = &anomalyTracker{
	failureCount: make(map[string]int),
	lastReset:    time.Now(),
}

func AnomalyDetection(auditLogger audit.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Basic reset logic every hour
			tracker.mu.Lock()
			if time.Since(tracker.lastReset) > 1*time.Hour {
				tracker.failureCount = make(map[string]int)
				tracker.lastReset = time.Now()
			}
			tracker.mu.Unlock()

			// Wrap response writer to capture status code
			ww := &statusWriter{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(ww, r)

			// Detect Anomalies
			ip := ClientIP(r)
			if ww.status == http.StatusUnauthorized || ww.status == http.StatusForbidden {
				tracker.mu.Lock()
				tracker.failureCount[ip]++
				count := tracker.failureCount[ip]
				tracker.mu.Unlock()

				if count >= 10 {
					_ = auditLogger.Write(r.Context(), "SECURITY_ANOMALY", "ip_address", ip, nil, map[string]any{
						"reason": "high_auth_failure_count",
						"count":  count,
						"path":   r.URL.Path,
						"method": r.Method,
					})
				}
			}

			// Detect Large Payloads to non-attachment endpoints
			if r.ContentLength > 1*1024*1024 && !strings.Contains(r.URL.Path, "attachments") && !strings.Contains(r.URL.Path, "receipts") {
				_ = auditLogger.Write(r.Context(), "SECURITY_ANOMALY", "ip_address", ip, nil, map[string]any{
					"reason": "large_payload_suspicious",
					"size":   r.ContentLength,
					"path":   r.URL.Path,
				})
			}
		})
	}
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}
