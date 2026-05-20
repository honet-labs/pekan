-- Migration: 0033_finance_reminder_payments.sql
CREATE TABLE IF NOT EXISTS finance_reminder_payments (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    reminder_id UUID NOT NULL REFERENCES finance_reminders(id) ON DELETE CASCADE,
    paid_at DATE NOT NULL,
    amount_minor BIGINT NOT NULL,
    status VARCHAR(50) NOT NULL, -- 'paid', 'partially_paid'
    notes TEXT,
    proof_image_url TEXT,
    created_by UUID,
    updated_by UUID,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_reminder_payments_reminder_id ON finance_reminder_payments(reminder_id);
CREATE INDEX IF NOT EXISTS idx_reminder_payments_tenant_id ON finance_reminder_payments(tenant_id);
