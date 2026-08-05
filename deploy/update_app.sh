#!/usr/bin/env bash
# Update Script for PEKAN Platform
# This script updates the code and rebuilds everything while keeping existing .env config.

set -Eeuo pipefail

INSTALL_DIR="/opt/pekan"
APP_USER="pekan"
APP_GROUP="pekan"
WEB_ROOT="/var/www/pekan-web"

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

as_app() {
  if [[ "${EUID}" -eq 0 ]]; then
    runuser -u "$APP_USER" -- bash -lc "$*"
  else
    sudo -u "$APP_USER" -H bash -lc "$*"
  fi
}

main() {
  log "Starting application update..."

  if [[ ! -d "$INSTALL_DIR" ]]; then
    die "Installation directory $INSTALL_DIR not found. Please run install_server.sh first."
  fi

  # 1. Sync files (excluding .env and node_modules)
  log "Syncing latest code to ${INSTALL_DIR}"
  as_root rsync -av --delete-after \
    --exclude ".git" \
    --exclude "frontend/node_modules" \
    --exclude "backend/.env" \
    --exclude "frontend/.env.production" \
    ./ "${INSTALL_DIR}/" || die "Sync failed. Check permissions or disk space."
    
  as_root chown -R "${APP_USER}:${APP_GROUP}" "$INSTALL_DIR"

  # 2. Load Existing Config
  if [[ ! -f "${INSTALL_DIR}/backend/.env" ]]; then
    die "Existing .env not found in ${INSTALL_DIR}/backend/. Update cannot proceed without config."
  fi
  DATABASE_URL=$(grep "^DATABASE_URL=" "${INSTALL_DIR}/backend/.env" | cut -d= -f2-)
  HTTP_PORT=$(grep "^HTTP_PORT=" "${INSTALL_DIR}/backend/.env" | cut -d= -f2- || echo "8080")

  # 3. Rebuild Backend
  log "Rebuilding backend binaries"
  
  # Verify go.mod existence
  if [[ ! -f "${INSTALL_DIR}/backend/go.mod" ]]; then
    die "go.mod not found in ${INSTALL_DIR}/backend. Sync may have failed silently or project structure is invalid."
  fi

  as_app "cd '${INSTALL_DIR}/backend' && /usr/local/go/bin/go mod tidy"
  as_app "cd '${INSTALL_DIR}/backend' && /usr/local/go/bin/go build -o '${INSTALL_DIR}/bin/pekan-api' ./cmd/api"
  as_app "cd '${INSTALL_DIR}/backend' && /usr/local/go/bin/go build -o '${INSTALL_DIR}/bin/pekan-worker' ./cmd/worker"
  as_app "cd '${INSTALL_DIR}/backend' && /usr/local/go/bin/go build -o '${INSTALL_DIR}/bin/pekan-ai' ./cmd/ai"

  # 4. Apply Migrations
  log "Applying database migrations"
  as_app "cd '${INSTALL_DIR}/backend' && chmod +x ./scripts/apply_migrations.sh && DATABASE_URL='${DATABASE_URL}' ./scripts/apply_migrations.sh"

  # 5. Rebuild Frontend
  log "Rebuilding frontend"
  # Generate frontend .env from backend .env if missing, or just keep it
  if [[ ! -f "${INSTALL_DIR}/frontend/.env.production" ]]; then
     log "Generating frontend .env.production"
     echo "VITE_API_BASE_URL=/api/v1" | as_root tee "${INSTALL_DIR}/frontend/.env.production" >/dev/null
     as_root chown "${APP_USER}:${APP_GROUP}" "${INSTALL_DIR}/frontend/.env.production"
  fi

  as_app "cd '${INSTALL_DIR}/frontend' && npm install --include=dev --no-audit && npm run build"
  
  log "Publishing frontend to ${WEB_ROOT}"
  as_root rm -rf "${WEB_ROOT:?}/"*
  as_root cp -r "${INSTALL_DIR}/frontend/dist/"* "${WEB_ROOT}/"

  # 6. Update and Restart Services
  log "Updating systemd service units"
  as_root cp "${INSTALL_DIR}/deploy/systemd/pekan-api.service" /etc/systemd/system/
  as_root cp "${INSTALL_DIR}/deploy/systemd/pekan-worker.service" /etc/systemd/system/
  as_root cp "${INSTALL_DIR}/deploy/systemd/pekan-ai.service" /etc/systemd/system/
  as_root systemctl daemon-reload
  
  log "Restarting systemd services"
  as_root systemctl restart pekan-api pekan-worker pekan-ai

  # 7. Health Check
  log "Verifying health..."
  sleep 3
  if curl -fsS "http://127.0.0.1:${HTTP_PORT}/api/v1/healthz" >/dev/null 2>&1; then
    log "Update completed successfully! API is healthy."
  else
    die "API is not healthy after update. Check logs: journalctl -u pekan-api -n 50"
  fi
}

main "$@"
