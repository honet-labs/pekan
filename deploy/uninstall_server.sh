#!/usr/bin/env bash
# Clean Uninstall Script for Pekan SaaS Platform
# This script will remove all components, data, and configurations.

set -Eeuo pipefail

INSTALL_DIR="/opt/pekan"
WEB_ROOT="/var/www/pekan-web"
APP_USER="pekan"
NGINX_CONF="/etc/nginx/conf.d/pekan.conf"
ONLY_SERVICES=0

# Parse arguments
for arg in "$@"; do
  case $arg in
    --only-services)
      ONLY_SERVICES=1
      shift
      ;;
  esac
done

log() {
  printf '[INFO] %s\n' "$*"
}

warn() {
  printf '[WARN] %s\n' "$*" >&2
}

as_root() {
  if [[ "${EUID}" -eq 0 ]]; then
    "$@"
  else
    sudo "$@"
  fi
}

confirm() {
  if [[ "$ONLY_SERVICES" == "1" ]]; then
    printf 'WARNING: This will remove ONLY services and Nginx config. Data and files will be KEPT. Continue? (y/N): '
  else
    printf 'WARNING: This will delete ALL data (database, uploads, config). Continue? (y/N): '
  fi
  read -r response
  if [[ ! "$response" =~ ^[Yy]$ ]]; then
    log "Uninstall cancelled."
    exit 0
  fi
}

remove_systemd_services() {
  log "Stopping and removing systemd services"
  for svc in pekan-api pekan-worker; do
    if systemctl list-unit-files | grep -q "${svc}.service"; then
      as_root systemctl stop "${svc}.service" || true
      as_root systemctl disable "${svc}.service" || true
      as_root rm -f "/etc/systemd/system/${svc}.service"
    fi
  done
  as_root systemctl daemon-reload
}

remove_nginx_config() {
  log "Removing Nginx configuration"
  if [[ -f "$NGINX_CONF" ]]; then
    as_root rm -f "$NGINX_CONF"
    if command -v nginx >/dev/null 2>&1; then
      as_root nginx -s reload || true
    fi
  fi
}

cleanup_docker() {
  log "Checking for Docker infrastructure"
  if [[ -f "${INSTALL_DIR}/deploy/docker-compose.server-test.yml" ]]; then
    as_root docker compose -f "${INSTALL_DIR}/deploy/docker-compose.server-test.yml" down -v || true
  fi
}

cleanup_standalone_db() {
  log "Checking for Standalone PostgreSQL database"
  if command -v psql >/dev/null 2>&1; then
    # Attempt to drop database if it exists
    # We use a subshell to avoid exit on failure if DB doesn't exist
    (as_root -u postgres psql -c "DROP DATABASE pekan;" || true) 2>/dev/null
    (as_root -u postgres psql -c "DROP USER pekan;" || true) 2>/dev/null
  fi
}

remove_files() {
  log "Removing installation and web directories"
  as_root rm -rf "$INSTALL_DIR"
  as_root rm -rf "$WEB_ROOT"
}

remove_user() {
  log "Removing application user: $APP_USER"
  if id -u "$APP_USER" >/dev/null 2>&1; then
    as_root userdel -r "$APP_USER" || warn "Could not remove user $APP_USER (it might be in use)"
  fi
}

main() {
  if [[ "${EUID}" -ne 0 ]] && ! command -v sudo >/dev/null 2>&1; then
    printf '[ERROR] sudo is required to run this script.\n' >&2
    exit 1
  fi

  confirm
  
  remove_systemd_services
  remove_nginx_config

  if [[ "$ONLY_SERVICES" == "0" ]]; then
    cleanup_docker
    cleanup_standalone_db
    remove_files
    remove_user
    log "Full uninstall completed successfully."
  else
    log "Service-only uninstall completed. Data and files were KEPT."
  fi
}

main "$@"
