package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func main() {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL is not set")
	}

	db, err := sql.Open("pgx", dbURL)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	email := "admin@honet.web.id"
	tenantCode := "pekanhonet"

	fmt.Printf("--- Checking User: %s ---\n", email)
	var userID, pwdHash string
	var isActive, mustChange bool
	err = db.QueryRow("SELECT id, password_hash, is_active, must_change_password FROM public.users WHERE email = $1", email).Scan(&userID, &pwdHash, &isActive, &mustChange)
	if err != nil {
		fmt.Printf("Error finding user: %v\n", err)
	} else {
		fmt.Printf("User ID: %s\n", userID)
		fmt.Printf("Is Active: %v\n", isActive)
		fmt.Printf("Must Change Password: %v\n", mustChange)
		fmt.Printf("Password Hash (Prefix): %s...\n", pwdHash[:10])
	}

	fmt.Printf("\n--- Checking Tenant: %s ---\n", tenantCode)
	var tenantID string
	err = db.QueryRow("SELECT id FROM public.tenants WHERE code = $1", tenantCode).Scan(&tenantID)
	if err != nil {
		fmt.Printf("Error finding tenant: %v\n", err)
	} else {
		fmt.Printf("Tenant ID: %s\n", tenantID)
	}

	if userID != "" && tenantID != "" {
		fmt.Printf("\n--- Checking Membership ---\n")
		var membershipID, status string
		err = db.QueryRow("SELECT id, status FROM public.tenant_memberships WHERE user_id = $1 AND tenant_id = $2", userID, tenantID).Scan(&membershipID, &status)
		if err != nil {
			fmt.Printf("Error finding membership: %v\n", err)
		} else {
			fmt.Printf("Membership ID: %s\n", membershipID)
			fmt.Printf("Status: %s\n", status)
		}
	}

	// Test a specific password if provided via args
	if len(os.Args) > 1 && pwdHash != "" {
		testPwd := os.Args[1]
		fmt.Printf("\n--- Testing Password Verification ---\n")
		fmt.Printf("Raw Hash from DB: %s (input: %s)\n", pwdHash, testPwd)
	}
}
