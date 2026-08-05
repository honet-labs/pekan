#!/usr/bin/env bash
if [ -z "${BASH_VERSION:-}" ]; then
  exec /usr/bin/env bash "$0" "$@"
fi
set -Eeuo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

INSTALL_DIR="${INSTALL_DIR:-/opt/pekan}"
APP_USER="${APP_USER:-pekan}"
APP_GROUP="${APP_GROUP:-pekan}"
STORAGE_LOCAL_PATH="${STORAGE_LOCAL_PATH:-/var/lib/pekan/storage}"
WEB_ROOT="${WEB_ROOT:-/var/www/pekan-web}"
NGINX_CONF_PATH="${NGINX_CONF_PATH:-/etc/nginx/conf.d/pekan.conf}"

REMOVE_INSTALL_DIR=0
REMOVE_DATA=0
REMOVE_USER_GROUP=0
REMOVE_SYSTEMD_UNITS=1
REMOVE_DOCKER_IMAGES=0

usage() {
  cat <<'EOF'
Usage: ./deploy/uninstall_server_rocky.sh [options]

Default behavior:
- stop + disable PEKAN services
- stop docker compose infra (postgres + redis)
- keep install directory and persistent data

Options:
  --install-dir <path>        Install path (default: /opt/pekan)
  --app-user <user>           Service user (default: pekan)
  --app-group <group>         Service group (default: pekan)
  --storage-path <path>       Local storage path (default: /var/lib/pekan/storage)
  --remove-install-dir        Delete install directory
  --remove-data               Delete docker volumes + local storage path
  --web-root <path>           Web root path (default: /var/www/pekan-web)
  --nginx-conf <path>         Nginx site path (default: /etc/nginx/conf.d/pekan.conf)
  --remove-user-group         Delete app user and group
  --keep-systemd-units        Keep unit files in /etc/systemd/system
  --remove-docker-images      Remove postgres/redis images used by compose
  -h, --help                  Show help

Examples:
  ./deploy/uninstall_server_rocky.sh
  ./deploy/uninstall_server_rocky.sh --remove-install-dir --remove-data --remove-user-group
EOF
}

log() {
  printf '[INFO] %s\n' "$*"
}

warn() {
  printf '[WARN] %s\n' "$*" >&2
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

parse_args() {
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --install-dir)
        INSTALL_DIR="$2"
        shift 2
        ;;
      --app-user)
        APP_USER="$2"
        shift 2
        ;;
      --app-group)
        APP_GROUP="$2"
        shift 2
        ;;
      --storage-path)
        STORAGE_LOCAL_PATH="$2"
        shift 2
        ;;
      --remove-install-dir)
        REMOVE_INSTALL_DIR=1
        shift
        ;;
      --remove-data)
        REMOVE_DATA=1
        shift
        ;;
      --web-root)
        WEB_ROOT="$2"
        shift 2
        ;;
      --nginx-conf)
        NGINX_CONF_PATH="$2"
        shift 2
        ;;
      --remove-user-group)
        REMOVE_USER_GROUP=1
        shift
        ;;
      --keep-systemd-units)
        REMOVE_SYSTEMD_UNITS=0
        shift
        ;;
      --remove-docker-images)
        REMOVE_DOCKER_IMAGES=1
        shift
        ;;
      -h|--help)
        usage
        exit 0
        ;;
      *)
        die "Unknown argument: $1"
        ;;
    esac
  done
}

resolve_compose_file() {
  local -a candidates=(
    "${INSTALL_DIR}/deploy/docker-compose.server-test.yml"
    "${REPO_DIR}/deploy/docker-compose.server-test.yml"
    "${SCRIPT_DIR}/docker-compose.server-test.yml"
  )
  local file
  for file in "${candidates[@]}"; do
    if [[ -f "$file" ]]; then
      printf '%s\n' "$file"
      return 0
    fi
  done
  return 1
}

cleanup_named_docker_resources() {
  log "Cleaning docker resources by fixed names (fallback safety)"
  as_root docker rm -f pekan-postgres pekan-redis >/dev/null 2>&1 || true
  as_root docker network rm deploy_default >/dev/null 2>&1 || true
  if [[ "$REMOVE_DATA" == "1" ]]; then
    as_root docker volume rm -f \
      deploy_pekan_postgres_data \
      deploy_pekan_redis_data \
      pekan_postgres_data \
      pekan_redis_data >/dev/null 2>&1 || true
  fi
}

