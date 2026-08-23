#!/usr/bin/env bash
# =============================================================================
# PEKAN Docker Installer
# Install PEKAN with Docker containers (all-in-one)
#
# Usage:
#   sudo bash installer-docker.sh [options]
#
# Options:
#   --branch <name>           Git branch to install (default: main)
#   --install-dir <path>      Installation directory (default: /opt/pekan)
#   --web-port <port>         Web/Nginx port (default: 80)
#   --postgres-pass <pass>    PostgreSQL password (auto-generated if empty)
#   --skip-build              Skip Docker image build (use existing images)
#   --help                    Show this help
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
BRANCH="${BRANCH:-dev}"
INSTALL_DIR="${INSTALL_DIR:-/opt/pekan}"
APP_ENV="${APP_ENV:-production}"
WEB_PORT="${WEB_PORT:-80}"
POSTGRES_PASSWORD="${POSTGRES_PASSWORD:-}"
DB_NAME="${DB_NAME:-pekan}"
DB_USER="${DB_USER:-postgres}"
SKIP_BUILD=0

# Resource limits (empty = no limit)
LIMIT_API_CPU="${LIMIT_API_CPU:-}"
LIMIT_API_MEM="${LIMIT_API_MEM:-}"
LIMIT_WORKER_CPU="${LIMIT_WORKER_CPU:-}"
LIMIT_WORKER_MEM="${LIMIT_WORKER_MEM:-}"
LIMIT_AI_CPU="${LIMIT_AI_CPU:-}"
LIMIT_AI_MEM="${LIMIT_AI_MEM:-}"
LIMIT_DB_CPU="${LIMIT_DB_CPU:-}"
LIMIT_DB_MEM="${LIMIT_DB_MEM:-}"
LIMIT_REDIS_CPU="${LIMIT_REDIS_CPU:-}"
LIMIT_REDIS_MEM="${LIMIT_REDIS_MEM:-}"
LIMIT_WEB_CPU="${LIMIT_WEB_CPU:-}"
LIMIT_WEB_MEM="${LIMIT_WEB_MEM:-}"

