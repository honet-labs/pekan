#!/usr/bin/env bash
# =============================================================================
# PEKAN Systemd Installer
# Install PEKAN with native PostgreSQL, Redis, Go binaries, Nginx, and systemd
#
# Usage:
#   sudo bash installer-systemd.sh [options]
#
# Options:
#   --branch <name>         Git branch to install (default: main)
#   --install-dir <path>    Installation directory (default: /opt/pekan)
#   --http-port <port>      API server port (default: 8080)
#   --web-port <port>       Web/Nginx port (default: 80)
#   --db-pass <password>    PostgreSQL password (auto-generated if empty)
#   --jwt-secret <secret>   JWT secret (auto-generated if empty)
#   --skip-deps             Skip dependency installation
#   --skip-migrate          Skip database migration
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
BRANCH="${BRANCH:-main}"
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
JWT_SECRET="${JWT_SECRET:-}"
SKIP_DEPS=0
SKIP_MIGRATE=0

show_help() {
  cat <<'EOF'
PEKAN Systemd Installer

Install PEKAN with native PostgreSQL, Redis, Go binaries, Nginx, and systemd.

Usage:
  sudo bash installer-systemd.sh [options]

Options:
  --branch <name>         Git branch to install (default: main)
                          Available: main, dev
  --install-dir <path>    Installation directory (default: /opt/pekan)
  --http-port <port>      API server port (default: 8080)
  --web-port <port>       Web/Nginx port (default: 80)
  --db-pass <password>    PostgreSQL password (auto-generated if empty)
  --jwt-secret <secret>   JWT secret (auto-generated if empty)
  --skip-deps             Skip dependency installation (Go, Node, PostgreSQL, Redis, Nginx)
  --skip-migrate          Skip database migration
  --help                  Show this help message

Examples:
  # Install from main branch (default)
  sudo bash installer-systemd.sh

  # Install from dev branch
  sudo bash installer-systemd.sh --branch dev

  # Install with custom ports
  sudo bash installer-systemd.sh --http-port 9090 --web-port 80

  # Install with pre-set secrets
  sudo bash installer-systemd.sh --db-pass "mypassword" --jwt-secret "my-jwt-secret"

Steps performed:
  1. Detect OS (Ubuntu/Debian or Rocky/AlmaLinux)
  2. Install dependencies (PostgreSQL, Redis, Go, Nginx)
  3. Configure PostgreSQL database
  4. Configure Redis
  5. Clone/update repository from specified branch
  6. Build backend binaries (api, worker, ai)
  7. Build frontend
  8. Run database migrations
  9. Configure systemd services
  10. Configure Nginx reverse proxy
  11. Start all services
  12. Verify installation
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
      --http-port)
        HTTP_PORT="$2"
        shift 2
        ;;
      --web-port)
        WEB_PORT="$2"
        shift 2
        ;;
      --db-pass)
        DB_PASS="$2"
        shift 2
        ;;
      --jwt-secret)
        JWT_SECRET="$2"
        shift 2
        ;;
      --skip-deps)
        SKIP_DEPS=1
        shift
        ;;
      --skip-migrate)
        SKIP_MIGRATE=1
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

