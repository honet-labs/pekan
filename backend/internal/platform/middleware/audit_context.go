package middleware

import (
	"net/http"

	"pekan/backend/internal/platform/audit"
	"pekan/backend/internal/platform/tenancy"
)

func AuditContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tc, _ := tenancy.FromContext(r.Context())
		host := ClientIP(r)

		ctx := audit.WithContext(r.Context(), audit.AuditContext{
			TenantID:    tc.TenantID,
			ActorUserID: tc.UserID,
			RequestID:   GetRequestID(r.Context()),
			IPAddress:   host,
			UserAgent:   r.UserAgent(),
		})

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

