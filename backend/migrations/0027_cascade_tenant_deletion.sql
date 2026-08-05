-- Add ON DELETE CASCADE to various foreign keys referencing tenants(id)
-- This ensures clean deletion of tenants and all associated data.

-- 1. tenant_features
ALTER TABLE tenant_features DROP CONSTRAINT IF EXISTS tenant_features_tenant_id_fkey;
ALTER TABLE tenant_features ADD CONSTRAINT tenant_features_tenant_id_fkey 
    FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE;

-- 2. auth_sessions
ALTER TABLE auth_sessions DROP CONSTRAINT IF EXISTS auth_sessions_tenant_id_fkey;
ALTER TABLE auth_sessions ADD CONSTRAINT auth_sessions_tenant_id_fkey 
    FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE;

-- 3. roles
ALTER TABLE roles DROP CONSTRAINT IF EXISTS roles_tenant_id_fkey;
ALTER TABLE roles ADD CONSTRAINT roles_tenant_id_fkey 
    FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE;

-- 4. tenant_memberships
ALTER TABLE tenant_memberships DROP CONSTRAINT IF EXISTS tenant_memberships_tenant_id_fkey;
ALTER TABLE tenant_memberships ADD CONSTRAINT tenant_memberships_tenant_id_fkey 
    FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE;

-- 5. files
ALTER TABLE files DROP CONSTRAINT IF EXISTS files_tenant_id_fkey;
ALTER TABLE files ADD CONSTRAINT files_tenant_id_fkey 
    FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE;

-- 6. finance_accounts
ALTER TABLE finance_accounts DROP CONSTRAINT IF EXISTS finance_accounts_tenant_id_fkey;
ALTER TABLE finance_accounts ADD CONSTRAINT finance_accounts_tenant_id_fkey 
    FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE;

-- 7. finance_categories
ALTER TABLE finance_categories DROP CONSTRAINT IF EXISTS finance_categories_tenant_id_fkey;
ALTER TABLE finance_categories ADD CONSTRAINT finance_categories_tenant_id_fkey 
    FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE;

-- 8. finance_transactions
ALTER TABLE finance_transactions DROP CONSTRAINT IF EXISTS finance_transactions_tenant_id_fkey;
ALTER TABLE finance_transactions ADD CONSTRAINT finance_transactions_tenant_id_fkey 
    FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE;

-- 9. finance_transaction_attachments
ALTER TABLE finance_transaction_attachments DROP CONSTRAINT IF EXISTS finance_transaction_attachments_tenant_id_fkey;
ALTER TABLE finance_transaction_attachments ADD CONSTRAINT finance_transaction_attachments_tenant_id_fkey 
    FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE;

-- 10. finance_savings
ALTER TABLE finance_savings DROP CONSTRAINT IF EXISTS finance_savings_tenant_id_fkey;
ALTER TABLE finance_savings ADD CONSTRAINT finance_savings_tenant_id_fkey 
    FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE;

-- 11. finance_budgets
ALTER TABLE finance_budgets DROP CONSTRAINT IF EXISTS finance_budgets_tenant_id_fkey;
ALTER TABLE finance_budgets ADD CONSTRAINT finance_budgets_tenant_id_fkey 
    FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE;

-- 12. finance_reminders
ALTER TABLE finance_reminders DROP CONSTRAINT IF EXISTS finance_reminders_tenant_id_fkey;
ALTER TABLE finance_reminders ADD CONSTRAINT finance_reminders_tenant_id_fkey 
    FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE;

-- 13. finance_reports
ALTER TABLE finance_reports DROP CONSTRAINT IF EXISTS finance_reports_tenant_id_fkey;
ALTER TABLE finance_reports ADD CONSTRAINT finance_reports_tenant_id_fkey 
    FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE;

-- 14. finance_transaction_items
ALTER TABLE finance_transaction_items DROP CONSTRAINT IF EXISTS finance_transaction_items_tenant_id_fkey;
ALTER TABLE finance_transaction_items ADD CONSTRAINT finance_transaction_items_tenant_id_fkey 
    FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE;

-- 15. audit_logs (Note: tenant_id can be NULL here, but cascade still helps if it's set)
ALTER TABLE audit_logs DROP CONSTRAINT IF EXISTS audit_logs_tenant_id_fkey;
ALTER TABLE audit_logs ADD CONSTRAINT audit_logs_tenant_id_fkey 
    FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE;

-- Also handle sub-tables that might not have CASCADE yet
ALTER TABLE role_permissions DROP CONSTRAINT IF EXISTS role_permissions_role_id_fkey;
ALTER TABLE role_permissions ADD CONSTRAINT role_permissions_role_id_fkey 
    FOREIGN KEY (role_id) REFERENCES roles(id) ON DELETE CASCADE;

ALTER TABLE membership_roles DROP CONSTRAINT IF EXISTS membership_roles_membership_id_fkey;
ALTER TABLE membership_roles ADD CONSTRAINT membership_roles_membership_id_fkey 
    FOREIGN KEY (membership_id) REFERENCES tenant_memberships(id) ON DELETE CASCADE;

ALTER TABLE membership_roles DROP CONSTRAINT IF EXISTS membership_roles_role_id_fkey;
ALTER TABLE membership_roles ADD CONSTRAINT membership_roles_role_id_fkey 
    FOREIGN KEY (role_id) REFERENCES roles(id) ON DELETE CASCADE;