prompt_config() {
  printf '\n'
  printf '============================================\n'
  printf '  PEKAN Configuration Setup\n'
  printf '============================================\n'
  printf '\n'

  # Branch
  printf 'Git branch [%s]: ' "$BRANCH"
  read -r input
  if [[ -n "$input" ]]; then
    BRANCH="$input"
  fi

  # Install directory
  printf 'Install directory [%s]: ' "$INSTALL_DIR"
  read -r input
  if [[ -n "$input" ]]; then
    INSTALL_DIR="$input"
  fi

  # API port
  printf 'API port [%s]: ' "$HTTP_PORT"
  read -r input
  if [[ -n "$input" ]]; then
    HTTP_PORT="$input"
  fi

  # Web port
  printf 'Web port [%s]: ' "$WEB_PORT"
  read -r input
  if [[ -n "$input" ]]; then
    WEB_PORT="$input"
  fi

  # Database name
  printf 'Database name [%s]: ' "$DB_NAME"
  read -r input
  if [[ -n "$input" ]]; then
    DB_NAME="$input"
  fi

  # Database user
  printf 'Database user [%s]: ' "$DB_USER"
  read -r input
  if [[ -n "$input" ]]; then
    DB_USER="$input"
  fi

  # Database host
  printf 'Database host [%s]: ' "$DB_HOST"
  read -r input
  if [[ -n "$input" ]]; then
    DB_HOST="$input"
  fi

  # Database port
  printf 'Database port [%s]: ' "$DB_PORT"
  read -r input
  if [[ -n "$input" ]]; then
    DB_PORT="$input"
  fi

  # Database password
  printf 'Database password (leave empty to auto-generate): '
  read -rs input
  printf '\n'
  if [[ -n "$input" ]]; then
    DB_PASS="$input"
  fi

  # JWT secret
  printf 'JWT secret (leave empty to auto-generate): '
  read -rs input
  printf '\n'
  if [[ -n "$input" ]]; then
    JWT_SECRET="$input"
  fi

  # Check for existing Docker containers and warn
  if docker ps --format '{{.Names}}' 2>/dev/null | grep -q "pekan-postgres"; then
    printf '\n'
    printf '[WARN] Docker container "pekan-postgres" detected!\n'
    printf '       Systemd installer will install NATIVE PostgreSQL service.\n'
    printf '       Existing Docker containers will NOT be used.\n'
    printf '\n'
    printf 'Do you want to continue with native systemd installation? [Y/n]: '
    read -r confirm
    if [[ "$confirm" =~ ^[Nn]$ ]]; then
      die "Installation cancelled. Use installer-docker.sh for Docker mode."
    fi
  fi

  # Confirm
  printf '\n'
  printf '============================================\n'
  printf '  Configuration Summary\n'
  printf '============================================\n'
  printf '  Branch:       %s\n' "$BRANCH"
  printf '  Install Dir:  %s\n' "$INSTALL_DIR"
  printf '  API Port:     %s\n' "$HTTP_PORT"
  printf '  Web Port:     %s\n' "$WEB_PORT"
  printf '  Database:     %s\n' "$DB_NAME"
  printf '  DB User:      %s\n' "$DB_USER"
  printf '  DB Host:      %s:%s\n' "$DB_HOST" "$DB_PORT"
  printf '  DB Password:  %s\n' "$(if [[ -n "$DB_PASS" ]]; then echo '***'; else echo '(auto-generate)'; fi)"
  printf '  JWT Secret:   %s\n' "$(if [[ -n "$JWT_SECRET" ]]; then echo '***'; else echo '(auto-generate)'; fi)"
  printf '  Mode:         Systemd (Native)\n'
  printf '============================================\n'
  printf '\n'

  printf 'Continue with installation? [Y/n]: '
  read -r confirm
  if [[ "$confirm" =~ ^[Nn]$ ]]; then
    die "Installation cancelled."
  fi
}

as_root() {
  if [[ "${EUID}" -eq 0 ]]; then "$@"; else sudo "$@"; fi
}

as_user() {
  local user="$1"
  shift
  if [[ "${EUID}" -eq 0 ]]; then
    runuser -u "$user" -- bash -lc "$*"
  else
    sudo -u "$user" -H bash -lc "$*"
  fi
}

check_root() {
  if [[ "${EUID}" -ne 0 ]]; then
    die "This script must be run as root. Use: sudo bash $0"
  fi
}

detect_os() {
  if [[ -f /etc/os-release ]]; then
    . /etc/os-release
    OS_ID="${ID:-}"
    OS_LIKE="${ID_LIKE:-}"
  else
    die "Cannot detect OS. This script supports Ubuntu/Debian and Rocky/AlmaLinux."
  fi

  if [[ "$OS_ID" =~ ^(rocky|almalinux|rhel|centos|ol)$ ]] || [[ "$OS_LIKE" =~ (rhel|fedora|centos) ]]; then
    PKG_MANAGER="dnf"
    PKG_INSTALL="dnf install -y"
  elif [[ "$OS_ID" =~ ^(ubuntu|debian)$ ]] || [[ "$OS_LIKE" =~ (ubuntu|debian) ]]; then
    PKG_MANAGER="apt"
    PKG_INSTALL="apt install -y"
  else
    die "Unsupported OS: $OS_ID. Use Ubuntu/Debian or Rocky/AlmaLinux."
  fi
  log "Detected OS: $OS_ID"
}

