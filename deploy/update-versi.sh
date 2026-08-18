#!/usr/bin/env bash
# =============================================================================
# PEKAN Version Updater
# Update PEKAN to latest version (supports both systemd and docker deployments)
# =============================================================================
if [ -z "${BASH_VERSION:-}" ]; then
  exec /usr/bin/env bash "$0" "$@"
fi
set -Eeuo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log()   { printf "${GREEN}[INFO]${NC} %s\n" "$*"; }
warn()  { printf "${YELLOW}[WARN]${NC} %s\n" "$*"; }
error() { printf "${RED}[ERROR]${NC} %s\n" "$*" >&2; }
die()   { error "$*"; exit 1; }

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

INSTALL_DIR="${INSTALL_DIR:-/opt/pekan}"
DEPLOY_MODE=""
BACKUP_BEFORE_UPDATE="${BACKUP_BEFORE_UPDATE:-true}"

as_root() {
  if [[ "${EUID}" -eq 0 ]]; then "$@"; else sudo "$@"; fi
}

detect_deploy_mode() {
  if [[ -f /etc/systemd/system/pekan-api.service ]]; then
    DEPLOY_MODE="systemd"
    log "Terdeteksi deployment: systemd"
  elif [[ -f "$INSTALL_DIR/docker-compose.yml" ]] && command -v docker &>/dev/null; then
    DEPLOY_MODE="docker"
    log "Terdeteksi deployment: docker"
  else
    die "Tidak bisa mendeteksi mode deployment. Pastikan PEKAN sudah terinstall di $INSTALL_DIR"
  fi
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
  log "Mengecek update..."

  cd "$REPO_DIR"
  git fetch origin --quiet 2>/dev/null || true

  local CURRENT_BRANCH
  CURRENT_BRANCH=$(git branch --show-current)

  local LOCAL_HASH
  LOCAL_HASH=$(git rev-parse HEAD)

  local REMOTE_HASH
  REMOTE_HASH=$(git rev-parse "origin/$CURRENT_BRANCH" 2>/dev/null || echo "$LOCAL_HASH")

  if [[ "$LOCAL_HASH" == "$REMOTE_HASH" ]]; then
    log "Repository sudah up-to-date"
  else
    log "Update tersedia di remote. Melakukan pull..."
    git pull origin "$CURRENT_BRANCH"
  fi
}

backup_database() {
  if [[ "$BACKUP_BEFORE_UPDATE" != "true" ]]; then
    return
  fi

  log "Melakukan backup database sebelum update..."

  local BACKUP_DIR="$INSTALL_DIR/backups"
  local DATE
  DATE=$(date +%Y%m%d_%H%M%S)
  as_root mkdir -p "$BACKUP_DIR"

  if [[ "$DEPLOY_MODE" == "docker" ]]; then
    cd "$INSTALL_DIR"
    as_root docker compose exec -T pekan-postgres pg_dump -U postgres pekan | \
      gzip > "$BACKUP_DIR/pekan_pre_update_${DATE}.sql.gz" 2>/dev/null || warn "Backup gagal, melanjutkan update..."
  elif [[ "$DEPLOY_MODE" == "systemd" ]]; then
    local DATABASE_URL
    DATABASE_URL=$(grep "^DATABASE_URL=" "$INSTALL_DIR/backend/.env" | cut -d= -f2-)
    if [[ -n "$DATABASE_URL" ]]; then
      pg_dump "$DATABASE_URL" | gzip > "$BACKUP_DIR/pekan_pre_update_${DATE}.sql.gz" 2>/dev/null || warn "Backup gagal, melanjutkan update..."
    fi
  fi

  log "Backup tersimpan di $BACKUP_DIR/pekan_pre_update_${DATE}.sql.gz"
}

sync_code() {
  log "Menyinkronkan kode terbaru ke $INSTALL_DIR..."

  as_root rsync -av --delete-after \
    --exclude ".git" \
    --exclude "frontend/node_modules" \
    --exclude "backend/.env" \
    --exclude ".env" \
    --exclude "backups" \
    --exclude "storage" \
    --exclude "logs" \
    "$REPO_DIR/" "$INSTALL_DIR/"
}

