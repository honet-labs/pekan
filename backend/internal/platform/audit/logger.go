package audit

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"
)

type Logger interface {
	Write(ctx context.Context, action, resourceType, resourceID string, before, after any) error
}

type DBLogger struct {
	db *sql.DB
}

func NewDBLogger(db *sql.DB) *DBLogger {
	return &DBLogger{db: db}
}

func (l *DBLogger) Write(ctx context.Context, action, resourceType, resourceID string, before, after any) error {
	tc, ok := ctx.Value(auditContextKey).(AuditContext)
	if !ok {
		tc = AuditContext{}
	}

	// Mask sensitive data before marshaling
	maskedBefore := maskSensitive(before)
	maskedAfter := maskSensitive(after)

	beforeJSON, _ := marshalJSON(maskedBefore)
	afterJSON, _ := marshalJSON(maskedAfter)

	const q = `
INSERT INTO audit_logs (
  tenant_id, actor_user_id, action, resource_type, resource_id, before_json, after_json, request_id, ip_address, user_agent, created_at
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,now())`

	_, err := l.db.ExecContext(ctx, q,
		nullIfEmpty(tc.TenantID),
		nullIfEmpty(tc.ActorUserID),
		action,
		resourceType,
		resourceID,
		beforeJSON,
		afterJSON,
		nullIfEmpty(tc.RequestID),
		nullIfEmpty(tc.IPAddress),
		nullIfEmpty(tc.UserAgent),
	)
	return err
}

func maskSensitive(v any) any {
	if v == nil {
		return nil
	}

	// If it's not a map or slice (e.g. a struct), marshal and unmarshal it to a map first
	// so we can generically iterate over its keys without reflection.
	switch v.(type) {
	case map[string]any, []any:
		// already generic, proceed
	default:
		b, err := json.Marshal(v)
		if err == nil {
			var generic any
			if err := json.Unmarshal(b, &generic); err == nil {
				v = generic
			}
		}
	}

	return maskValue(v)
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
				strings.Contains(lowerK, "phone") ||
				strings.Contains(lowerK, "email") {
				newMap[k] = "[REDACTED]"
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

func marshalJSON(v any) ([]byte, error) {
	if v == nil {
		return nil, nil
	}
	return json.Marshal(v)
}

func nullIfEmpty(v string) any {
	if v == "" {
		return nil
	}
	return v
}