show_help() {
  cat <<'EOF'
PEKAN Docker Installer

Install PEKAN with Docker containers (all-in-one).

Usage:
  sudo bash installer-docker.sh [options]

Options:
  --branch <name>           Git branch to install (default: dev)
                            Available: main, dev
                            NOTE: Docker files only exist on 'dev' branch
  --install-dir <path>      Installation directory (default: /opt/pekan)
  --web-port <port>         Web/Nginx port (default: 80)
  --db-name <name>          Database name (default: pekan)
  --db-user <user>          Database user (default: postgres)
  --postgres-pass <pass>    PostgreSQL password (auto-generated if empty)
  --skip-build              Skip Docker image build (use existing images)
  --help                    Show this help message

Examples:
  # Install with defaults
  sudo bash installer-docker.sh

  # Install with custom database credentials
  sudo bash installer-docker.sh --db-name mydb --db-user myuser --postgres-pass mypass

  # Install with custom port
  sudo bash installer-docker.sh --web-port 8080

Steps performed:
  1. Detect OS
  2. Install Docker and Docker Compose
  3. Clone repository from specified branch
  4. Generate secrets (JWT, admin, etc.)
  5. Write configuration files
  6. Build Docker images
  7. Start all containers
  8. Run database migrations
  9. Configure automatic daily backups
  10. Verify installation

Containers created:
  - pekan-postgres  (PostgreSQL 16)
  - pekan-redis     (Redis 7)
  - pekan-api       (API Server)
  - pekan-worker    (Background Worker)
  - pekan-ai        (AI Queue Worker)
  - pekan-web       (Frontend Nginx)
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
      --web-port)
        WEB_PORT="$2"
        shift 2
        ;;
      --db-name)
        DB_NAME="$2"
        shift 2
        ;;
      --db-user)
        DB_USER="$2"
        shift 2
        ;;
      --postgres-pass)
        POSTGRES_PASSWORD="$2"
        shift 2
        ;;
      --skip-build)
        SKIP_BUILD=1
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

  # PostgreSQL password
  printf 'Database password (leave empty to auto-generate): '
  read -rs input
  printf '\n'
  if [[ -n "$input" ]]; then
    POSTGRES_PASSWORD="$input"
  fi

  # Resource limits
  printf '\n'
  printf '--- Resource Limits (leave empty for no limit) ---\n'
  printf 'Format: CPU (e.g., 0.5, 1, 2) | Memory (e.g., 256m, 1g)\n'
  printf '\n'

  printf 'PostgreSQL - CPU limit [%s]: ' "$(if [[ -n "$LIMIT_DB_CPU" ]]; then echo "$LIMIT_DB_CPU"; else echo "none"; fi)"
  read -r input
  if [[ -n "$input" ]]; then LIMIT_DB_CPU="$input"; fi

  printf 'PostgreSQL - Memory limit [%s]: ' "$(if [[ -n "$LIMIT_DB_MEM" ]]; then echo "$LIMIT_DB_MEM"; else echo "none"; fi)"
  read -r input
  if [[ -n "$input" ]]; then LIMIT_DB_MEM="$input"; fi

  printf 'Redis - CPU limit [%s]: ' "$(if [[ -n "$LIMIT_REDIS_CPU" ]]; then echo "$LIMIT_REDIS_CPU"; else echo "none"; fi)"
  read -r input
  if [[ -n "$input" ]]; then LIMIT_REDIS_CPU="$input"; fi

  printf 'Redis - Memory limit [%s]: ' "$(if [[ -n "$LIMIT_REDIS_MEM" ]]; then echo "$LIMIT_REDIS_MEM"; else echo "none"; fi)"
  read -r input
  if [[ -n "$input" ]]; then LIMIT_REDIS_MEM="$input"; fi

  printf 'API Server - CPU limit [%s]: ' "$(if [[ -n "$LIMIT_API_CPU" ]]; then echo "$LIMIT_API_CPU"; else echo "none"; fi)"
  read -r input
  if [[ -n "$input" ]]; then LIMIT_API_CPU="$input"; fi

  printf 'API Server - Memory limit [%s]: ' "$(if [[ -n "$LIMIT_API_MEM" ]]; then echo "$LIMIT_API_MEM"; else echo "none"; fi)"
  read -r input
  if [[ -n "$input" ]]; then LIMIT_API_MEM="$input"; fi

  printf 'Worker - CPU limit [%s]: ' "$(if [[ -n "$LIMIT_WORKER_CPU" ]]; then echo "$LIMIT_WORKER_CPU"; else echo "none"; fi)"
  read -r input
  if [[ -n "$input" ]]; then LIMIT_WORKER_CPU="$input"; fi

  printf 'Worker - Memory limit [%s]: ' "$(if [[ -n "$LIMIT_WORKER_MEM" ]]; then echo "$LIMIT_WORKER_MEM"; else echo "none"; fi)"
  read -r input
  if [[ -n "$input" ]]; then LIMIT_WORKER_MEM="$input"; fi

  printf 'AI Worker - CPU limit [%s]: ' "$(if [[ -n "$LIMIT_AI_CPU" ]]; then echo "$LIMIT_AI_CPU"; else echo "none"; fi)"
  read -r input
  if [[ -n "$input" ]]; then LIMIT_AI_CPU="$input"; fi

  printf 'AI Worker - Memory limit [%s]: ' "$(if [[ -n "$LIMIT_AI_MEM" ]]; then echo "$LIMIT_AI_MEM"; else echo "none"; fi)"
  read -r input
  if [[ -n "$input" ]]; then LIMIT_AI_MEM="$input"; fi

  printf 'Frontend (Nginx) - CPU limit [%s]: ' "$(if [[ -n "$LIMIT_WEB_CPU" ]]; then echo "$LIMIT_WEB_CPU"; else echo "none"; fi)"
  read -r input
  if [[ -n "$input" ]]; then LIMIT_WEB_CPU="$input"; fi

  printf 'Frontend (Nginx) - Memory limit [%s]: ' "$(if [[ -n "$LIMIT_WEB_MEM" ]]; then echo "$LIMIT_WEB_MEM"; else echo "none"; fi)"
  read -r input
  if [[ -n "$input" ]]; then LIMIT_WEB_MEM="$input"; fi

  # Confirm
  printf '\n'
  printf '============================================\n'
  printf '  Configuration Summary\n'
  printf '============================================\n'
  printf '  Branch:       %s\n' "$BRANCH"
  printf '  Install Dir:  %s\n' "$INSTALL_DIR"
  printf '  Web Port:     %s\n' "$WEB_PORT"
  printf '  Database:     %s\n' "$DB_NAME"
  printf '  DB User:      %s\n' "$DB_USER"
  printf '  DB Password:  %s\n' "$(if [[ -n "$POSTGRES_PASSWORD" ]]; then echo '***'; else echo '(auto-generate)'; fi)"
  printf '\n'
  printf '  Resource Limits:\n'
  printf '    PostgreSQL:   CPU=%s MEM=%s\n' "$(if [[ -n "$LIMIT_DB_CPU" ]]; then echo "$LIMIT_DB_CPU"; else echo "unlimited"; fi)" "$(if [[ -n "$LIMIT_DB_MEM" ]]; then echo "$LIMIT_DB_MEM"; else echo "unlimited"; fi)"
  printf '    Redis:        CPU=%s MEM=%s\n' "$(if [[ -n "$LIMIT_REDIS_CPU" ]]; then echo "$LIMIT_REDIS_CPU"; else echo "unlimited"; fi)" "$(if [[ -n "$LIMIT_REDIS_MEM" ]]; then echo "$LIMIT_REDIS_MEM"; else echo "unlimited"; fi)"
  printf '    API Server:   CPU=%s MEM=%s\n' "$(if [[ -n "$LIMIT_API_CPU" ]]; then echo "$LIMIT_API_CPU"; else echo "unlimited"; fi)" "$(if [[ -n "$LIMIT_API_MEM" ]]; then echo "$LIMIT_API_MEM"; else echo "unlimited"; fi)"
  printf '    Worker:       CPU=%s MEM=%s\n' "$(if [[ -n "$LIMIT_WORKER_CPU" ]]; then echo "$LIMIT_WORKER_CPU"; else echo "unlimited"; fi)" "$(if [[ -n "$LIMIT_WORKER_MEM" ]]; then echo "$LIMIT_WORKER_MEM"; else echo "unlimited"; fi)"
  printf '    AI Worker:    CPU=%s MEM=%s\n' "$(if [[ -n "$LIMIT_AI_CPU" ]]; then echo "$LIMIT_AI_CPU"; else echo "unlimited"; fi)" "$(if [[ -n "$LIMIT_AI_MEM" ]]; then echo "$LIMIT_AI_MEM"; else echo "unlimited"; fi)"
  printf '    Frontend:     CPU=%s MEM=%s\n' "$(if [[ -n "$LIMIT_WEB_CPU" ]]; then echo "$LIMIT_WEB_CPU"; else echo "unlimited"; fi)" "$(if [[ -n "$LIMIT_WEB_MEM" ]]; then echo "$LIMIT_WEB_MEM"; else echo "unlimited"; fi)"
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

