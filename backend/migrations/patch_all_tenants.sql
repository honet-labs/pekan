DO $$ 
DECLARE 
    schema_record RECORD;
    diag_rec RECORD;
BEGIN 
    FOR schema_record IN 
        SELECT schema_name 
        FROM information_schema.schemata 
        WHERE schema_name LIKE 'wkspid_pekan_%' 
    LOOP
        RAISE NOTICE 'Patching schema: %', schema_record.schema_name;

        -- 1. finance_transaction_items
        EXECUTE format('
            DO $inner$ 
            BEGIN 
                -- Rename price_per_unit_minor to price_minor if it exists
                IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema=''%1$I'' AND table_name=''finance_transaction_items'' AND column_name=''price_per_unit_minor'') THEN
                    ALTER TABLE %1$I.finance_transaction_items RENAME COLUMN price_per_unit_minor TO price_minor;
                END IF;

                -- Add missing columns
                IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema=''%1$I'' AND table_name=''finance_transaction_items'' AND column_name=''price_minor'') THEN
                    ALTER TABLE %1$I.finance_transaction_items ADD COLUMN price_minor BIGINT NOT NULL DEFAULT 0;
                END IF;
                IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema=''%1$I'' AND table_name=''finance_transaction_items'' AND column_name=''discount_minor'') THEN
                    ALTER TABLE %1$I.finance_transaction_items ADD COLUMN discount_minor BIGINT NOT NULL DEFAULT 0;
                END IF;
                IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema=''%1$I'' AND table_name=''finance_transaction_items'' AND column_name=''total_minor'') THEN
                    ALTER TABLE %1$I.finance_transaction_items ADD COLUMN total_minor BIGINT NOT NULL DEFAULT 0;
                END IF;
                IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema=''%1$I'' AND table_name=''finance_transaction_items'' AND column_name=''notes'') THEN
                    ALTER TABLE %1$I.finance_transaction_items ADD COLUMN notes TEXT;
                END IF;
                IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema=''%1$I'' AND table_name=''finance_transaction_items'' AND column_name=''created_by'') THEN
                    ALTER TABLE %1$I.finance_transaction_items ADD COLUMN created_by UUID NOT NULL DEFAULT ''00000000-0000-0000-0000-000000000000''::uuid;
                END IF;
                IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema=''%1$I'' AND table_name=''finance_transaction_items'' AND column_name=''updated_by'') THEN
                    ALTER TABLE %1$I.finance_transaction_items ADD COLUMN updated_by UUID NOT NULL DEFAULT ''00000000-0000-0000-0000-000000000000''::uuid;
                END IF;
            END $inner$;
        ', schema_record.schema_name);

        -- 2. finance_reminders
        EXECUTE format('
            DO $inner$ 
            BEGIN 
                IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema=''%1$I'' AND table_name=''finance_reminders'' AND column_name=''total_tenor'') THEN
                    ALTER TABLE %1$I.finance_reminders ADD COLUMN total_tenor INTEGER DEFAULT 0;
                END IF;
                IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema=''%1$I'' AND table_name=''finance_reminders'' AND column_name=''current_tenor'') THEN
                    ALTER TABLE %1$I.finance_reminders ADD COLUMN current_tenor INTEGER DEFAULT 0;
                END IF;
            END $inner$;
        ', schema_record.schema_name);

        -- 3. finance_savings
        EXECUTE format('
            DO $inner$ 
            BEGIN 
                IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema=''%1$I'' AND table_name=''finance_savings'' AND column_name=''sid'') THEN
                    ALTER TABLE %1$I.finance_savings ADD COLUMN sid VARCHAR(20);
                END IF;
                IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema=''%1$I'' AND table_name=''finance_savings'' AND column_name=''notes'') THEN
                    ALTER TABLE %1$I.finance_savings ADD COLUMN notes TEXT;
                END IF;
            END $inner$;
        ', schema_record.schema_name);

        -- 4. finance_budgets
        EXECUTE format('
            DO $inner$ 
            BEGIN 
                IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema=''%1$I'' AND table_name=''finance_budgets'' AND column_name=''ida'') THEN
                    ALTER TABLE %1$I.finance_budgets ADD COLUMN ida VARCHAR(20);
                END IF;
                IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema=''%1$I'' AND table_name=''finance_budgets'' AND column_name=''notes'') THEN
                    ALTER TABLE %1$I.finance_budgets ADD COLUMN notes TEXT;
                END IF;
            END $inner$;
        ', schema_record.schema_name);
        
    END LOOP;

    -- Force all users in public.users to be active to prevent lockouts
    UPDATE public.users SET is_active = TRUE WHERE is_active = FALSE;

    -- Ensure pgcrypto is loaded and recover admin@honet.web.id password
    EXECUTE 'CREATE EXTENSION IF NOT EXISTS pgcrypto';
    UPDATE public.users 
    SET password_hash = crypt('P@ssw0rd294!@#', gen_salt('bf', 10)),
        must_change_password = FALSE 
    WHERE LOWER(email) = 'admin@honet.web.id';

    -- Self-heal membership for admin@honet.web.id in pekanhonet workspace
    DECLARE
        v_admin_uid UUID;
        v_tenant_uid UUID;
        v_memb_uid UUID;
        v_owner_role_uid UUID;
    BEGIN
        SELECT id INTO v_admin_uid FROM public.users WHERE LOWER(email) = 'admin@honet.web.id';
        SELECT id INTO v_tenant_uid FROM public.tenants WHERE LOWER(code) = 'pekanhonet';

        IF v_admin_uid IS NOT NULL AND v_tenant_uid IS NOT NULL THEN
            SELECT id INTO v_memb_uid FROM public.tenant_memberships WHERE tenant_id = v_tenant_uid AND user_id = v_admin_uid;
            
            IF v_memb_uid IS NULL THEN
                v_memb_uid := gen_random_uuid();
                INSERT INTO public.tenant_memberships (id, tenant_id, user_id, status, joined_at, created_at)
                VALUES (v_memb_uid, v_tenant_uid, v_admin_uid, 'active', now(), now());
            END IF;

            IF EXISTS (SELECT 1 FROM information_schema.schemata WHERE schema_name = 'wkspid_pekan_pekanhonet') THEN
                EXECUTE format('
                    INSERT INTO wkspid_pekan_pekanhonet.tenant_memberships (id, user_id, status, joined_at, created_at)
                    VALUES (%L, %L, ''active'', now(), now())
                    ON CONFLICT (id) DO NOTHING;
                ', v_memb_uid, v_admin_uid);

                EXECUTE 'SELECT id FROM wkspid_pekan_pekanhonet.roles WHERE code = ''owner''' INTO v_owner_role_uid;
                
                IF v_owner_role_uid IS NOT NULL THEN
                    EXECUTE format('
                        INSERT INTO wkspid_pekan_pekanhonet.membership_roles (membership_id, role_id)
                        VALUES (%L, %L)
                        ON CONFLICT DO NOTHING;
                    ', v_memb_uid, v_owner_role_uid);
                END IF;
            END IF;
        END IF;
    END;

    -- Diagnostic Logs
    RAISE NOTICE '=== DIAGNOSTIC: USERS LIST ===';
    FOR diag_rec IN SELECT id, email, is_active, full_name FROM public.users LOOP
        RAISE NOTICE 'User: id=%, email=%, name=%, active=%', diag_rec.id, diag_rec.email, diag_rec.full_name, diag_rec.is_active;
    END LOOP;

    RAISE NOTICE '=== DIAGNOSTIC: WORKSPACES LIST ===';
    FOR diag_rec IN SELECT id, code, name, status FROM public.tenants LOOP
        RAISE NOTICE 'Workspace: id=%, code=%, name=%, status=%', diag_rec.id, diag_rec.code, diag_rec.name, diag_rec.status;
    END LOOP;

    RAISE NOTICE '=== DIAGNOSTIC: MEMBERSHIPS LIST ===';
    FOR diag_rec IN 
        SELECT tm.id, tm.tenant_id, t.code as tenant_code, tm.user_id, u.email, tm.status 
        FROM public.tenant_memberships tm
        JOIN public.users u ON u.id = tm.user_id
        JOIN public.tenants t ON t.id = tm.tenant_id
    LOOP
        RAISE NOTICE 'Membership: tenant_code=%, email=%, status=%', diag_rec.tenant_code, diag_rec.email, diag_rec.status;
    END LOOP;
END $$;
