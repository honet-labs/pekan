package security

import (
	"strings"
	"github.com/microcosm-cc/bluemonday"
)

var policy = bluemonday.UGCPolicy()

// SanitizeHTML removes potentially malicious HTML from a string
func SanitizeHTML(s string) string {
	if s == "" {
		return ""
	}
	return policy.Sanitize(s)
}

// MaskEmail masks an email address: a*******@example.com
func MaskEmail(email string) string {
	if email == "" {
		return ""
	}
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return email // Not a valid email
	}
	user := parts[0]
	domain := parts[1]

	if len(user) <= 1 {
		return "*" + "@" + domain
	}
	if len(user) <= 2 {
		return string(user[0]) + "*" + "@" + domain
	}

	maskedUser := string(user[0]) + strings.Repeat("*", 7)
	return maskedUser + "@" + domain
}

// MaskPhone masks a phone number: 62812*******42
func MaskPhone(phone string) string {
	if phone == "" {
		return ""
	}
	// Remove non-digits for masking logic
	clean := ""
	for _, c := range phone {
		if c >= '0' && c <= '9' {
			clean += string(c)
		}
	}

	if len(clean) <= 6 {
		return strings.Repeat("*", len(clean))
	}

	return clean[:5] + strings.Repeat("*", 7) + clean[len(clean)-2:]
}

// MaskString masks a string by keeping only first and last N characters
func MaskString(s string, keepFirst, keepLast int) string {
	if s == "" {
		return ""
	}
	if len(s) <= keepFirst+keepLast {
		return strings.Repeat("*", len(s))
	}
	return s[:keepFirst] + strings.Repeat("*", 7) + s[len(s)-keepLast:]
}
