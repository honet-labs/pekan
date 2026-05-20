#!/usr/bin/env bash
# Full Restore Script for PEKAN Platform

set -Eeuo pipefail

INSTALL_DIR="/opt/pekan"
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

usage() {
  echo "Usage: $0 <path_to_backup_archive.tar.gz>"
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
  
  # Find the actual data folder inside (it might be named pekan_backup_...)
  local data_dir
  data_dir=$(find "$temp_extract" -maxdepth 1 -type d -name "pekan_backup_*" | head -n 1)
  if [[ -z "$data_dir" ]]; then
    die "Invalid backup archive format."
  fi

  # 1. Stop Services
  log "Stopping services..."
  as_root systemctl stop pekan-api pekan-worker || true

  # 2. Restore Config
  log "Restoring configuration (.env)..."
  as_root cp "${data_dir}/.env" "${INSTALL_DIR}/backend/.env"
  
  # Load env to get DATABASE_URL and STORAGE_LOCAL_PATH
  DATABASE_URL=$(grep "^DATABASE_URL=" "${INSTALL_DIR}/backend/.env" | cut -d= -f2-)
  STORAGE_LOCAL_PATH=$(grep "^STORAGE_LOCAL_PATH=" "${INSTALL_DIR}/backend/.env" | cut -d= -f2-)

  # 3. Restore Storage
  if [[ -d "${data_dir}/storage" ]]; then
    log "Restoring storage files to ${STORAGE_LOCAL_PATH}..."
    as_root mkdir -p "$STORAGE_LOCAL_PATH"
    as_root cp -r "${data_dir}/storage/"* "$STORAGE_LOCAL_PATH/"
    as_root chown -R pekan:pekan "$STORAGE_LOCAL_PATH"
  fi

  # 4. Restore Database
  log "Restoring database..."
  if [[ -f "${data_dir}/database.sql" ]]; then
    if [[ "$DATABASE_URL" == *"127.0.0.1"* ]] || [[ "$DATABASE_URL" == *"localhost"* ]]; then
      # Local DB
      if as_root docker ps | grep -q pekan-postgres; then
         as_root docker exec -i pekan-postgres psql -U postgres pekan < "${data_dir}/database.sql"
      else
         as_root -u postgres psql pekan < "${data_dir}/database.sql"
      fi
    else
      # External DB
      psql "$DATABASE_URL" < "${data_dir}/database.sql"
    fi
  fi

  # 5. Start Services
  log "Restarting services..."
  as_root systemctl start pekan-api pekan-worker
  
  # Cleanup
  rm -rf "$temp_extract"
  
  log "Restore completed successfully."
}

main "$@"
