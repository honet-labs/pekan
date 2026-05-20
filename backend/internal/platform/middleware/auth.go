package middleware

import (
	"net/http"
	"strings"

	"pekan/backend/internal/platform/auth"
	"pekan/backend/internal/platform/httpx"
	"pekan/backend/internal/platform/session"
	"pekan/backend/internal/platform/tenancy"
)

func Auth(jwtService *auth.Service, sessionStore session.Store) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var token string
			
			// Extract token from HttpOnly cookie first
			if cookie, err := r.Cookie("pekan_access_token"); err == nil {
				token = cookie.Value
			}
			
			// Fallback to Bearer Header
			if token == "" {
				authHeader := r.Header.Get("Authorization")
				if strings.HasPrefix(authHeader, "Bearer ") {
					token = strings.TrimPrefix(authHeader, "Bearer ")
				}
			}

			if token == "" {
				httpx.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing access token", GetRequestID(r.Context()))
				return
			}

			claims, err := jwtService.ParseAccessToken(token)
			if err != nil {
				httpx.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "invalid access token", GetRequestID(r.Context()))
				return
			}
			if claims.TokenType != "" && claims.TokenType != "access" {
				httpx.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "token type is not access", GetRequestID(r.Context()))
				return
			}

			if sessionStore != nil {
				revoked, err := sessionStore.IsTokenRevoked(r.Context(), claims.SessionID)
				if err == nil && revoked {
					httpx.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "session has been revoked", GetRequestID(r.Context()))
					return
				}
			}

			ctx := tenancy.WithContext(r.Context(), tenancy.Context{
				UserID:      claims.UserID,
				TenantID:    claims.TenantID,
				SchemaName:  tenancy.GetSchemaName(claims.TenantCode),
				Email:       claims.Email,
				Permissions: toSet(claims.Permissions),
				Features:    toSet(claims.Features),
				Modules:     toSet(claims.Modules),
			})

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func toSet(values []string) map[string]struct{} {
	out := make(map[string]struct{}, len(values))
	for _, v := range values {
		out[v] = struct{}{}
	}
	return out
}
