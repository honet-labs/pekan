ALTER TABLE finance_transactions
    ADD COLUMN IF NOT EXISTS merchant_name TEXT,
    ADD COLUMN IF NOT EXISTS receipt_number TEXT,
    ADD COLUMN IF NOT EXISTS payment_method TEXT,
    ADD COLUMN IF NOT EXISTS subtotal_minor BIGINT NOT NULL DEFAULT 0 CHECK (subtotal_minor >= 0),
    ADD COLUMN IF NOT EXISTS tax_minor BIGINT NOT NULL DEFAULT 0 CHECK (tax_minor >= 0),
    ADD COLUMN IF NOT EXISTS service_charge_minor BIGINT NOT NULL DEFAULT 0 CHECK (service_charge_minor >= 0),
    ADD COLUMN IF NOT EXISTS receipt_discount_minor BIGINT NOT NULL DEFAULT 0 CHECK (receipt_discount_minor >= 0);
