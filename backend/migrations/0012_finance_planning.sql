CREATE TABLE IF NOT EXISTS finance_savings (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    name VARCHAR(150) NOT NULL,
    target_amount_minor BIGINT NOT NULL,
    current_amount_minor BIGINT NOT NULL DEFAULT 0,
    currency VARCHAR(10) NOT NULL,
    start_date DATE NULL,
    target_date DATE NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    created_by UUID NULL REFERENCES users(id),
    updated_by UUID NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ NULL
);

CREATE INDEX IF NOT EXISTS idx_finance_savings_tenant_status ON finance_savings (tenant_id, status);
CREATE INDEX IF NOT EXISTS idx_finance_savings_tenant_target_date ON finance_savings (tenant_id, target_date);

CREATE TABLE IF NOT EXISTS finance_budgets (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    name VARCHAR(150) NOT NULL,
    category_id UUID NULL REFERENCES finance_categories(id),
    amount_limit_minor BIGINT NOT NULL,
    currency VARCHAR(10) NOT NULL,
    period VARCHAR(20) NOT NULL,
    start_date DATE NOT NULL,
    end_date DATE NULL,
    alert_threshold_pct INT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    created_by UUID NULL REFERENCES users(id),
    updated_by UUID NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ NULL
);

CREATE INDEX IF NOT EXISTS idx_finance_budgets_tenant_status ON finance_budgets (tenant_id, status);
CREATE INDEX IF NOT EXISTS idx_finance_budgets_tenant_period ON finance_budgets (tenant_id, start_date, end_date);

CREATE TABLE IF NOT EXISTS finance_reminders (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    title VARCHAR(200) NOT NULL,
    description TEXT NULL,
    amount_minor BIGINT NULL,
    currency VARCHAR(10) NULL,
    due_date DATE NOT NULL,
    repeat_interval VARCHAR(20) NOT NULL DEFAULT 'none',
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    last_triggered_at TIMESTAMPTZ NULL,
    created_by UUID NULL REFERENCES users(id),
    updated_by UUID NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ NULL
);

CREATE INDEX IF NOT EXISTS idx_finance_reminders_tenant_due ON finance_reminders (tenant_id, due_date, status);

CREATE TABLE IF NOT EXISTS finance_notifications (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    notification_type VARCHAR(60) NOT NULL,
    title VARCHAR(200) NOT NULL,
    message TEXT NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'unread',
    metadata JSONB NULL,
    created_by UUID NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    read_at TIMESTAMPTZ NULL
);

CREATE INDEX IF NOT EXISTS idx_finance_notifications_tenant_status ON finance_notifications (tenant_id, status, created_at DESC);

CREATE TABLE IF NOT EXISTS finance_reports (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    report_type VARCHAR(60) NOT NULL,
    format VARCHAR(10) NOT NULL,
    status VARCHAR(20) NOT NULL,
    params JSONB NOT NULL DEFAULT '{}'::jsonb,
    storage_provider VARCHAR(30) NULL,
    storage_key VARCHAR(255) NULL,
    created_by UUID NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_finance_reports_tenant_created ON finance_reports (tenant_id, created_at DESC);
