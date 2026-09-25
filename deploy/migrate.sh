#!/usr/bin/env bash
# =============================================================================
# PEKAN Database Migration Runner for Docker
# Run this script to apply all database migrations and seed default tenant
# =============================================================================
set -euo pipefail

log() { printf '[INFO] %s\n' "$*"; }
warn() { printf '[WARN] %s\n' "$*"; }
error() { printf '[ERROR] %s\n' "$*" >&2; }

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
INSTALL_DIR="${INSTALL_DIR:-/opt/pekan}"

# Detect install directory
if [ -f "./docker-compose.yml" ]; then
  INSTALL_DIR="$(pwd)"
elif [ -f "$SCRIPT_DIR/../docker-compose.yml" ]; then
  INSTALL_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
fi

cd "$INSTALL_DIR"

# Load environment variables from .env
DB_USER="${DB_USER:-postgres}"
DB_NAME="${DB_NAME:-pekan}"
POSTGRES_PASSWORD="${POSTGRES_PASSWORD:-}"

if [ -f "$INSTALL_DIR/.env" ]; then
  ENV_PASS=$(grep -E '^POSTGRES_PASSWORD=' "$INSTALL_DIR/.env" | head -n 1 | cut -d= -f2- | tr -d '\r"' || true)
  if [ -n "$ENV_PASS" ]; then
    POSTGRES_PASSWORD="$ENV_PASS"
  fi
  ENV_USER=$(grep -E '^DB_USER=' "$INSTALL_DIR/.env" | head -n 1 | cut -d= -f2- | tr -d '\r"' || true)
  if [ -n "$ENV_USER" ]; then
    DB_USER="$ENV_USER"
  fi
  ENV_DB=$(grep -E '^DB_NAME=' "$INSTALL_DIR/.env" | head -n 1 | cut -d= -f2- | tr -d '\r"' || true)
  if [ -n "$ENV_DB" ]; then
    DB_NAME="$ENV_DB"
  fi
fi

log "Starting database migration..."
log "Database: $DB_NAME"
log "User: $DB_USER"

# Wait for PostgreSQL
log "Waiting for PostgreSQL..."
until docker compose exec -T -e PGPASSWORD="$POSTGRES_PASSWORD" pekan-postgres pg_isready -U "$DB_USER" -d "$DB_NAME" 2>/dev/null; do
  sleep 2
done
log "PostgreSQL is ready"

# Make sure password is synchronized inside postgres
if [ -n "$POSTGRES_PASSWORD" ]; then
  docker compose exec -T pekan-postgres psql -U postgres -c "ALTER USER \"$DB_USER\" WITH PASSWORD '$POSTGRES_PASSWORD';" 2>/dev/null || true
fi

# Clean up & prepare migration directory in container
docker compose exec -T pekan-postgres rm -rf /tmp/migrations /tmp/tenant_migrations
docker compose exec -T pekan-postgres mkdir -p /tmp/migrations /tmp/tenant_migrations

# Copy migrations
log "Copying migration files..."
docker compose cp backend/migrations/. pekan-postgres:/tmp/migrations/
if [ -d "backend/migrations/tenant" ]; then
  docker compose cp backend/migrations/tenant/. pekan-postgres:/tmp/tenant_migrations/
fi

# List files using container shell (Alpine /bin/sh)
log "Migration files:"
docker compose exec -T pekan-postgres sh -c 'ls -1 /tmp/migrations/*.sql 2>/dev/null' | head -10 || true

# Run public migrations (Alpine /bin/sh)
log "Applying public migrations..."
docker compose exec -T -e PGPASSWORD="$POSTGRES_PASSWORD" pekan-postgres sh -c '
  cd /tmp/migrations
  SUCCESS=0
  FAILED=0
  for f in $(ls *.sql 2>/dev/null | sort); do
    printf "  -> %s" "$f"
    if psql -U "'"$DB_USER"'" -d "'"$DB_NAME"'" -f "$f" >/dev/null 2>&1; then
      echo " [OK]"
      SUCCESS=$((SUCCESS + 1))
    else
      echo " [WARN/SKIP]"
      FAILED=$((FAILED + 1))
    fi
  done
  echo ""
  echo "Public migrations result: $SUCCESS succeeded, $FAILED skipped/warn"
'

