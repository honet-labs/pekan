#!/usr/bin/env bash
# =============================================================================
# PEKAN Docker Uninstaller
# Remove PEKAN and all data (Docker mode)
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

as_root() {
  if [[ "${EUID}" -eq 0 ]]; then "$@"; else sudo "$@"; fi
}

show_help() {
  cat <<'EOF'
PEKAN Docker Uninstaller

Remove PEKAN containers, images, volumes, and ALL DATA.

Usage:
  sudo bash uninstall-docker.sh [options]

Options:
  --install-dir <path>    Installation directory (default: /opt/pekan)
  --keep-volumes          Keep Docker volumes (database data)
  --keep-images           Keep Docker images
  --help                  Show this help message

WARNING: This will permanently delete:
  - All PEKAN containers (pekan-api, pekan-worker, pekan-ai, pekan-web, pekan-postgres, pekan-redis)
  - Docker volumes (database data, Redis data)
  - Docker images
  - Application files in /opt/pekan
  - All backups and storage files
EOF
}

parse_args() {
  KEEP_VOLUMES=false
  KEEP_IMAGES=false

  while [[ $# -gt 0 ]]; do
    case "$1" in
      --install-dir)
        INSTALL_DIR="$2"
        shift 2
        ;;
      --keep-volumes)
        KEEP_VOLUMES=true
        shift
        ;;
      --keep-images)
        KEEP_IMAGES=true
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
  printf '  PEKAN Docker Uninstaller\n'
  printf '============================================\n'
  printf '\n'
  printf '  This will permanently delete:\n'
  printf '    - Containers: pekan-api, pekan-worker, pekan-ai, pekan-web\n'
  printf '    - Containers: pekan-postgres, pekan-redis\n'
  if [[ "$KEEP_VOLUMES" == "false" ]]; then
    printf '    - Volumes: pekan_postgres_data, pekan_redis_data (ALL DATA)\n'
  fi
  if [[ "$KEEP_IMAGES" == "false" ]]; then
    printf '    - Docker images\n'
  fi
  printf '    - Application: %s\n' "$INSTALL_DIR"
  printf '    - Network: pekan-network\n'
  printf '\n'

  printf 'Are you sure you want to continue? [y/N]: '
  read -r confirm
  if [[ ! "$confirm" =~ ^[Yy]$ ]]; then
    die "Uninstall cancelled."
  fi
}

stop_containers() {
  log "Stopping PEKAN containers..."

  if [[ -f "$INSTALL_DIR/docker-compose.yml" ]]; then
    cd "$INSTALL_DIR"
    as_root docker compose down 2>/dev/null || true
  fi

  # Force remove any remaining containers
  for container in pekan-api pekan-worker pekan-ai pekan-web pekan-postgres pekan-redis; do
    as_root docker rm -f "$container" 2>/dev/null || true
  done

  log "  Containers stopped and removed"
}

remove_volumes() {
  if [[ "$KEEP_VOLUMES" == "true" ]]; then
    log "Skipping volume deletion (--keep-volumes)"
    return
  fi

  log "Removing Docker volumes..."

  as_root docker volume rm pekan_postgres_data 2>/dev/null || true
  as_root docker volume rm pekan_redis_data 2>/dev/null || true

  log "  Volumes removed"
}

remove_images() {
  if [[ "$KEEP_IMAGES" == "true" ]]; then
    log "Skipping image deletion (--keep-images)"
    return
  fi

  log "Removing Docker images..."

  # Remove PEKAN images
  as_root docker images --format '{{.Repository}}:{{.Tag}}' | grep -i pekan | while read -r img; do
    as_root docker rmi "$img" 2>/dev/null || true
  done

  log "  Images removed"
}

remove_network() {
  log "Removing Docker network..."

  as_root docker network rm pekan-network 2>/dev/null || true

  log "  Network removed"
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
  printf '    - Containers: pekan-api, pekan-worker, pekan-ai, pekan-web\n'
  printf '    - Containers: pekan-postgres, pekan-redis\n'
  if [[ "$KEEP_VOLUMES" == "false" ]]; then
    printf '    - Volumes: pekan_postgres_data, pekan_redis_data\n'
  fi
  if [[ "$KEEP_IMAGES" == "false" ]]; then
    printf '    - Docker images\n'
  fi
  printf '    - Application: %s\n' "$INSTALL_DIR"
  printf '    - Network: pekan-network\n'
  printf '\n'
  printf '  Docker is still installed. To remove it:\n'
  printf '    sudo apt remove docker-ce docker-ce-cli containerd.io\n'
  printf '\n'
}

main() {
  parse_args "$@"
  confirm_uninstall
  stop_containers
  remove_volumes
  remove_images
  remove_network
  remove_application
  cleanup_cron
  show_summary
}

main "$@"
