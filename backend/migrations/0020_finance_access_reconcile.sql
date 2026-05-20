-- Reconcile access grants for legacy tenants after finance module expansion.
-- This migration is intentionally idempotent and safe to rerun.

-- Ensure owner/admin roles can access all finance permissions.
INSERT INTO role_permissions (role_id, permission_id, created_at)
SELECT r.id, p.id, now()
FROM roles r
JOIN permissions p ON p.code LIKE 'finance.%'
WHERE r.code IN ('owner', 'admin')
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- Ensure all finance modules are enabled for active tenants.
INSERT INTO tenant_modules (id, tenant_id, module_code, is_enabled, source, created_at, updated_at)
SELECT uuid_generate_v4(), t.id, m.code, TRUE, 'manual', now(), now()
FROM tenants t
JOIN modules m ON (m.code = 'finance' OR m.code LIKE 'finance.%')
WHERE t.status = 'active'
ON CONFLICT (tenant_id, module_code) DO UPDATE
SET is_enabled = TRUE,
    updated_at = now();

-- Ensure all finance features are enabled for active tenants.
INSERT INTO tenant_features (id, tenant_id, feature_code, is_enabled, source, created_at, updated_at)
SELECT uuid_generate_v4(), t.id, f.code, TRUE, 'manual', now(), now()
FROM tenants t
JOIN features f ON TRUE
JOIN modules m ON m.id = f.module_id
WHERE t.status = 'active'
  AND (m.code = 'finance' OR m.code LIKE 'finance.%')
ON CONFLICT (tenant_id, feature_code) DO UPDATE
SET is_enabled = TRUE,
    updated_at = now();
