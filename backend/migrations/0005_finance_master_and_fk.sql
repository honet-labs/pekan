CREATE TABLE IF NOT EXISTS finance_accounts (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    name VARCHAR(150) NOT NULL,
    account_type VARCHAR(30) NOT NULL CHECK (account_type IN ('cash','bank','ewallet','credit')),
    currency CHAR(3) NOT NULL,
    opening_balance_minor BIGINT NOT NULL DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, name)
);

CREATE INDEX IF NOT EXISTS idx_finance_accounts_tenant_active ON finance_accounts (tenant_id, is_active);

CREATE TABLE IF NOT EXISTS finance_categories (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    name VARCHAR(150) NOT NULL,
    category_type VARCHAR(20) NOT NULL CHECK (category_type IN ('income','expense')),
    parent_id UUID NULL REFERENCES finance_categories(id),
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, category_type, name)
);

CREATE INDEX IF NOT EXISTS idx_finance_categories_tenant_type ON finance_categories (tenant_id, category_type, is_active);

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_finance_transactions_account_id'
    ) THEN
        ALTER TABLE finance_transactions
            ADD CONSTRAINT fk_finance_transactions_account_id
            FOREIGN KEY (account_id) REFERENCES finance_accounts(id);
    END IF;
END $$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_finance_transactions_category_id'
    ) THEN
        ALTER TABLE finance_transactions
            ADD CONSTRAINT fk_finance_transactions_category_id
            FOREIGN KEY (category_id) REFERENCES finance_categories(id);
    END IF;
END $$;

