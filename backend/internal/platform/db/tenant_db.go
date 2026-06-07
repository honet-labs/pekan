package db

import (
	"context"
	"database/sql"
	"fmt"
	"pekan/backend/internal/platform/tenancy"
)

// WithTenantTx executes a function within a transaction that has the search_path set to the tenant's schema.
// This is the SAFER way to handle separate schemas with sql.DB.
func WithTenantTx(ctx context.Context, db *sql.DB, fn func(tx *sql.Tx) error) error {
	tc, err := tenancy.FromContext(ctx)
	if err != nil {
		// If no tenant context (e.g. background job), use public schema
		return withSchema(ctx, db, "public", fn)
	}

	return withSchema(ctx, db, tc.SchemaName, fn)
}

func withSchema(ctx context.Context, db *sql.DB, schemaName string, fn func(tx *sql.Tx) error) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, fmt.Sprintf("SET LOCAL search_path TO \"%s\", public", schemaName))
	if err != nil {
		return fmt.Errorf("failed to set search_path: %w", err)
	}

	if err := fn(tx); err != nil {
		return err
	}

	return tx.Commit()
}
