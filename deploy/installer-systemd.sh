#!/usr/bin/env bash
# =============================================================================
# PEKAN Systemd Installer
# Install PEKAN with native PostgreSQL, Redis, Go binaries, Nginx, and systemd
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
APP_USER="${APP_USER:-pekan}"
APP_GROUP="${APP_GROUP:-pekan}"
GO_VERSION="${GO_VERSION:-1.23.8}"
APP_ENV="${APP_ENV:-production}"
HTTP_PORT="${HTTP_PORT:-8080}"
WEB_PORT="${WEB_PORT:-80}"
DB_NAME="${DB_NAME:-pekan}"
DB_USER="${DB_USER:-postgres}"
DB_PASS="${DB_PASS:-}"
DB_HOST="${DB_HOST:-127.0.0.1}"
DB_PORT="${DB_PORT:-5432}"

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
    OS_LIKE="${ID_LIKE:-}"
  else
    die "Tidak bisa mendeteksi OS. Script ini mendukung Ubuntu/Debian."
  fi

  if [[ "$OS_ID" =~ ^(rocky|almalinux|rhel|centos|ol)$ ]] || [[ "$OS_LIKE" =~ (rhel|fedora|centos) ]]; then
    PKG_MANAGER="dnf"
    PKG_INSTALL="dnf install -y"
  elif [[ "$OS_ID" =~ ^(ubuntu|debian)$ ]] || [[ "$OS_LIKE" =~ (ubuntu|debian) ]]; then
    PKG_MANAGER="apt"
    PKG_INSTALL="apt install -y"
  else
    die "OS tidak didukung: $OS_ID. Gunakan Ubuntu/Debian atau Rocky/AlmaLinux."
  fi
  log "Terdeteksi OS: $OS_ID"
}

generate_secrets() {
  if [[ -z "$DB_PASS" ]]; then
    DB_PASS="$(openssl rand -base64 24 | tr -d '/+=' | head -c 24)"
    log "Password PostgreSQL di-generate otomatis"
  fi

  JWT_SECRET="$(openssl rand -base64 48)"
  RECEIPT_SCAN_SECRET="$(openssl rand -base64 32)"
  ADMIN_SECRET="$(openssl rand -base64 32)"
  log "Secret keys di-generate"
}

install_packages() {
  log "Menginstall packages yang dibutuhkan..."

  if [[ "$PKG_MANAGER" == "apt" ]]; then
    as_root apt update
    as_root $PKG_INSTALL -y curl wget git build-essential ca-certificates gnupg lsb-release

    # PostgreSQL 16
    if ! command -v psql &>/dev/null; then
      curl -fsSL https://www.postgresql.org/media/keys/ACCC4CF8.asc | as_root gpg --dearmor -o /usr/share/keyrings/postgresql-keyring.gpg
      echo "deb [signed-by=/usr/share/keyrings/postgresql-keyring.gpg] http://apt.postgresql.org/pub/repos/apt $(lsb_release -cs)-pgdg main" | as_root tee /etc/apt/sources.list.d/pgdg.list
      as_root apt update
      as_root $PKG_INSTALL -y postgresql-16 postgresql-client-16
    fi

    # Redis
    if ! command -v redis-server &>/dev/null; then
      curl -fsSL https://packages.redis.io/gpg | as_root gpg --dearmor -o /usr/share/keyrings/redis-archive-keyring.gpg
      echo "deb [signed-by=/usr/share/keyrings/redis-archive-keyring.gpg] https://packages.redis.io/deb $(lsb_release -cs) main" | as_root tee /etc/apt/sources.list.d/redis.list
      as_root apt update
      as_root $PKG_INSTALL -y redis
    fi

    # Nginx
    if ! command -v nginx &>/dev/null; then
      as_root $PKG_INSTALL -y nginx
    fi

  elif [[ "$PKG_MANAGER" == "dnf" ]]; then
    as_root dnf install -y curl wget git gcc make ca-certificates

    # PostgreSQL 16
    if ! command -v psql &>/dev/null; then
      as_root dnf install -y https://download.postgresql.org/pub/repos/yum/reporpms/EL-9-x86_64/pgdg-redhat-repo-latest.noarch.rpm
      as_root dnf -y install postgresql16-server postgresql16
      as_root /usr/pgsql-16/bin/postgresql-16-setup initdb
      as_root systemctl enable postgresql-16
      as_root systemctl start postgresql-16
    fi

    # Redis
    if ! command -v redis-server &>/dev/null; then
      as_root dnf install -y epel-release
      as_root dnf install -y redis
      as_root systemctl enable redis
      as_root systemctl start redis
    fi

    # Nginx
    if ! command -v nginx &>/dev/null; then
      as_root dnf install -y nginx
    fi
  fi

  log "Packages berhasil diinstall"
}

