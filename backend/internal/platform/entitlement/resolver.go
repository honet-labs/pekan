package entitlement

import (
	"context"
	"database/sql"
	"sort"
	"strings"
)

type ResolveResult struct {
	Modules  []string
	Features []string
}

type Resolver interface {
	ResolveTenant(ctx context.Context, tenantID string) (ResolveResult, error)
}

type PGResolver struct {
	db *sql.DB
}

func NewPGResolver(db *sql.DB) *PGResolver {
	return &PGResolver{db: db}
}

func (r *PGResolver) ResolveTenant(ctx context.Context, tenantID string) (ResolveResult, error) {
	modules := map[string]bool{}
	features := map[string]bool{}

	// Legacy/manual module overrides.
	moduleOverrides, err := r.queryTenantModuleOverrides(ctx, tenantID)
	if err != nil {
		return ResolveResult{}, err
	}
	for _, row := range moduleOverrides {
		modules[row.Code] = row.Enabled
	}

	// Legacy/manual feature overrides.
	featureOverrides, err := r.queryTenantFeatureOverridesLegacy(ctx, tenantID)
	if err != nil {
		return ResolveResult{}, err
	}
	for _, row := range featureOverrides {
		features[row.FeatureCode] = row.Enabled
		if row.Enabled && row.ModuleCode != "" {
			modules[row.ModuleCode] = true
		}
	}

	// Feature overrides with expiry support.
	overrideRows, err := r.queryTenantFeatureOverrides(ctx, tenantID)
	if err != nil {
		return ResolveResult{}, err
	}
	for _, row := range overrideRows {
		features[row.FeatureCode] = row.Enabled
		if row.Enabled && row.ModuleCode != "" {
			modules[row.ModuleCode] = true
		}
	}

	moduleList := make([]string, 0)
	for code, enabled := range modules {
		if enabled {
			moduleList = append(moduleList, code)
		}
	}
	sort.Strings(moduleList)

	featureList := make([]string, 0)
	for code, enabled := range features {
		if enabled {
			featureList = append(featureList, code)
		}
	}
	sort.Strings(featureList)

	return ResolveResult{
		Modules:  moduleList,
		Features: featureList,
	}, nil
}

type featureWithModule struct {
	FeatureCode string
	ModuleCode  string
	Enabled     bool
}

type moduleOverride struct {
	Code    string
	Enabled bool
}

func (r *PGResolver) queryTenantModuleOverrides(ctx context.Context, tenantID string) ([]moduleOverride, error) {
	const q = `
SELECT module_code, is_enabled
FROM public.tenant_modules
WHERE tenant_id = $1`

	rows, err := r.db.QueryContext(ctx, q, tenantID)
	if err != nil {
		if isUndefinedTable(err) {
			return nil, nil
		}
		return nil, err
	}
	defer rows.Close()

	out := make([]moduleOverride, 0)
	for rows.Next() {
		var row moduleOverride
		if err := rows.Scan(&row.Code, &row.Enabled); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (r *PGResolver) queryTenantFeatureOverridesLegacy(ctx context.Context, tenantID string) ([]featureWithModule, error) {
	const q = `
SELECT tf.feature_code, COALESCE(m.code, ''), tf.is_enabled
FROM public.tenant_features tf
LEFT JOIN public.features f ON f.code = tf.feature_code
LEFT JOIN public.modules m ON m.id = f.module_id
WHERE tf.tenant_id = $1`

	rows, err := r.db.QueryContext(ctx, q, tenantID)
	if err != nil {
		if isUndefinedTable(err) {
			return nil, nil
		}
		return nil, err
	}
	defer rows.Close()

	out := make([]featureWithModule, 0)
	for rows.Next() {
		var row featureWithModule
		if err := rows.Scan(&row.FeatureCode, &row.ModuleCode, &row.Enabled); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (r *PGResolver) queryTenantFeatureOverrides(ctx context.Context, tenantID string) ([]featureWithModule, error) {
	const q = `
SELECT f.code, m.code, tfo.is_enabled
FROM public.tenant_feature_overrides tfo
JOIN public.features f ON f.id = tfo.feature_id
JOIN public.modules m ON m.id = f.module_id
WHERE tfo.tenant_id = $1
  AND (tfo.expires_at IS NULL OR tfo.expires_at > now())`

	rows, err := r.db.QueryContext(ctx, q, tenantID)
	if err != nil {
		if isUndefinedTable(err) {
			return nil, nil
		}
		return nil, err
	}
	defer rows.Close()

	out := make([]featureWithModule, 0)
	for rows.Next() {
		var row featureWithModule
		if err := rows.Scan(&row.FeatureCode, &row.ModuleCode, &row.Enabled); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func isUndefinedTable(err error) bool {
	// Avoid hard dependency on driver-specific errors for now.
	return err != nil && (strings.Contains(err.Error(), "does not exist") || strings.Contains(err.Error(), "undefined_table"))
}