check_root() {
  if [[ "${EUID}" -ne 0 ]]; then
    die "This script must be run as root. Use: sudo bash $0"
  fi
}

detect_os() {
  if [[ -f /etc/os-release ]]; then
    . /etc/os-release
    OS_ID="${ID:-}"
  else
    die "Cannot detect OS."
  fi
  log "Detected OS: $OS_ID"
}

install_docker() {
  log "Step 1/9: Installing Docker..."

  if command -v docker &>/dev/null; then
    log "  Docker already installed: $(docker --version)"
    return
  fi

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
  log "  Docker installed"
}

clone_repository() {
  log "Step 2/9: Cloning repository (branch: $BRANCH)..."

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

  # Verify docker-compose.yml exists
  if [[ ! -f "$INSTALL_DIR/docker-compose.yml" ]]; then
    die "docker-compose.yml not found in branch '$BRANCH'. Use --branch dev for Docker installation."
  fi

  log "  Repository ready at $INSTALL_DIR"
}

generate_secrets() {
  log "Step 3/9: Generating secrets..."

  if [[ -z "$POSTGRES_PASSWORD" ]]; then
    POSTGRES_PASSWORD="$(openssl rand -base64 24 | tr -d '/+=' | head -c 24)"
    log "  PostgreSQL password generated"
  fi

  JWT_SECRET="$(openssl rand -base64 48)"
  RECEIPT_SCAN_SECRET="$(openssl rand -base64 32)"
  ADMIN_SECRET="$(openssl rand -base64 32)"
  log "  All secrets generated"
}

