package middleware

import (
	"net/http"

	"pekan/backend/internal/platform/httpx"
	"pekan/backend/internal/platform/tenancy"
)

const tenantHeader = "X-Tenant-ID"

func Tenant(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tc, err := tenancy.FromContext(r.Context())
		if err != nil || tc.TenantID == "" {
			httpx.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "tenant context missing or invalid", GetRequestID(r.Context()))
			return
		}

		headerTenant := r.Header.Get(tenantHeader)
		if headerTenant != "" && headerTenant != tc.TenantID {
			httpx.WriteError(w, http.StatusForbidden, "FORBIDDEN_TENANT_MISMATCH", "token tenant and header tenant mismatch", GetRequestID(r.Context()))
			return
		}

		next.ServeHTTP(w, r)
	})
}

