package middleware

import (
	"net/http"
	"strings"

	"pekan/backend/internal/platform/httpx"
)

var (
	defaultCORSMethods = "GET,POST,PUT,PATCH,DELETE,OPTIONS"
	defaultCORSHeaders = "Authorization,Content-Type,X-Tenant-ID,X-Request-ID"
)

func CORS(allowedOrigins []string) func(http.Handler) http.Handler {
	allowAll := false
	allowedMap := make(map[string]struct{})
	for _, origin := range allowedOrigins {
		trimmed := strings.TrimSpace(origin)
		if trimmed == "" {
			continue
		}
		if trimmed == "*" {
			allowAll = true
			continue
		}
		allowedMap[trimmed] = struct{}{}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := strings.TrimSpace(r.Header.Get("Origin"))
			if origin == "" {
				next.ServeHTTP(w, r)
				return
			}

			if !allowAll {
				if _, ok := allowedMap[origin]; !ok {
					if isPreflight(r) {
						httpx.WriteError(w, http.StatusForbidden, "CORS_ORIGIN_FORBIDDEN", "origin is not allowed", GetRequestID(r.Context()))
						return
					}
					next.ServeHTTP(w, r)
					return
				}
			}

			w.Header().Set("Vary", "Origin")
			if allowAll {
				w.Header().Set("Access-Control-Allow-Origin", "*")
			} else {
				w.Header().Set("Access-Control-Allow-Origin", origin)
			}
			w.Header().Set("Access-Control-Allow-Methods", defaultCORSMethods)
			w.Header().Set("Access-Control-Allow-Headers", defaultCORSHeaders)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Expose-Headers", "X-Request-ID")
			w.Header().Set("Access-Control-Max-Age", "600")

			if isPreflight(r) {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func isPreflight(r *http.Request) bool {
	return r.Method == http.MethodOptions && strings.TrimSpace(r.Header.Get("Access-Control-Request-Method")) != ""
}
