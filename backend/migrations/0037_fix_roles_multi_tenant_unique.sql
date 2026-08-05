-- Fix bad unique constraint on roles table that breaks multi-tenancy
-- Some environments might have a legacy 'roles_code_key' instead of (tenant_id, code)

DO $$ 
BEGIN
    -- 1. Remove the old single-column unique constraint if it exists
    -- The name 'roles_code_key' is the standard Postgres name for UNIQUE(code)
    ALTER TABLE IF EXISTS public.roles DROP CONSTRAINT IF EXISTS roles_code_key;
    
    -- 2. Ensure the correct multi-tenant unique constraint exists
    -- We use IF NOT EXISTS pattern (by checking pg_constraint)
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'roles_tenant_id_code_key') THEN
        ALTER TABLE public.roles ADD CONSTRAINT roles_tenant_id_code_key UNIQUE (tenant_id, code);
    END IF;

    -- 3. Also check tenant_modules and tenant_features just in case
    ALTER TABLE IF EXISTS public.tenant_modules DROP CONSTRAINT IF EXISTS tenant_modules_module_code_key;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'tenant_modules_tenant_id_module_code_key') THEN
        ALTER TABLE public.tenant_modules ADD CONSTRAINT tenant_modules_tenant_id_module_code_key UNIQUE (tenant_id, module_code);
    END IF;

    ALTER TABLE IF EXISTS public.tenant_features DROP CONSTRAINT IF EXISTS tenant_features_feature_code_key;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'tenant_features_tenant_id_feature_code_key') THEN
        ALTER TABLE public.tenant_features ADD CONSTRAINT tenant_features_tenant_id_feature_code_key UNIQUE (tenant_id, feature_code);
    END IF;
END $$;
