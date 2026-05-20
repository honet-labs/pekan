# Auth + RBAC Bootstrap Guide

Setelah migration `0001` sampai `0011`, lakukan bootstrap data minimum agar login, finance master data, transactions, entitlement resolver, dan attachment scan flow bisa dipakai.

## 1) Buat tenant
```sql
INSERT INTO tenants (id, code, name, status, timezone)
VALUES ('11111111-1111-1111-1111-111111111111', 'default', 'Default Tenant', 'active', 'Asia/Jakarta');
```

## 2) Buat user
Gunakan hash bcrypt untuk password (jangan plaintext).

```sql
INSERT INTO users (id, email, password_hash, full_name, is_active)
VALUES (
  '22222222-2222-2222-2222-222222222222',
  'owner@pekan.local',
  '$2a$10$replace_with_real_bcrypt_hash',
  'Owner',
  TRUE
);
```

## 3) Buat membership
```sql
INSERT INTO tenant_memberships (id, tenant_id, user_id, status, joined_at)
VALUES (
  '33333333-3333-3333-3333-333333333333',
  '11111111-1111-1111-1111-111111111111',
  '22222222-2222-2222-2222-222222222222',
  'active',
  now()
);
```

## 4) Buat role owner + assign permissions
```sql
INSERT INTO roles (id, tenant_id, code, name, is_system)
VALUES ('44444444-4444-4444-4444-444444444444', '11111111-1111-1111-1111-111111111111', 'owner', 'Owner', TRUE);

INSERT INTO role_permissions (role_id, permission_id)
SELECT '44444444-4444-4444-4444-444444444444', p.id
FROM permissions p
WHERE p.code IN (
  'finance.transactions.create',
  'finance.transactions.read',
  'finance.transactions.update',
  'finance.transactions.delete',
  'finance.transactions.attach',
  'finance.transactions.scan.manage',
  'finance.transactions.attachment.read',
  'finance.accounts.create',
  'finance.accounts.read',
  'finance.categories.create',
  'finance.categories.read',
  'core.entitlement.manage',
  'core.entitlement.read'
)
ON CONFLICT DO NOTHING;

INSERT INTO membership_roles (membership_id, role_id)
VALUES ('33333333-3333-3333-3333-333333333333', '44444444-4444-4444-4444-444444444444')
ON CONFLICT DO NOTHING;
```

## 5) Aktifkan module + feature tenant (opsional manual override)
```sql
INSERT INTO tenant_modules (id, tenant_id, module_code, is_enabled, source)
VALUES (uuid_generate_v4(), '11111111-1111-1111-1111-111111111111', 'finance', TRUE, 'manual')
ON CONFLICT (tenant_id, module_code) DO UPDATE SET is_enabled = EXCLUDED.is_enabled;

INSERT INTO tenant_features (id, tenant_id, feature_code, is_enabled, source)
VALUES
  (uuid_generate_v4(), '11111111-1111-1111-1111-111111111111', 'finance.transactions.read', TRUE, 'manual'),
  (uuid_generate_v4(), '11111111-1111-1111-1111-111111111111', 'finance.transactions.write', TRUE, 'manual'),
  (uuid_generate_v4(), '11111111-1111-1111-1111-111111111111', 'finance.masterdata.read', TRUE, 'manual'),
  (uuid_generate_v4(), '11111111-1111-1111-1111-111111111111', 'finance.masterdata.write', TRUE, 'manual')
ON CONFLICT (tenant_id, feature_code) DO UPDATE SET is_enabled = EXCLUDED.is_enabled;
```

## 6) Opsional: override feature tertentu
```sql
INSERT INTO tenant_feature_overrides (id, tenant_id, feature_id, is_enabled, reason, created_at, updated_at)
SELECT
  uuid_generate_v4(),
  '11111111-1111-1111-1111-111111111111',
  f.id,
  FALSE,
  'temporary lock',
  now(),
  now()
FROM features f
WHERE f.code = 'finance.transactions.write'
ON CONFLICT (tenant_id, feature_id)
DO UPDATE SET is_enabled = EXCLUDED.is_enabled, reason = EXCLUDED.reason, updated_at = now();
```
