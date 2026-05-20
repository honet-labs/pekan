-- Defense-in-depth for tenant isolation at DB level.
-- Adds composite tenant-aware constraints to prevent cross-tenant references.

CREATE UNIQUE INDEX IF NOT EXISTS uq_finance_accounts_tenant_id_id ON finance_accounts (tenant_id, id);
CREATE UNIQUE INDEX IF NOT EXISTS uq_finance_categories_tenant_id_id ON finance_categories (tenant_id, id);
CREATE UNIQUE INDEX IF NOT EXISTS uq_finance_transactions_tenant_id_id ON finance_transactions (tenant_id, id);
CREATE UNIQUE INDEX IF NOT EXISTS uq_files_tenant_id_id ON files (tenant_id, id);

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_finance_transactions_tenant_account'
    ) THEN
        ALTER TABLE finance_transactions
            ADD CONSTRAINT fk_finance_transactions_tenant_account
            FOREIGN KEY (tenant_id, account_id)
            REFERENCES finance_accounts (tenant_id, id);
    END IF;
END $$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_finance_transactions_tenant_category'
    ) THEN
        ALTER TABLE finance_transactions
            ADD CONSTRAINT fk_finance_transactions_tenant_category
            FOREIGN KEY (tenant_id, category_id)
            REFERENCES finance_categories (tenant_id, id);
    END IF;
END $$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_finance_trx_attachments_tenant_transaction'
    ) THEN
        ALTER TABLE finance_transaction_attachments
            ADD CONSTRAINT fk_finance_trx_attachments_tenant_transaction
            FOREIGN KEY (tenant_id, transaction_id)
            REFERENCES finance_transactions (tenant_id, id);
    END IF;
END $$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'fk_finance_trx_attachments_tenant_file'
    ) THEN
        ALTER TABLE finance_transaction_attachments
            ADD CONSTRAINT fk_finance_trx_attachments_tenant_file
            FOREIGN KEY (tenant_id, file_id)
            REFERENCES files (tenant_id, id);
    END IF;
END $$;
