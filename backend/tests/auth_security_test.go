package tests

import (
	"testing"
	"time"

	platformauth "pekan/backend/internal/platform/auth"
)

func TestPasswordHashVerify(t *testing.T) {
	t.Parallel()

	hash, err := platformauth.HashPassword("Pekan#123")
	if err != nil {
		t.Fatalf("hash error: %v", err)
	}

	if err := platformauth.VerifyPassword(hash, "Pekan#123"); err != nil {
		t.Fatalf("verify should succeed: %v", err)
	}
	if err := platformauth.VerifyPassword(hash, "wrong-password"); err == nil {
		t.Fatalf("verify should fail for wrong password")
	}
}

func TestJWTIssueAndParse(t *testing.T) {
	t.Parallel()

	svc := platformauth.NewService("pekan-test", "secret", 15*time.Minute)
	raw, _, err := svc.IssueAccessToken(platformauth.IssueAccessTokenInput{
		UserID:      "user-1",
		Email:       "user@example.com",
		TenantID:    "tenant-1",
		SessionID:   "session-1",
		Permissions: []string{"finance.transactions.read"},
		Features:    []string{"finance.transactions.read"},
		Modules:     []string{"finance"},
	})
	if err != nil {
		t.Fatalf("issue token error: %v", err)
	}

	claims, err := svc.ParseAccessToken(raw)
	if err != nil {
		t.Fatalf("parse token error: %v", err)
	}
	if claims.UserID != "user-1" || claims.TenantID != "tenant-1" || claims.TokenType != "access" {
		t.Fatalf("unexpected claims parsed: %+v", claims)
	}
}

