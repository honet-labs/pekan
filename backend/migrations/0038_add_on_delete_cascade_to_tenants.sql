-- Add missing ON DELETE CASCADE to global tenant-related tables
-- This ensures that deleting a tenant automatically cleans up its modules, features, etc.

DO $$ 
BEGIN
    -- 1. tenant_modules
    IF EXISTS (SELECT 1 FROM information_schema.table_constraints WHERE constraint_name = 'tenant_modules_tenant_id_fkey') THEN
        ALTER TABLE public.tenant_modules DROP CONSTRAINT tenant_modules_tenant_id_fkey;
    END IF;
    ALTER TABLE public.tenant_modules ADD CONSTRAINT tenant_modules_tenant_id_fkey 
        FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE;

    -- 2. tenant_features
    IF EXISTS (SELECT 1 FROM information_schema.table_constraints WHERE constraint_name = 'tenant_features_tenant_id_fkey') THEN
        ALTER TABLE public.tenant_features DROP CONSTRAINT tenant_features_tenant_id_fkey;
    END IF;
    ALTER TABLE public.tenant_features ADD CONSTRAINT tenant_features_tenant_id_fkey 
        FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE;

    -- 3. roles
    IF EXISTS (SELECT 1 FROM information_schema.table_constraints WHERE constraint_name = 'roles_tenant_id_fkey') THEN
        ALTER TABLE public.roles DROP CONSTRAINT roles_tenant_id_fkey;
    END IF;
    ALTER TABLE public.roles ADD CONSTRAINT roles_tenant_id_fkey 
        FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE;

    -- 4. tenant_memberships
    IF EXISTS (SELECT 1 FROM information_schema.table_constraints WHERE constraint_name = 'tenant_memberships_tenant_id_fkey') THEN
        ALTER TABLE public.tenant_memberships DROP CONSTRAINT tenant_memberships_tenant_id_fkey;
    END IF;
    ALTER TABLE public.tenant_memberships ADD CONSTRAINT tenant_memberships_tenant_id_fkey 
        FOREIGN KEY (tenant_id) REFERENCES public.tenants(id) ON DELETE CASCADE;

END $$;
