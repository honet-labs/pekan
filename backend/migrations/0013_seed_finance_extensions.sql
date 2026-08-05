-- Seed additional finance modules/features/permissions

-- Modules
WITH p AS (SELECT id FROM products WHERE code = 'finance')
INSERT INTO modules (id, product_id, code, name, is_active, created_at, updated_at)
SELECT uuid_generate_v4(), p.id, 'finance.savings', 'Finance Savings', TRUE, now(), now() FROM p
ON CONFLICT (code) DO UPDATE SET name = EXCLUDED.name, is_active = EXCLUDED.is_active, updated_at = now();

WITH p AS (SELECT id FROM products WHERE code = 'finance')
INSERT INTO modules (id, product_id, code, name, is_active, created_at, updated_at)
SELECT uuid_generate_v4(), p.id, 'finance.budgets', 'Finance Budgets', TRUE, now(), now() FROM p
ON CONFLICT (code) DO UPDATE SET name = EXCLUDED.name, is_active = EXCLUDED.is_active, updated_at = now();

WITH p AS (SELECT id FROM products WHERE code = 'finance')
INSERT INTO modules (id, product_id, code, name, is_active, created_at, updated_at)
SELECT uuid_generate_v4(), p.id, 'finance.reminders', 'Finance Reminders', TRUE, now(), now() FROM p
ON CONFLICT (code) DO UPDATE SET name = EXCLUDED.name, is_active = EXCLUDED.is_active, updated_at = now();

WITH p AS (SELECT id FROM products WHERE code = 'finance')
INSERT INTO modules (id, product_id, code, name, is_active, created_at, updated_at)
SELECT uuid_generate_v4(), p.id, 'finance.reports', 'Finance Reports', TRUE, now(), now() FROM p
ON CONFLICT (code) DO UPDATE SET name = EXCLUDED.name, is_active = EXCLUDED.is_active, updated_at = now();

WITH p AS (SELECT id FROM products WHERE code = 'finance')
INSERT INTO modules (id, product_id, code, name, is_active, created_at, updated_at)
SELECT uuid_generate_v4(), p.id, 'finance.dashboard', 'Finance Dashboard', TRUE, now(), now() FROM p
ON CONFLICT (code) DO UPDATE SET name = EXCLUDED.name, is_active = EXCLUDED.is_active, updated_at = now();

WITH p AS (SELECT id FROM products WHERE code = 'finance')
INSERT INTO modules (id, product_id, code, name, is_active, created_at, updated_at)
SELECT uuid_generate_v4(), p.id, 'finance.notifications', 'Finance Notifications', TRUE, now(), now() FROM p
ON CONFLICT (code) DO UPDATE SET name = EXCLUDED.name, is_active = EXCLUDED.is_active, updated_at = now();

-- Features
WITH m AS (SELECT id FROM modules WHERE code = 'finance.savings')
INSERT INTO features (id, module_id, code, name, is_active, created_at, updated_at)
SELECT uuid_generate_v4(), m.id, 'finance.savings.read', 'Read Savings', TRUE, now(), now() FROM m
ON CONFLICT (code) DO UPDATE SET name = EXCLUDED.name, is_active = EXCLUDED.is_active, updated_at = now();

WITH m AS (SELECT id FROM modules WHERE code = 'finance.savings')
INSERT INTO features (id, module_id, code, name, is_active, created_at, updated_at)
SELECT uuid_generate_v4(), m.id, 'finance.savings.write', 'Write Savings', TRUE, now(), now() FROM m
ON CONFLICT (code) DO UPDATE SET name = EXCLUDED.name, is_active = EXCLUDED.is_active, updated_at = now();

WITH m AS (SELECT id FROM modules WHERE code = 'finance.budgets')
INSERT INTO features (id, module_id, code, name, is_active, created_at, updated_at)
SELECT uuid_generate_v4(), m.id, 'finance.budgets.read', 'Read Budgets', TRUE, now(), now() FROM m
ON CONFLICT (code) DO UPDATE SET name = EXCLUDED.name, is_active = EXCLUDED.is_active, updated_at = now();

WITH m AS (SELECT id FROM modules WHERE code = 'finance.budgets')
INSERT INTO features (id, module_id, code, name, is_active, created_at, updated_at)
SELECT uuid_generate_v4(), m.id, 'finance.budgets.write', 'Write Budgets', TRUE, now(), now() FROM m
ON CONFLICT (code) DO UPDATE SET name = EXCLUDED.name, is_active = EXCLUDED.is_active, updated_at = now();

WITH m AS (SELECT id FROM modules WHERE code = 'finance.reminders')
INSERT INTO features (id, module_id, code, name, is_active, created_at, updated_at)
SELECT uuid_generate_v4(), m.id, 'finance.reminders.read', 'Read Reminders', TRUE, now(), now() FROM m
ON CONFLICT (code) DO UPDATE SET name = EXCLUDED.name, is_active = EXCLUDED.is_active, updated_at = now();

