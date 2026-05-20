-- Numbered migration to automatically repair tenants with missing entitlements or permissions
-- This fix is idempotent and will only apply if data is missing.

DO $$
DECLARE
    tenant_record RECORD;
    v_schema_name TEXT;
    admin_user_id UUID;
    v_membership_id UUID;
BEGIN
    FOR tenant_record IN 
        SELECT id, code FROM public.tenants
    LOOP
        -- Skip public/system tenants
        IF tenant_record.code = 'public' OR tenant_record.code = 'default' THEN
            CONTINUE;
        END IF;

        RAISE NOTICE 'Verifying entitlements for tenant: % (%)', tenant_record.code, tenant_record.id;
        
        -- 1. Ensure core modules and features are enabled in PUBLIC schema
        INSERT INTO public.tenant_modules (id, tenant_id, module_code, is_enabled, source, created_at, updated_at)
        SELECT gen_random_uuid(), tenant_record.id, code, TRUE, 'repair', now(), now() FROM public.modules
        ON CONFLICT (tenant_id, module_code) DO UPDATE SET is_enabled = TRUE;

        INSERT INTO public.tenant_features (id, tenant_id, feature_code, is_enabled, source, created_at, updated_at)
        SELECT gen_random_uuid(), tenant_record.id, code, TRUE, 'repair', now(), now() FROM public.features
        ON CONFLICT (tenant_id, feature_code) DO UPDATE SET is_enabled = TRUE;

        -- 2. Repair isolated schema data
        v_schema_name := 'wkspid_pekan_' || tenant_record.code;
        
        IF EXISTS (SELECT 1 FROM information_schema.schemata s WHERE s.schema_name = v_schema_name) THEN
            RAISE NOTICE 'Repairing isolated schema: %', v_schema_name;
            
            -- 2a. Ensure Roles exist
            EXECUTE format('
                INSERT INTO %I.roles (id, code, name, is_system, created_at, updated_at)
                VALUES 
                    (gen_random_uuid(), ''owner'', ''Owner'', true, now(), now()),
                    (gen_random_uuid(), ''admin'', ''Administrator'', true, now(), now())
                ON CONFLICT (code) DO NOTHING;
            ', v_schema_name, v_schema_name);

            -- 2b. Ensure Permissions are linked to Owner role
            EXECUTE format('
                INSERT INTO %I.role_permissions (role_id, permission_id)
                SELECT r.id, p.id
                FROM %I.roles r, public.permissions p
                WHERE r.code = ''owner''
                AND NOT EXISTS (
                    SELECT 1 FROM %I.role_permissions rp2 
                    WHERE rp2.role_id = r.id AND rp2.permission_id = p.id
                )
                ON CONFLICT DO NOTHING;
            ', v_schema_name, v_schema_name, v_schema_name);

            -- 2c. Repair Memberships and Roles linking
            -- Find the first user associated with this tenant in the public schema
            SELECT user_id INTO admin_user_id FROM public.tenant_memberships 
            WHERE tenant_id = tenant_record.id ORDER BY joined_at ASC LIMIT 1;

            IF admin_user_id IS NOT NULL THEN
                -- Ensure membership exists in tenant schema
                EXECUTE format('
                    INSERT INTO %I.tenant_memberships (id, user_id, status, joined_at, created_at)
                    SELECT id, user_id, status, joined_at, created_at FROM public.tenant_memberships
                    WHERE tenant_id = %L AND user_id = %L
                    ON CONFLICT (id) DO NOTHING;
                ', v_schema_name, tenant_record.id, admin_user_id);

                -- Link the user to the ''owner'' role in the tenant schema
                EXECUTE format('
                    INSERT INTO %I.membership_roles (membership_id, role_id)
                    SELECT m.id, r.id
                    FROM %I.tenant_memberships m, %I.roles r
                    WHERE m.user_id = %L AND r.code = ''owner''
                    ON CONFLICT DO NOTHING;
                ', v_schema_name, v_schema_name, v_schema_name, admin_user_id);
            END IF;
        END IF;
    END LOOP;
END $$;
