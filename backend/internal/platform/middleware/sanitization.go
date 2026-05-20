package middleware

import (
	"bytes"
	"io"
	"net/http"
	"strings"

	"github.com/microcosm-cc/bluemonday"
)

// SanitizeInput adalah middleware untuk membersihkan request body dari potensi XSS.
func SanitizeInput(next http.Handler) http.Handler {
	policy := bluemonday.UGCPolicy()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost || r.Method == http.MethodPut || r.Method == http.MethodPatch {
			if strings.Contains(r.Header.Get("Content-Type"), "application/json") {
				body, err := io.ReadAll(r.Body)
				if err == nil {
					// Bersihkan body
					sanitizedBody := policy.SanitizeBytes(body)
					r.Body = io.NopCloser(bytes.NewBuffer(sanitizedBody))
				}
			}
		}
		next.ServeHTTP(w, r)
	})
}
