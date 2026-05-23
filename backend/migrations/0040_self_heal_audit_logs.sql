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

-- 6. Reconcile finance.settings permissions into tenant-specific schemas for owner and admin roles
DO $$
DECLARE
    tenant_record RECORD;
    v_schema_name TEXT;
    v_owner_role_id UUID;
    v_admin_role_id UUID;
BEGIN
    FOR tenant_record IN 
        SELECT id, code FROM public.tenants
    LOOP
        -- Skip system/public/default tenants
        IF tenant_record.code = 'public' OR tenant_record.code = 'default' THEN
            CONTINUE;
        END IF;

        IF tenant_record.code = 'pekan' OR tenant_record.code = 'pekanhonet' THEN
            v_schema_name := 'wkspid_pekan_pekanhonet';
        ELSE
            v_schema_name := 'wkspid_pekan_' || tenant_record.code;
        END IF;
        
        -- Check if schema exists and has roles table
        IF EXISTS (SELECT 1 FROM information_schema.schemata s WHERE s.schema_name = v_schema_name) THEN
            RAISE NOTICE '=== Aligning finance permissions in schema: % ===', v_schema_name;

            -- Get role IDs for owner and admin in the tenant schema
            EXECUTE format('SELECT id FROM %I.roles WHERE code = ''owner'' LIMIT 1', v_schema_name) INTO v_owner_role_id;
            EXECUTE format('SELECT id FROM %I.roles WHERE code = ''admin'' LIMIT 1', v_schema_name) INTO v_admin_role_id;

            -- Link finance.settings permissions to owner role in tenant schema
            IF v_owner_role_id IS NOT NULL THEN
                EXECUTE format('
                    INSERT INTO %I.role_permissions (role_id, permission_id)
                    SELECT %L, p.id
                    FROM public.permissions p
                    WHERE p.code IN (
                        ''finance.settings.read'',
                        ''finance.settings.update'',
                        ''finance.settings.roles.manage'',
                        ''finance.settings.audit.read''
                    )
                    ON CONFLICT DO NOTHING;
                ', v_schema_name, v_owner_role_id);
            END IF;

            -- Link finance.settings permissions to admin role in tenant schema
            IF v_admin_role_id IS NOT NULL THEN
                EXECUTE format('
                    INSERT INTO %I.role_permissions (role_id, permission_id)
                    SELECT %L, p.id
                    FROM public.permissions p
                    WHERE p.code IN (
                        ''finance.settings.read'',
                        ''finance.settings.update'',
                        ''finance.settings.roles.manage'',
                        ''finance.settings.audit.read''
                    )
                    ON CONFLICT DO NOTHING;
                ', v_schema_name, v_admin_role_id);
            END IF;
        END IF;
    END LOOP;
END $$;

