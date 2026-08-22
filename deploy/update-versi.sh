#!/usr/bin/env bash
# =============================================================================
# PEKAN Version Updater
# Update PEKAN to latest version (supports both systemd and docker deployments)
#
# Usage:
#   sudo bash update-versi.sh [options]
#
# Options:
#   --branch <name>         Git branch to update to (default: current branch)
#   --install-dir <path>    Installation directory (default: /opt/pekan)
#   --no-backup             Skip database backup before update
#   --mode <systemd|docker> Override deployment mode (auto-detect if not set)
#   --help                  Show this help
# =============================================================================
if [ -z "${BASH_VERSION:-}" ]; then
  exec /usr/bin/env bash "$0" "$@"
fi
set -Eeuo pipefail

log()   { printf '[INFO] %s\n' "$*"; }
warn()  { printf '[WARN] %s\n' "$*"; }
error() { printf '[ERROR] %s\n' "$*" >&2; }
die()   { error "$*"; exit 1; }

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

# Default values
BRANCH=""
INSTALL_DIR="${INSTALL_DIR:-/opt/pekan}"
DEPLOY_MODE=""
BACKUP_BEFORE_UPDATE="${BACKUP_BEFORE_UPDATE:-true}"

show_help() {
  cat <<'EOF'
PEKAN Version Updater

Update PEKAN to the latest version. Supports both systemd and docker deployments.

Usage:
  sudo bash update-versi.sh [options]

Options:
  --branch <name>         Git branch to update to (default: current branch)
                          Available: main, dev
  --install-dir <path>    Installation directory (default: /opt/pekan)
  --no-backup             Skip database backup before update
  --mode <systemd|docker> Override deployment mode (auto-detect if not set)
  --help                  Show this help message

Examples:
  # Update current branch (auto-detect mode)
  sudo bash update-versi.sh

  # Switch to dev branch
  sudo bash update-versi.sh --branch dev

  # Switch to main branch
  sudo bash update-versi.sh --branch main

  # Update without backup
  sudo bash update-versi.sh --no-backup

  # Force docker mode
  sudo bash update-versi.sh --mode docker

Steps performed:
  1. Detect deployment mode (systemd or docker)
  2. Fetch latest code from repository
  3. Switch to specified branch (if --branch provided)
  4. Backup database (unless --no-backup)
  5. Sync code to installation directory
  6. Rebuild backend binaries (systemd) or Docker images (docker)
  7. Rebuild frontend
  8. Run database migrations
  9. Restart services/containers
  10. Verify health check
EOF
}

parse_args() {
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --branch)
        BRANCH="$2"
        shift 2
        ;;
      --install-dir)
        INSTALL_DIR="$2"
        shift 2
        ;;
      --no-backup)
        BACKUP_BEFORE_UPDATE="false"
        shift
        ;;
      --mode)
        DEPLOY_MODE="$2"
        shift 2
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

as_root() {
  if [[ "${EUID}" -eq 0 ]]; then "$@"; else sudo "$@"; fi
}

detect_deploy_mode() {
  if [[ -n "$DEPLOY_MODE" ]]; then
    log "Deployment mode: $DEPLOY_MODE (manual)"
    return
  fi

  if [[ -f /etc/systemd/system/pekan-api.service ]]; then
    DEPLOY_MODE="systemd"
  elif [[ -f "$INSTALL_DIR/docker-compose.yml" ]] && command -v docker &>/dev/null; then
    DEPLOY_MODE="docker"
  else
    die "Cannot detect deployment mode. Use --mode to specify."
  fi

  log "Deployment mode: $DEPLOY_MODE (auto-detected)"
}

get_current_version() {
  if [[ -f "$INSTALL_DIR/.version" ]]; then
    cat "$INSTALL_DIR/.version"
  else
    echo "unknown"
  fi
}

get_latest_commit() {
  cd "$REPO_DIR"
  git rev-parse --short HEAD
}

check_for_updates() {
  log "Step 1/10: Checking for updates..."

  cd "$REPO_DIR"
  git fetch origin --quiet 2>/dev/null || true

  local CURRENT_BRANCH
  CURRENT_BRANCH=$(git branch --show-current)

  # If --branch specified, switch to it
  if [[ -n "$BRANCH" ]]; then
    log "  Switching to branch: $BRANCH"
    git checkout "$BRANCH"
    git pull origin "$BRANCH"
  else
    BRANCH="$CURRENT_BRANCH"
    log "  Current branch: $BRANCH"

    local LOCAL_HASH
    LOCAL_HASH=$(git rev-parse HEAD)

    local REMOTE_HASH
    REMOTE_HASH=$(git rev-parse "origin/$BRANCH" 2>/dev/null || echo "$LOCAL_HASH")

    if [[ "$LOCAL_HASH" == "$REMOTE_HASH" ]]; then
      log "  Already up-to-date"
    else
      log "  Pulling latest changes..."
      git pull origin "$BRANCH"
    fi
  fi
}

backup_database() {
  if [[ "$BACKUP_BEFORE_UPDATE" != "true" ]]; then
    log "Step 2/10: Skipping backup (--no-backup)"
    return
  fi

  log "Step 2/10: Backing up database..."

  local BACKUP_DIR="$INSTALL_DIR/backups"
  local DATE
  DATE=$(date +%Y%m%d_%H%M%S)
  as_root mkdir -p "$BACKUP_DIR"

  if [[ "$DEPLOY_MODE" == "docker" ]]; then
    cd "$INSTALL_DIR"
    as_root docker compose exec -T pekan-postgres pg_dump -U postgres pekan | \
      gzip > "$BACKUP_DIR/pekan_pre_update_${DATE}.sql.gz" 2>/dev/null || warn "  Backup failed, continuing update..."
  elif [[ "$DEPLOY_MODE" == "systemd" ]]; then
    local DATABASE_URL
    DATABASE_URL=$(grep "^DATABASE_URL=" "$INSTALL_DIR/backend/.env" | cut -d= -f2-)
    if [[ -n "$DATABASE_URL" ]]; then
      pg_dump "$DATABASE_URL" | gzip > "$BACKUP_DIR/pekan_pre_update_${DATE}.sql.gz" 2>/dev/null || warn "  Backup failed, continuing update..."
    fi
  fi

  log "  Backup saved to $BACKUP_DIR/pekan_pre_update_${DATE}.sql.gz"
}

