# PEKAN SaaS PostgreSQL Schema (Tenant-Aware)

## 1) Desain Umum

- Model: shared database, shared schema, tenant isolation by `tenant_id`.
- Semua tabel domain penting wajib punya `tenant_id`.
- Gunakan `uuid` sebagai PK untuk distribusi dan keamanan enumeration.
- Timestamp standar: `created_at`, `updated_at`, optional `deleted_at` (soft delete).
- Semua query domain wajib ada filter `tenant_id`.

## 2) Core Platform Tables

### `tenants`
- `id uuid pk`
- `code varchar(50) unique`
- `name varchar(200)`
- `status varchar(20)` (`active|suspended|deleted`)
- `timezone varchar(64)`
- `created_at timestamptz`
- `updated_at timestamptz`

### `users`
- `id uuid pk`
- `email varchar(255) unique`
- `password_hash varchar(255)`
- `full_name varchar(200)`
- `is_active boolean`
- `last_login_at timestamptz null`
- `created_at timestamptz`
- `updated_at timestamptz`

### `tenant_memberships`
- `id uuid pk`
- `tenant_id uuid fk -> tenants.id`
- `user_id uuid fk -> users.id`
- `status varchar(20)` (`active|invited|suspended`)
- `joined_at timestamptz`
- `created_at timestamptz`
- Unique: `(tenant_id, user_id)`

### `roles`
- `id uuid pk`
- `tenant_id uuid fk -> tenants.id` (allow tenant custom role)
- `code varchar(100)`
- `name varchar(150)`
- `is_system boolean`
- Unique: `(tenant_id, code)`

### `permissions`
- `id uuid pk`
- `code varchar(150) unique` (e.g. `finance.transactions.create`)
- `name varchar(150)`
- `module_code varchar(100)`
- `action varchar(50)`

### `role_permissions`
- `role_id uuid fk -> roles.id`
- `permission_id uuid fk -> permissions.id`
- PK: `(role_id, permission_id)`

### `membership_roles`
- `membership_id uuid fk -> tenant_memberships.id`
- `role_id uuid fk -> roles.id`
- PK: `(membership_id, role_id)`

### `auth_sessions`
- `id uuid pk`
- `user_id uuid fk -> users.id`
- `tenant_id uuid fk -> tenants.id`
- `refresh_token_hash varchar(255)`
- `expires_at timestamptz`
- `revoked_at timestamptz null`
- `ip_address inet`
- `user_agent text`
- Index: `(user_id, tenant_id, revoked_at)`

### `products`
- `id uuid pk`
- `code varchar(100) unique` (e.g. `finance`)
- `name varchar(150)`
- `is_active boolean`

### `modules`
- `id uuid pk`
- `product_id uuid fk -> products.id`
- `code varchar(100) unique` (e.g. `finance.transactions`)
- `name varchar(150)`
- `is_active boolean`

### `features`
- `id uuid pk`
- `module_id uuid fk -> modules.id`
- `code varchar(150) unique` (e.g. `finance.transactions.write`)
- `name varchar(150)`
- `is_active boolean`

### `tenant_feature_overrides`
- `id uuid pk`
- `tenant_id uuid fk -> tenants.id`
- `feature_id uuid fk -> features.id`
- `is_enabled boolean`
- `reason varchar(255)`
- `expires_at timestamptz null`
- Unique: `(tenant_id, feature_id)`

### `tenant_modules`
- `id uuid pk`
- `tenant_id uuid fk -> tenants.id`
- `module_code varchar(120)`
- `is_enabled boolean`
- `source varchar(20)` (`override|manual`)
- Unique: `(tenant_id, module_code)`

