ALTER TABLE finance_transaction_items
    ADD COLUMN IF NOT EXISTS discount_minor BIGINT NOT NULL DEFAULT 0 CHECK (discount_minor >= 0);
