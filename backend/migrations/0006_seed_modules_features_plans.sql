-- Seed module/feature catalog for free SaaS (no subscription/plan tables used)

-- Products
INSERT INTO products (id, code, name, is_active, created_at, updated_at)
VALUES (uuid_generate_v4(), 'finance', 'Finance', TRUE, now(), now())
ON CONFLICT (code) DO UPDATE SET name = EXCLUDED.name, is_active = EXCLUDED.is_active, updated_at = now();

-- Modules
WITH p AS (SELECT id FROM products WHERE code = 'finance')
INSERT INTO modules (id, product_id, code, name, is_active, created_at, updated_at)
SELECT uuid_generate_v4(), p.id, 'finance', 'Finance Core', TRUE, now(), now() FROM p
ON CONFLICT (code) DO UPDATE SET name = EXCLUDED.name, is_active = EXCLUDED.is_active, updated_at = now();

WITH p AS (SELECT id FROM products WHERE code = 'finance')
INSERT INTO modules (id, product_id, code, name, is_active, created_at, updated_at)
SELECT uuid_generate_v4(), p.id, 'finance.masterdata', 'Finance Master Data', TRUE, now(), now() FROM p
ON CONFLICT (code) DO UPDATE SET name = EXCLUDED.name, is_active = EXCLUDED.is_active, updated_at = now();

WITH p AS (SELECT id FROM products WHERE code = 'finance')
INSERT INTO modules (id, product_id, code, name, is_active, created_at, updated_at)
SELECT uuid_generate_v4(), p.id, 'finance.transactions', 'Finance Transactions', TRUE, now(), now() FROM p
ON CONFLICT (code) DO UPDATE SET name = EXCLUDED.name, is_active = EXCLUDED.is_active, updated_at = now();

-- Features
WITH m AS (SELECT id FROM modules WHERE code = 'finance.transactions')
INSERT INTO features (id, module_id, code, name, is_active, created_at, updated_at)
SELECT uuid_generate_v4(), m.id, 'finance.transactions.read', 'Read Transactions', TRUE, now(), now() FROM m
ON CONFLICT (code) DO UPDATE SET name = EXCLUDED.name, is_active = EXCLUDED.is_active, updated_at = now();

WITH m AS (SELECT id FROM modules WHERE code = 'finance.transactions')
INSERT INTO features (id, module_id, code, name, is_active, created_at, updated_at)
SELECT uuid_generate_v4(), m.id, 'finance.transactions.write', 'Write Transactions', TRUE, now(), now() FROM m
ON CONFLICT (code) DO UPDATE SET name = EXCLUDED.name, is_active = EXCLUDED.is_active, updated_at = now();

WITH m AS (SELECT id FROM modules WHERE code = 'finance.masterdata')
INSERT INTO features (id, module_id, code, name, is_active, created_at, updated_at)
SELECT uuid_generate_v4(), m.id, 'finance.masterdata.read', 'Read Finance Master Data', TRUE, now(), now() FROM m
ON CONFLICT (code) DO UPDATE SET name = EXCLUDED.name, is_active = EXCLUDED.is_active, updated_at = now();

WITH m AS (SELECT id FROM modules WHERE code = 'finance.masterdata')
INSERT INTO features (id, module_id, code, name, is_active, created_at, updated_at)
SELECT uuid_generate_v4(), m.id, 'finance.masterdata.write', 'Write Finance Master Data', TRUE, now(), now() FROM m
ON CONFLICT (code) DO UPDATE SET name = EXCLUDED.name, is_active = EXCLUDED.is_active, updated_at = now();

WITH m AS (SELECT id FROM modules WHERE code = 'finance')
INSERT INTO features (id, module_id, code, name, is_active, created_at, updated_at)
SELECT uuid_generate_v4(), m.id, 'core.entitlement.read', 'Read effective entitlements', TRUE, now(), now() FROM m
ON CONFLICT (code) DO UPDATE SET name = EXCLUDED.name, is_active = EXCLUDED.is_active, updated_at = now();

WITH m AS (SELECT id FROM modules WHERE code = 'finance')
INSERT INTO features (id, module_id, code, name, is_active, created_at, updated_at)
SELECT uuid_generate_v4(), m.id, 'core.entitlement.manage', 'Manage feature overrides', TRUE, now(), now() FROM m
ON CONFLICT (code) DO UPDATE SET name = EXCLUDED.name, is_active = EXCLUDED.is_active, updated_at = now();

-- Permissions
INSERT INTO permissions (id, code, name, module_code, action, created_at)
VALUES
    (uuid_generate_v4(), 'finance.accounts.create', 'Create finance account', 'finance.masterdata', 'create', now()),
    (uuid_generate_v4(), 'finance.accounts.read', 'Read finance account', 'finance.masterdata', 'read', now()),
    (uuid_generate_v4(), 'finance.categories.create', 'Create finance category', 'finance.masterdata', 'create', now()),
    (uuid_generate_v4(), 'finance.categories.read', 'Read finance category', 'finance.masterdata', 'read', now()),
    (uuid_generate_v4(), 'core.entitlement.read', 'Read tenant effective entitlements', 'core.entitlement', 'read', now()),
    (uuid_generate_v4(), 'core.entitlement.manage', 'Manage tenant feature override', 'core.entitlement', 'manage', now())
ON CONFLICT (code) DO NOTHING;

