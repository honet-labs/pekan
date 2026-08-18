package middleware

import (
	"log/slog"
	"net/http"
	"os"
	"time"

	"pekan/backend/internal/platform/tenancy"
)

type responseWriter struct {
	http.ResponseWriter
	statusCode int
	bytes      int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	n, err := rw.ResponseWriter.Write(b)
	rw.bytes += n
	return n, err
}

func StructuredLogger(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			rw := &responseWriter{
				ResponseWriter: w,
				statusCode:     http.StatusOK,
			}

			next.ServeHTTP(rw, r)

			duration := time.Since(start)

			attrs := []slog.Attr{
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.String("query", r.URL.RawQuery),
				slog.Int("status", rw.statusCode),
				slog.Int("bytes", rw.bytes),
				slog.String("duration", duration.String()),
				slog.String("remote_addr", r.RemoteAddr),
				slog.String("user_agent", r.UserAgent()),
				slog.String("request_id", GetRequestID(r.Context())),
			}

			if tc, err := tenancy.FromContext(r.Context()); err == nil {
				attrs = append(attrs, slog.String("user_id", tc.UserID))
				if tc.TenantID != "" {
					attrs = append(attrs, slog.String("tenant_id", tc.TenantID))
				}
			}

			level := slog.LevelInfo
			if rw.statusCode >= 500 {
				level = slog.LevelError
			} else if rw.statusCode >= 400 {
				level = slog.LevelWarn
			}

			logger.LogAttrs(r.Context(), level, "HTTP request", attrs...)
		})
	}
}

func NewLogger() *slog.Logger {
	level := slog.LevelInfo
	if os.Getenv("APP_ENV") == "development" {
		level = slog.LevelDebug
	}

	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	}))
}
