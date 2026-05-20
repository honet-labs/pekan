-- Backfill permission grants for legacy tenants after finance module expansion.
-- Ensures owner/admin roles can access newly added settings + attachment endpoints.

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

WITH p AS (SELECT id FROM products WHERE code = 'finance')
INSERT INTO modules (id, product_id, code, name, is_active, created_at, updated_at)
SELECT uuid_generate_v4(), p.id, 'finance.settings', 'Finance Settings', TRUE, now(), now() FROM p
ON CONFLICT (code) DO UPDATE
SET name = EXCLUDED.name, is_active = EXCLUDED.is_active, updated_at = now();

WITH m AS (SELECT id FROM modules WHERE code = 'finance.settings')
INSERT INTO features (id, module_id, code, name, is_active, created_at, updated_at)
SELECT uuid_generate_v4(), m.id, 'finance.settings.read', 'Read finance settings', TRUE, now(), now() FROM m
ON CONFLICT (code) DO UPDATE
SET name = EXCLUDED.name, is_active = EXCLUDED.is_active, updated_at = now();

WITH m AS (SELECT id FROM modules WHERE code = 'finance.settings')
INSERT INTO features (id, module_id, code, name, is_active, created_at, updated_at)
SELECT uuid_generate_v4(), m.id, 'finance.settings.write', 'Write finance settings', TRUE, now(), now() FROM m
ON CONFLICT (code) DO UPDATE
SET name = EXCLUDED.name, is_active = EXCLUDED.is_active, updated_at = now();

INSERT INTO role_permissions (role_id, permission_id, created_at)
SELECT r.id, p.id, now()
FROM roles r
JOIN permissions p ON p.code LIKE 'finance.%'
WHERE r.code IN ('owner', 'admin')
ON CONFLICT (role_id, permission_id) DO NOTHING;

INSERT INTO tenant_modules (id, tenant_id, module_code, is_enabled, source, created_at, updated_at)
SELECT uuid_generate_v4(), t.id, 'finance.settings', TRUE, 'manual', now(), now()
FROM tenants t
WHERE t.status = 'active'
ON CONFLICT (tenant_id, module_code) DO NOTHING;

INSERT INTO tenant_features (id, tenant_id, feature_code, is_enabled, source, created_at, updated_at)
SELECT uuid_generate_v4(), t.id, 'finance.settings.read', TRUE, 'manual', now(), now()
FROM tenants t
WHERE t.status = 'active'
ON CONFLICT (tenant_id, feature_code) DO NOTHING;

INSERT INTO tenant_features (id, tenant_id, feature_code, is_enabled, source, created_at, updated_at)
SELECT uuid_generate_v4(), t.id, 'finance.settings.write', TRUE, 'manual', now(), now()
FROM tenants t
WHERE t.status = 'active'
ON CONFLICT (tenant_id, feature_code) DO NOTHING;