sync_code() {
  log "Step 3/10: Syncing code to $INSTALL_DIR..."

  as_root rsync -av --delete-after \
    --exclude ".git" \
    --exclude "frontend/node_modules" \
    --exclude "backend/.env" \
    --exclude ".env" \
    --exclude "backups" \
    --exclude "storage" \
    --exclude "logs" \
    "$REPO_DIR/" "$INSTALL_DIR/"

  log "  Code synced"
}

update_systemd() {
  log "Step 4/10: Stopping services..."
  as_root systemctl stop pekan-api pekan-worker pekan-ai 2>/dev/null || true

  sync_code

  log "Step 5/10: Building backend binaries..."
  export PATH=$PATH:/usr/local/go/bin
  local APP_USER
  APP_USER=$(stat -c '%U' "$INSTALL_DIR/bin/pekan-api" 2>/dev/null || echo "pekan")

  log "  Running go mod tidy..."
  as_root -u "$APP_USER" bash -c "cd '$INSTALL_DIR/backend' && /usr/local/go/bin/go mod tidy"

  log "  Building pekan-api..."
  as_root -u "$APP_USER" bash -c "cd '$INSTALL_DIR/backend' && CGO_ENABLED=0 /usr/local/go/bin/go build -ldflags='-s -w' -o '$INSTALL_DIR/bin/pekan-api' ./cmd/api"

  log "  Building pekan-worker..."
  as_root -u "$APP_USER" bash -c "cd '$INSTALL_DIR/backend' && CGO_ENABLED=0 /usr/local/go/bin/go build -ldflags='-s -w' -o '$INSTALL_DIR/bin/pekan-worker' ./cmd/worker"

  log "  Building pekan-ai..."
  as_root -u "$APP_USER" bash -c "cd '$INSTALL_DIR/backend' && CGO_ENABLED=0 /usr/local/go/bin/go build -ldflags='-s -w' -o '$INSTALL_DIR/bin/pekan-ai' ./cmd/ai"

  log "Step 6/10: Building frontend..."
  as_root bash -c "cd '$INSTALL_DIR/frontend' && npm ci && npm run build"

  log "Step 7/10: Running migrations..."
  local DATABASE_URL
  DATABASE_URL=$(grep "^DATABASE_URL=" "$INSTALL_DIR/backend/.env" | cut -d= -f2-)
  if [[ -f "$INSTALL_DIR/backend/scripts/apply_migrations.sh" ]]; then
    as_root -u "$APP_USER" bash -c "cd '$INSTALL_DIR/backend' && chmod +x ./scripts/apply_migrations.sh && DATABASE_URL='${DATABASE_URL}' ./scripts/apply_migrations.sh"
  fi

  log "Step 8/10: Starting services..."
  as_root systemctl start pekan-api pekan-worker pekan-ai

  log "  Systemd update complete!"
}

update_docker() {
  sync_code

  log "Step 5/10: Building Docker images..."
  cd "$INSTALL_DIR"
  as_root docker compose build --no-cache

  log "Step 6/10: Restarting containers..."
  as_root docker compose down
  as_root docker compose up -d

  log "Step 7/10: Waiting for database..."
  sleep 10

  log "  Docker update complete!"
}

save_version() {
  log "Step 9/10: Saving version..."

  local VERSION
  VERSION=$(get_latest_commit)
  echo "$VERSION" | as_root tee "$INSTALL_DIR/.version" >/dev/null

  log "  Version: $VERSION"
}

verify_update() {
  log "Step 10/10: Verifying update..."
  sleep 3

  local STATUS
  STATUS=$(curl -sf "http://127.0.0.1:8080/api/v1/healthz" 2>/dev/null || echo "FAILED")

  if [[ "$STATUS" == *"FAILED"* ]]; then
    warn "Health check failed. Check logs for details."
    if [[ "$DEPLOY_MODE" == "systemd" ]]; then
      warn "Logs: journalctl -u pekan-api -n 50"
    else
      warn "Logs: docker compose -f $INSTALL_DIR/docker-compose.yml logs --tail=50 pekan-api"
    fi
  else
    log "Health check passed!"
  fi

  printf '\n'
  printf '============================================\n'
  printf '  PEKAN Update Complete\n'
  printf '============================================\n'
  printf '\n'
  printf '  Branch:    %s\n' "${BRANCH}"
  printf '  Mode:      %s\n' "${DEPLOY_MODE}"
  printf '  Version:   %s\n' "$(get_latest_commit)"
  printf '  Health:    http://localhost:8080/api/v1/healthz\n'
  printf '\n'
}

main() {
  parse_args "$@"

  printf '============================================\n'
  printf '  PEKAN Version Updater\n'
  printf '============================================\n'
  printf '\n'

  log "Current version: $(get_current_version)"

  detect_deploy_mode
  check_for_updates
  backup_database

  case "$DEPLOY_MODE" in
    systemd)
      update_systemd
      ;;
    docker)
      update_docker
      ;;
    *)
      die "Invalid deployment mode: $DEPLOY_MODE"
      ;;
  esac

  save_version
  verify_update
}

main "$@"
