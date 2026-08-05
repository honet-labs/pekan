package session

import (
	"context"
	"time"
)

// Store defines the interface for managing distributed session security state.
type Store interface {
	// IsAccountLocked checks if the account is locked without incrementing the counter.
	IsAccountLocked(ctx context.Context, email, tenantID string) (locked bool, retryAfter time.Duration, err error)

	// RecordFailedLogin increments the failed login counter for an account.
	// Returns locked=true if the account is currently locked, along with the retryAfter duration.
	RecordFailedLogin(ctx context.Context, email, tenantID string) (locked bool, retryAfter time.Duration, err error)

	// ClearFailedLogin resets the failed login counter after a successful login.
	ClearFailedLogin(ctx context.Context, email, tenantID string) error

	// RevokeToken adds a token signature or ID to a denylist until its natural expiration.
	RevokeToken(ctx context.Context, tokenID string, expiration time.Duration) error

	// IsTokenRevoked checks if a given token signature or ID is currently in the denylist.
	IsTokenRevoked(ctx context.Context, tokenID string) (bool, error)
}