### `audit_logs`
- `id bigserial pk`
- `tenant_id uuid null` (null for global/system events)
- `actor_user_id uuid null fk -> users.id`
- `action varchar(150)`
- `resource_type varchar(100)`
- `resource_id varchar(100)`
- `before_json jsonb null`
- `after_json jsonb null`
- `ip_address inet`
- `user_agent text`
- `request_id varchar(100)`
- `created_at timestamptz`
- Index: `(tenant_id, created_at desc)`
- Index: `(actor_user_id, created_at desc)`
- Index: `(action, created_at desc)`

### `files`
- `id uuid pk`
- `tenant_id uuid fk -> tenants.id`
- `module_code varchar(100)` (e.g. `finance.transactions`)
- `owner_type varchar(100)` (e.g. `transaction`)
- `owner_id uuid`
- `provider varchar(20)` (`local|s3|gdrive`)
- `bucket_or_container varchar(255) null`
- `object_key varchar(500)`
- `original_filename varchar(255)`
- `stored_filename varchar(255)`
- `mime_type varchar(120)`
- `size_bytes bigint`
- `checksum_sha256 char(64)`
- `scan_status varchar(20)` (`pending|clean|infected|failed`)
- `uploaded_by uuid fk -> users.id`
- `created_at timestamptz`
- Index: `(tenant_id, owner_type, owner_id)`
- Unique (recommended): `(tenant_id, provider, object_key)`

## 3) Finance Module Tables

### `finance_accounts`
- `id uuid pk`
- `tenant_id uuid fk -> tenants.id`
- `name varchar(150)`
- `account_type varchar(30)` (`cash|bank|ewallet|credit`)
- `currency char(3)`
- `opening_balance numeric(18,2)`
- `is_active boolean`
- `created_by uuid fk -> users.id`
- `created_at timestamptz`
- `updated_at timestamptz`
- Unique: `(tenant_id, name)`

### `finance_categories`
- `id uuid pk`
- `tenant_id uuid fk -> tenants.id`
- `name varchar(150)`
- `category_type varchar(20)` (`income|expense`)
- `parent_id uuid null fk -> finance_categories.id`
- `is_active boolean`
- Unique: `(tenant_id, category_type, name)`

### `finance_transactions`
- `id uuid pk`
- `tenant_id uuid fk -> tenants.id`
- `account_id uuid fk -> finance_accounts.id`
- `category_id uuid null fk -> finance_categories.id`
- `type varchar(20)` (`income|expense|transfer`)
- `amount numeric(18,2)` check (`amount > 0`)
- `currency char(3)`
- `transaction_date date`
- `description text null`
- `reference_no varchar(100) null`
- `created_by uuid fk -> users.id`
- `updated_by uuid fk -> users.id`
- `deleted_at timestamptz null`
- `created_at timestamptz`
- `updated_at timestamptz`
- Index: `(tenant_id, transaction_date desc, created_at desc)`
- Index: `(tenant_id, account_id, transaction_date desc)`
- Index: `(tenant_id, category_id, transaction_date desc)`

### `finance_transaction_attachments`
- `id uuid pk`
- `tenant_id uuid fk -> tenants.id`
- `transaction_id uuid fk -> finance_transactions.id`
- `file_id uuid fk -> files.id`
- `created_by uuid fk -> users.id`
- `created_at timestamptz`
- Unique: `(tenant_id, transaction_id, file_id)`

### `finance_savings_goals`
- `id uuid pk`
- `tenant_id uuid fk -> tenants.id`
- `name varchar(150)`
- `target_amount numeric(18,2)`
- `current_amount numeric(18,2)`
- `target_date date null`
- `status varchar(20)` (`active|achieved|paused|cancelled`)
- `created_at timestamptz`
- `updated_at timestamptz`
- Index: `(tenant_id, status, target_date)`

### `finance_budgets`
- `id uuid pk`
- `tenant_id uuid fk -> tenants.id`
- `category_id uuid fk -> finance_categories.id`
- `period_type varchar(20)` (`monthly|yearly|custom`)
- `period_start date`
- `period_end date`
- `amount_limit numeric(18,2)`
- `alert_threshold_pct numeric(5,2)`
- `created_at timestamptz`
- `updated_at timestamptz`
- Unique: `(tenant_id, category_id, period_start, period_end)`

