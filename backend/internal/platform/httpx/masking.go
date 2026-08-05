package httpx

import (
	"encoding/json"
	"strings"

	"pekan/backend/internal/platform/security"
)

// MaskEmail menyamarkan alamat email (contoh: u***r@example.com)
func MaskEmail(email string) string {
	return security.MaskEmail(email)
}

// MaskPhone menyamarkan nomor telepon (contoh: 0812****5678)
func MaskPhone(phone string) string {
	return security.MaskPhone(phone)
}

// MaskString menyamarkan teks umum (contoh: AB****YZ)
func MaskString(s string) string {
	return security.MaskString(s, 2, 2)
}

// MaskJSON menyamarkan nilai sensitif dalam string JSON
func MaskJSON(rawJSON string) string {
	if rawJSON == "" || rawJSON == "null" || rawJSON == "{}" {
		return rawJSON
	}

	var data any
	if err := json.Unmarshal([]byte(rawJSON), &data); err != nil {
		return rawJSON
	}

	masked := maskValue(data)
	b, err := json.Marshal(masked)
	if err != nil {
		return rawJSON
	}
	return string(b)
}

func maskValue(v any) any {
	if v == nil {
		return nil
	}

	if m, ok := v.(map[string]any); ok {
		newMap := make(map[string]any)
		for k, val := range m {
			lowerK := strings.ToLower(k)
			if strings.Contains(lowerK, "password") ||
				strings.Contains(lowerK, "token") ||
				strings.Contains(lowerK, "secret") ||
				strings.Contains(lowerK, "key") ||
				strings.Contains(lowerK, "api_key") ||
				strings.Contains(lowerK, "auth") {
				newMap[k] = "[REDACTED]"
			} else if lowerK == "email" {
				if s, ok := val.(string); ok {
					newMap[k] = security.MaskEmail(s)
				} else {
					newMap[k] = val
				}
			} else if lowerK == "phone" || lowerK == "whatsapp" {
				if s, ok := val.(string); ok {
					newMap[k] = security.MaskPhone(s)
				} else {
					newMap[k] = val
				}
			} else {
				newMap[k] = maskValue(val)
			}
		}
		return newMap
	}

	if s, ok := v.([]any); ok {
		newSlice := make([]any, len(s))
		for i, item := range s {
			newSlice[i] = maskValue(item)
		}
		return newSlice
	}

	return v
}
