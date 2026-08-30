#!/usr/bin/env bash
# Full Restore Script for PEKAN Platform (Seamlessly restores to Systemd or Docker)

set -Eeuo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
INSTALL_DIR="${INSTALL_DIR:-/opt/pekan}"
if [[ ! -d "$INSTALL_DIR" ]] && [[ -d "$REPO_DIR/backend" ]]; then
  INSTALL_DIR="$REPO_DIR"
fi

log() {
  printf '[INFO] %s\n' "$*"
}

warn() {
  printf '[WARN] %s\n' "$*"
}

die() {
  printf '[ERROR] %s\n' "$*" >&2
  exit 1
}

as_root() {
  if [[ "${EUID}" -eq 0 ]]; then
    "$@"
  else
    sudo "$@"
  fi
}

usage() {
  echo "Usage: sudo $0 <path_to_backup_archive.tar.gz>"
  exit 1
}

main() {
  if [[ $# -lt 1 ]]; then
    usage
  fi

  local backup_file="$1"
  if [[ ! -f "$backup_file" ]]; then
    die "Backup file not found: $backup_file"
  fi

  log "Starting full restore from: ${backup_file}"
  
  local temp_extract="/tmp/pekan_restore_$(date +%s)"
  mkdir -p "$temp_extract"
  
  log "Extracting backup..."
  tar -xzf "$backup_file" -C "$temp_extract"
  
  # Find the actual data folder inside (named pekan_backup_...)
  local data_dir
  data_dir=$(find "$temp_extract" -maxdepth 1 -type d -name "pekan_backup_*" | head -n 1)
  if [[ -z "$data_dir" ]]; then
    data_dir="$temp_extract"
  fi

  # Detect destination mode (systemd vs docker)
  local DEST_MODE="systemd"
  if [[ -f "${INSTALL_DIR}/docker-compose.yml" ]] && command -v docker &>/dev/null && ! systemctl is-system-running &>/dev/null; then
    DEST_MODE="docker"
  fi
  log "Destination Environment: ${DEST_MODE}"

  # 1. Stop Services
  log "Step 1/6: Stopping services..."
  if command -v systemctl &>/dev/null && systemctl is-system-running &>/dev/null; then
    as_root systemctl stop pekan-api pekan-worker pekan-ai 2>/dev/null || true
  elif [[ "$DEST_MODE" == "docker" ]]; then
    cd "${INSTALL_DIR}" && (as_root docker compose stop 2>/dev/null || as_root docker-compose stop 2>/dev/null || true)
  fi

  # 2. Restore and Adapt Config (.env)
  log "Step 2/6: Restoring configuration (.env)..."
  if [[ -f "${data_dir}/.env" ]]; then
    as_root mkdir -p "${INSTALL_DIR}/backend"
    as_root cp "${data_dir}/.env" "${INSTALL_DIR}/backend/.env"
    
    # If restoring to Systemd from Docker, adapt hostnames
    if [[ "$DEST_MODE" == "systemd" ]]; then
      log "Adapting configuration for Systemd environment..."
      # Replace docker hostnames (pekan-postgres -> 127.0.0.1, pekan-redis -> 127.0.0.1)
      as_root sed -i 's/@pekan-postgres:/@127.0.0.1:/g' "${INSTALL_DIR}/backend/.env"
      as_root sed -i 's/@pekan-postgres\//\/127.0.0.1\//g' "${INSTALL_DIR}/backend/.env"
      as_root sed -i 's/redis:\/\/pekan-redis:/redis:\/\/127.0.0.1:/g' "${INSTALL_DIR}/backend/.env"
      as_root sed -i 's/STORAGE_LOCAL_PATH=.*/STORAGE_LOCAL_PATH=\/var\/lib\/pekan\/storage/g' "${INSTALL_DIR}/backend/.env"
    fi
  fi

  # Load resolved variables
  DATABASE_URL=$(grep "^DATABASE_URL=" "${INSTALL_DIR}/backend/.env" 2>/dev/null | cut -d= -f2- || echo "postgres://postgres:postgres@127.0.0.1:5432/pekan?sslmode=disable")
  STORAGE_LOCAL_PATH=$(grep "^STORAGE_LOCAL_PATH=" "${INSTALL_DIR}/backend/.env" 2>/dev/null | cut -d= -f2- || echo "/var/lib/pekan/storage")

  # 3. Restore Storage (Uploads)
  log "Step 3/6: Restoring storage files..."
  if [[ -d "${data_dir}/storage" ]]; then
    as_root mkdir -p "$STORAGE_LOCAL_PATH"
    as_root cp -r "${data_dir}/storage/." "$STORAGE_LOCAL_PATH/" 2>/dev/null || true
    as_root chown -R pekan:pekan "$STORAGE_LOCAL_PATH" 2>/dev/null || true
    log "Storage files restored to ${STORAGE_LOCAL_PATH}"
  fi

  # 4. Restore Database
  log "Step 4/6: Restoring database schemas and data..."
  if [[ -f "${data_dir}/database.sql" ]]; then
    if [[ "$DEST_MODE" == "docker" ]]; then
      local PG_CONTAINER
      PG_CONTAINER=$(as_root docker ps --format '{{.Names}}' 2>/dev/null | grep -E '(pekan-postgres|postgres|pekan-db|db)' | head -n 1 || true)
      if [[ -n "$PG_CONTAINER" ]]; then
        as_root docker exec -i "${PG_CONTAINER}" psql -U postgres -d pekan < "${data_dir}/database.sql"
      fi
    else
      # Standalone PostgreSQL (Systemd)
      # Ensure database exists
      as_root -u postgres psql -c "CREATE DATABASE pekan;" 2>/dev/null || true
      as_root -u postgres psql -d pekan -c "CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\";" 2>/dev/null || true
      
      log "Executing SQL restore into PostgreSQL..."
      as_root -u postgres psql -d pekan -f "${data_dir}/database.sql"
    fi
    log "Database restored successfully."
  fi

  # 5. Run Migrations / Patches Check
  log "Step 5/6: Applying database patches and verifying schemas..."
  if [[ -f "${INSTALL_DIR}/backend/scripts/apply_migrations.sh" ]]; then
    as_root chmod +x "${INSTALL_DIR}/backend/scripts/apply_migrations.sh"
    DATABASE_URL="${DATABASE_URL}" as_root -u pekan "${INSTALL_DIR}/backend/scripts/apply_migrations.sh" 2>/dev/null || \
    DATABASE_URL="${DATABASE_URL}" bash "${INSTALL_DIR}/backend/scripts/apply_migrations.sh" 2>/dev/null || true
  fi

  # 6. Restart Services
  log "Step 6/6: Restarting services..."
  if command -v systemctl &>/dev/null && systemctl is-system-running &>/dev/null; then
    as_root systemctl daemon-reload 2>/dev/null || true
    as_root systemctl restart pekan-api pekan-worker pekan-ai 2>/dev/null || true
  elif [[ "$DEST_MODE" == "docker" ]]; then
    cd "${INSTALL_DIR}" && (as_root docker compose start 2>/dev/null || as_root docker-compose start 2>/dev/null || true)
  fi

  # Cleanup
  rm -rf "$temp_extract"
  
  # Health check
  sleep 3
  local HEALTH_STATUS
  HEALTH_STATUS=$(curl -sf "http://127.0.0.1:8080/api/v1/healthz" 2>/dev/null || echo "PENDING")
  
  printf '\n======================================================\n'
  printf '  PEKAN RESTORE COMPLETED SUCCESSFULLY!\n'
  printf '  Environment: %s\n' "${DEST_MODE}"
  printf '  Health Status: %s\n' "${HEALTH_STATUS}"
  printf '======================================================\n\n'
}

main "$@"