write_config() {
  log "Step 4/9: Writing configuration..."

  # Backend .env
  cat <<EOF | as_root tee "$INSTALL_DIR/backend/.env" >/dev/null
APP_ENV=${APP_ENV}
HTTP_PORT=8080
DATABASE_URL=postgres://${DB_USER}:${POSTGRES_PASSWORD}@pekan-postgres:5432/${DB_NAME}?sslmode=prefer
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
RATE_LIMIT_REDIS_URL=redis://pekan-redis:6379/0
RATE_LIMIT_REDIS_PREFIX=pekan:ratelimit
STORAGE_PROVIDER=local
STORAGE_LOCAL_PATH=/app/storage
EOF

  # Root .env for docker-compose
  cat <<EOF | as_root tee "$INSTALL_DIR/.env" >/dev/null
POSTGRES_PASSWORD=${POSTGRES_PASSWORD}
DB_NAME=${DB_NAME}
DB_USER=${DB_USER}
EOF

  log "  Configuration written"
}

write_docker_compose() {
  log "  Writing docker-compose.yml with resource limits..."

  # Helper function to generate deploy section
  generate_deploy() {
    local cpu="$1"
    local mem="$2"
    if [[ -n "$cpu" || -n "$mem" ]]; then
      echo "    deploy:"
      echo "      resources:"
      echo "        limits:"
      if [[ -n "$cpu" ]]; then
        echo "          cpus: \"${cpu}\""
      fi
      if [[ -n "$mem" ]]; then
        echo "          memory: ${mem}"
      fi
    fi
  }

  cat <<EOF | as_root tee "$INSTALL_DIR/docker-compose.yml" >/dev/null
services:
  pekan-postgres:
    image: postgres:16-alpine
    container_name: pekan-postgres
    restart: unless-stopped
    environment:
      POSTGRES_DB: \${DB_NAME:-pekan}
      POSTGRES_USER: \${DB_USER:-postgres}
      POSTGRES_PASSWORD: \${POSTGRES_PASSWORD:-postgres}
    command: postgres -c ssl=off -c max_connections=100
    ports:
      - "5432:5432"
    volumes:
      - pekan_postgres_data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U \${DB_USER:-postgres} -d \${DB_NAME:-pekan}"]
      interval: 10s
      timeout: 5s
      retries: 10
$(generate_deploy "$LIMIT_DB_CPU" "$LIMIT_DB_MEM")
    networks:
      - pekan-network

  pekan-redis:
    image: redis:7-alpine
    container_name: pekan-redis
    restart: unless-stopped
    command: ["redis-server", "--appendonly", "yes", "--maxmemory", "256mb", "--maxmemory-policy", "allkeys-lru"]
    ports:
      - "6379:6379"
    volumes:
      - pekan_redis_data:/data
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 10s
      timeout: 5s
      retries: 10
$(generate_deploy "$LIMIT_REDIS_CPU" "$LIMIT_REDIS_MEM")
    networks:
      - pekan-network

  pekan-api:
    build:
      context: ./backend
      dockerfile: Dockerfile
    container_name: pekan-api
    restart: unless-stopped
    entrypoint: ["/app/pekan-api"]
    env_file:
      - ./backend/.env
    environment:
      HTTP_PORT: 8080
      DATABASE_URL: postgres://\${DB_USER:-postgres}:\${POSTGRES_PASSWORD:-postgres}@pekan-postgres:5432/\${DB_NAME:-pekan}?sslmode=prefer
      RATE_LIMIT_REDIS_URL: redis://pekan-redis:6379/0
    ports:
      - "8080:8080"
    depends_on:
      pekan-postgres:
        condition: service_healthy
      pekan-redis:
        condition: service_healthy
    healthcheck:
      test: ["CMD", "wget", "-qO-", "http://localhost:8080/api/v1/healthz"]
      interval: 30s
      timeout: 5s
      start_period: 15s
      retries: 3
$(generate_deploy "$LIMIT_API_CPU" "$LIMIT_API_MEM")
    networks:
      - pekan-network

  pekan-worker:
    build:
      context: ./backend
      dockerfile: Dockerfile
    container_name: pekan-worker
    restart: unless-stopped
    entrypoint: ["/app/pekan-worker"]
    env_file:
      - ./backend/.env
    environment:
      DATABASE_URL: postgres://\${DB_USER:-postgres}:\${POSTGRES_PASSWORD:-postgres}@pekan-postgres:5432/\${DB_NAME:-pekan}?sslmode=prefer
    depends_on:
      pekan-postgres:
        condition: service_healthy
    healthcheck:
      test: ["CMD-SHELL", "pgrep -f pekan-worker || exit 1"]
      interval: 30s
      timeout: 5s
      start_period: 15s
      retries: 3
$(generate_deploy "$LIMIT_WORKER_CPU" "$LIMIT_WORKER_MEM")
    networks:
      - pekan-network

  pekan-ai:
    build:
      context: ./backend
      dockerfile: Dockerfile
    container_name: pekan-ai
    restart: unless-stopped
    entrypoint: ["/app/pekan-ai"]
    env_file:
      - ./backend/.env
    environment:
      DATABASE_URL: postgres://\${DB_USER:-postgres}:\${POSTGRES_PASSWORD:-postgres}@pekan-postgres:5432/\${DB_NAME:-pekan}?sslmode=prefer
    depends_on:
      pekan-postgres:
        condition: service_healthy
    healthcheck:
      test: ["CMD-SHELL", "pgrep -f pekan-ai || exit 1"]
      interval: 30s
      timeout: 5s
      start_period: 15s
      retries: 3
$(generate_deploy "$LIMIT_AI_CPU" "$LIMIT_AI_MEM")
    networks:
      - pekan-network

  pekan-web:
    build:
      context: ./frontend
      dockerfile: Dockerfile
    container_name: pekan-web
    restart: unless-stopped
    ports:
      - "\${WEB_PORT:-80}:80"
    depends_on:
      pekan-api:
        condition: service_healthy
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:80/healthz"]
      interval: 30s
      timeout: 5s
      start_period: 10s
      retries: 3
$(generate_deploy "$LIMIT_WEB_CPU" "$LIMIT_WEB_MEM")
    networks:
      - pekan-network

volumes:
  pekan_postgres_data:
    name: pekan_postgres_data
  pekan_redis_data:
    name: pekan_redis_data

networks:
  pekan-network:
    name: pekan-network
    driver: bridge
EOF

  log "  docker-compose.yml written"
}

