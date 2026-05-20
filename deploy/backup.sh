#!/usr/bin/env bash
# Full Backup Script for PEKAN Platform

set -Eeuo pipefail

INSTALL_DIR="/opt/pekan"
BACKUP_DIR="${BACKUP_DIR:-/opt/pekan/backups}"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
BACKUP_NAME="pekan_backup_${TIMESTAMP}"
TEMP_DIR="/tmp/${BACKUP_NAME}"

log() {
  printf '[INFO] %s\n' "$*"
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
  if [[ -f "${INSTALL_DIR}/backend/.env" ]]; then
    # shellcheck disable=SC1091
    # We use a temporary way to source or grep because .env might not be valid bash
    DATABASE_URL=$(grep "^DATABASE_URL=" "${INSTALL_DIR}/backend/.env" | cut -d= -f2-)
    STORAGE_LOCAL_PATH=$(grep "^STORAGE_LOCAL_PATH=" "${INSTALL_DIR}/backend/.env" | cut -d= -f2-)
  else
    die ".env file not found in ${INSTALL_DIR}/backend"
  fi

  mkdir -p "$TEMP_DIR"
  as_root mkdir -p "$BACKUP_DIR"

  # 1. Backup Database
  log "Exporting database..."
  if [[ "$DATABASE_URL" == *"127.0.0.1"* ]] || [[ "$DATABASE_URL" == *"localhost"* ]]; then
    # Local DB (Docker or Standalone)
    # Check if Docker compose is used
    if as_root docker ps | grep -q pekan-postgres; then
       as_root docker exec pekan-postgres pg_dump -U postgres pekan > "${TEMP_DIR}/database.sql"
    else
       as_root -u postgres pg_dump pekan > "${TEMP_DIR}/database.sql"
    fi
  else
    # External DB
    pg_dump "$DATABASE_URL" > "${TEMP_DIR}/database.sql"
  fi

  # 2. Backup Config
  log "Copying configuration..."
  cp "${INSTALL_DIR}/backend/.env" "${TEMP_DIR}/.env"

  # 3. Backup Storage (Uploads)
  if [[ -d "$STORAGE_LOCAL_PATH" ]]; then
    log "Copying storage files from ${STORAGE_LOCAL_PATH}..."
    as_root cp -r "$STORAGE_LOCAL_PATH" "${TEMP_DIR}/storage"
  else
    warn "Storage path $STORAGE_LOCAL_PATH not found, skipping storage backup."
  fi

  # 4. Create Archive
  log "Creating compressed archive..."
  cd /tmp
  tar -czf "${BACKUP_DIR}/${BACKUP_NAME}.tar.gz" "$BACKUP_NAME"
  
  # Cleanup
  rm -rf "$TEMP_DIR"
  
  log "Backup completed: ${BACKUP_DIR}/${BACKUP_NAME}.tar.gz"
  as_root chown "$(id -un):$(id -gn)" "${BACKUP_DIR}/${BACKUP_NAME}.tar.gz"
}

main "$@"
