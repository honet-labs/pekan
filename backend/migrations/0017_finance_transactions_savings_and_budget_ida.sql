DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'finance_transactions_type_check'
    ) THEN
        ALTER TABLE finance_transactions
            DROP CONSTRAINT finance_transactions_type_check;
    END IF;
END $$;

ALTER TABLE finance_transactions
    ADD CONSTRAINT finance_transactions_type_check
    CHECK (type IN ('income', 'expense', 'transfer', 'savings'));

CREATE UNIQUE INDEX IF NOT EXISTS uq_finance_savings_tenant_id_id
    ON finance_savings (tenant_id, id);

CREATE SEQUENCE IF NOT EXISTS finance_budgets_ida_seq START 1;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = 'public'
          AND table_name = 'finance_budgets'
          AND column_name = 'ida'
    ) THEN
        ALTER TABLE finance_budgets
            ADD COLUMN ida VARCHAR(20);
    END IF;
END $$;

UPDATE finance_budgets
SET ida = 'BGT-' || LPAD(nextval('finance_budgets_ida_seq')::text, 6, '0')
WHERE ida IS NULL;

ALTER TABLE finance_budgets
    ALTER COLUMN ida SET DEFAULT ('BGT-' || LPAD(nextval('finance_budgets_ida_seq')::text, 6, '0'));

ALTER TABLE finance_budgets
    ALTER COLUMN ida SET NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS uq_finance_budgets_tenant_ida
    ON finance_budgets (tenant_id, ida)
    WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS finance_transaction_savings_links (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    transaction_id UUID NOT NULL,
    savings_id UUID NOT NULL,
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, transaction_id, savings_id),
    CONSTRAINT fk_transaction_savings_transaction
        FOREIGN KEY (tenant_id, transaction_id)
        REFERENCES finance_transactions (tenant_id, id),
    CONSTRAINT fk_transaction_savings_goal
        FOREIGN KEY (tenant_id, savings_id)
        REFERENCES finance_savings (tenant_id, id)
);

CREATE INDEX IF NOT EXISTS idx_transaction_savings_links_transaction
    ON finance_transaction_savings_links (tenant_id, transaction_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_transaction_savings_links_savings
    ON finance_transaction_savings_links (tenant_id, savings_id, created_at DESC);

CREATE TABLE IF NOT EXISTS user_profiles (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    username VARCHAR(120) NOT NULL UNIQUE,
    phone VARCHAR(40) NULL,
    address TEXT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

INSERT INTO user_profiles (user_id, username, created_at, updated_at)
SELECT
    u.id,
    LOWER(REGEXP_REPLACE(SPLIT_PART(u.email, '@', 1), '[^a-zA-Z0-9_]+', '_', 'g')) || '_' || SUBSTRING(u.id::text, 1, 6),
    now(),
    now()
FROM users u
ON CONFLICT (user_id) DO NOTHING;
