-- Migration to handle cascading effects when a user is deleted.
-- This is primarily used when a tenant is deleted and its orphaned users are cleaned up.

-- 1. audit_logs: set actor_user_id to NULL if the user is deleted
ALTER TABLE audit_logs DROP CONSTRAINT IF EXISTS audit_logs_actor_user_id_fkey;
ALTER TABLE audit_logs ADD CONSTRAINT audit_logs_actor_user_id_fkey 
    FOREIGN KEY (actor_user_id) REFERENCES users(id) ON DELETE SET NULL;

-- 2. auth_sessions: delete sessions if user is deleted
ALTER TABLE auth_sessions DROP CONSTRAINT IF EXISTS auth_sessions_user_id_fkey;
ALTER TABLE auth_sessions ADD CONSTRAINT auth_sessions_user_id_fkey 
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

-- 3. auth_refresh_tokens: delete tokens if user is deleted
ALTER TABLE auth_refresh_tokens DROP CONSTRAINT IF EXISTS auth_refresh_tokens_user_id_fkey;
ALTER TABLE auth_refresh_tokens ADD CONSTRAINT auth_refresh_tokens_user_id_fkey 
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

-- 4. tenant_memberships: delete memberships if user is deleted (already handled by cascade usually, but good to be explicit)
ALTER TABLE tenant_memberships DROP CONSTRAINT IF EXISTS tenant_memberships_user_id_fkey;
ALTER TABLE tenant_memberships ADD CONSTRAINT tenant_memberships_user_id_fkey 
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;
