-- Add ON DELETE CASCADE to auth_refresh_tokens referencing tenants(id)
-- This fixes the foreign key constraint error when deleting a tenant.

ALTER TABLE auth_refresh_tokens DROP CONSTRAINT IF EXISTS auth_refresh_tokens_tenant_id_fkey;
ALTER TABLE auth_refresh_tokens ADD CONSTRAINT auth_refresh_tokens_tenant_id_fkey 
    FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE;
