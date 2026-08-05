ALTER TABLE files
    ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ NULL;

CREATE INDEX IF NOT EXISTS idx_files_tenant_owner_active
    ON files (tenant_id, owner_type, owner_id, created_at DESC)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_audit_logs_tenant_action_created
    ON audit_logs (tenant_id, action, created_at DESC);

CREATE TABLE IF NOT EXISTS finance_entity_attachments (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    owner_type VARCHAR(30) NOT NULL CHECK (owner_type IN ('savings', 'budgets', 'reminders')),
    owner_id UUID NOT NULL,
    file_id UUID NOT NULL REFERENCES files(id),
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ NULL,
    deleted_by UUID NULL REFERENCES users(id),
    UNIQUE (tenant_id, owner_type, owner_id, file_id)
);

CREATE INDEX IF NOT EXISTS idx_finance_entity_attachments_owner
    ON finance_entity_attachments (tenant_id, owner_type, owner_id, created_at DESC)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_finance_entity_attachments_file
    ON finance_entity_attachments (file_id);

CREATE TABLE IF NOT EXISTS finance_notification_channels (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    channel_code VARCHAR(40) NOT NULL CHECK (
        channel_code IN ('email', 'telegram', 'whatsapp_official', 'whatsapp_gowa', 'whatsapp_fonte')
    ),
    is_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    config_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_by UUID NULL REFERENCES users(id),
    updated_by UUID NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, channel_code)
);

CREATE INDEX IF NOT EXISTS idx_finance_notification_channels_tenant
    ON finance_notification_channels (tenant_id, channel_code);

CREATE TABLE IF NOT EXISTS finance_message_templates (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    template_code VARCHAR(100) NOT NULL,
    channel_code VARCHAR(40) NOT NULL CHECK (
        channel_code IN ('any', 'email', 'telegram', 'whatsapp_official', 'whatsapp_gowa', 'whatsapp_fonte')
    ),
    language_code VARCHAR(8) NOT NULL DEFAULT 'id',
    title_template VARCHAR(255) NULL,
    body_template TEXT NOT NULL,
    is_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_by UUID NULL REFERENCES users(id),
    updated_by UUID NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, template_code, channel_code, language_code)
);

CREATE INDEX IF NOT EXISTS idx_finance_message_templates_tenant_code
    ON finance_message_templates (tenant_id, template_code, channel_code, language_code);

CREATE INDEX IF NOT EXISTS idx_tenant_memberships_tenant_status_user
    ON tenant_memberships (tenant_id, status, user_id);

CREATE INDEX IF NOT EXISTS idx_membership_roles_role
    ON membership_roles (role_id);

CREATE INDEX IF NOT EXISTS idx_finance_reports_tenant_type_created
    ON finance_reports (tenant_id, report_type, created_at DESC);

WITH p AS (SELECT id FROM products WHERE code = 'finance')
INSERT INTO modules (id, product_id, code, name, is_active, created_at, updated_at)
SELECT uuid_generate_v4(), p.id, 'finance.settings', 'Finance Settings', TRUE, now(), now() FROM p
ON CONFLICT (code) DO UPDATE SET name = EXCLUDED.name, is_active = EXCLUDED.is_active, updated_at = now();

WITH m AS (SELECT id FROM modules WHERE code = 'finance.settings')
INSERT INTO features (id, module_id, code, name, is_active, created_at, updated_at)
SELECT uuid_generate_v4(), m.id, 'finance.settings.read', 'Read finance settings', TRUE, now(), now() FROM m
ON CONFLICT (code) DO UPDATE SET name = EXCLUDED.name, is_active = EXCLUDED.is_active, updated_at = now();

WITH m AS (SELECT id FROM modules WHERE code = 'finance.settings')
INSERT INTO features (id, module_id, code, name, is_active, created_at, updated_at)
SELECT uuid_generate_v4(), m.id, 'finance.settings.write', 'Write finance settings', TRUE, now(), now() FROM m
ON CONFLICT (code) DO UPDATE SET name = EXCLUDED.name, is_active = EXCLUDED.is_active, updated_at = now();

INSERT INTO permissions (id, code, name, module_code, action, created_at)
VALUES
    (uuid_generate_v4(), 'finance.savings.attach', 'Upload savings attachment', 'finance.savings', 'attach', now()),
    (uuid_generate_v4(), 'finance.savings.attachment.read', 'Read savings attachment', 'finance.savings', 'attachment.read', now()),
    (uuid_generate_v4(), 'finance.budgets.attach', 'Upload budget attachment', 'finance.budgets', 'attach', now()),
    (uuid_generate_v4(), 'finance.budgets.attachment.read', 'Read budget attachment', 'finance.budgets', 'attachment.read', now()),
    (uuid_generate_v4(), 'finance.reminders.attach', 'Upload reminder attachment', 'finance.reminders', 'attach', now()),
    (uuid_generate_v4(), 'finance.reminders.attachment.read', 'Read reminder attachment', 'finance.reminders', 'attachment.read', now()),
    (uuid_generate_v4(), 'finance.settings.read', 'Read finance settings', 'finance.settings', 'read', now()),
    (uuid_generate_v4(), 'finance.settings.update', 'Update finance settings', 'finance.settings', 'update', now()),
    (uuid_generate_v4(), 'finance.settings.roles.manage', 'Manage tenant roles', 'finance.settings', 'manage_roles', now()),
    (uuid_generate_v4(), 'finance.settings.audit.read', 'Read audit logs', 'finance.settings', 'read_audit', now())
ON CONFLICT (code) DO NOTHING;