update_systemd() {
  log "Update mode systemd..."

  # Stop services
  log "Menghentikan services..."
  as_root systemctl stop pekan-api pekan-worker pekan-ai 2>/dev/null || true

  sync_code

  # Rebuild backend
  log "Membuild backend binaries..."
  export PATH=$PATH:/usr/local/go/bin
  local APP_USER
  APP_USER=$(stat -c '%U' "$INSTALL_DIR/bin/pekan-api" 2>/dev/null || echo "pekan")

  as_root -u "$APP_USER" bash -c "cd '$INSTALL_DIR/backend' && /usr/local/go/bin/go mod tidy"
  as_root -u "$APP_USER" bash -c "cd '$INSTALL_DIR/backend' && CGO_ENABLED=0 /usr/local/go/bin/go build -ldflags='-s -w' -o '$INSTALL_DIR/bin/pekan-api' ./cmd/api"
  as_root -u "$APP_USER" bash -c "cd '$INSTALL_DIR/backend' && CGO_ENABLED=0 /usr/local/go/bin/go build -ldflags='-s -w' -o '$INSTALL_DIR/bin/pekan-worker' ./cmd/worker"
  as_root -u "$APP_USER" bash -c "cd '$INSTALL_DIR/backend' && CGO_ENABLED=0 /usr/local/go/bin/go build -ldflags='-s -w' -o '$INSTALL_DIR/bin/pekan-ai' ./cmd/ai"

  # Rebuild frontend
  log "Membuild frontend..."
  as_root bash -c "cd '$INSTALL_DIR/frontend' && npm ci && npm run build"

  # Run migrations
  log "Menjalankan migrasi database..."
  local DATABASE_URL
  DATABASE_URL=$(grep "^DATABASE_URL=" "$INSTALL_DIR/backend/.env" | cut -d= -f2-)
  as_root -u "$APP_USER" bash -c "cd '$INSTALL_DIR/backend' && chmod +x ./scripts/apply_migrations.sh && DATABASE_URL='${DATABASE_URL}' ./scripts/apply_migrations.sh"

  # Restart services
  log "Memulai services..."
  as_root systemctl start pekan-api pekan-worker pekan-ai

  log "Update systemd selesai!"
}

update_docker() {
  log "Update mode docker..."

  cd "$INSTALL_DIR"

  # Sync code first (preserve .env)
  sync_code

  # Rebuild and restart
  log "Membangun ulang Docker images..."
  as_root docker compose build --no-cache

  log "Merestart containers..."
  as_root docker compose down
  as_root docker compose up -d

  # Wait for DB to be ready
  log "Menunggu database siap..."
  sleep 10

  log "Update docker selesai!"
}

save_version() {
  local VERSION
  VERSION=$(get_latest_commit)
  echo "$VERSION" | as_root tee "$INSTALL_DIR/.version" >/dev/null
}

verify_update() {
  log "Memverifikasi update..."
  sleep 3

  local STATUS
  STATUS=$(curl -sf "http://127.0.0.1:8080/api/v1/healthz" 2>/dev/null || echo "FAILED")

  if [[ "$STATUS" == *"FAILED"* ]]; then
    warn "Health check gagal. Cek logs untuk detail."
    if [[ "$DEPLOY_MODE" == "systemd" ]]; then
      warn "Logs: journalctl -u pekan-api -n 50"
    else
      warn "Logs: docker compose -f $INSTALL_DIR/docker-compose.yml logs --tail=50 pekan-api"
    fi
  else
    log "Health check berhasil!"
  fi

  printf "\n"
  printf "${GREEN}============================================${NC}\n"
  printf "${GREEN}  PEKAN Berhasil Diupdate${NC}\n"
  printf "${GREEN}============================================${NC}\n"
  printf "\n"
  printf "  Mode     : ${DEPLOY_MODE}\n"
  printf "  Version  : $(get_latest_commit)\n"
  printf "  Health   : http://localhost:8080/api/v1/healthz\n"
  printf "\n"
}

show_help() {
  cat <<EOF
PEKAN Version Updater

Penggunaan: $0 [opsi]

Opsi:
  --install-dir <path>    Lokasi instalasi (default: /opt/pekan)
  --no-backup             Skip backup database sebelum update
  --mode <systemd|docker> Override mode deployment (auto-detect default)
  --help                  Tampilkan bantuan ini

Contoh:
  sudo bash $0                        # Auto-detect dan update
  sudo bash $0 --mode docker          # Force docker mode
  sudo bash $0 --no-backup            # Skip backup
  sudo bash $0 --install-dir /app     # Custom install directory
EOF
}

main() {
  while [[ $# -gt 0 ]]; do
    case "$1" in
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
        die "Opsi tidak dikenal: $1. Gunakan --help untuk bantuan."
        ;;
    esac
  done

  printf "${BLUE}"
  printf "╔═══════════════════════════════════════════╗\n"
  printf "║     PEKAN Version Updater                 ║\n"
  printf "║     Update ke versi terbaru               ║\n"
  printf "╚═══════════════════════════════════════════╝\n"
  printf "${NC}\n"

  log "Versi saat ini: $(get_current_version)"

  if [[ -z "$DEPLOY_MODE" ]]; then
    detect_deploy_mode
  fi

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
      die "Mode deployment tidak valid: $DEPLOY_MODE"
      ;;
  esac

  save_version
  verify_update
}

main "$@"