WITH m AS (SELECT id FROM modules WHERE code = 'finance.reminders')
INSERT INTO features (id, module_id, code, name, is_active, created_at, updated_at)
SELECT uuid_generate_v4(), m.id, 'finance.reminders.write', 'Write Reminders', TRUE, now(), now() FROM m
ON CONFLICT (code) DO UPDATE SET name = EXCLUDED.name, is_active = EXCLUDED.is_active, updated_at = now();

WITH m AS (SELECT id FROM modules WHERE code = 'finance.reports')
INSERT INTO features (id, module_id, code, name, is_active, created_at, updated_at)
SELECT uuid_generate_v4(), m.id, 'finance.reports.read', 'Read Reports', TRUE, now(), now() FROM m
ON CONFLICT (code) DO UPDATE SET name = EXCLUDED.name, is_active = EXCLUDED.is_active, updated_at = now();

WITH m AS (SELECT id FROM modules WHERE code = 'finance.reports')
INSERT INTO features (id, module_id, code, name, is_active, created_at, updated_at)
SELECT uuid_generate_v4(), m.id, 'finance.reports.write', 'Write Reports', TRUE, now(), now() FROM m
ON CONFLICT (code) DO UPDATE SET name = EXCLUDED.name, is_active = EXCLUDED.is_active, updated_at = now();

WITH m AS (SELECT id FROM modules WHERE code = 'finance.dashboard')
INSERT INTO features (id, module_id, code, name, is_active, created_at, updated_at)
SELECT uuid_generate_v4(), m.id, 'finance.dashboard.read', 'Read Dashboard', TRUE, now(), now() FROM m
ON CONFLICT (code) DO UPDATE SET name = EXCLUDED.name, is_active = EXCLUDED.is_active, updated_at = now();

WITH m AS (SELECT id FROM modules WHERE code = 'finance.notifications')
INSERT INTO features (id, module_id, code, name, is_active, created_at, updated_at)
SELECT uuid_generate_v4(), m.id, 'finance.notifications.read', 'Read Notifications', TRUE, now(), now() FROM m
ON CONFLICT (code) DO UPDATE SET name = EXCLUDED.name, is_active = EXCLUDED.is_active, updated_at = now();

WITH m AS (SELECT id FROM modules WHERE code = 'finance.notifications')
INSERT INTO features (id, module_id, code, name, is_active, created_at, updated_at)
SELECT uuid_generate_v4(), m.id, 'finance.notifications.write', 'Manage Notifications', TRUE, now(), now() FROM m
ON CONFLICT (code) DO UPDATE SET name = EXCLUDED.name, is_active = EXCLUDED.is_active, updated_at = now();

-- Permissions
INSERT INTO permissions (id, code, name, module_code, action, created_at)
VALUES
    (uuid_generate_v4(), 'finance.savings.create', 'Create savings goal', 'finance.savings', 'create', now()),
    (uuid_generate_v4(), 'finance.savings.read', 'Read savings goal', 'finance.savings', 'read', now()),
    (uuid_generate_v4(), 'finance.savings.update', 'Update savings goal', 'finance.savings', 'update', now()),
    (uuid_generate_v4(), 'finance.savings.delete', 'Delete savings goal', 'finance.savings', 'delete', now()),

    (uuid_generate_v4(), 'finance.budgets.create', 'Create budget', 'finance.budgets', 'create', now()),
    (uuid_generate_v4(), 'finance.budgets.read', 'Read budget', 'finance.budgets', 'read', now()),
    (uuid_generate_v4(), 'finance.budgets.update', 'Update budget', 'finance.budgets', 'update', now()),
    (uuid_generate_v4(), 'finance.budgets.delete', 'Delete budget', 'finance.budgets', 'delete', now()),

    (uuid_generate_v4(), 'finance.reminders.create', 'Create reminder', 'finance.reminders', 'create', now()),
    (uuid_generate_v4(), 'finance.reminders.read', 'Read reminder', 'finance.reminders', 'read', now()),
    (uuid_generate_v4(), 'finance.reminders.update', 'Update reminder', 'finance.reminders', 'update', now()),
    (uuid_generate_v4(), 'finance.reminders.delete', 'Delete reminder', 'finance.reminders', 'delete', now()),
    (uuid_generate_v4(), 'finance.reminders.mark', 'Mark reminder status', 'finance.reminders', 'update', now()),

    (uuid_generate_v4(), 'finance.reports.create', 'Create report', 'finance.reports', 'create', now()),
    (uuid_generate_v4(), 'finance.reports.read', 'Read report', 'finance.reports', 'read', now()),
    (uuid_generate_v4(), 'finance.reports.download', 'Download report', 'finance.reports', 'read', now()),

    (uuid_generate_v4(), 'finance.dashboard.read', 'Read finance dashboard', 'finance.dashboard', 'read', now()),

    (uuid_generate_v4(), 'finance.notifications.create', 'Create notification', 'finance.notifications', 'create', now()),
    (uuid_generate_v4(), 'finance.notifications.read', 'Read notifications', 'finance.notifications', 'read', now()),
    (uuid_generate_v4(), 'finance.notifications.update', 'Update notification', 'finance.notifications', 'update', now())
ON CONFLICT (code) DO NOTHING;
