#!/usr/bin/env bash
# =============================================================================
# PEKAN Database Migration Runner for Docker
# Run this script to apply all database migrations
# =============================================================================
set -euo pipefail

log() { printf '[INFO] %s\n' "$*"; }
warn() { printf '[WARN] %s\n' "$*"; }
error() { printf '[ERROR] %s\n' "$*" >&2; }

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
INSTALL_DIR="${INSTALL_DIR:-/opt/pekan}"
DB_USER="${DB_USER:-postgres}"
DB_NAME="${DB_NAME:-pekan}"

cd "$INSTALL_DIR"

log "Starting database migration..."
log "Database: $DB_NAME"
log "User: $DB_USER"

# Wait for PostgreSQL
log "Waiting for PostgreSQL..."
until docker compose exec -T pekan-postgres pg_isready -U "$DB_USER" -d "$DB_NAME" 2>/dev/null; do
  sleep 2
done
log "PostgreSQL is ready"

# Copy migrations
log "Copying migration files..."
docker compose cp backend/migrations pekan-postgres:/tmp/migrations

# List files
log "Migration files:"
docker compose exec -T pekan-postgres ls /tmp/migrations/*.sql | head -10

# Run migrations
log "Applying migrations..."
docker compose exec -T pekan-postgres bash -c '
  cd /tmp/migrations
  SUCCESS=0
  FAILED=0
  for f in $(ls *.sql 2>/dev/null | sort); do
    echo "  -> $f"
    if psql -U '"$DB_USER"' -d '"$DB_NAME"' -f "$f" 2>&1; then
      SUCCESS=$((SUCCESS + 1))
    else
      echo "     ERROR applying $f"
      FAILED=$((FAILED + 1))
    fi
  done
  echo ""
  echo "Results: $SUCCESS succeeded, $FAILED failed"
'

# Cleanup
docker compose exec -T pekan-postgres rm -rf /tmp/migrations

# Verify
log "Verifying tables..."
TABLES=$(docker compose exec -T pekan-postgres psql -U "$DB_USER" -d "$DB_NAME" -t -c "SELECT count(*) FROM information_schema.tables WHERE table_schema = 'public';" 2>/dev/null | tr -d ' ')
log "Found $TABLES tables in database"

# List key tables
log "Key tables:"
docker compose exec -T pekan-postgres psql -U "$DB_USER" -d "$DB_NAME" -c "SELECT table_name FROM information_schema.tables WHERE table_schema = 'public' AND table_name IN ('tenants', 'users', 'file_scan_jobs', 'finance_reminders') ORDER BY table_name;" 2>/dev/null

log "Migration complete!"
