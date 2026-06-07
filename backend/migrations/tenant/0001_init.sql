-- Tenant Schema Initialization
-- Defines all tables required for a tenant-specific isolated schema.

-- 1. Memberships and Access Control
CREATE TABLE IF NOT EXISTS tenant_memberships (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES public.users(id) ON DELETE CASCADE,
    status VARCHAR(20) NOT NULL CHECK (status IN ('active', 'invited', 'suspended')),
    joined_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS roles (
    id UUID PRIMARY KEY,
    code VARCHAR(100) NOT NULL UNIQUE,
    name VARCHAR(150) NOT NULL,
    is_system BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS role_permissions (
    role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    permission_id UUID NOT NULL REFERENCES public.permissions(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (role_id, permission_id)
);

CREATE TABLE IF NOT EXISTS membership_roles (
    membership_id UUID NOT NULL REFERENCES tenant_memberships(id) ON DELETE CASCADE,
    role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (membership_id, role_id)
);

-- 2. Finance Tables (Core)
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
    parent_id UUID NULL REFERENCES finance_categories(id) ON DELETE SET NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_by UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS finance_transactions (
    id UUID PRIMARY KEY,
    account_id UUID NOT NULL REFERENCES finance_accounts(id) ON DELETE CASCADE,
    category_id UUID NULL REFERENCES finance_categories(id) ON DELETE SET NULL,
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
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    notes TEXT NULL
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

-- 3. Savings, Budgets and Reminders
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
    category_id TEXT NOT NULL,
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

CREATE TABLE IF NOT EXISTS finance_reminder_payments (
    id UUID PRIMARY KEY,
    reminder_id UUID NOT NULL REFERENCES finance_reminders(id) ON DELETE CASCADE,
    paid_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    amount_minor BIGINT NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'completed',
    notes TEXT NULL,
    proof_image_url TEXT NULL,
    created_by UUID NOT NULL,
    updated_by UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- 4. Attachments, Receipts and Links
CREATE TABLE IF NOT EXISTS finance_entity_attachments (
    id UUID PRIMARY KEY,
    owner_type VARCHAR(50) NOT NULL,
    owner_id UUID NOT NULL,
    file_id UUID NOT NULL REFERENCES public.files(id) ON DELETE CASCADE,
    created_by UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (owner_type, owner_id, file_id)
);

CREATE TABLE IF NOT EXISTS finance_receipt_scan_jobs (
    id UUID PRIMARY KEY,
    file_id UUID NOT NULL REFERENCES public.files(id) ON DELETE CASCADE,
    status VARCHAR(20) NOT NULL,
    result_json JSONB NULL,
    error_message TEXT NULL,
    created_by UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS finance_transaction_savings_links (
    id UUID PRIMARY KEY,
    transaction_id UUID NOT NULL REFERENCES finance_transactions(id) ON DELETE CASCADE,
    savings_id UUID NOT NULL REFERENCES finance_savings(id) ON DELETE CASCADE,
    allocated_amount_minor BIGINT NOT NULL DEFAULT 0,
    created_by UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (transaction_id, savings_id)
);

-- 5. WhatsApp Integration & Settings
CREATE TABLE IF NOT EXISTS whatsapp_otp_tokens (
    token VARCHAR(10) PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES public.users(id) ON DELETE CASCADE,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS whatsapp_sessions (
    phone_number VARCHAR(20) PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES public.users(id) ON DELETE CASCADE,
    last_active TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS finance_settings (
    default_currency CHAR(3) NOT NULL DEFAULT 'IDR',
    fiscal_year_start_month INTEGER NOT NULL DEFAULT 1,
    multi_currency_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    id_fixed INTEGER PRIMARY KEY DEFAULT 1 CHECK (id_fixed = 1)
);

CREATE TABLE IF NOT EXISTS finance_audit_logs (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES public.users(id) ON DELETE CASCADE,
    action VARCHAR(100) NOT NULL,
    entity_type VARCHAR(100) NOT NULL,
    entity_id UUID NOT NULL,
    old_value TEXT NULL,
    new_value TEXT NULL,
    ip_address INET NULL,
    user_agent TEXT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
