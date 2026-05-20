package audit

import "context"

type AuditContext struct {
	TenantID    string
	ActorUserID string
	RequestID   string
	IPAddress   string
	UserAgent   string
}

type contextKey string

const auditContextKey contextKey = "audit_context"

func WithContext(ctx context.Context, ac AuditContext) context.Context {
	return context.WithValue(ctx, auditContextKey, ac)
}

