-- Migration 0040: Self-healing audit logs and module settings for active tenants
-- Date: 2026-05-23

-- 1. Backfill tenant_id for audit logs that currently have NULL tenant_id
UPDATE public.audit_logs l
SET tenant_id = tm.tenant_id
FROM public.tenant_memberships tm
WHERE l.tenant_id IS NULL
  AND l.actor_user_id = tm.user_id;

-- 2. Ensure finance.settings module is enabled for all active tenants
INSERT INTO public.tenant_modules (id, tenant_id, module_code, is_enabled, source, created_at, updated_at)
SELECT uuid_generate_v4(), t.id, 'finance.settings', TRUE, 'manual', now(), now()
FROM public.tenants t
WHERE t.status = 'active'
ON CONFLICT (tenant_id, module_code) DO NOTHING;

-- 3. Ensure finance.settings.read feature is enabled for all active tenants
INSERT INTO public.tenant_features (id, tenant_id, feature_code, is_enabled, source, created_at, updated_at)
SELECT uuid_generate_v4(), t.id, 'finance.settings.read', TRUE, 'manual', now(), now()
FROM public.tenants t
WHERE t.status = 'active'
ON CONFLICT (tenant_id, feature_code) DO NOTHING;

-- 4. Ensure finance.settings.write feature is enabled for all active tenants
INSERT INTO public.tenant_features (id, tenant_id, feature_code, is_enabled, source, created_at, updated_at)
SELECT uuid_generate_v4(), t.id, 'finance.settings.write', TRUE, 'manual', now(), now()
FROM public.tenants t
WHERE t.status = 'active'
ON CONFLICT (tenant_id, feature_code) DO NOTHING;

-- 5. Backfill finance permissions to owner and admin roles for all active tenants
INSERT INTO public.role_permissions (role_id, permission_id, created_at)
SELECT r.id, p.id, now()
FROM public.roles r
CROSS JOIN public.permissions p
WHERE r.code IN ('owner', 'admin')
  AND p.code IN (
      'finance.settings.read',
      'finance.settings.update',
      'finance.settings.roles.manage',
      'finance.settings.audit.read'
  )
ON CONFLICT (role_id, permission_id) DO NOTHING;
