CREATE SEQUENCE IF NOT EXISTS finance_savings_sid_seq START 1;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = 'public'
          AND table_name = 'finance_savings'
          AND column_name = 'sid'
    ) THEN
        ALTER TABLE finance_savings
            ADD COLUMN sid VARCHAR(20);
    END IF;
END $$;

UPDATE finance_savings
SET sid = 'SVG-' || LPAD(nextval('finance_savings_sid_seq')::text, 6, '0')
WHERE sid IS NULL;

ALTER TABLE finance_savings
    ALTER COLUMN sid SET DEFAULT ('SVG-' || LPAD(nextval('finance_savings_sid_seq')::text, 6, '0'));

ALTER TABLE finance_savings
    ALTER COLUMN sid SET NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS uq_finance_savings_tenant_sid
    ON finance_savings (tenant_id, sid)
    WHERE deleted_at IS NULL;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = 'public'
          AND table_name = 'finance_transaction_savings_links'
          AND column_name = 'allocated_amount_minor'
    ) THEN
        ALTER TABLE finance_transaction_savings_links
            ADD COLUMN allocated_amount_minor BIGINT NOT NULL DEFAULT 0;
    END IF;
END $$;

WITH ranked_links AS (
    SELECT
        l.id,
        GREATEST(t.amount_minor, 0) AS amount_minor,
        COUNT(*) OVER (PARTITION BY l.tenant_id, l.transaction_id) AS link_count,
        ROW_NUMBER() OVER (PARTITION BY l.tenant_id, l.transaction_id ORDER BY l.created_at, l.id) AS row_num
    FROM finance_transaction_savings_links l
    JOIN finance_transactions t
      ON t.tenant_id = l.tenant_id
     AND t.id = l.transaction_id
    WHERE l.allocated_amount_minor = 0
      AND t.deleted_at IS NULL
)
UPDATE finance_transaction_savings_links l
SET allocated_amount_minor = (
    CASE
        WHEN rl.link_count <= 0 THEN 0
        ELSE (rl.amount_minor / rl.link_count)
             + CASE WHEN rl.row_num <= (rl.amount_minor % rl.link_count) THEN 1 ELSE 0 END
    END
)
FROM ranked_links rl
WHERE l.id = rl.id;

INSERT INTO permissions (id, code, name, module_code, action, created_at)
VALUES
    (uuid_generate_v4(), 'finance.reports.delete', 'Delete report', 'finance.reports', 'delete', now())
ON CONFLICT (code) DO NOTHING;
