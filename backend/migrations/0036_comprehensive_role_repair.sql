-- Migration to repair missing roles and memberships for all users across all tenants
-- Ensures every user in public.tenant_memberships has a corresponding record and role in their tenant schema

DO $$
DECLARE
    tenant_record RECORD;
    user_record RECORD;
    v_schema_name TEXT;
    v_role_id UUID;
    v_owner_role_id UUID;
    v_admin_role_id UUID;
BEGIN
    FOR tenant_record IN 
        SELECT id, code FROM public.tenants
    LOOP
        -- Skip system/public tenants
        IF tenant_record.code = 'public' OR tenant_record.code = 'default' THEN
            CONTINUE;
        END IF;

        v_schema_name := 'wkspid_pekan_' || tenant_record.code;
        
        -- Check if schema exists
        IF EXISTS (SELECT 1 FROM information_schema.schemata s WHERE s.schema_name = v_schema_name) THEN
            RAISE NOTICE 'Repairing memberships for tenant: %', tenant_record.code;

            -- 1. Ensure 'owner' and 'admin' roles exist in tenant schema
            EXECUTE format('
                INSERT INTO %I.roles (id, code, name, is_system, created_at, updated_at)
                VALUES 
                    (gen_random_uuid(), ''owner'', ''Owner'', true, now(), now()),
                    (gen_random_uuid(), ''admin'', ''Administrator'', true, now(), now())
                ON CONFLICT (code) DO NOTHING;
            ', v_schema_name, v_schema_name);

            -- Get role IDs for assignment
            EXECUTE format('SELECT id FROM %I.roles WHERE code = ''owner'' LIMIT 1', v_schema_name) INTO v_owner_role_id;
            EXECUTE format('SELECT id FROM %I.roles WHERE code = ''admin'' LIMIT 1', v_schema_name) INTO v_admin_role_id;

            -- 2. Ensure all permissions are linked to 'owner' role
            EXECUTE format('
                INSERT INTO %I.role_permissions (role_id, permission_id)
                SELECT %L, p.id
                FROM public.permissions p
                WHERE NOT EXISTS (
                    SELECT 1 FROM %I.role_permissions rp2 
                    WHERE rp2.role_id = %L AND rp2.permission_id = p.id
                )
                ON CONFLICT DO NOTHING;
            ', v_schema_name, v_owner_role_id, v_schema_name, v_owner_role_id);

            -- 3. Loop through all users in public.tenant_memberships for this tenant
            FOR user_record IN 
                SELECT id, user_id, status, joined_at, created_at 
                FROM public.tenant_memberships 
                WHERE tenant_id = tenant_record.id
            LOOP
                -- 3a. Ensure membership exists in tenant schema
                EXECUTE format('
                    INSERT INTO %I.tenant_memberships (id, user_id, status, joined_at, created_at)
                    VALUES (%L, %L, %L, %L, %L)
                    ON CONFLICT (id) DO NOTHING;
                ', v_schema_name, user_record.id, user_record.user_id, user_record.status, user_record.joined_at, user_record.created_at);

                -- 3b. If user has no roles in tenant schema, assign one
                -- Check if membership already has a role
                DECLARE
                    v_has_role BOOLEAN;
                BEGIN
                    EXECUTE format('SELECT EXISTS(SELECT 1 FROM %I.membership_roles WHERE membership_id = %L)', v_schema_name, user_record.id) INTO v_has_role;
                    
                    IF NOT v_has_role THEN
                        -- If it is the first user (by joined_at), give owner, else give admin
                        DECLARE
                            v_is_first BOOLEAN;
                        BEGIN
                            SELECT (user_record.id = (
                                SELECT id FROM public.tenant_memberships 
                                WHERE tenant_id = tenant_record.id 
                                ORDER BY joined_at ASC LIMIT 1
                            )) INTO v_is_first;

                            IF v_is_first THEN
                                v_role_id := v_owner_role_id;
                            ELSE
                                v_role_id := v_admin_role_id;
                            END IF;

                            EXECUTE format('
                                INSERT INTO %I.membership_roles (membership_id, role_id)
                                VALUES (%L, %L)
                                ON CONFLICT DO NOTHING;
                            ', v_schema_name, user_record.id, v_role_id);
                        END;
                    END IF;
                END;
            END LOOP;
        END IF;
    END LOOP;
END $$;