generate_secrets() {
  if [[ -z "$DB_PASS" ]]; then
    DB_PASS="$(openssl rand -base64 24 | tr -d '/+=' | head -c 24)"
    log "PostgreSQL password auto-generated"
  fi

  if [[ -z "$JWT_SECRET" ]]; then
    JWT_SECRET="$(openssl rand -base64 48)"
    log "JWT secret auto-generated"
  fi

  RECEIPT_SCAN_SECRET="$(openssl rand -base64 32)"
  ADMIN_SECRET="$(openssl rand -base64 32)"
  log "All secrets generated"
}

install_dependencies() {
  if [[ "$SKIP_DEPS" -eq 1 ]]; then
    log "Skipping dependency installation (--skip-deps)"
    return
  fi

  log "Step 1/12: Installing dependencies..."

  if [[ "$PKG_MANAGER" == "apt" ]]; then
    as_root apt update
    as_root $PKG_INSTALL -y curl wget git build-essential ca-certificates gnupg lsb-release rsync

    # PostgreSQL 16
    if ! command -v psql &>/dev/null; then
      log "  Installing PostgreSQL 16..."
      curl -fsSL https://www.postgresql.org/media/keys/ACCC4CF8.asc | as_root gpg --dearmor -o /usr/share/keyrings/postgresql-keyring.gpg
      echo "deb [signed-by=/usr/share/keyrings/postgresql-keyring.gpg] http://apt.postgresql.org/pub/repos/apt $(lsb_release -cs)-pgdg main" | as_root tee /etc/apt/sources.list.d/pgdg.list
      as_root apt update
      as_root $PKG_INSTALL -y postgresql-16 postgresql-client-16
    else
      log "  PostgreSQL already installed"
    fi

    # Redis
    if ! command -v redis-server &>/dev/null; then
      log "  Installing Redis..."
      curl -fsSL https://packages.redis.io/gpg | as_root gpg --dearmor -o /usr/share/keyrings/redis-archive-keyring.gpg
      echo "deb [signed-by=/usr/share/keyrings/redis-archive-keyring.gpg] https://packages.redis.io/deb $(lsb_release -cs) main" | as_root tee /etc/apt/sources.list.d/redis.list
      as_root apt update
      as_root $PKG_INSTALL -y redis
    else
      log "  Redis already installed"
    fi

    # Nginx
    if ! command -v nginx &>/dev/null; then
      log "  Installing Nginx..."
      as_root $PKG_INSTALL -y nginx
    else
      log "  Nginx already installed"
    fi

  elif [[ "$PKG_MANAGER" == "dnf" ]]; then
    as_root dnf install -y curl wget git gcc make ca-certificates rsync

    # PostgreSQL 16
    if ! command -v psql &>/dev/null; then
      log "  Installing PostgreSQL 16..."
      as_root dnf install -y https://download.postgresql.org/pub/repos/yum/reporpms/EL-9-x86_64/pgdg-redhat-repo-latest.noarch.rpm
      as_root dnf -y install postgresql16-server postgresql16
      as_root /usr/pgsql-16/bin/postgresql-16-setup initdb
      as_root systemctl enable postgresql-16
      as_root systemctl start postgresql-16
    else
      log "  PostgreSQL already installed"
    fi

    # Redis
    if ! command -v redis-server &>/dev/null; then
      log "  Installing Redis..."
      as_root dnf install -y epel-release
      as_root dnf install -y redis
      as_root systemctl enable redis
      as_root systemctl start redis
    else
      log "  Redis already installed"
    fi

    # Nginx
    if ! command -v nginx &>/dev/null; then
      log "  Installing Nginx..."
      as_root dnf install -y nginx
    else
      log "  Nginx already installed"
    fi
  fi

  log "  Dependencies installed"
}

install_go() {
  if [[ "$SKIP_DEPS" -eq 1 ]]; then
    return
  fi

  log "Step 2/12: Installing Go..."

  if command -v go &>/dev/null; then
    CURRENT_GO=$(go version | awk '{print $3}' | sed 's/go//')
    log "  Go already installed: $CURRENT_GO"
    return
  fi

  log "  Downloading Go $GO_VERSION..."
  wget -q "https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz" -O /tmp/go.tar.gz
  as_root rm -rf /usr/local/go
  as_root tar -C /usr/local -xzf /tmp/go.tar.gz
  rm -f /tmp/go.tar.gz

  echo 'export PATH=$PATH:/usr/local/go/bin' | as_root tee /etc/profile.d/go.sh >/dev/null
  export PATH=$PATH:/usr/local/go/bin
  log "  Go $GO_VERSION installed"
}

