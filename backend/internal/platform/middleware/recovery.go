package middleware

import (
	"log"
	"net/http"

	"pekan/backend/internal/platform/httpx"
)

func Recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				log.Printf("panic recovered request_id=%s panic=%v", GetRequestID(r.Context()), rec)
				httpx.WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error", GetRequestID(r.Context()))
			}
		}()
		next.ServeHTTP(w, r)
	})
}

