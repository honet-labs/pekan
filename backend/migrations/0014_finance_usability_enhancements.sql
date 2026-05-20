DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = 'public'
          AND table_name = 'finance_savings'
          AND column_name = 'progress_percent'
    ) THEN
        ALTER TABLE finance_savings
            ADD COLUMN progress_percent NUMERIC(7,2)
            GENERATED ALWAYS AS (
                CASE
                    WHEN target_amount_minor > 0 THEN ROUND((current_amount_minor::numeric * 100.0) / target_amount_minor::numeric, 2)
                    ELSE 0
                END
            ) STORED;
    END IF;
END $$;

ALTER TABLE finance_transactions
    ADD COLUMN IF NOT EXISTS input_date DATE;

UPDATE finance_transactions
SET input_date = COALESCE(input_date, created_at::date)
WHERE input_date IS NULL;

ALTER TABLE finance_transactions
    ALTER COLUMN input_date SET DEFAULT CURRENT_DATE;

ALTER TABLE finance_transactions
    ALTER COLUMN input_date SET NOT NULL;

CREATE INDEX IF NOT EXISTS idx_finance_transactions_tenant_input_date
    ON finance_transactions (tenant_id, input_date DESC)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_finance_transactions_tenant_type_date
    ON finance_transactions (tenant_id, type, transaction_date DESC)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_finance_categories_tenant_type_name
    ON finance_categories (tenant_id, category_type, name);