# Run tenant schema initialization and seed default tenant
log "Bootstrapping default tenant and owner credentials..."
docker compose exec -T -e PGPASSWORD="$POSTGRES_PASSWORD" pekan-postgres psql -U "$DB_USER" -d "$DB_NAME" <<'EOSQL'
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- 1. Default Tenant in public schema
INSERT INTO public.tenants (id, code, name, status, timezone, created_at, updated_at)
VALUES (
  '11111111-1111-1111-1111-111111111111',
  'default',
  'Default Tenant',
  'active',
  'Asia/Jakarta',
  now(),
  now()
)
ON CONFLICT (code) DO NOTHING;

-- 2. Default User in public schema (email: owner@pekan.local, password: password)
INSERT INTO public.users (id, email, password_hash, full_name, is_active, created_at, updated_at)
VALUES (
  '22222222-2222-2222-2222-222222222222',
  'owner@pekan.local',
  '$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi',
  'Owner',
  TRUE,
  now(),
  now()
)
ON CONFLICT (email) DO NOTHING;

-- 3. Global Membership in public schema
INSERT INTO public.tenant_memberships (id, tenant_id, user_id, status, joined_at, created_at)
VALUES (
  '33333333-3333-3333-3333-333333333333',
  '11111111-1111-1111-1111-111111111111',
  '22222222-2222-2222-2222-222222222222',
  'active',
  now(),
  now()
)
ON CONFLICT (tenant_id, user_id) DO NOTHING;

-- 4. Roles & permissions in public schema
INSERT INTO public.roles (id, tenant_id, code, name, is_system, created_at, updated_at)
VALUES ('44444444-4444-4444-4444-444444444444', '11111111-1111-1111-1111-111111111111', 'owner', 'Owner', TRUE, now(), now())
ON CONFLICT (tenant_id, code) DO NOTHING;

INSERT INTO public.membership_roles (membership_id, role_id, created_at)
VALUES ('33333333-3333-3333-3333-333333333333', '44444444-4444-4444-4444-444444444444', now())
ON CONFLICT (membership_id, role_id) DO NOTHING;

INSERT INTO public.role_permissions (role_id, permission_id, created_at)
SELECT '44444444-4444-4444-4444-444444444444', p.id, now()
FROM public.permissions p
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- 5. Enable modules and features for default tenant
INSERT INTO public.tenant_modules (id, tenant_id, module_code, is_enabled, source, created_at, updated_at)
SELECT gen_random_uuid(), '11111111-1111-1111-1111-111111111111', code, TRUE, 'system', now(), now()
FROM public.modules WHERE is_active = TRUE
ON CONFLICT (tenant_id, module_code) DO NOTHING;

INSERT INTO public.tenant_features (id, tenant_id, feature_code, is_enabled, source, created_at, updated_at)
SELECT gen_random_uuid(), '11111111-1111-1111-1111-111111111111', code, TRUE, 'system', now(), now()
FROM public.features WHERE is_active = TRUE
ON CONFLICT (tenant_id, feature_code) DO NOTHING;

-- 6. Create isolated schema for default tenant
CREATE SCHEMA IF NOT EXISTS wkspid_pekan_default;
EOSQL

# Apply tenant migrations to wkspid_pekan_default
log "Applying isolated tenant schema tables for default tenant..."
docker compose exec -T -e PGPASSWORD="$POSTGRES_PASSWORD" pekan-postgres sh -c '
  if [ -f /tmp/tenant_migrations/0001_init.sql ]; then
    psql -U "'"$DB_USER"'" -d "'"$DB_NAME"'" -c "SET search_path TO wkspid_pekan_default, public;" -f /tmp/tenant_migrations/0001_init.sql >/dev/null 2>&1 || true
  fi
'

# Seed roles and default accounts in wkspid_pekan_default
docker compose exec -T -e PGPASSWORD="$POSTGRES_PASSWORD" pekan-postgres psql -U "$DB_USER" -d "$DB_NAME" <<'EOSQL'
SET search_path TO wkspid_pekan_default, public;

-- Local membership in schema
INSERT INTO wkspid_pekan_default.tenant_memberships (id, user_id, status, joined_at, created_at)
VALUES ('33333333-3333-3333-3333-333333333333', '22222222-2222-2222-2222-222222222222', 'active', now(), now())
ON CONFLICT (id) DO NOTHING;

