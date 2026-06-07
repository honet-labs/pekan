-- 1. Create Schema (Clean Start)
DROP SCHEMA IF EXISTS wkspid_pekan_default CASCADE;
CREATE SCHEMA wkspid_pekan_default;

-- 2. Switch to schema context for table creation
SET search_path TO wkspid_pekan_default;

-- 3. Create Tables (Aligned with tenant_init.sql)
CREATE TABLE IF NOT EXISTS finance_accounts (
    id UUID PRIMARY KEY,
    name VARCHAR(200) NOT NULL,
    account_type VARCHAR(50) NOT NULL,
    currency CHAR(3) NOT NULL DEFAULT 'IDR',
    opening_balance_minor BIGINT NOT NULL DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_by UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS finance_categories (
    id UUID PRIMARY KEY,
    name VARCHAR(200) NOT NULL,
    category_type VARCHAR(20) NOT NULL CHECK (category_type IN ('income', 'expense')),
    parent_id UUID NULL REFERENCES finance_categories(id),
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_by UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS finance_transactions (
    id UUID PRIMARY KEY,
    account_id UUID NOT NULL REFERENCES finance_accounts(id),
    category_id UUID NULL REFERENCES finance_categories(id),
    type VARCHAR(20) NOT NULL CHECK (type IN ('income', 'expense', 'transfer', 'savings')),
    amount_minor BIGINT NOT NULL CHECK (amount_minor > 0),
    currency CHAR(3) NOT NULL,
    input_date TIMESTAMPTZ NOT NULL DEFAULT now(),
    transaction_date DATE NOT NULL,
    description TEXT NULL,
    merchant_name TEXT NULL,
    receipt_number TEXT NULL,
    payment_method TEXT NULL,
    subtotal_minor BIGINT NOT NULL DEFAULT 0,
    tax_minor BIGINT NOT NULL DEFAULT 0,
    service_charge_minor BIGINT NOT NULL DEFAULT 0,
    receipt_discount_minor BIGINT NOT NULL DEFAULT 0,
    created_by UUID NOT NULL,
    updated_by UUID NOT NULL,
    deleted_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS finance_transaction_items (
    id UUID PRIMARY KEY,
    transaction_id UUID NOT NULL REFERENCES finance_transactions(id) ON DELETE CASCADE,
    item_name VARCHAR(255) NOT NULL,
    quantity DECIMAL(12,2) NOT NULL DEFAULT 1,
    price_minor BIGINT NOT NULL,
    discount_minor BIGINT NOT NULL DEFAULT 0,
    total_minor BIGINT NOT NULL,
    notes TEXT NULL,
    created_by UUID NOT NULL,
    updated_by UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS finance_savings (
    id UUID PRIMARY KEY,
    sid VARCHAR(20) NOT NULL,
    name VARCHAR(200) NOT NULL,
    target_amount_minor BIGINT NOT NULL,
    current_amount_minor BIGINT NOT NULL DEFAULT 0,
    currency CHAR(3) NOT NULL DEFAULT 'IDR',
    start_date DATE NULL,
    target_date DATE NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    notes TEXT NULL,
    created_by UUID NOT NULL,
    updated_by UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ NULL
);

CREATE TABLE IF NOT EXISTS finance_budgets (
    id UUID PRIMARY KEY,
    ida VARCHAR(20) NOT NULL,
    name VARCHAR(200) NOT NULL,
    category_id UUID NOT NULL REFERENCES finance_categories(id),
    amount_limit_minor BIGINT NOT NULL,
    currency CHAR(3) NOT NULL DEFAULT 'IDR',
    period VARCHAR(20) NOT NULL CHECK (period IN ('monthly', 'weekly', 'yearly')),
    start_date DATE NOT NULL,
    end_date DATE NULL,
    alert_threshold_pct INT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    notes TEXT NULL,
    created_by UUID NOT NULL,
    updated_by UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ NULL
);

CREATE TABLE IF NOT EXISTS finance_reminders (
    id UUID PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    description TEXT NULL,
    amount_minor BIGINT NOT NULL,
    currency CHAR(3) NOT NULL DEFAULT 'IDR',
    due_date DATE NOT NULL,
    repeat_interval VARCHAR(20) NOT NULL DEFAULT 'none',
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    total_tenor INTEGER DEFAULT 0,
    current_tenor INTEGER DEFAULT 0,
    last_triggered_at TIMESTAMPTZ NULL,
    created_by UUID NOT NULL,
    updated_by UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ NULL
);

CREATE TABLE IF NOT EXISTS finance_audit_logs (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL,
    action VARCHAR(100) NOT NULL,
    entity_type VARCHAR(100) NOT NULL,
    entity_id UUID NOT NULL,
    old_value TEXT NULL,
    new_value TEXT NULL,
    ip_address INET NULL,
    user_agent TEXT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- 4. Migrate Data from public
DO $$
DECLARE
    tid UUID;
BEGIN
    SELECT id INTO tid FROM public.tenants WHERE code = 'default' LIMIT 1;

    IF tid IS NOT NULL THEN
        -- Move Accounts
        INSERT INTO wkspid_pekan_default.finance_accounts (id, name, account_type, currency, opening_balance_minor, is_active, created_by, created_at, updated_at)
        SELECT id, name, account_type, currency, opening_balance_minor, is_active, created_by, created_at, updated_at
        FROM public.finance_accounts WHERE tenant_id = tid
        ON CONFLICT (id) DO NOTHING;

        -- Move Categories
        INSERT INTO wkspid_pekan_default.finance_categories (id, name, category_type, parent_id, is_active, created_by, created_at, updated_at)
        SELECT id, name, category_type, parent_id, is_active, created_by, created_at, updated_at
        FROM public.finance_categories WHERE tenant_id = tid
        ON CONFLICT (id) DO NOTHING;

        -- Move Transactions
        INSERT INTO wkspid_pekan_default.finance_transactions (
            id, account_id, category_id, type, amount_minor, currency, input_date, transaction_date, 
            description, merchant_name, receipt_number, payment_method, subtotal_minor, 
            tax_minor, service_charge_minor, receipt_discount_minor, created_by, updated_by, 
            deleted_at, created_at, updated_at
        )
        SELECT 
            id, account_id, category_id, type, amount_minor, currency, 
            COALESCE(created_at, now()), transaction_date, 
            description, merchant_name, receipt_number, payment_method, 
            COALESCE(subtotal_minor, amount_minor), 
            COALESCE(tax_minor, 0), 
            COALESCE(service_charge_minor, 0), 
            COALESCE(receipt_discount_minor, 0), 
            created_by, updated_by, 
            deleted_at, created_at, updated_at
        FROM public.finance_transactions WHERE tenant_id = tid
        ON CONFLICT (id) DO NOTHING;

        -- Move Transaction Items
        INSERT INTO wkspid_pekan_default.finance_transaction_items (
            id, transaction_id, item_name, quantity, price_minor, discount_minor, total_minor, notes, created_by, updated_by, created_at, updated_at
        )
        SELECT 
            ti.id, ti.transaction_id, ti.item_name, ti.quantity, 
            ti.price_per_unit_minor, 
            0, -- Default discount to 0 if not present in source
            ti.total_minor, 
            ti.notes, 
            ti.created_by,
            ti.updated_by,
            ti.created_at, 
            ti.updated_at
        FROM public.finance_transaction_items ti
        WHERE ti.transaction_id IN (SELECT t.id FROM public.finance_transactions t WHERE t.tenant_id = tid)
        ON CONFLICT (id) DO NOTHING;

        -- Move Savings
        INSERT INTO wkspid_pekan_default.finance_savings (
            id, sid, name, target_amount_minor, current_amount_minor, currency, start_date, target_date, 
            status, notes, created_by, updated_by, created_at, updated_at, deleted_at
        )
        SELECT 
            id, sid, name, target_amount_minor, current_amount_minor, currency, start_date, target_date, 
            status, NULL, created_by, updated_by, created_at, updated_at, deleted_at
        FROM public.finance_savings WHERE tenant_id = tid
        ON CONFLICT (id) DO NOTHING;

        -- Move Budgets
        INSERT INTO wkspid_pekan_default.finance_budgets (
            id, ida, name, category_id, amount_limit_minor, currency, period, start_date, end_date, 
            alert_threshold_pct, status, notes, created_by, updated_by, created_at, updated_at, deleted_at
        )
        SELECT 
            id, ida, name, category_id, amount_limit_minor, currency, period, start_date, end_date, 
            alert_threshold_pct, status, NULL, created_by, updated_by, created_at, updated_at, deleted_at
        FROM public.finance_budgets WHERE tenant_id = tid
        ON CONFLICT (id) DO NOTHING;

        -- Move Reminders
        INSERT INTO wkspid_pekan_default.finance_reminders (
            id, title, description, amount_minor, currency, due_date, repeat_interval, status, 
            total_tenor, current_tenor, last_triggered_at, created_by, updated_by, created_at, updated_at, deleted_at
        )
        SELECT 
            id, title, description, amount_minor, currency, due_date, repeat_interval, status, 
            COALESCE(total_tenor, 0), COALESCE(current_tenor, 0), last_triggered_at, created_by, updated_by, created_at, updated_at, deleted_at
        FROM public.finance_reminders WHERE tenant_id = tid
        ON CONFLICT (id) DO NOTHING;
    END IF;
END $$;