install_go() {
  if command -v go &>/dev/null; then
    CURRENT_GO=$(go version | awk '{print $3}' | sed 's/go//')
    log "Go sudah terinstall: $CURRENT_GO"
    return
  fi

  log "Menginstall Go $GO_VERSION..."
  wget -q "https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz" -O /tmp/go.tar.gz
  as_root rm -rf /usr/local/go
  as_root tar -C /usr/local -xzf /tmp/go.tar.gz
  rm -f /tmp/go.tar.gz

  echo 'export PATH=$PATH:/usr/local/go/bin' >> /etc/profile.d/go.sh
  export PATH=$PATH:/usr/local/go/bin
  log "Go $GO_VERSION berhasil diinstall"
}

setup_database() {
  log "Mengkonfigurasi PostgreSQL..."

  as_root systemctl enable postgresql
  as_root systemctl start postgresql

  # Set password for postgres user
  as_root -u postgres psql -c "ALTER USER postgres WITH PASSWORD '${DB_PASS}';" 2>/dev/null || true

  # Create database if not exists
  as_root -u postgres psql -tc "SELECT 1 FROM pg_database WHERE datname = '${DB_NAME}'" | grep -q 1 || \
    as_root -u postgres psql -c "CREATE DATABASE ${DB_NAME};"

  # Configure pg_hba for local password auth
  PG_HBA=$(as_root -u postgres psql -t -c "SHOW hba_file;" | tr -d ' ')
  if ! grep -q "host.*${DB_NAME}.*127.0.0.1/32.*md5" "$PG_HBA" 2>/dev/null; then
    echo "host ${DB_NAME} ${DB_USER} 127.0.0.1/32 md5" | as_root tee -a "$PG_HBA" >/dev/null
    as_root systemctl restart postgresql
  fi

  log "PostgreSQL berhasil dikonfigurasi"
}

setup_redis() {
  log "Mengkonfigurasi Redis..."
  as_root systemctl enable redis-server 2>/dev/null || as_root systemctl enable redis
  as_root systemctl start redis-server 2>/dev/null || as_root systemctl start redis
  log "Redis berhasil dikonfigurasi"
}

setup_app_user() {
  if ! id "$APP_USER" &>/dev/null; then
    as_root useradd --system --shell /usr/sbin/nologin --home-dir "$INSTALL_DIR" "$APP_USER"
    log "User '$APP_USER' dibuat"
  fi
}

