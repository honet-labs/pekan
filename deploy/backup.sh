#!/usr/bin/env bash
# Full Backup Script for PEKAN Platform (Supports Docker and Systemd environments)

set -Eeuo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
INSTALL_DIR="${INSTALL_DIR:-/opt/pekan}"
if [[ ! -d "$INSTALL_DIR" ]] && [[ -d "$REPO_DIR/backend" ]]; then
  INSTALL_DIR="$REPO_DIR"
fi

BACKUP_DIR="${BACKUP_DIR:-${INSTALL_DIR}/backups}"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
BACKUP_NAME="pekan_backup_${TIMESTAMP}"
TEMP_DIR="/tmp/${BACKUP_NAME}"

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

main() {
  log "Starting full backup: ${BACKUP_NAME}"
  
  if [[ ! -d "$INSTALL_DIR" ]]; then
    die "Installation directory $INSTALL_DIR not found."
  fi

  # Load env to get DATABASE_URL and STORAGE_LOCAL_PATH
  local ENV_FILE=""
  if [[ -f "${INSTALL_DIR}/backend/.env" ]]; then
    ENV_FILE="${INSTALL_DIR}/backend/.env"
  elif [[ -f "${INSTALL_DIR}/.env" ]]; then
    ENV_FILE="${INSTALL_DIR}/.env"
  fi

  DATABASE_URL=""
  STORAGE_LOCAL_PATH=""
  if [[ -n "$ENV_FILE" ]]; then
    DATABASE_URL=$(grep "^DATABASE_URL=" "$ENV_FILE" | cut -d= -f2- || true)
    STORAGE_LOCAL_PATH=$(grep "^STORAGE_LOCAL_PATH=" "$ENV_FILE" | cut -d= -f2- || true)
  fi

  mkdir -p "$TEMP_DIR"
  as_root mkdir -p "$BACKUP_DIR"

  # 1. Backup Database
  log "Exporting database..."
  local PG_CONTAINER=""
  if command -v docker &>/dev/null; then
    PG_CONTAINER=$(as_root docker ps --format '{{.Names}}' 2>/dev/null | grep -E '(pekan-postgres|postgres|pekan-db|db)' | head -n 1 || true)
  fi

  if [[ -n "$PG_CONTAINER" ]]; then
    log "Exporting from Docker container: ${PG_CONTAINER}..."
    as_root docker exec "${PG_CONTAINER}" pg_dump -U postgres --clean --if-exists --no-owner --no-privileges pekan > "${TEMP_DIR}/database.sql"
  elif [[ -n "$DATABASE_URL" && "$DATABASE_URL" != *"127.0.0.1"* && "$DATABASE_URL" != *"localhost"* && "$DATABASE_URL" != *"pekan-postgres"* ]]; then
    log "Exporting from external database URL..."
    pg_dump --clean --if-exists --no-owner --no-privileges "$DATABASE_URL" > "${TEMP_DIR}/database.sql"
  else
    log "Exporting from local PostgreSQL server..."
    as_root -u postgres pg_dump --clean --if-exists --no-owner --no-privileges pekan > "${TEMP_DIR}/database.sql"
  fi

  # 2. Backup Config
  log "Copying configuration..."
  if [[ -n "$ENV_FILE" ]]; then
    cp "$ENV_FILE" "${TEMP_DIR}/.env"
  fi

  # 3. Backup Storage (Uploads)
  log "Copying storage files..."
  local STORAGE_COPIED=0
  if [[ -n "$STORAGE_LOCAL_PATH" && -d "$STORAGE_LOCAL_PATH" ]]; then
    as_root cp -r "$STORAGE_LOCAL_PATH" "${TEMP_DIR}/storage"
    STORAGE_COPIED=1
  fi

  if [[ $STORAGE_COPIED -eq 0 && -d "${INSTALL_DIR}/data/storage" ]]; then
    as_root cp -r "${INSTALL_DIR}/data/storage" "${TEMP_DIR}/storage"
    STORAGE_COPIED=1
  fi

  if [[ $STORAGE_COPIED -eq 0 && -d "${INSTALL_DIR}/storage" ]]; then
    as_root cp -r "${INSTALL_DIR}/storage" "${TEMP_DIR}/storage"
    STORAGE_COPIED=1
  fi

  if [[ $STORAGE_COPIED -eq 0 && -d "/var/lib/pekan/storage" ]]; then
    as_root cp -r "/var/lib/pekan/storage" "${TEMP_DIR}/storage"
    STORAGE_COPIED=1
  fi

  if [[ $STORAGE_COPIED -eq 0 && command -v docker &>/dev/null ]]; then
    local APP_CONTAINER
    APP_CONTAINER=$(as_root docker ps --format '{{.Names}}' 2>/dev/null | grep -E '(pekan-api|pekan-app|pekan)' | head -n 1 || true)
    if [[ -n "$APP_CONTAINER" ]]; then
      as_root docker cp "${APP_CONTAINER}:/var/lib/pekan/storage" "${TEMP_DIR}/storage" 2>/dev/null || true
    fi
  fi

  # 4. Create Archive
  log "Creating compressed archive..."
  local ARCHIVE_PATH="${BACKUP_DIR}/${BACKUP_NAME}.tar.gz"
  tar -czf "$ARCHIVE_PATH" -C /tmp "$BACKUP_NAME"
  
  # Cleanup
  rm -rf "$TEMP_DIR"
  
  log "Backup completed successfully!"
  log "Archive location: ${ARCHIVE_PATH}"
  as_root chown "$(id -un):$(id -gn)" "$ARCHIVE_PATH" 2>/dev/null || true
  
  printf '\n======================================================\n'
  printf '  PEKAN FULL BACKUP SUCCESSFUL\n'
  printf '  File: %s\n' "${ARCHIVE_PATH}"
  printf '  Size: %s\n' "$(du -sh "${ARCHIVE_PATH}" | cut -f1)"
  printf '======================================================\n\n'
}

main "$@"