### `finance_payment_reminders`
- `id uuid pk`
- `tenant_id uuid fk -> tenants.id`
- `title varchar(150)`
- `amount numeric(18,2)`
- `due_date date`
- `recurrence_rule varchar(100) null`
- `status varchar(20)` (`upcoming|paid|overdue|cancelled`)
- `notification_channel varchar(20)` (`inapp|email|both`)
- `created_at timestamptz`
- `updated_at timestamptz`
- Index: `(tenant_id, status, due_date)`

### `finance_reports`
- `id uuid pk`
- `tenant_id uuid fk -> tenants.id`
- `report_type varchar(50)` (`cashflow|expense_by_category|budget_vs_actual`)
- `format varchar(10)` (`csv|pdf`)
- `period_start date`
- `period_end date`
- `status varchar(20)` (`queued|processing|ready|failed`)
- `file_id uuid null fk -> files.id`
- `requested_by uuid fk -> users.id`
- `created_at timestamptz`
- `updated_at timestamptz`
- Index: `(tenant_id, status, created_at desc)`

## 4) Relasi Utama

- `users` many-to-many `tenants` via `tenant_memberships`.
- `tenant_memberships` many-to-many `roles` via `membership_roles`.
- `roles` many-to-many `permissions` via `role_permissions`.
- `tenant_modules` menentukan baseline module aktif per tenant.
- `tenant_features` menentukan baseline feature aktif per tenant.
- `tenant_feature_overrides` override granular features.
- `finance_transactions` belongs to `finance_accounts`, optional `finance_categories`.
- `finance_transaction_attachments` connects transaction to `files`.

## 5) Index dan Constraint Penting

- Semua tabel tenant-owned: index komposit yang diawali `tenant_id`.
- Constraints unique scoped by tenant untuk mencegah bentrok antar tenant.
- FK seluruh domain table ke `tenants.id`.
- Check constraints untuk enum-like kolom kritikal (`status`, `type`, `provider`) atau gunakan PostgreSQL enum jika tim setuju.
- Tambahkan partial index untuk data aktif:
  - Contoh `finance_transactions` aktif:
  - `CREATE INDEX ... ON finance_transactions (tenant_id, transaction_date DESC) WHERE deleted_at IS NULL;`

## 6) Keamanan dan Performa

- Query wajib tenant-filter, bukan hanya rely ke frontend.
- Gunakan prepared statements/parameterized queries.
- Pertimbangkan PostgreSQL RLS:
  - policy by `tenant_id = current_setting('app.tenant_id')::uuid`.
- Audit table bisa tumbuh cepat:
  - partisi bulanan/kuartalan setelah volume meningkat.
- Gunakan cursor/keyset pagination untuk list transaksi volume besar.
- Gunakan background jobs untuk report generation agar request API tetap cepat.

## 7) Migration Order (Disarankan)

1. `tenants`, `users`, `tenant_memberships`
2. `roles`, `permissions`, `role_permissions`, `membership_roles`
3. `products`, `modules`, `features`
4. `tenant_modules`, `tenant_features`, `tenant_feature_overrides`
5. `auth_sessions`, `audit_logs`
6. `files`
7. `finance_accounts`, `finance_categories`
8. `finance_transactions`, `finance_transaction_attachments`
9. `finance_savings_goals`, `finance_budgets`, `finance_payment_reminders`, `finance_reports`
10. add advanced indexes, partial indexes, optional RLS policies

## 8) Alasan Desain

- **Modular monolith compatible**: core tables reusable untuk module masa depan.
- **Tenant-aware by design**: isolasi tenant enforced dari level schema dan query.
- **Entitlement fleksibel**: tenant baseline + override granular.
- **Auditability**: event penting dapat ditelusuri end-to-end.
- **Storage decoupling**: metadata file unified terpisah dari provider fisik.