deploy_application() {
  log "Mendeploy aplikasi ke $INSTALL_DIR..."

  as_root mkdir -p "$INSTALL_DIR"/{bin,logs,storage}
  as_root rsync -av --delete-after \
    --exclude ".git" \
    --exclude "frontend/node_modules" \
    --exclude "backend/.env" \
    --exclude "*.md" \
    --exclude "deploy" \
    --exclude "docs" \
    "$REPO_DIR/" "$INSTALL_DIR/"

  # Build backend
  log "Membuild backend binaries..."
  export PATH=$PATH:/usr/local/go/bin
  as_root -u "$APP_USER" bash -c "cd '$INSTALL_DIR/backend' && /usr/local/go/bin/go mod tidy"
  as_root -u "$APP_USER" bash -c "cd '$INSTALL_DIR/backend' && CGO_ENABLED=0 /usr/local/go/bin/go build -ldflags='-s -w' -o '$INSTALL_DIR/bin/pekan-api' ./cmd/api"
  as_root -u "$APP_USER" bash -c "cd '$INSTALL_DIR/backend' && CGO_ENABLED=0 /usr/local/go/bin/go build -ldflags='-s -w' -o '$INSTALL_DIR/bin/pekan-worker' ./cmd/worker"
  as_root -u "$APP_USER" bash -c "cd '$INSTALL_DIR/backend' && CGO_ENABLED=0 /usr/local/go/bin/go build -ldflags='-s -w' -o '$INSTALL_DIR/bin/pekan-ai' ./cmd/ai"

  # Build frontend
  log "Membuild frontend..."
  as_root bash -c "cd '$INSTALL_DIR/frontend' && npm ci && npm run build"

  # Write .env
  DATABASE_URL="postgres://${DB_USER}:${DB_PASS}@${DB_HOST}:${DB_PORT}/${DB_NAME}?sslmode=disable"
  cat <<EOF | as_root tee "$INSTALL_DIR/backend/.env" >/dev/null
APP_ENV=${APP_ENV}
HTTP_PORT=${HTTP_PORT}
DATABASE_URL=${DATABASE_URL}
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
RATE_LIMIT_REDIS_URL=redis://127.0.0.1:6379/0
RATE_LIMIT_REDIS_PREFIX=pekan:ratelimit
STORAGE_PROVIDER=local
STORAGE_LOCAL_PATH=${INSTALL_DIR}/storage
ADMIN_SECRET=${ADMIN_SECRET}
EOF

  as_root chown -R "${APP_USER}:${APP_GROUP}" "$INSTALL_DIR"
  log "Aplikasi berhasil dideploy"
}

run_migrations() {
  log "Menjalankan migrasi database..."
  DATABASE_URL="postgres://${DB_USER}:${DB_PASS}@${DB_HOST}:${DB_PORT}/${DB_NAME}?sslmode=disable"
  as_root -u "$APP_USER" bash -c "cd '$INSTALL_DIR/backend' && chmod +x ./scripts/apply_migrations.sh && DATABASE_URL='${DATABASE_URL}' ./scripts/apply_migrations.sh"
  log "Migrasi berhasil"
}

setup_systemd() {
  log "Mengkonfigurasi systemd services..."

  cat <<EOF | as_root tee /etc/systemd/system/pekan-api.service >/dev/null
[Unit]
Description=PEKAN API Server
After=network.target postgresql.service redis.service
Wants=postgresql.service redis.service

[Service]
Type=simple
User=${APP_USER}
Group=${APP_GROUP}
WorkingDirectory=${INSTALL_DIR}
ExecStart=${INSTALL_DIR}/bin/pekan-api
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal
SyslogIdentifier=pekan-api
Environment=PATH=/usr/local/go/bin:/usr/bin:/bin

[Install]
WantedBy=multi-user.target
EOF

  cat <<EOF | as_root tee /etc/systemd/system/pekan-worker.service >/dev/null
[Unit]
Description=PEKAN Background Worker
After=network.target postgresql.service redis.service
Wants=postgresql.service redis.service

[Service]
Type=simple
User=${APP_USER}
Group=${APP_GROUP}
WorkingDirectory=${INSTALL_DIR}
ExecStart=${INSTALL_DIR}/bin/pekan-worker
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal
SyslogIdentifier=pekan-worker
Environment=PATH=/usr/local/go/bin:/usr/bin:/bin

[Install]
WantedBy=multi-user.target
EOF

  cat <<EOF | as_root tee /etc/systemd/system/pekan-ai.service >/dev/null
[Unit]
Description=PEKAN AI Queue Worker
After=network.target postgresql.service redis.service
Wants=postgresql.service redis.service

[Service]
Type=simple
User=${APP_USER}
Group=${APP_GROUP}
WorkingDirectory=${INSTALL_DIR}
ExecStart=${INSTALL_DIR}/bin/pekan-ai
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal
SyslogIdentifier=pekan-ai
Environment=PATH=/usr/local/go/bin:/usr/bin:/bin

[Install]
WantedBy=multi-user.target
EOF

  as_root systemctl daemon-reload
  as_root systemctl enable pekan-api pekan-worker pekan-ai
  log "Systemd services berhasil dikonfigurasi"
}

