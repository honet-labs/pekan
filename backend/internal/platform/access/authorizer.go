package access

import (
	"context"
	"errors"

	"pekan/backend/internal/platform/tenancy"
)

var (
	ErrModuleDisabled   = errors.New("module disabled")
	ErrFeatureLocked    = errors.New("feature locked")
	ErrPermissionDenied = errors.New("permission denied")
)

type Authorizer struct{}

func NewAuthorizer() *Authorizer {
	return &Authorizer{}
}

func (a *Authorizer) EnsureModule(ctx context.Context, module string) error {
	tc, err := tenancy.FromContext(ctx)
	if err != nil {
		return err
	}
	if _, ok := tc.Modules[module]; !ok {
		return ErrModuleDisabled
	}
	return nil
}

func (a *Authorizer) EnsureFeature(ctx context.Context, feature string) error {
	tc, err := tenancy.FromContext(ctx)
	if err != nil {
		return err
	}
	if _, ok := tc.Features[feature]; !ok {
		return ErrFeatureLocked
	}
	return nil
}

func (a *Authorizer) EnsurePermission(ctx context.Context, permission string) error {
	tc, err := tenancy.FromContext(ctx)
	if err != nil {
		return err
	}
	if _, ok := tc.Permissions[permission]; !ok {
		return ErrPermissionDenied
	}
	return nil
}

