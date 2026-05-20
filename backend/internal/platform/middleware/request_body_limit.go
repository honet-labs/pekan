package middleware

import (
	"net/http"
	"strings"

	"pekan/backend/internal/platform/httpx"
)

func RequestBodyLimit(maxBytes int64) func(http.Handler) http.Handler {
	if maxBytes <= 0 {
		maxBytes = 1_048_576
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if shouldSkipBodyLimit(r) {
				next.ServeHTTP(w, r)
				return
			}

			if r.ContentLength > maxBytes {
				httpx.WriteError(
					w,
					http.StatusRequestEntityTooLarge,
					"REQUEST_BODY_TOO_LARGE",
					"request body exceeds limit",
					GetRequestID(r.Context()),
				)
				return
			}

			r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
			next.ServeHTTP(w, r)
		})
	}
}

func shouldSkipBodyLimit(r *http.Request) bool {
	switch r.Method {
	case http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodTrace:
		return true
	}

	contentType := strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type")))
	if strings.HasPrefix(contentType, "multipart/form-data") {
		return true
	}

	return false
}
