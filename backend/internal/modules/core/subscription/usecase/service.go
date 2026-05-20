package usecase

import (
	"context"
	"time"

	"pekan/backend/internal/modules/core/subscription/domain"
)

type Authorizer interface {
	EnsurePermission(ctx context.Context, permission string) error
}

type AuditLogger interface {
	Write(ctx context.Context, action, resourceType, resourceID string, before, after any) error
}

type Service struct {
	repo  domain.Repository
	authz Authorizer
	audit AuditLogger
}

func NewService(repo domain.Repository, authz Authorizer, audit AuditLogger) *Service {
	return &Service{
		repo:  repo,
		authz: authz,
		audit: audit,
	}
}

func (s *Service) GetEffectiveEntitlements(ctx context.Context, tenantID string) (domain.EffectiveEntitlements, error) {
	if err := s.authz.EnsurePermission(ctx, "core.entitlement.read"); err != nil {
		return domain.EffectiveEntitlements{}, err
	}
	return s.repo.GetEffectiveEntitlements(ctx, tenantID)
}

type SetFeatureOverrideInput struct {
	TenantID   string
	FeatureCode string
	IsEnabled  bool
	Reason     *string
	ExpiresAt  *time.Time
}

func (s *Service) SetFeatureOverride(ctx context.Context, in SetFeatureOverrideInput) error {
	if err := s.authz.EnsurePermission(ctx, "core.entitlement.manage"); err != nil {
		return err
	}
	if err := s.repo.SetFeatureOverride(ctx, in.TenantID, in.FeatureCode, in.IsEnabled, in.Reason, in.ExpiresAt); err != nil {
		return err
	}
	_ = s.audit.Write(ctx, "core.entitlement.feature_override.set", "tenant_feature_override", in.TenantID, nil, in)
	return nil
}
