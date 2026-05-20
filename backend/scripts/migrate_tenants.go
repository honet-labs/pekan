package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"pekan/backend/internal/platform/db"
	"pekan/backend/internal/platform/tenancy"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func main() {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL environment variable is required")
	}

	// Use absolute path for migrations if possible, or relative to current dir
	migrationsPath := "./migrations/tenant"
	if _, err := os.Stat(migrationsPath); os.IsNotExist(err) {
		// Try one level up if run from scripts/
		migrationsPath = "../migrations/tenant"
		if _, err := os.Stat(migrationsPath); os.IsNotExist(err) {
			log.Fatal("Tenant migrations directory not found at ./migrations/tenant or ../migrations/tenant")
		}
	}
	absPath, _ := filepath.Abs(migrationsPath)
	log.Printf("Using migrations from: %s", absPath)

	conn, err := sql.Open("pgx", dbURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer conn.Close()

	if err := conn.Ping(); err != nil {
		log.Fatalf("database is unreachable: %v", err)
	}

	ctx := context.Background()
	migrator := db.NewMigrator(conn)

	// 1. Fetch all active tenants from public schema
	rows, err := conn.QueryContext(ctx, "SELECT code FROM public.tenants WHERE status != 'deleted'")
	if err != nil {
		log.Fatalf("failed to query tenants: %v", err)
	}
	defer rows.Close()

	successCount := 0
	failCount := 0
	tenantCodes := []string{}

	for rows.Next() {
		var code string
		if err := rows.Scan(&code); err != nil {
			log.Printf("[ERROR] Failed to scan tenant row: %v", err)
			continue
		}
		tenantCodes = append(tenantCodes, code)
	}

	if len(tenantCodes) == 0 {
		log.Println("No tenants found to migrate.")
		return
	}

	log.Printf("Found %d tenants to migrate.", len(tenantCodes))

	for _, code := range tenantCodes {
		schemaName := tenancy.GetSchemaName(code)
		fmt.Printf("--------------------------------------------------\n")
		log.Printf("[...] Migrating tenant: %s (schema: %s)", code, schemaName)
		
		if err := migrator.MigrateTenantSchema(ctx, schemaName, migrationsPath); err != nil {
			log.Printf("[FAIL] Tenant %s migration failed: %v", code, err)
			failCount++
		} else {
			log.Printf("[DONE] Tenant %s migrated successfully", code)
			successCount++
		}
	}

	fmt.Printf("--------------------------------------------------\n")
	log.Printf("Migration finished. Success: %d, Failed: %d", successCount, failCount)
	
	if failCount > 0 {
		os.Exit(1)
	}
}
