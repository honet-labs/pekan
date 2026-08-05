-- Transaction Items: Support for itemized transactions
-- Allows users to add multiple line items to a single transaction
-- Each item has: name (description), quantity, price per unit, and total

CREATE TABLE IF NOT EXISTS finance_transaction_items (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    transaction_id UUID NOT NULL,
    item_name VARCHAR(255) NOT NULL,
    quantity NUMERIC(12, 2) NOT NULL CHECK (quantity > 0),
    price_per_unit_minor BIGINT NOT NULL CHECK (price_per_unit_minor > 0),
    total_minor BIGINT NOT NULL CHECK (total_minor > 0),
    notes TEXT NULL,
    created_by UUID NOT NULL REFERENCES users(id),
    updated_by UUID NOT NULL REFERENCES users(id),
    deleted_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT fk_transaction_items_transaction
        FOREIGN KEY (tenant_id, transaction_id)
        REFERENCES finance_transactions (tenant_id, id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_finance_transaction_items_tenant_transaction
    ON finance_transaction_items (tenant_id, transaction_id, deleted_at)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_finance_transaction_items_created_at
    ON finance_transaction_items (created_at DESC);


-- Entity Change History: Track all field-level changes for audit purposes
-- Stores before/after snapshot of changes per entity
-- owner_type: 'transaction', 'savings', 'budget', 'reminder'

CREATE TABLE IF NOT EXISTS finance_entity_change_history (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    owner_type VARCHAR(50) NOT NULL CHECK (owner_type IN ('transaction', 'savings', 'budget', 'reminder')),
    owner_id UUID NOT NULL,
    field_name VARCHAR(100) NOT NULL,
    old_value_text TEXT NULL,
    new_value_text TEXT NULL,
    change_type VARCHAR(20) NOT NULL CHECK (change_type IN ('create', 'update', 'delete')),
    changed_by UUID NOT NULL REFERENCES users(id),
    ip_address INET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_finance_entity_change_history_owner
    ON finance_entity_change_history (tenant_id, owner_type, owner_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_finance_entity_change_history_changed_by
    ON finance_entity_change_history (changed_by, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_finance_entity_change_history_field
    ON finance_entity_change_history (owner_type, field_name, created_at DESC);