setup_nginx() {
  log "Mengkonfigurasi Nginx..."

  cat <<EOF | as_root tee /etc/nginx/sites-available/pekan >/dev/null
server {
    listen ${WEB_PORT};
    server_name _;

    root ${INSTALL_DIR}/frontend/dist;
    index index.html;

    gzip on;
    gzip_types text/plain text/css application/json application/javascript text/xml application/xml image/svg+xml;
    gzip_min_length 256;

    location / {
        try_files \$uri \$uri/ /index.html;
    }

    location /api/ {
        proxy_pass http://127.0.0.1:${HTTP_PORT};
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
        proxy_set_header X-Request-ID \$request_id;
        proxy_connect_timeout 30s;
        proxy_send_timeout 30s;
        proxy_read_timeout 30s;
    }

    location /webhook/ {
        proxy_pass http://127.0.0.1:${HTTP_PORT};
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
    }

    location ~* \.(js|css|png|jpg|jpeg|gif|ico|svg|woff|woff2|ttf|eot)$ {
        expires 1y;
        add_header Cache-Control "public, immutable";
    }

    location /healthz {
        access_log off;
        proxy_pass http://127.0.0.1:${HTTP_PORT}/api/v1/healthz;
    }
}
EOF

  as_root ln -sf /etc/nginx/sites-available/pekan /etc/nginx/sites-enabled/pekan
  as_root rm -f /etc/nginx/sites-enabled/default
  as_root nginx -t
  as_root systemctl enable nginx
  as_root systemctl restart nginx
  log "Nginx berhasil dikonfigurasi"
}

start_services() {
  log "Memulai services..."
  as_root systemctl start pekan-api
  as_root systemctl start pekan-worker
  as_root systemctl start pekan-ai
  log "Semua services berhasil dimulai"
}

verify_installation() {
  log "Memverifikasi instalasi..."
  sleep 3

  local STATUS
  STATUS=$(curl -sf "http://127.0.0.1:${HTTP_PORT}/api/v1/healthz" 2>/dev/null || echo "FAILED")

  if [[ "$STATUS" == *"FAILED"* ]]; then
    warn "Health check gagal. Cek logs: journalctl -u pekan-api -f"
  else
    log "Health check berhasil!"
  fi

  printf "\n"
  printf "${GREEN}============================================${NC}\n"
  printf "${GREEN}  PEKAN Berhasil Diinstall (Systemd)${NC}\n"
  printf "${GREEN}============================================${NC}\n"
  printf "\n"
  printf "  Aplikasi : http://localhost:${WEB_PORT}\n"
  printf "  API      : http://localhost:${HTTP_PORT}/api/v1\n"
  printf "  Health   : http://localhost:${HTTP_PORT}/api/v1/healthz\n"
  printf "\n"
  printf "  ${YELLOW}Konfigurasi:${NC} ${INSTALL_DIR}/backend/.env\n"
  printf "  ${YELLOW}Logs:${NC}       journalctl -u pekan-api -f\n"
  printf "  ${YELLOW}Storage:${NC}    ${INSTALL_DIR}/storage\n"
  printf "\n"
  printf "  ${BLUE}Service Commands:${NC}\n"
  printf "    systemctl status pekan-api\n"
  printf "    systemctl restart pekan-api\n"
  printf "    systemctl status pekan-worker\n"
  printf "    systemctl status pekan-ai\n"
  printf "\n"
}

main() {
  printf "${BLUE}"
  printf "╔═══════════════════════════════════════════╗\n"
  printf "║     PEKAN Systemd Installer               ║\n"
  printf "║     Platform Pencatatan Keuangan          ║\n"
  printf "╚═══════════════════════════════════════════╝\n"
  printf "${NC}\n"

  check_root
  detect_os
  generate_secrets
  install_packages
  install_go
  setup_database
  setup_redis
  setup_app_user
  deploy_application
  run_migrations
  setup_systemd
  setup_nginx
  start_services
  verify_installation
}

main "$@"
