package infra

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"

	"pekan/backend/internal/modules/core/subscription/domain"
	"pekan/backend/internal/platform/entitlement"
)

type RepositoryPG struct {
	db       *sql.DB
	resolver entitlement.Resolver
}

func NewRepositoryPG(db *sql.DB) *RepositoryPG {
	return &RepositoryPG{
		db:       db,
		resolver: entitlement.NewPGResolver(db),
	}
}

func (r *RepositoryPG) SetFeatureOverride(ctx context.Context, tenantID, featureCode string, isEnabled bool, reason *string, expiresAt *time.Time) error {
	const featureQuery = `SELECT id FROM features WHERE code = $1`
	var featureID string
	if err := r.db.QueryRowContext(ctx, featureQuery, featureCode).Scan(&featureID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.ErrFeatureNotFound
		}
		return err
	}

	const upsertQuery = `
INSERT INTO tenant_feature_overrides (
  id, tenant_id, feature_id, is_enabled, reason, expires_at, created_at, updated_at
) VALUES ($1,$2,$3,$4,$5,$6,now(),now())
ON CONFLICT (tenant_id, feature_id)
DO UPDATE SET
  is_enabled = EXCLUDED.is_enabled,
  reason = EXCLUDED.reason,
  expires_at = EXCLUDED.expires_at,
  updated_at = now()`

	_, err := r.db.ExecContext(ctx, upsertQuery,
		uuid.NewString(), tenantID, featureID, isEnabled, reason, expiresAt,
	)
	return err
}

func (r *RepositoryPG) GetEffectiveEntitlements(ctx context.Context, tenantID string) (domain.EffectiveEntitlements, error) {
	resolved, err := r.resolver.ResolveTenant(ctx, tenantID)
	if err != nil {
		return domain.EffectiveEntitlements{}, err
	}
	return domain.EffectiveEntitlements{
		Modules:  resolved.Modules,
		Features: resolved.Features,
	}, nil
}
