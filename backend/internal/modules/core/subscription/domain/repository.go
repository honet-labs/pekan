package domain

import (
	"context"
	"time"
)

type Repository interface {
	SetFeatureOverride(ctx context.Context, tenantID, featureCode string, isEnabled bool, reason *string, expiresAt *time.Time) error
	GetEffectiveEntitlements(ctx context.Context, tenantID string) (EffectiveEntitlements, error)
}