build_images() {
  if [[ "$SKIP_BUILD" -eq 1 ]]; then
    log "Step 5/9: Skipping Docker build (--skip-build)"
    return
  fi

  log "Step 5/9: Building Docker images..."
  cd "$INSTALL_DIR"
  as_root docker compose build --no-cache
  log "  Docker images built"
}

start_containers() {
  log "Step 6/9: Starting containers..."

  cd "$INSTALL_DIR"

  # Start database and cache first
  log "  Starting PostgreSQL and Redis..."
  as_root docker compose up -d pekan-postgres pekan-redis

  log "  Waiting for database to be ready..."
  sleep 10

  # Start all services
  log "  Starting all services..."
  as_root docker compose up -d

  log "  All containers started"
}

run_migrations() {
  log "Step 7/11: Running database migrations..."

  cd "$INSTALL_DIR"

  # Wait for PostgreSQL to be fully ready
  log "  Waiting for PostgreSQL to be ready..."
  sleep 5

  # Check if migrations directory exists
  if [[ ! -d "$INSTALL_DIR/backend/migrations" ]]; then
    warn "  Migrations directory not found, skipping"
    return
  fi

  # Count migration files
  MIGRATION_COUNT=$(ls "$INSTALL_DIR/backend/migrations/"*.sql 2>/dev/null | wc -l)
  log "  Found $MIGRATION_COUNT migration files"

  # Run migrations one by one from host
  log "  Applying migrations..."
  
  SUCCESS=0
  FAILED=0
  
  for sql_file in $(ls "$INSTALL_DIR/backend/migrations/"*.sql 2>/dev/null | sort); do
    filename=$(basename "$sql_file")
    printf "    -> %s" "$filename"
    
    # Copy file to container and execute
    as_root docker compose cp "$sql_file" pekan-postgres:/tmp/current_migration.sql
    if as_root docker compose exec -T pekan-postgres psql -U "$DB_USER" -d "$DB_NAME" -f /tmp/current_migration.sql > /dev/null 2>&1; then
      printf " [OK]\n"
      SUCCESS=$((SUCCESS + 1))
    else
      printf " [SKIP]\n"
      FAILED=$((FAILED + 1))
    fi
  done

  # Cleanup
  as_root docker compose exec -T pekan-postgres rm -f /tmp/current_migration.sql

  log "  Migration results: $SUCCESS succeeded, $FAILED skipped"

  # Verify key tables exist
  log "  Verifying tables..."
  TABLES=$(as_root docker compose exec -T pekan-postgres psql -U "$DB_USER" -d "$DB_NAME" -t -c "SELECT count(*) FROM information_schema.tables WHERE table_schema = 'public';" 2>/dev/null | tr -d ' ')
  log "  Found $TABLES tables in database"

  # List key tables
  log "  Key tables:"
  as_root docker compose exec -T pekan-postgres psql -U "$DB_USER" -d "$DB_NAME" -c "SELECT table_name FROM information_schema.tables WHERE table_schema = 'public' AND table_name IN ('tenants', 'users', 'file_scan_jobs', 'finance_reminders') ORDER BY table_name;" 2>/dev/null

  log "  Migrations completed"
}