setup_database() {
  log "Step 3/12: Installing and configuring PostgreSQL..."

  # Check if PostgreSQL server is installed (not just client)
  PG_SERVER_INSTALLED=false
  if command -v pg_isready &>/dev/null || command -v postgres &>/dev/null; then
    PG_SERVER_INSTALLED=true
  fi

  # Install PostgreSQL server if not installed
  if [[ "$PG_SERVER_INSTALLED" == "false" ]]; then
    log "  Installing PostgreSQL server..."
    if [[ "$PKG_MANAGER" == "apt" ]]; then
      curl -fsSL https://www.postgresql.org/media/keys/ACCC4CF8.asc | as_root gpg --dearmor -o /usr/share/keyrings/postgresql-keyring.gpg
      echo "deb [signed-by=/usr/share/keyrings/postgresql-keyring.gpg] http://apt.postgresql.org/pub/repos/apt $(lsb_release -cs)-pgdg main" | as_root tee /etc/apt/sources.list.d/pgdg.list
      as_root apt update
      as_root $PKG_INSTALL -y postgresql-16 postgresql-client-16
    elif [[ "$PKG_MANAGER" == "dnf" ]]; then
      as_root dnf install -y https://download.postgresql.org/pub/repos/yum/reporpms/EL-9-x86_64/pgdg-redhat-repo-latest.noarch.rpm
      as_root dnf -y install postgresql16-server postgresql16
      as_root /usr/pgsql-16/bin/postgresql-16-setup initdb
    fi
  else
    log "  PostgreSQL server already installed"
  fi

  # Find PostgreSQL service name
  PG_SERVICE=""
  log "  Searching for PostgreSQL service..."
  for svc in postgresql postgresql-16 postgresql@16-main; do
    if systemctl list-unit-files "${svc}.service" 2>/dev/null | grep -q "${svc}"; then
      PG_SERVICE="$svc"
      log "  Found service: $svc"
      break
    fi
  done

  if [[ -z "$PG_SERVICE" ]]; then
    # Try to find any postgresql service
    PG_SERVICE=$(systemctl list-units --type=service --all 2>/dev/null | grep -o 'postgresql[^ ]*\.service' | head -1 | sed 's/\.service//')
    if [[ -n "$PG_SERVICE" ]]; then
      log "  Found service via search: $PG_SERVICE"
    fi
  fi

  if [[ -z "$PG_SERVICE" ]]; then
    warn "  PostgreSQL service not found. Available services:"
    systemctl list-units --type=service 2>/dev/null | grep -i postgres || echo "  (none)"
    die "PostgreSQL service not found. Please install PostgreSQL manually: sudo apt install postgresql-16"
  fi

  log "  Using PostgreSQL service: $PG_SERVICE"

  # Start and enable PostgreSQL
  as_root systemctl enable "$PG_SERVICE"
  as_root systemctl start "$PG_SERVICE"

  # Wait for PostgreSQL to be ready
  log "  Waiting for PostgreSQL to be ready..."
  sleep 3

  # Set password for postgres user
  as_root -u postgres psql -c "ALTER USER postgres WITH PASSWORD '${DB_PASS}';" 2>/dev/null || true

  # Create database if not exists
  as_root -u postgres psql -tc "SELECT 1 FROM pg_database WHERE datname = '${DB_NAME}'" | grep -q 1 || \
    as_root -u postgres psql -c "CREATE DATABASE ${DB_NAME};"

  # Create application user if not exists
  as_root -u postgres psql -tc "SELECT 1 FROM pg_roles WHERE rolname='${DB_USER}'" | grep -q 1 || \
    as_root -u postgres psql -c "CREATE USER ${DB_USER} WITH PASSWORD '${DB_PASS}';"

  # Grant privileges
  as_root -u postgres psql -c "GRANT ALL PRIVILEGES ON DATABASE ${DB_NAME} TO ${DB_USER};" 2>/dev/null || true

  # Configure pg_hba for local password auth
  PG_HBA=$(as_root -u postgres psql -t -c "SHOW hba_file;" | tr -d ' ')
  if ! grep -q "host.*${DB_NAME}.*127.0.0.1/32.*md5" "$PG_HBA" 2>/dev/null; then
    echo "host ${DB_NAME} ${DB_USER} 127.0.0.1/32 md5" | as_root tee -a "$PG_HBA" >/dev/null
  fi
  
  # Also add trust for local connections (for initial setup)
  if ! grep -q "local.*all.*all.*trust" "$PG_HBA" 2>/dev/null; then
    echo "local all all trust" | as_root tee -a "$PG_HBA" >/dev/null
  fi

  # Disable SSL for local connections
  PG_CONF=$(as_root -u postgres psql -t -c "SHOW config_file;" | tr -d ' ')
  if ! grep -q "ssl = off" "$PG_CONF" 2>/dev/null; then
    echo "ssl = off" | as_root tee -a "$PG_CONF" >/dev/null
  fi

  as_root systemctl restart "$PG_SERVICE"
  sleep 2

  log "  PostgreSQL installed and configured"
}

