package tenancy

import (
	"context"
	"errors"
)

type Context struct {
	UserID      string
	TenantID    string
	SchemaName  string
	Email       string
	Permissions map[string]struct{}
	Features    map[string]struct{}
	Modules     map[string]struct{}
}

type contextKey string

const tenantContextKey contextKey = "tenant_context"

func WithContext(ctx context.Context, tc Context) context.Context {
	return context.WithValue(ctx, tenantContextKey, tc)
}

func FromContext(ctx context.Context) (Context, error) {
	v := ctx.Value(tenantContextKey)
	if v == nil {
		return Context{}, errors.New("tenant context not found")
	}

	tc, ok := v.(Context)
	if !ok {
		return Context{}, errors.New("invalid tenant context type")
	}

	return tc, nil
}

func MustFromContext(ctx context.Context) Context {
	tc, err := FromContext(ctx)
	if err != nil {
		panic(err)
	}
	return tc
}