restart_services() {
  log "Step 8/10: Restarting services..."

  cd "$INSTALL_DIR"

  # Restart all services to pick up the new database schema
  as_root docker compose restart

  # Wait for services to be healthy
  log "  Waiting for services to be healthy..."
  sleep 10

  log "  Services restarted"
}

setup_backup() {
  log "Step 9/10: Configuring automatic backups..."

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

  log "  Automatic backup configured (daily at 02:00)"
}

save_version() {
  log "Step 10/10: Saving version..."

  cd "$INSTALL_DIR"
  local VERSION
  VERSION=$(git rev-parse --short HEAD)
  echo "$VERSION" | as_root tee "$INSTALL_DIR/.version" >/dev/null

  log "  Version: $VERSION"
}

verify_installation() {
  log "Step 11/11: Verifying installation..."
  sleep 5

  cd "$INSTALL_DIR"
  as_root docker compose ps

  local STATUS
  STATUS=$(curl -sf "http://127.0.0.1:8080/api/v1/healthz" 2>/dev/null || echo "FAILED")

  if [[ "$STATUS" == *"FAILED"* ]]; then
    warn "Health check not ready yet. Containers may still be starting."
    warn "Check logs: docker compose -f $INSTALL_DIR/docker-compose.yml logs -f pekan-api"
  else
    log "Health check passed!"
  fi

  printf '\n'
  printf '============================================\n'
  printf '  PEKAN Installation Complete (Docker)\n'
  printf '============================================\n'
  printf '\n'
  printf '  Branch:      %s\n' "${BRANCH}"
  printf '  Install Dir: %s\n' "${INSTALL_DIR}"
  printf '  Web URL:     http://localhost:%s\n' "${WEB_PORT}"
  printf '  API URL:     http://localhost:8080/api/v1\n'
  printf '  Health:      http://localhost:8080/api/v1/healthz\n'
  printf '\n'
  printf '  Config:      %s/backend/.env\n' "${INSTALL_DIR}"
  printf '  Compose:     %s/docker-compose.yml\n' "${INSTALL_DIR}"
  printf '  Logs:        docker compose -f %s/docker-compose.yml logs -f\n' "${INSTALL_DIR}"
  printf '\n'
  printf '  Containers:\n'
  printf '    pekan-postgres  (PostgreSQL 16)\n'
  printf '    pekan-redis     (Redis 7)\n'
  printf '    pekan-api       (API Server - port 8080)\n'
  printf '    pekan-worker    (Background Worker)\n'
  printf '    pekan-ai        (AI Queue Worker)\n'
  printf '    pekan-web       (Frontend Nginx - port %s)\n' "${WEB_PORT}"
  printf '\n'
  printf '  Commands:\n'
  printf '    docker compose ps                    # Container status\n'
  printf '    docker compose logs -f pekan-api     # View API logs\n'
  printf '    docker compose restart pekan-api     # Restart API\n'
  printf '    docker compose down                  # Stop all\n'
  printf '    docker compose up -d                 # Start all\n'
  printf '\n'
  printf '  Update:\n'
  printf '    cd %s && git pull && sudo bash deploy/update-versi.sh\n' "${INSTALL_DIR}"
  printf '\n'
}

main() {
  parse_args "$@"

  prompt_config

  printf '============================================\n'
  printf '  PEKAN Docker Installer\n'
  printf '  Branch: %s\n' "${BRANCH}"
  printf '============================================\n'
  printf '\n'

  check_root
  detect_os
  install_docker
  clone_repository
  generate_secrets
  write_config
  write_docker_compose
  build_images
  start_containers
  run_migrations
  restart_services
  setup_backup
  save_version
  verify_installation
}

main "$@"