setup_redis() {
  log "Step 4/12: Installing and configuring Redis..."

  # Install Redis if not installed
  if ! command -v redis-server &>/dev/null; then
    log "  Installing Redis..."
    if [[ "$PKG_MANAGER" == "apt" ]]; then
      curl -fsSL https://packages.redis.io/gpg | as_root gpg --dearmor -o /usr/share/keyrings/redis-archive-keyring.gpg
      echo "deb [signed-by=/usr/share/keyrings/redis-archive-keyring.gpg] https://packages.redis.io/deb $(lsb_release -cs) main" | as_root tee /etc/apt/sources.list.d/redis.list
      as_root apt update
      as_root $PKG_INSTALL -y redis
    elif [[ "$PKG_MANAGER" == "dnf" ]]; then
      as_root dnf install -y epel-release
      as_root dnf install -y redis
    fi
  else
    log "  Redis already installed"
  fi

  # Find Redis service name
  REDIS_SERVICE=""
  for svc in redis redis-server; do
    if systemctl list-unit-files "${svc}.service" 2>/dev/null | grep -q "${svc}"; then
      REDIS_SERVICE="$svc"
      break
    fi
  done

  if [[ -z "$REDIS_SERVICE" ]]; then
    warn "  Redis service not found, skipping"
    return
  fi

  log "  Using Redis service: $REDIS_SERVICE"

  # Start and enable Redis
  as_root systemctl enable "$REDIS_SERVICE"
  as_root systemctl start "$REDIS_SERVICE"

  log "  Redis installed and configured"
}

setup_app_user() {
  log "Step 5/12: Creating application user..."

  if ! id "$APP_USER" &>/dev/null; then
    as_root useradd --system --shell /usr/sbin/nologin --home-dir "$INSTALL_DIR" "$APP_USER"
    log "  User '$APP_USER' created"
  else
    log "  User '$APP_USER' already exists"
  fi
}

clone_repository() {
  log "Step 6/12: Cloning repository (branch: $BRANCH)..."

  REPO_URL="https://github.com/honet-labs/pekan.git"

  if [[ -d "$INSTALL_DIR/.git" ]]; then
    log "  Repository exists, updating..."
    cd "$INSTALL_DIR"
    as_root git fetch origin
    as_root git checkout "$BRANCH"
    as_root git pull origin "$BRANCH"
  else
    log "  Cloning from $REPO_URL (branch: $BRANCH)..."
    as_root git clone --branch "$BRANCH" --single-branch "$REPO_URL" "$INSTALL_DIR"
  fi

  as_root chown -R "${APP_USER}:${APP_GROUP}" "$INSTALL_DIR"
  log "  Repository ready at $INSTALL_DIR"
}

build_backend() {
  log "Step 7/12: Building backend binaries..."

  export PATH=$PATH:/usr/local/go/bin

  as_root mkdir -p "$INSTALL_DIR/bin"
  as_root chown -R "${APP_USER}:${APP_GROUP}" "$INSTALL_DIR/bin"

  log "  Running go mod tidy..."
  as_user "$APP_USER" "cd '$INSTALL_DIR/backend' && /usr/local/go/bin/go mod tidy"

  log "  Building pekan-api..."
  as_user "$APP_USER" "cd '$INSTALL_DIR/backend' && CGO_ENABLED=0 /usr/local/go/bin/go build -ldflags='-s -w' -o '$INSTALL_DIR/bin/pekan-api' ./cmd/api"

  log "  Building pekan-worker..."
  as_user "$APP_USER" "cd '$INSTALL_DIR/backend' && CGO_ENABLED=0 /usr/local/go/bin/go build -ldflags='-s -w' -o '$INSTALL_DIR/bin/pekan-worker' ./cmd/worker"

  log "  Building pekan-ai..."
  as_user "$APP_USER" "cd '$INSTALL_DIR/backend' && CGO_ENABLED=0 /usr/local/go/bin/go build -ldflags='-s -w' -o '$INSTALL_DIR/bin/pekan-ai' ./cmd/ai"

  log "  Backend binaries built"
}

