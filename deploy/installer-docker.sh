#!/usr/bin/env bash
# =============================================================================
# PEKAN Docker Installer
# Install PEKAN with Docker containers (all-in-one)
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

INSTALL_DIR="${INSTALL_DIR:-/opt/pekan}"
APP_ENV="${APP_ENV:-production}"
WEB_PORT="${WEB_PORT:-80}"
POSTGRES_PASSWORD="${POSTGRES_PASSWORD:-}"

as_root() {
  if [[ "${EUID}" -eq 0 ]]; then "$@"; else sudo "$@"; fi
}

check_root() {
  if [[ "${EUID}" -ne 0 ]]; then
    die "Script ini harus dijalankan sebagai root. Gunakan: sudo bash $0"
  fi
}

detect_os() {
  if [[ -f /etc/os-release ]]; then
    . /etc/os-release
    OS_ID="${ID:-}"
  else
    die "Tidak bisa mendeteksi OS."
  fi
  log "Terdeteksi OS: $OS_ID"
}

install_docker() {
  if command -v docker &>/dev/null; then
    log "Docker sudah terinstall: $(docker --version)"
    return
  fi

  log "Menginstall Docker..."

  if command -v apt &>/dev/null; then
    as_root apt update
    as_root apt install -y ca-certificates curl gnupg lsb-release

    as_root mkdir -p /etc/apt/keyrings
    curl -fsSL https://download.docker.com/linux/ubuntu/gpg | as_root gpg --dearmor -o /etc/apt/keyrings/docker.gpg

    echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable" | \
      as_root tee /etc/apt/sources.list.d/docker.list >/dev/null

    as_root apt update
    as_root apt install -y docker-ce docker-ce-cli containerd.io docker-compose-plugin

  elif command -v dnf &>/dev/null; then
    as_root dnf install -y dnf-plugins-core
    as_root dnf config-manager --add-repo https://download.docker.com/linux/centos/docker-ce.repo
    as_root dnf install -y docker-ce docker-ce-cli containerd.io docker-compose-plugin

  elif command -v yum &>/dev/null; then
    as_root yum install -y yum-utils
    as_root yum-config-manager --add-repo https://download.docker.com/linux/centos/docker-ce.repo
    as_root yum install -y docker-ce docker-ce-cli containerd.io docker-compose-plugin
  fi

  as_root systemctl enable docker
  as_root systemctl start docker
  log "Docker berhasil diinstall"
}

generate_secrets() {
  if [[ -z "$POSTGRES_PASSWORD" ]]; then
    POSTGRES_PASSWORD="$(openssl rand -base64 24 | tr -d '/+=' | head -c 24)"
    log "Password PostgreSQL di-generate otomatis"
  fi

  JWT_SECRET="$(openssl rand -base64 48)"
  RECEIPT_SCAN_SECRET="$(openssl rand -base64 32)"
  ADMIN_SECRET="$(openssl rand -base64 32)"
  log "Secret keys di-generate"
}

deploy_application() {
  log "Mendeploy aplikasi ke $INSTALL_DIR..."

  as_root mkdir -p "$INSTALL_DIR"
  as_root rsync -av --delete-after \
    --exclude ".git" \
    --exclude "frontend/node_modules" \
    --exclude "backend/.env" \
    --exclude "*.md" \
    --exclude "docs" \
    --exclude "deploy/install_server.sh" \
    --exclude "deploy/install_server_rocky.sh" \
    "$REPO_DIR/" "$INSTALL_DIR/"

  # Write .env
  cat <<EOF | as_root tee "$INSTALL_DIR/backend/.env" >/dev/null
APP_ENV=${APP_ENV}
HTTP_PORT=8080
DATABASE_URL=postgres://postgres:${POSTGRES_PASSWORD}@pekan-postgres:5432/pekan?sslmode=disable
DB_MAX_OPEN_CONNS=25
DB_MAX_IDLE_CONNS=25
DB_CONN_MAX_LIFETIME_MINUTES=30
DB_CONN_MAX_IDLE_MINUTES=5
JWT_SECRET=${JWT_SECRET}
RECEIPT_SCAN_SECRET=${RECEIPT_SCAN_SECRET}
JWT_ISSUER=pekan-api
JWT_ACCESS_TTL_MINUTES=15
JWT_REFRESH_TTL_HOURS=720
CORS_ALLOWED_ORIGINS=http://localhost:${WEB_PORT}
REQUEST_BODY_MAX_BYTES=1048576
API_RATE_LIMIT_PER_MINUTE=1000
API_RATE_LIMIT_WINDOW_SECONDS=60
API_REQUEST_TIMEOUT_SECONDS=30
MAX_HEADER_BYTES=1048576
RATE_LIMIT_REDIS_URL=redis://pekan-redis:6379/0
RATE_LIMIT_REDIS_PREFIX=pekan:ratelimit
STORAGE_PROVIDER=local
STORAGE_LOCAL_PATH=/app/storage
ADMIN_SECRET=${ADMIN_SECRET}
EOF

  # Write root .env for docker-compose
  cat <<EOF | as_root tee "$INSTALL_DIR/.env" >/dev/null
POSTGRES_PASSWORD=${POSTGRES_PASSWORD}
EOF

  log "Aplikasi berhasil dideploy"
}