-- Default roles in schema
INSERT INTO wkspid_pekan_default.roles (id, code, name, is_system, created_at, updated_at)
VALUES 
  ('44444444-4444-4444-4444-444444444444', 'owner', 'Owner', TRUE, now(), now()),
  (gen_random_uuid(), 'admin', 'Administrator', TRUE, now(), now())
ON CONFLICT (code) DO NOTHING;

-- Assign owner role in schema
INSERT INTO wkspid_pekan_default.membership_roles (membership_id, role_id, created_at)
SELECT '33333333-3333-3333-3333-333333333333', id, now()
FROM wkspid_pekan_default.roles WHERE code = 'owner' LIMIT 1
ON CONFLICT (membership_id, role_id) DO NOTHING;

-- Role permissions in schema
INSERT INTO wkspid_pekan_default.role_permissions (role_id, permission_id, created_at)
SELECT r.id, p.id, now()
FROM wkspid_pekan_default.roles r, public.permissions p 
WHERE r.code = 'owner'
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- Default account in schema
INSERT INTO wkspid_pekan_default.finance_accounts (id, name, account_type, currency, opening_balance_minor, is_active, created_by, created_at, updated_at)
VALUES (gen_random_uuid(), 'Kas Utama', 'cash', 'IDR', 0, TRUE, '22222222-2222-2222-2222-222222222222', now(), now())
ON CONFLICT DO NOTHING;

-- Default categories in schema
INSERT INTO wkspid_pekan_default.finance_categories (id, name, category_type, is_active, created_by, created_at, updated_at)
VALUES
  (gen_random_uuid(), 'Gaji & Pendapatan', 'income', TRUE, '22222222-2222-2222-2222-222222222222', now(), now()),
  (gen_random_uuid(), 'Makanan & Minuman', 'expense', TRUE, '22222222-2222-2222-2222-222222222222', now(), now()),
  (gen_random_uuid(), 'Transportasi', 'expense', TRUE, '22222222-2222-2222-2222-222222222222', now(), now()),
  (gen_random_uuid(), 'Tagihan & Utilitas', 'expense', TRUE, '22222222-2222-2222-2222-222222222222', now(), now()),
  (gen_random_uuid(), 'Belanja Kebutuhan', 'expense', TRUE, '22222222-2222-2222-2222-222222222222', now(), now())
ON CONFLICT DO NOTHING;
EOSQL

# Run patch_all_tenants and fix_all_tenants to ensure schema sync
log "Running tenant repair / synchronization..."
docker compose exec -T -e PGPASSWORD="$POSTGRES_PASSWORD" pekan-postgres sh -c '
  if [ -f /tmp/migrations/patch_all_tenants.sql ]; then
    psql -U "'"$DB_USER"'" -d "'"$DB_NAME"'" -f /tmp/migrations/patch_all_tenants.sql >/dev/null 2>&1 || true
  fi
  if [ -f /tmp/migrations/fix_all_tenants.sql ]; then
    psql -U "'"$DB_USER"'" -d "'"$DB_NAME"'" -f /tmp/migrations/fix_all_tenants.sql >/dev/null 2>&1 || true
  fi
'

# Cleanup
docker compose exec -T pekan-postgres rm -rf /tmp/migrations /tmp/tenant_migrations

# Verify
log "Verifying tables..."
TABLES=$(docker compose exec -T -e PGPASSWORD="$POSTGRES_PASSWORD" pekan-postgres psql -U "$DB_USER" -d "$DB_NAME" -t -c "SELECT count(*) FROM information_schema.tables WHERE table_schema = 'public';" 2>/dev/null | tr -d ' ')
log "Found $TABLES tables in public database"

TENANT_TABLES=$(docker compose exec -T -e PGPASSWORD="$POSTGRES_PASSWORD" pekan-postgres psql -U "$DB_USER" -d "$DB_NAME" -t -c "SELECT count(*) FROM information_schema.tables WHERE table_schema = 'wkspid_pekan_default';" 2>/dev/null | tr -d ' ')
log "Found $TENANT_TABLES tables in default tenant schema (wkspid_pekan_default)"

# List key tables
log "Key public tables:"
docker compose exec -T -e PGPASSWORD="$POSTGRES_PASSWORD" pekan-postgres psql -U "$DB_USER" -d "$DB_NAME" -c "SELECT table_name FROM information_schema.tables WHERE table_schema = 'public' AND table_name IN ('tenants', 'users', 'file_scan_jobs', 'finance_reminders') ORDER BY table_name;" 2>/dev/null

log "Migration complete!"
