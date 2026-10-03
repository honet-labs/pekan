package middleware

import (
	"bytes"
	"io"
	"net/http"
	"regexp"
	"strings"
)

var htmlTagRegex = regexp.MustCompile(`<[^>]*>`)

// SanitizeInput adalah middleware untuk membersihkan request body dari potensi XSS.
func SanitizeInput(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost || r.Method == http.MethodPut || r.Method == http.MethodPatch {
			if strings.Contains(r.Header.Get("Content-Type"), "application/json") {
				body, err := io.ReadAll(r.Body)
				if err == nil {
					clean := htmlTagRegex.ReplaceAll(body, []byte(""))
					r.Body = io.NopCloser(bytes.NewBuffer(clean))
				}
			}
		}
		next.ServeHTTP(w, r)
	})
}