stop_services() {
  log "Stopping systemd services (if installed)"
  as_root systemctl disable --now pekan-api.service pekan-worker.service >/dev/null 2>&1 || true
}

remove_systemd_units() {
  if [[ "$REMOVE_SYSTEMD_UNITS" != "1" ]]; then
    warn "Keeping systemd unit files (--keep-systemd-units)"
    return
  fi

  log "Removing systemd unit files"
  as_root rm -f /etc/systemd/system/pekan-api.service
  as_root rm -f /etc/systemd/system/pekan-worker.service
  as_root systemctl daemon-reload
}

stop_compose_infra() {
  local compose_file=""
  if ! compose_file="$(resolve_compose_file)"; then
    warn "Compose file not found in known paths (skip docker compose down)"
    cleanup_named_docker_resources
    return
  fi

  if [[ "$REMOVE_DATA" == "1" ]]; then
    log "Stopping docker compose and removing volumes (${compose_file})"
    as_root docker compose -f "$compose_file" down -v || true
  else
    log "Stopping docker compose (volumes kept) (${compose_file})"
    as_root docker compose -f "$compose_file" down || true
  fi

  # Extra safety in case project name differs and compose does not catch named resources.
  cleanup_named_docker_resources
}

remove_docker_images() {
  if [[ "$REMOVE_DOCKER_IMAGES" != "1" ]]; then
    return
  fi
  log "Removing docker images (postgres:16-alpine, redis:7-alpine)"
  as_root docker image rm postgres:16-alpine redis:7-alpine >/dev/null 2>&1 || true
}

remove_install_dir() {
  if [[ "$REMOVE_INSTALL_DIR" != "1" ]]; then
    warn "Keeping install directory: ${INSTALL_DIR}"
    return
  fi
  log "Removing install directory: ${INSTALL_DIR}"
  as_root rm -rf "$INSTALL_DIR"
}

remove_web_artifacts() {
  if [[ "$REMOVE_INSTALL_DIR" != "1" ]]; then
    warn "Keeping web artifacts (use --remove-install-dir to clean web root/nginx site)"
    return
  fi

  log "Removing web root: ${WEB_ROOT}"
  as_root rm -rf "$WEB_ROOT"

  if [[ -f "$NGINX_CONF_PATH" ]]; then
    log "Removing nginx site config: ${NGINX_CONF_PATH}"
    as_root rm -f "$NGINX_CONF_PATH"
    as_root systemctl reload nginx >/dev/null 2>&1 || true
  fi
}

remove_data_paths() {
  if [[ "$REMOVE_DATA" != "1" ]]; then
    warn "Keeping persistent data path: ${STORAGE_LOCAL_PATH}"
    return
  fi
  log "Removing persistent data path: ${STORAGE_LOCAL_PATH}"
  as_root rm -rf "$STORAGE_LOCAL_PATH"
}

remove_user_group() {
  if [[ "$REMOVE_USER_GROUP" != "1" ]]; then
    warn "Keeping app user/group: ${APP_USER}:${APP_GROUP}"
    return
  fi

  if id -u "$APP_USER" >/dev/null 2>&1; then
    log "Removing user: ${APP_USER}"
    as_root userdel "$APP_USER" >/dev/null 2>&1 || true
  fi

  if getent group "$APP_GROUP" >/dev/null 2>&1; then
    log "Removing group: ${APP_GROUP}"
    as_root groupdel "$APP_GROUP" >/dev/null 2>&1 || true
  fi
}

print_summary() {
  cat <<EOF

Uninstall completed.

Summary:
- Services stopped: pekan-api, pekan-worker
- Systemd units removed: ${REMOVE_SYSTEMD_UNITS}
- Install dir removed: ${REMOVE_INSTALL_DIR}
- Web root/nginx site removed: ${REMOVE_INSTALL_DIR}
- Persistent data removed: ${REMOVE_DATA}
- User/group removed: ${REMOVE_USER_GROUP}
- Docker images removed: ${REMOVE_DOCKER_IMAGES}

EOF
}

main() {
  parse_args "$@"
  stop_services
  remove_systemd_units
  stop_compose_infra
  remove_docker_images
  remove_install_dir
  remove_web_artifacts
  remove_data_paths
  remove_user_group
  print_summary
}

main "$@"
