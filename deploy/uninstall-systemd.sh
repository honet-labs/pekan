#!/usr/bin/env bash
# =============================================================================
# PEKAN Systemd Uninstaller
# Remove PEKAN and all data (systemd mode)
# =============================================================================
if [ -z "${BASH_VERSION:-}" ]; then
  exec /usr/bin/env bash "$0" "$@"
fi
set -Eeuo pipefail

log()   { printf '[INFO] %s\n' "$*"; }
warn()  { printf '[WARN] %s\n' "$*"; }
error() { printf '[ERROR] %s\n' "$*" >&2; }
die()   { error "$*"; exit 1; }

INSTALL_DIR="${INSTALL_DIR:-/opt/pekan}"
APP_USER="${APP_USER:-pekan}"
DB_NAME="${DB_NAME:-pekan}"

as_root() {
  if [[ "${EUID}" -eq 0 ]]; then "$@"; else sudo "$@"; fi
}

show_help() {
  cat <<'EOF'
PEKAN Systemd Uninstaller

Remove PEKAN application, services, and ALL DATA.

Usage:
  sudo bash uninstall-systemd.sh [options]

Options:
  --install-dir <path>    Installation directory (default: /opt/pekan)
  --keep-data             Keep database and storage data
  --keep-user             Keep system user 'pekan'
  --help                  Show this help message

WARNING: This will permanently delete:
  - All PEKAN services (pekan-api, pekan-worker, pekan-ai)
  - Application files in /opt/pekan
  - Database 'pekan' and all data
  - All backups and storage files
  - System user 'pekan'
  - Nginx configuration
EOF
}

parse_args() {
  KEEP_DATA=false
  KEEP_USER=false

  while [[ $# -gt 0 ]]; do
    case "$1" in
      --install-dir)
        INSTALL_DIR="$2"
        shift 2
        ;;
      --keep-data)
        KEEP_DATA=true
        shift
        ;;
      --keep-user)
        KEEP_USER=true
        shift
        ;;
      --help|-h)
        show_help
        exit 0
        ;;
      *)
        die "Unknown option: $1. Use --help for usage."
        ;;
    esac
  done
}

confirm_uninstall() {
  printf '\n'
  printf '============================================\n'
  printf '  PEKAN Systemd Uninstaller\n'
  printf '============================================\n'
  printf '\n'
  printf '  This will permanently delete:\n'
  printf '    - Services: pekan-api, pekan-worker, pekan-ai\n'
  printf '    - Application: %s\n' "$INSTALL_DIR"
  if [[ "$KEEP_DATA" == "false" ]]; then
    printf '    - Database: %s (ALL DATA)\n' "$DB_NAME"
    printf '    - Backups and storage files\n'
  fi
  if [[ "$KEEP_USER" == "false" ]]; then
    printf '    - System user: %s\n' "$APP_USER"
  fi
  printf '    - Nginx config: /etc/nginx/sites-available/pekan\n'
  printf '\n'

  printf 'Are you sure you want to continue? [y/N]: '
  read -r confirm
  if [[ ! "$confirm" =~ ^[Yy]$ ]]; then
    die "Uninstall cancelled."
  fi
}

stop_services() {
  log "Stopping PEKAN services..."

  as_root systemctl stop pekan-api 2>/dev/null || true
  as_root systemctl stop pekan-worker 2>/dev/null || true
  as_root systemctl stop pekan-ai 2>/dev/null || true

  as_root systemctl disable pekan-api 2>/dev/null || true
  as_root systemctl disable pekan-worker 2>/dev/null || true
  as_root systemctl disable pekan-ai 2>/dev/null || true

  log "  Services stopped"
}

remove_services() {
  log "Removing systemd service files..."

  as_root rm -f /etc/systemd/system/pekan-api.service
  as_root rm -f /etc/systemd/system/pekan-worker.service
  as_root rm -f /etc/systemd/system/pekan-ai.service

  as_root systemctl daemon-reload

  log "  Service files removed"
}

