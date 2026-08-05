package tests

import (
	"context"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"

	"pekan/backend/internal/platform/entitlement"
)

func TestEntitlementResolverTenantAndOverride(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock new: %v", err)
	}
	defer db.Close()

	resolver := entitlement.NewPGResolver(db)

	// Tenant module toggles.
	mock.ExpectQuery(regexp.QuoteMeta(`
SELECT module_code, is_enabled
FROM public.tenant_modules
WHERE tenant_id = $1`)).
		WithArgs("tenant-a").
		WillReturnRows(sqlmock.NewRows([]string{"module_code", "is_enabled"}).
			AddRow("finance", true).
			AddRow("finance.transactions", true))

	// Legacy/manual tenant features.
	mock.ExpectQuery(regexp.QuoteMeta(`
SELECT tf.feature_code, COALESCE(m.code, ''), tf.is_enabled
FROM public.tenant_features tf
LEFT JOIN public.features f ON f.code = tf.feature_code
LEFT JOIN public.modules m ON m.id = f.module_id
WHERE tf.tenant_id = $1`)).
		WithArgs("tenant-a").
		WillReturnRows(sqlmock.NewRows([]string{"feature_code", "module_code", "is_enabled"}).
			AddRow("finance.transactions.read", "finance.transactions", true).
			AddRow("finance.transactions.write", "finance.transactions", true))

	// Tenant feature overrides disabling write.
	mock.ExpectQuery(regexp.QuoteMeta(`
SELECT f.code, m.code, tfo.is_enabled
FROM public.tenant_feature_overrides tfo
JOIN public.features f ON f.id = tfo.feature_id
JOIN public.modules m ON m.id = f.module_id
WHERE tfo.tenant_id = $1
  AND (tfo.expires_at IS NULL OR tfo.expires_at > now())`)).
		WithArgs("tenant-a").
		WillReturnRows(sqlmock.NewRows([]string{"feature_code", "module_code", "is_enabled"}).
			AddRow("finance.transactions.write", "finance.transactions", false))

	out, err := resolver.ResolveTenant(context.Background(), "tenant-a")
	if err != nil {
		t.Fatalf("resolve error: %v", err)
	}

	if contains(out.Features, "finance.transactions.write") {
		t.Fatalf("expected overridden feature to be disabled")
	}
	if !contains(out.Features, "finance.transactions.read") {
		t.Fatalf("expected read feature to remain enabled")
	}
	if !contains(out.Modules, "finance.transactions") {
		t.Fatalf("expected finance.transactions module to stay enabled")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet SQL expectations: %v", err)
	}
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
