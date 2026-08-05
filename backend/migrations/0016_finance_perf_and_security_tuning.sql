CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE INDEX IF NOT EXISTS idx_finance_transactions_desc_trgm
    ON finance_transactions
    USING gin (LOWER(COALESCE(description, '')) gin_trgm_ops)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_finance_accounts_name_trgm
    ON finance_accounts
    USING gin (LOWER(name) gin_trgm_ops)
    WHERE is_active = TRUE;

CREATE INDEX IF NOT EXISTS idx_finance_categories_name_trgm
    ON finance_categories
    USING gin (LOWER(name) gin_trgm_ops)
    WHERE is_active = TRUE;

CREATE INDEX IF NOT EXISTS idx_audit_logs_tenant_actor_created
    ON audit_logs (tenant_id, actor_user_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_finance_transactions_tenant_created
    ON finance_transactions (tenant_id, created_at DESC)
    WHERE deleted_at IS NULL;