remove_database() {
  if [[ "$KEEP_DATA" == "true" ]]; then
    log "Skipping database deletion (--keep-data)"
    return
  fi

  log "Removing database..."

  # Check if PostgreSQL is running in Docker
  if docker ps --format '{{.Names}}' 2>/dev/null | grep -q "postgres"; then
    PG_CONTAINER=$(docker ps --format '{{.Names}}' | grep postgres | head -1)
    docker exec "$PG_CONTAINER" psql -U postgres -c "DROP DATABASE IF EXISTS ${DB_NAME};" 2>/dev/null || true
    docker exec "$PG_CONTAINER" psql -U postgres -c "DROP USER IF EXISTS ${APP_USER};" 2>/dev/null || true
    log "  Database dropped (Docker)"
  elif command -v psql &>/dev/null; then
    as_root -u postgres psql -c "DROP DATABASE IF EXISTS ${DB_NAME};" 2>/dev/null || true
    as_root -u postgres psql -c "DROP USER IF EXISTS ${APP_USER};" 2>/dev/null || true
    log "  Database dropped"
  else
    warn "  PostgreSQL not found, skipping database deletion"
  fi
}

remove_nginx() {
  log "Removing Nginx configuration..."

  as_root rm -f /etc/nginx/sites-available/pekan
  as_root rm -f /etc/nginx/sites-enabled/pekan

  # Restore default nginx config if exists
  if [[ -f /etc/nginx/sites-available/default ]]; then
    as_root ln -sf /etc/nginx/sites-available/default /etc/nginx/sites-enabled/default
  fi

  as_root nginx -t 2>/dev/null && as_root systemctl reload nginx 2>/dev/null || true

  log "  Nginx configuration removed"
}

remove_application() {
  log "Removing application files..."

  if [[ -d "$INSTALL_DIR" ]]; then
    as_root rm -rf "$INSTALL_DIR"
    log "  Application removed from $INSTALL_DIR"
  else
    warn "  Installation directory not found: $INSTALL_DIR"
  fi
}

remove_user() {
  if [[ "$KEEP_USER" == "true" ]]; then
    log "Skipping user deletion (--keep-user)"
    return
  fi

  log "Removing system user..."

  if id "$APP_USER" &>/dev/null; then
    as_root userdel -r "$APP_USER" 2>/dev/null || as_root userdel "$APP_USER" 2>/dev/null || true
    log "  User '$APP_USER' removed"
  else
    log "  User '$APP_USER' not found"
  fi
}

cleanup_cron() {
  log "Cleaning up cron jobs..."

  # Remove PEKAN backup cron jobs
  as_root crontab -l 2>/dev/null | grep -v "pekan" | as_root crontab - 2>/dev/null || true

  log "  Cron jobs cleaned"
}

show_summary() {
  printf '\n'
  printf '============================================\n'
  printf '  PEKAN Uninstalled Successfully\n'
  printf '============================================\n'
  printf '\n'
  printf '  Removed:\n'
  printf '    - Services: pekan-api, pekan-worker, pekan-ai\n'
  printf '    - Application: %s\n' "$INSTALL_DIR"
  if [[ "$KEEP_DATA" == "false" ]]; then
    printf '    - Database: %s\n' "$DB_NAME"
    printf '    - Backups and storage\n'
  fi
  if [[ "$KEEP_USER" == "false" ]]; then
    printf '    - User: %s\n' "$APP_USER"
  fi
  printf '    - Nginx config\n'
  printf '\n'
  printf '  Note: PostgreSQL and Redis services are still running.\n'
  printf '  To remove them completely:\n'
  printf '    sudo apt remove postgresql redis-server\n'
  printf '\n'
}

main() {
  parse_args "$@"
  confirm_uninstall
  stop_services
  remove_services
  remove_database
  remove_nginx
  remove_application
  remove_user
  cleanup_cron
  show_summary
}

main "$@"