build_and_start() {
  log "Building dan memulai Docker containers..."

  cd "$INSTALL_DIR"

  as_root docker compose build --no-cache

  as_root docker compose up -d pekan-postgres pekan-redis

  log "Menunggu database siap..."
  sleep 10

  # Start all services
  as_root docker compose up -d

  log "Semua containers berhasil dimulai"
}

verify_installation() {
  log "Memverifikasi instalasi..."
  sleep 5

  cd "$INSTALL_DIR"
  as_root docker compose ps

  local STATUS
  STATUS=$(curl -sf "http://127.0.0.1:8080/api/v1/healthz" 2>/dev/null || echo "FAILED")

  if [[ "$STATUS" == *"FAILED"* ]]; then
    warn "Health check belum berhasil. Container mungkin masih starting. Cek: docker compose logs pekan-api"
  else
    log "Health check berhasil!"
  fi

  printf '\n'
  printf '============================================\n'
  printf '  PEKAN Berhasil Diinstall (Docker)\n'
  printf '============================================\n'
  printf '\n'
  printf '  Aplikasi : http://localhost:%s\n' "${WEB_PORT}"
  printf '  API      : http://localhost:8080/api/v1\n'
  printf '  Health   : http://localhost:8080/api/v1/healthz\n'
  printf '\n'
  printf '  Konfigurasi: %s/backend/.env\n' "${INSTALL_DIR}"
  printf '  Compose:    %s/docker-compose.yml\n' "${INSTALL_DIR}"
  printf '  Logs:       docker compose -f %s/docker-compose.yml logs -f\n' "${INSTALL_DIR}"
  printf '\n'
  printf '  Container Commands:\n'
  printf '    docker compose ps                    # Status containers\n'
  printf '    docker compose logs -f pekan-api     # Logs API\n'
  printf '    docker compose restart pekan-api     # Restart API\n'
  printf '    docker compose down                  # Stop semua\n'
  printf '    docker compose up -d                 # Start semua\n'
  printf '\n'
  printf '  Services:\n'
  printf '    pekan-postgres  (PostgreSQL 16)\n'
  printf '    pekan-redis     (Redis 7)\n'
  printf '    pekan-api       (API Server - port 8080)\n'
  printf '    pekan-worker    (Background Worker)\n'
  printf '    pekan-ai        (AI Queue Worker)\n'
  printf '    pekan-web       (Frontend Nginx - port %s)\n' "${WEB_PORT}"
  printf '\n'
}

setup_backup_cron() {
  log "Mengkonfigurasi backup otomatis..."

  BACKUP_SCRIPT="$INSTALL_DIR/scripts/backup-docker.sh"
  as_root mkdir -p "$INSTALL_DIR/scripts"

  cat <<'BACKUP_EOF' | as_root tee "$BACKUP_SCRIPT" >/dev/null
#!/usr/bin/env bash
set -euo pipefail
INSTALL_DIR="/opt/pekan"
BACKUP_DIR="$INSTALL_DIR/backups"
DATE=$(date +%Y%m%d_%H%M%S)

mkdir -p "$BACKUP_DIR"

cd "$INSTALL_DIR"
docker compose exec -T pekan-postgres pg_dump -U postgres pekan | gzip > "$BACKUP_DIR/pekan_${DATE}.sql.gz"

# Keep last 7 backups
ls -t "$BACKUP_DIR"/pekan_*.sql.gz | tail -n +8 | xargs -r rm

echo "Backup completed: pekan_${DATE}.sql.gz"
BACKUP_EOF

  as_root chmod +x "$BACKUP_SCRIPT"

  # Add cron job for daily backup at 2 AM
  (crontab -l 2>/dev/null || true; echo "0 2 * * * $BACKUP_SCRIPT >> $INSTALL_DIR/logs/backup.log 2>&1") | as_root crontab -

  log "Backup otomatis dikonfigurasi (setiap jam 02:00)"
}

main() {
  printf '============================================\n'
  printf '  PEKAN Docker Installer\n'
  printf '  Platform Pencatatan Keuangan\n'
  printf '============================================\n'
  printf '\n'

  check_root
  detect_os
  install_docker
  generate_secrets
  deploy_application
  build_and_start
  setup_backup_cron
  verify_installation
}

main "$@"