build_frontend() {
  log "Step 8/12: Building frontend..."

  if ! command -v node &>/dev/null; then
    log "  Installing Node.js..."
    if [[ "$PKG_MANAGER" == "apt" ]]; then
      curl -fsSL https://deb.nodesource.com/setup_20.x | as_root bash -
      as_root apt install -y nodejs
    else
      as_root dnf install -y nodejs npm
    fi
  fi

  log "  Running npm ci..."
  as_root bash -c "cd '$INSTALL_DIR/frontend' && npm ci"

  log "  Running npm run build..."
  as_root bash -c "cd '$INSTALL_DIR/frontend' && npm run build"

  log "  Frontend built"
}

write_env_file() {
  log "Step 9/12: Writing configuration..."

  # Detect database host (Docker container or localhost)
  DB_HOST_CONFIG="$DB_HOST"
  if docker ps --format '{{.Names}}' 2>/dev/null | grep -q "postgres"; then
    PG_CONTAINER=$(docker ps --format '{{.Names}}' | grep postgres | head -1)
    DB_HOST_CONFIG="$PG_CONTAINER"
    log "  Using Docker PostgreSQL container: $PG_CONTAINER"
  fi

  DATABASE_URL="postgres://${DB_USER}:${DB_PASS}@${DB_HOST_CONFIG}:${DB_PORT}/${DB_NAME}?sslmode=disable"

  # Detect Redis URL
  REDIS_URL="redis://127.0.0.1:6379/0"
  if docker ps --format '{{.Names}}' 2>/dev/null | grep -q "redis"; then
    REDIS_CONTAINER=$(docker ps --format '{{.Names}}' | grep redis | head -1)
    REDIS_URL="redis://${REDIS_CONTAINER}:6379/0"
    log "  Using Docker Redis container: $REDIS_CONTAINER"
  fi

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
ADMIN_SECRET=${ADMIN_SECRET}
JWT_ISSUER=pekan-api
JWT_ACCESS_TTL_MINUTES=15
JWT_REFRESH_TTL_HOURS=720
CORS_ALLOWED_ORIGINS=http://localhost:${WEB_PORT}
REQUEST_BODY_MAX_BYTES=1048576
API_RATE_LIMIT_PER_MINUTE=1000
API_RATE_LIMIT_WINDOW_SECONDS=60
API_REQUEST_TIMEOUT_SECONDS=30
MAX_HEADER_BYTES=1048576
RATE_LIMIT_REDIS_URL=${REDIS_URL}
RATE_LIMIT_REDIS_PREFIX=pekan:ratelimit
STORAGE_PROVIDER=local
STORAGE_LOCAL_PATH=${INSTALL_DIR}/storage
EOF

  as_root chown -R "${APP_USER}:${APP_GROUP}" "$INSTALL_DIR"
  log "  Configuration written to $INSTALL_DIR/backend/.env"
}

run_migrations() {
  if [[ "$SKIP_MIGRATE" -eq 1 ]]; then
    log "Skipping database migration (--skip-migrate)"
    return
  fi

  log "Step 10/12: Running database migrations..."

  DATABASE_URL="postgres://${DB_USER}:${DB_PASS}@${DB_HOST}:${DB_PORT}/${DB_NAME}?sslmode=disable"

  if [[ -f "$INSTALL_DIR/backend/scripts/apply_migrations.sh" ]]; then
    as_user "$APP_USER" "cd '$INSTALL_DIR/backend' && chmod +x ./scripts/apply_migrations.sh && DATABASE_URL='${DATABASE_URL}' ./scripts/apply_migrations.sh"
    log "  Migrations applied"
  else
    warn "  Migration script not found, skipping"
  fi
}

