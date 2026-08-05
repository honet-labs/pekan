package db

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// Migrator handles database migrations for both global and tenant schemas.
type Migrator struct {
	db *sql.DB
}

func NewMigrator(db *sql.DB) *Migrator {
	return &Migrator{db: db}
}

// MigrateTenantSchema creates a new schema and runs all migrations in the specified directory.
func (m *Migrator) MigrateTenantSchema(ctx context.Context, schemaName string, migrationsDir string) error {
	log.Printf("[Migrator] Starting migration for schema: %s", schemaName)

	// 1. Create Schema
	if _, err := m.db.ExecContext(ctx, fmt.Sprintf("CREATE SCHEMA IF NOT EXISTS \"%s\"", schemaName)); err != nil {
		return fmt.Errorf("failed to create schema %s: %w", schemaName, err)
	}

	// 2. Set search_path and run migrations
	// We use a transaction to ensure atomicity
	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Set search_path for the transaction
	if _, err := tx.ExecContext(ctx, fmt.Sprintf("SET LOCAL search_path TO \"%s\"", schemaName)); err != nil {
		return fmt.Errorf("failed to set search_path to %s: %w", schemaName, err)
	}

	// 3. Read migration files
	files, err := os.ReadDir(migrationsDir)
	if err != nil {
		return fmt.Errorf("failed to read migrations directory %s: %w", migrationsDir, err)
	}

	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".sql") {
			continue
		}

		filePath := filepath.Join(migrationsDir, file.Name())
		log.Printf("[Migrator] Applying tenant migration: %s", file.Name())
		
		content, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("failed to read migration file %s: %w", file.Name(), err)
		}

		if _, err := tx.ExecContext(ctx, string(content)); err != nil {
			return fmt.Errorf("failed to execute migration %s: %w", file.Name(), err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit migrations for schema %s: %w", schemaName, err)
	}

	log.Printf("[Migrator] Successfully migrated schema: %s", schemaName)
	return nil
}