setup_systemd() {
  log "Step 11/12: Configuring systemd services..."

  # Detect if PostgreSQL/Redis are in Docker
  PG_DEPS=""
  REDIS_DEPS=""
  if ! docker ps --format '{{.Names}}' 2>/dev/null | grep -q "postgres"; then
    PG_DEPS="postgresql.service"
  fi
  if ! docker ps --format '{{.Names}}' 2>/dev/null | grep -q "redis"; then
    REDIS_DEPS="redis.service"
  fi

  # pekan-api.service
  cat <<EOF | as_root tee /etc/systemd/system/pekan-api.service >/dev/null
[Unit]
Description=PEKAN API Server
After=network.target ${PG_DEPS} ${REDIS_DEPS}
Wants=${PG_DEPS} ${REDIS_DEPS}

[Service]
Type=simple
User=${APP_USER}
Group=${APP_GROUP}
WorkingDirectory=${INSTALL_DIR}
ExecStartPre=/bin/sleep 5
ExecStart=${INSTALL_DIR}/bin/pekan-api
Restart=always
RestartSec=10
StartLimitIntervalSec=60
StartLimitBurst=5
StandardOutput=journal
StandardError=journal
SyslogIdentifier=pekan-api
Environment=PATH=/usr/local/go/bin:/usr/bin:/bin

[Install]
WantedBy=multi-user.target
EOF

  # pekan-worker.service
  cat <<EOF | as_root tee /etc/systemd/system/pekan-worker.service >/dev/null
[Unit]
Description=PEKAN Background Worker
After=network.target ${PG_DEPS} ${REDIS_DEPS}
Wants=${PG_DEPS} ${REDIS_DEPS}

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

  # pekan-ai.service
  cat <<EOF | as_root tee /etc/systemd/system/pekan-ai.service >/dev/null
[Unit]
Description=PEKAN AI Queue Worker
After=network.target ${PG_DEPS} ${REDIS_DEPS}
Wants=${PG_DEPS} ${REDIS_DEPS}

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

  log "  Systemd services configured"
}

setup_nginx() {
  log "Step 12/12: Configuring Nginx..."

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

  log "  Nginx configured"
}

start_services() {
  log "Starting services..."

  as_root systemctl start pekan-api
  as_root systemctl start pekan-worker
  as_root systemctl start pekan-ai

  log "  All services started"
}

verify_installation() {
  log "Verifying installation..."
  sleep 3

  local STATUS
  STATUS=$(curl -sf "http://127.0.0.1:${HTTP_PORT}/api/v1/healthz" 2>/dev/null || echo "FAILED")

  if [[ "$STATUS" == *"FAILED"* ]]; then
    warn "Health check failed. Check logs: journalctl -u pekan-api -f"
  else
    log "Health check passed!"
  fi

  printf '\n'
  printf '============================================\n'
  printf '  PEKAN Installation Complete (Systemd)\n'
  printf '============================================\n'
  printf '\n'
  printf '  Branch:      %s\n' "${BRANCH}"
  printf '  Install Dir: %s\n' "${INSTALL_DIR}"
  printf '  Web URL:     http://localhost:%s\n' "${WEB_PORT}"
  printf '  API URL:     http://localhost:%s/api/v1\n' "${HTTP_PORT}"
  printf '  Health:      http://localhost:%s/api/v1/healthz\n' "${HTTP_PORT}"
  printf '\n'
  printf '  Config:      %s/backend/.env\n' "${INSTALL_DIR}"
  printf '  Logs:        journalctl -u pekan-api -f\n'
  printf '  Storage:     %s/storage\n' "${INSTALL_DIR}"
  printf '\n'
  printf '  Services:\n'
  printf '    systemctl status pekan-api\n'
  printf '    systemctl status pekan-worker\n'
  printf '    systemctl status pekan-ai\n'
  printf '\n'
  printf '  Update:\n'
  printf '    cd %s && git pull && sudo bash deploy/update-versi.sh\n' "${INSTALL_DIR}"
  printf '\n'
}

main() {
  parse_args "$@"

  prompt_config

  printf '============================================\n'
  printf '  PEKAN Systemd Installer\n'
  printf '  Branch: %s\n' "${BRANCH}"
  printf '============================================\n'
  printf '\n'

  check_root
  detect_os
  generate_secrets
  install_dependencies
  install_go
  setup_database
  setup_redis
  setup_app_user
  clone_repository
  build_backend
  build_frontend
  write_env_file
  run_migrations
  setup_systemd
  setup_nginx
  start_services
  verify_installation
}

main "$@"
