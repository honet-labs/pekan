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
GO_VERSION="${GO_VERSION:-1.25.8}"
APP_ENV="${APP_ENV:-development}"
HTTP_PORT="${HTTP_PORT:-8080}"
JWT_ISSUER="${JWT_ISSUER:-pekan-api}"
JWT_SECRET="${JWT_SECRET:-}"
RECEIPT_SCAN_SECRET="${RECEIPT_SCAN_SECRET:-}"
CORS_ALLOWED_ORIGINS="${CORS_ALLOWED_ORIGINS:-http://localhost:5173}"
DATABASE_URL="${DATABASE_URL:-}"
DB_USER="${DB_USER:-postgres}"
DB_PASS="${DB_PASS:-postgres}"
DB_NAME="${DB_NAME:-pekan}"
DB_HOST="${DB_HOST:-127.0.0.1}"
DB_PORT="${DB_PORT:-5432}"
DB_SCHEMA="${DB_SCHEMA:-public}"
DB_SSLMODE="${DB_SSLMODE:-disable}"
_DATABASE_URL_SPECIFIED=0
_DB_PART_SPECIFIED=0
RATE_LIMIT_REDIS_URL="${RATE_LIMIT_REDIS_URL:-redis://127.0.0.1:6379/0}"
RATE_LIMIT_REDIS_PREFIX="${RATE_LIMIT_REDIS_PREFIX:-pekan:ratelimit}"
STORAGE_PROVIDER="${STORAGE_PROVIDER:-local}"
STORAGE_LOCAL_PATH="${STORAGE_LOCAL_PATH:-/var/lib/pekan/storage}"
RUN_TESTS="${RUN_TESTS:-1}"
SEED_DEMO="${SEED_DEMO:-1}"
ENABLE_SERVICES="${ENABLE_SERVICES:-1}"
ENABLE_WEB="${ENABLE_WEB:-1}"
WEB_PORT="${WEB_PORT:-80}"
WEB_ROOT="${WEB_ROOT:-/var/www/pekan-web}"
FRONTEND_API_BASE_URL="${FRONTEND_API_BASE_URL:-/api/v1}"
NPM_REGISTRY="${NPM_REGISTRY:-https://registry.npmjs.org/}"
INFRA_MODE="${INFRA_MODE:-docker}"
INSTALL_DB=1
INSTALL_REDIS=1
INSTALL_APP=1
INSTALL_WEB=1
_COMP_USER_SPECIFIED=0

usage() {
  cat <<'EOF'
Usage: ./deploy/install_server_rocky.sh [options]

Options:
  --install-dir <path>          Install path (default: /opt/pekan)
  --app-user <user>             Service user (default: pekan)
  --app-group <group>           Service group (default: pekan)
  --go-version <version>        Go version (default: 1.25.8)
  --app-env <env>               APP_ENV (default: development)
  --http-port <port>            HTTP_PORT (default: 8080)
  --jwt-secret <secret>         JWT secret (auto-generate if empty)
  --cors <origins>              CORS_ALLOWED_ORIGINS
  --database-url <url>          DATABASE_URL (takes precedence over --db-* flags)
  --db-user <user>              Database user (default: postgres)
  --db-pass <pass>              Database password (default: postgres)
  --db-name <name>              Database name (default: pekan)
  --db-host <host>              Database host (default: 127.0.0.1)
  --db-port <port>              Database port (default: 5432)
  --db-schema <schema>          Database schema (default: public)
  --redis-url <url>             RATE_LIMIT_REDIS_URL
  --redis-prefix <prefix>       RATE_LIMIT_REDIS_PREFIX
  --storage-provider <provider> STORAGE_PROVIDER (default: local)
  --storage-local-path <path>   STORAGE_LOCAL_PATH
  --skip-tests                  Skip go test ./...
  --skip-seed                   Skip demo seed
  --no-enable-services          Do not enable/start systemd services
  --skip-web                    Skip frontend build + nginx web publish
  --web-port <port>             Nginx web port (default: 80)
  --web-root <path>             Frontend publish path (default: /var/www/pekan-web)
  --frontend-api-base-url <url> VITE_API_BASE_URL for frontend build (default: /api/v1)
  --npm-registry <url>         NPM registry for frontend install (default: https://registry.npmjs.org/)
  --infra-mode <mode>           Infrastructure mode: docker or standalone (default: docker)
  --all-services                Install all components (default)
  --database, --only-db         Only install PostgreSQL
  --redis, --only-redis        Only install Redis
  --app, --only-app            Only install Go API + Worker
  --web, --only-web            Only install Nginx + Frontend
  -h, --help                    Show this help

Examples:
  ./deploy/install_server_rocky.sh
  ./deploy/install_server_rocky.sh --app-env production --jwt-secret "very-strong-secret-xxx"
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

as_app() {
  if [[ "${EUID}" -eq 0 ]]; then
    runuser -u "$APP_USER" -- bash -lc "$*"
  else
    sudo -u "$APP_USER" -H bash -lc "$*"
  fi
}

require_linux() {
  if [[ "$(uname -s)" != "Linux" ]]; then
    die "This installer only supports Linux."
  fi
  if [[ "${EUID}" -ne 0 ]] && ! command -v sudo >/dev/null 2>&1; then
    die "sudo is required when running installer as non-root user."
  fi
}

require_rocky_family() {
  local distro
  distro="$(. /etc/os-release && echo "${ID}")"
  if [[ "$distro" != "rocky" && "$distro" != "almalinux" && "$distro" != "rhel" && "$distro" != "centos" ]]; then
    die "Unsupported distro: ${distro}. Use Rocky Linux / AlmaLinux / RHEL / CentOS."
  fi
}

validate_runtime_config() {
  local app_env_lc
  app_env_lc="$(printf "%s" "$APP_ENV" | tr '[:upper:]' '[:lower:]')"
  if [[ "$app_env_lc" == "production" ]] && [[ "$DATABASE_URL" == *"sslmode=disable"* ]]; then
    die "APP_ENV=production cannot be used with DATABASE_URL sslmode=disable. Use sslmode=require (TLS DB) or set --app-env development for local docker DB."
  fi
}

normalize_cors_origins() {
  local original="$CORS_ALLOWED_ORIGINS"
  local -a normalized=()
  local -a parts=()
  IFS=',' read -r -a parts <<< "$CORS_ALLOWED_ORIGINS"

  local raw
  for raw in "${parts[@]}"; do
    local origin="$raw"
    origin="${origin#"${origin%%[![:space:]]*}"}"
    origin="${origin%"${origin##*[![:space:]]}"}"
    if [[ -z "$origin" ]]; then
      continue
    fi

    if [[ "$origin" =~ ^https?: ]]; then
      origin="$(printf '%s' "$origin" | sed -E 's#^(https?):/*#\1://#')"
    elif [[ "$origin" != "*" ]]; then
      origin="http://${origin}"
    fi

    origin="${origin%/}"
    if [[ "$origin" != "*" ]] && [[ ! "$origin" =~ ^https?://[^/]+$ ]]; then
      die "Invalid CORS origin: ${origin}. Example valid value: http://172.18.29.119:5173"
    fi
    normalized+=("$origin")
  done

  if [[ "${#normalized[@]}" -eq 0 ]]; then
    die "CORS_ALLOWED_ORIGINS must contain at least one valid origin."
  fi

  CORS_ALLOWED_ORIGINS="${normalized[0]}"
  local i
  for (( i=1; i<${#normalized[@]}; i++ )); do
    CORS_ALLOWED_ORIGINS+=",${normalized[i]}"
  done

  if [[ "$CORS_ALLOWED_ORIGINS" != "$original" ]]; then
    warn "Normalized CORS origins: ${CORS_ALLOWED_ORIGINS}"
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
      --go-version)
        GO_VERSION="$2"
        shift 2
        ;;
      --app-env)
        APP_ENV="$2"
        shift 2
        ;;
      --http-port)
        HTTP_PORT="$2"
        shift 2
        ;;
      --jwt-secret)
        JWT_SECRET="$2"
        shift 2
        ;;
      --cors)
        CORS_ALLOWED_ORIGINS="$2"
        shift 2
        ;;
      --database-url)
        DATABASE_URL="$2"
        _DATABASE_URL_SPECIFIED=1
        shift 2
        ;;
      --db-user)
        DB_USER="$2"
        _DB_PART_SPECIFIED=1
        shift 2
        ;;
      --db-pass)
        DB_PASS="$2"
        _DB_PART_SPECIFIED=1
        shift 2
        ;;
      --db-name)
        DB_NAME="$2"
        _DB_PART_SPECIFIED=1
        shift 2
        ;;
      --db-host)
        DB_HOST="$2"
        _DB_PART_SPECIFIED=1
        shift 2
        ;;
      --db-port)
        DB_PORT="$2"
        _DB_PART_SPECIFIED=1
        shift 2
        ;;
      --db-schema)
        DB_SCHEMA="$2"
        _DB_PART_SPECIFIED=1
        shift 2
        ;;
      --redis-url)
        RATE_LIMIT_REDIS_URL="$2"
        shift 2
        ;;
      --redis-prefix)
        RATE_LIMIT_REDIS_PREFIX="$2"
        shift 2
        ;;
      --storage-provider)
        STORAGE_PROVIDER="$2"
        shift 2
        ;;
      --storage-local-path)
        STORAGE_LOCAL_PATH="$2"
        shift 2
        ;;
      --skip-tests)
        RUN_TESTS=0
        shift
        ;;
      --skip-seed)
        SEED_DEMO=0
        shift
        ;;
      --no-enable-services)
        ENABLE_SERVICES=0
        shift
        ;;
      --skip-web)
        ENABLE_WEB=0
        shift
        ;;
      --web-port)
        WEB_PORT="$2"
        shift 2
        ;;
      --web-root)
        WEB_ROOT="$2"
        shift 2
        ;;
      --frontend-api-base-url)
        FRONTEND_API_BASE_URL="$2"
        shift 2
        ;;
      --npm-registry)
        NPM_REGISTRY="$2"
        shift 2
        ;;
      --infra-mode)
        INFRA_MODE="$2"
        shift 2
        ;;
      --all-services)
        INSTALL_DB=1; INSTALL_REDIS=1; INSTALL_APP=1; INSTALL_WEB=1
        _COMP_USER_SPECIFIED=1
        shift
        ;;
      --database|--only-db)
        if [[ "$_COMP_USER_SPECIFIED" == "0" ]]; then
           INSTALL_DB=0; INSTALL_REDIS=0; INSTALL_APP=0; INSTALL_WEB=0
           _COMP_USER_SPECIFIED=1
        fi
        INSTALL_DB=1
        shift
        ;;
      --redis|--only-redis)
        if [[ "$_COMP_USER_SPECIFIED" == "0" ]]; then
           INSTALL_DB=0; INSTALL_REDIS=0; INSTALL_APP=0; INSTALL_WEB=0
           _COMP_USER_SPECIFIED=1
        fi
        INSTALL_REDIS=1
        shift
        ;;
      --app|--only-app)
        if [[ "$_COMP_USER_SPECIFIED" == "0" ]]; then
           INSTALL_DB=0; INSTALL_REDIS=0; INSTALL_APP=0; INSTALL_WEB=0
           _COMP_USER_SPECIFIED=1
        fi
        INSTALL_APP=1
        shift
        ;;
      --web|--only-web)
        if [[ "$_COMP_USER_SPECIFIED" == "0" ]]; then
           INSTALL_DB=0; INSTALL_REDIS=0; INSTALL_APP=0; INSTALL_WEB=0
           _COMP_USER_SPECIFIED=1
        fi
        INSTALL_WEB=1
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

install_base_packages() {
  log "Installing base packages (Rocky family)"
  as_root dnf -y makecache
  as_root dnf -y install \
    ca-certificates \
    curl \
    gnupg2 \
    jq \
    unzip \
    rsync \
    openssl \
    postgresql \
    tar \
    dnf-plugins-core
}

install_web_packages() {
  if [[ "$ENABLE_WEB" != "1" ]]; then
    warn "Skipping web package install (--skip-web)"
    return
  fi

  log "Installing web runtime packages (Node.js + Nginx)"
  as_root dnf -y module reset nodejs >/dev/null 2>&1 || true
  as_root dnf -y module enable nodejs:20 >/dev/null 2>&1 || true
  as_root dnf -y install nodejs nginx
}

install_docker() {
  if [[ "$INFRA_MODE" != "docker" ]]; then
    warn "Skipping Docker install (infra-mode: ${INFRA_MODE})"
    return
  fi

  if command -v docker >/dev/null 2>&1; then
    log "Docker already installed, skipping"
    return
  fi

  log "Installing Docker Engine + Compose plugin"
  as_root dnf config-manager --add-repo https://download.docker.com/linux/centos/docker-ce.repo
  as_root dnf -y install docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
  as_root systemctl enable --now docker
}

install_go() {
  local go_bin="/usr/local/go/bin/go"
  if [[ -x "$go_bin" ]]; then
    local current
    current="$("$go_bin" version | awk '{print $3}' | sed 's/go//')"
    if [[ "$current" == "$GO_VERSION" ]]; then
      log "Go ${GO_VERSION} already installed, skipping"
      return
    fi
    warn "Go version ${current} found, upgrading to ${GO_VERSION}"
  else
    log "Installing Go ${GO_VERSION}"
  fi

  local tmp_dir
  tmp_dir="$(mktemp -d)"

  local tarball="go${GO_VERSION}.linux-amd64.tar.gz"
  curl -fsSL "https://go.dev/dl/${tarball}" -o "${tmp_dir}/${tarball}"

  as_root rm -rf /usr/local/go
  as_root tar -C /usr/local -xzf "${tmp_dir}/${tarball}"
  printf 'export PATH=$PATH:/usr/local/go/bin\n' | as_root tee /etc/profile.d/go-path.sh >/dev/null
  rm -rf "${tmp_dir}"
}

ensure_app_user() {
  if ! getent group "$APP_GROUP" >/dev/null 2>&1; then
    as_root groupadd --system "$APP_GROUP"
  fi
  if ! id -u "$APP_USER" >/dev/null 2>&1; then
    as_root useradd --system --create-home --home-dir "/home/${APP_USER}" --shell /bin/bash --gid "$APP_GROUP" "$APP_USER"
  fi
}

sync_project() {
  log "Sync project to ${INSTALL_DIR}"
  as_root mkdir -p "$INSTALL_DIR"
  as_root rsync -a --delete \
    --exclude ".git" \
    --exclude "frontend/node_modules" \
    --exclude "backend/.env" \
    "${REPO_DIR}/" "${INSTALL_DIR}/"
  as_root chown -R "${APP_USER}:${APP_GROUP}" "$INSTALL_DIR"
}

normalize_frontend_i18n_imports() {
  local frontend_dir="${INSTALL_DIR}/frontend/src/features/finance"
  if [[ ! -d "$frontend_dir" ]]; then
    return
  fi

  # Normalize relative import depth to src/core for all finance feature files.
  as_root bash -lc "find '$frontend_dir' -type f \\( -name '*.ts' -o -name '*.tsx' \\) -print0 | xargs -0 -r sed -i -e 's#\\.\\./\\.\\./\\.\\./core/i18n/i18n#../../../../core/i18n/i18n#g' -e 's#\\.\\./\\.\\./\\.\\./\\.\\./\\.\\./core/i18n/i18n#../../../../core/i18n/i18n#g'"
}

check_frontend_i18n() {
  local i18n_dir="${INSTALL_DIR}/frontend/src/core/i18n"
  if [[ ! -f "${i18n_dir}/i18n.tsx" || ! -f "${i18n_dir}/translations.ts" ]]; then
    die "Frontend i18n files are missing at ${i18n_dir}. Please update the project source (git pull / re-upload) and retry."
  fi
}

ensure_auth_route_mount_fix() {
  local target="${INSTALL_DIR}/backend/internal/modules/core/auth/delivery/http/routes.go"
  if [[ ! -f "$target" ]]; then
    die "Auth routes file not found: ${target}"
  fi

  if ! grep -A20 -F 'func (h *Handler) RegisterProtectedRoutes(r chi.Router)' "$target" | grep -Fq 'r.Route("/auth"'; then
    return
  fi

  warn "Detected legacy /auth protected-route mount pattern. Applying compatibility hotfix."
  cat <<'EOF' | as_root tee "$target" >/dev/null
package http

import (
	"time"

	"github.com/go-chi/chi/v5"

	"pekan/backend/internal/platform/middleware"
)

const (
	DefaultLoginRateLimitPerMinute   = 10
	DefaultRefreshRateLimitPerMinute = 30
)

func (h *Handler) RegisterPublicRoutes(r chi.Router) {
	loginLimiter := h.loginLimiter
	if loginLimiter == nil {
		loginLimiter = middleware.NewIPRateLimiter(DefaultLoginRateLimitPerMinute, time.Minute)
	}
	refreshLimiter := h.refreshLimiter
	if refreshLimiter == nil {
		refreshLimiter = middleware.NewIPRateLimiter(DefaultRefreshRateLimitPerMinute, time.Minute)
	}

	r.Route("/auth", func(r chi.Router) {
		r.With(loginLimiter).Post("/login", h.Login)
		r.With(refreshLimiter).Post("/refresh", h.Refresh)
	})
}

func (h *Handler) RegisterProtectedRoutes(r chi.Router) {
	r.Post("/auth/logout", h.Logout)
	r.Post("/auth/logout-all", h.LogoutAll)
	r.Get("/me/context", h.MeContext)
	r.Post("/tenants/{tenantID}/switch", h.SwitchTenant)
}
EOF
  as_root chown "${APP_USER}:${APP_GROUP}" "$target"
}

prepare_env() {
  log "Preparing backend .env"
  if [[ -z "${JWT_SECRET}" ]]; then
    JWT_SECRET="$(openssl rand -base64 48 | tr -d '\n')"
    log "Generated JWT secret automatically"
  fi
  if [[ -z "${RECEIPT_SCAN_SECRET}" ]]; then
    RECEIPT_SCAN_SECRET="${JWT_SECRET}"
  fi

  local env_file="${INSTALL_DIR}/backend/.env"
  cat <<EOF | as_root tee "$env_file" >/dev/null
APP_ENV=${APP_ENV}
HTTP_PORT=${HTTP_PORT}
DATABASE_URL=${DATABASE_URL}
JWT_SECRET=${JWT_SECRET}
RECEIPT_SCAN_SECRET=${RECEIPT_SCAN_SECRET}
JWT_ISSUER=${JWT_ISSUER}
JWT_ACCESS_TTL_MINUTES=15
JWT_REFRESH_TTL_HOURS=720
CORS_ALLOWED_ORIGINS=${CORS_ALLOWED_ORIGINS}
REQUEST_BODY_MAX_BYTES=1048576
RATE_LIMIT_REDIS_URL=${RATE_LIMIT_REDIS_URL}
RATE_LIMIT_REDIS_PREFIX=${RATE_LIMIT_REDIS_PREFIX}
FILE_SCAN_POLL_SECONDS=5
FILE_SCAN_MAX_ATTEMPTS=3
FILE_SCAN_RETRY_DELAY_SECONDS=60
STORAGE_PROVIDER=${STORAGE_PROVIDER}
STORAGE_LOCAL_PATH=${STORAGE_LOCAL_PATH}
STORAGE_S3_BUCKET=
STORAGE_S3_REGION=
STORAGE_GDRIVE_FOLDER_ID=
EOF
  as_root chown "${APP_USER}:${APP_GROUP}" "$env_file"
  as_root chmod 640 "$env_file"
}

start_infra() {
  if [[ "$INFRA_MODE" == "standalone" ]]; then
    install_infra_standalone
    return
  fi

  if [[ "$INSTALL_DB" == "0" ]] || [[ "$INSTALL_REDIS" == "0" ]]; then
    warn "Docker mode detected with modular component selection. Docker Compose will start both Postgres and Redis regardless of --only-X flags."
  fi

  log "Starting PostgreSQL + Redis via docker compose"
  as_root docker compose -f "${INSTALL_DIR}/deploy/docker-compose.server-test.yml" up -d
}

install_infra_standalone() {
  log "Installing native PostgreSQL + Redis (standalone mode)"
  if [[ "$INSTALL_DB" == "1" ]]; then
    as_root dnf install -y postgresql-server postgresql-contrib
  fi
  if [[ "$INSTALL_REDIS" == "1" ]]; then
    as_root dnf install -y redis
  fi

  log "Starting and enabling services"
  if [[ "$INSTALL_DB" == "1" ]]; then
    if ! as_root postgresql-setup --check >/dev/null 2>&1; then
      as_root postgresql-setup --initdb || true
    fi
    as_root systemctl enable --now postgresql
  fi
  if [[ "$INSTALL_REDIS" == "1" ]]; then
    as_root systemctl enable --now redis
  fi

  # Attempt to create database and user if pointing to local
  if [[ "$INSTALL_DB" == "1" ]] && ([[ "$DATABASE_URL" == *"127.0.0.1"* ]] || [[ "$DATABASE_URL" == *"localhost"* ]]); then
    log "Configuring local PostgreSQL database"
    local db_user="${DB_USER}"
    local db_name="${DB_NAME}"
    local db_pass="${DB_PASS}"

    log "Ensuring database user '${db_user}' exists"
    if ! as_root -u postgres psql -tAc "SELECT 1 FROM pg_roles WHERE rolname='$db_user'" | grep -q 1; then
      as_root -u postgres psql -c "CREATE USER $db_user WITH PASSWORD '$db_pass' SUPERUSER;" || true
    fi

    log "Ensuring database '${db_name}' exists"
    if ! as_root -u postgres psql -lqt | cut -d \| -f 1 | grep -qw "$db_name"; then
      as_root -u postgres createdb -O "$db_user" "$db_name" || true
    fi
  fi
}

wait_for_infra() {
  if [[ "$INFRA_MODE" == "standalone" ]]; then
    wait_for_infra_standalone
    return
  fi

  log "Waiting PostgreSQL readiness (Docker)"
  local pg_ready=0
  for _ in $(seq 1 30); do
    if as_root docker exec pekan-postgres pg_isready -U postgres -d pekan >/dev/null 2>&1; then
      pg_ready=1
      break
    fi
    sleep 2
  done
  if [[ "$pg_ready" != "1" ]]; then
    die "PostgreSQL (Docker) is not ready after timeout."
  fi

  log "Waiting Redis readiness (Docker)"
  local redis_ready=0
  for _ in $(seq 1 30); do
    if as_root docker exec pekan-redis redis-cli ping 2>/dev/null | grep -q PONG; then
      redis_ready=1
      break
    fi
    sleep 2
  done
  if [[ "$redis_ready" != "1" ]]; then
    die "Redis (Docker) is not ready after timeout."
  fi
}

wait_for_infra_standalone() {
  log "Waiting PostgreSQL readiness (Standalone)"
  local pg_ready=0
  for _ in $(seq 1 30); do
    if as_root -u postgres pg_isready >/dev/null 2>&1; then
      pg_ready=1
      break
    fi
    sleep 2
  done
  if [[ "$pg_ready" != "1" ]]; then
    die "PostgreSQL (Standalone) is not ready after timeout."
  fi

  log "Waiting Redis readiness (Standalone)"
  local redis_ready=0
  for _ in $(seq 1 30); do
    if redis-cli ping 2>/dev/null | grep -q PONG; then
      redis_ready=1
      break
    fi
    sleep 2
  done
  if [[ "$redis_ready" != "1" ]]; then
    die "Redis (Standalone) is not ready after timeout."
  fi
}

run_backend_bootstrap() {
  log "Installing Go module dependencies"
  as_app "cd '${INSTALL_DIR}/backend' && /usr/local/go/bin/go mod tidy"

  if [[ "$RUN_TESTS" == "1" ]]; then
    log "Running backend tests"
    as_app "cd '${INSTALL_DIR}/backend' && /usr/local/go/bin/go test ./..."
  else
    warn "Skipping tests (--skip-tests)"
  fi

  log "Applying migrations"
  as_app "cd '${INSTALL_DIR}/backend' && chmod +x ./scripts/apply_migrations.sh && DATABASE_URL='${DATABASE_URL}' ./scripts/apply_migrations.sh"

  if [[ "$SEED_DEMO" == "1" ]]; then
    log "Applying demo seed"
    as_app "cd '${INSTALL_DIR}/backend' && chmod +x ./scripts/apply_demo_seed.sh && DATABASE_URL='${DATABASE_URL}' ./scripts/apply_demo_seed.sh"
  else
    warn "Skipping demo seed (--skip-seed)"
  fi
}

build_binaries() {
  log "Building API and worker binaries"
  as_root mkdir -p "${INSTALL_DIR}/bin" "${STORAGE_LOCAL_PATH}"
  as_root chown -R "${APP_USER}:${APP_GROUP}" "${INSTALL_DIR}/bin" "${STORAGE_LOCAL_PATH}"

  as_app "cd '${INSTALL_DIR}/backend' && /usr/local/go/bin/go build -o '${INSTALL_DIR}/bin/pekan-api' ./cmd/api"
  as_app "cd '${INSTALL_DIR}/backend' && /usr/local/go/bin/go build -o '${INSTALL_DIR}/bin/pekan-worker' ./cmd/worker"
}

prepare_frontend_env() {
  if [[ "$ENABLE_WEB" != "1" ]]; then
    return
  fi

  local env_file="${INSTALL_DIR}/frontend/.env.production"
  log "Preparing frontend production env"
  cat <<EOF | as_root tee "$env_file" >/dev/null
VITE_API_BASE_URL=${FRONTEND_API_BASE_URL}
EOF
  as_root chown "${APP_USER}:${APP_GROUP}" "$env_file"
  as_root chmod 640 "$env_file"
}

normalize_frontend_npm() {
  if [[ "$ENABLE_WEB" != "1" ]]; then
    return
  fi

  local frontend_dir="${INSTALL_DIR}/frontend"
  local lock_file="${frontend_dir}/package-lock.json"
  local npmrc_file="${frontend_dir}/.npmrc"

  log "Normalizing frontend npm registry"
  cat <<EOF | as_root tee "$npmrc_file" >/dev/null
registry=${NPM_REGISTRY}
fund=false
audit=false
EOF
  as_root chown "${APP_USER}:${APP_GROUP}" "$npmrc_file"
  as_root chmod 644 "$npmrc_file"

  if [[ -f "$lock_file" ]]; then
    as_root sed -i "s#https://packages.applied-caas-gateway1.internal.api.openai.org/artifactory/api/npm/npm-public/#${NPM_REGISTRY}#g" "$lock_file"
  fi

  as_app "npm config delete proxy >/dev/null 2>&1 || true"
  as_app "npm config delete https-proxy >/dev/null 2>&1 || true"
  as_app "npm config set registry '${NPM_REGISTRY}' >/dev/null 2>&1"
}

build_frontend() {
  if [[ "$ENABLE_WEB" != "1" ]]; then
    return
  fi

  normalize_frontend_npm

  log "Installing frontend dependencies"
  if [[ -f "${INSTALL_DIR}/frontend/package-lock.json" ]]; then
    as_app "cd '${INSTALL_DIR}/frontend' && npm ci --include=dev --registry='${NPM_REGISTRY}' --no-audit --fetch-retries=5 --fetch-retry-mintimeout=20000 --fetch-retry-maxtimeout=120000 || npm install --include=dev --registry='${NPM_REGISTRY}' --no-audit --fetch-retries=5 --fetch-retry-mintimeout=20000 --fetch-retry-maxtimeout=120000"
  else
    as_app "cd '${INSTALL_DIR}/frontend' && npm install --include=dev --registry='${NPM_REGISTRY}' --no-audit --fetch-retries=5 --fetch-retry-mintimeout=20000 --fetch-retry-maxtimeout=120000"
  fi

  log "Building frontend bundle"
  as_app "cd '${INSTALL_DIR}/frontend' && npm run build"

  if [[ ! -d "${INSTALL_DIR}/frontend/dist" ]]; then
    die "Frontend build output not found: ${INSTALL_DIR}/frontend/dist"
  fi
}

publish_frontend() {
  if [[ "$ENABLE_WEB" != "1" ]]; then
    return
  fi

  log "Publishing frontend to ${WEB_ROOT}"
  as_root mkdir -p "${WEB_ROOT}"
  as_root rm -rf "${WEB_ROOT:?}/"*
  as_root cp -r "${INSTALL_DIR}/frontend/dist/"* "${WEB_ROOT}/"
}

check_web_port_conflict() {
  if [[ "$ENABLE_WEB" != "1" ]]; then
    return
  fi

  local listeners
  listeners="$(as_root ss -ltnp "( sport = :${WEB_PORT} )" 2>/dev/null | awk 'NR>1 {print}')"
  if [[ -z "$listeners" ]]; then
    return
  fi

  if printf '%s\n' "$listeners" | grep -qi 'nginx'; then
    return
  fi

  warn "Detected existing listener on port ${WEB_PORT}:"
  printf '%s\n' "$listeners" >&2
  die "WEB_PORT ${WEB_PORT} is already in use by another process. Use --web-port <free_port> or stop the conflicting service."
}

start_or_restart_nginx() {
  if as_root systemctl is-active --quiet nginx; then
    as_root systemctl restart nginx
  else
    as_root systemctl enable --now nginx
  fi
}

install_fail2ban() {
  log "Installing and configuring Fail2Ban for security (Rocky/CentOS)"
  as_root dnf install -y fail2ban

  # Create custom filter for login failures
  cat <<EOF | as_root tee /etc/fail2ban/filter.d/pekan-auth.conf >/dev/null
[Definition]
failregex = ^<HOST> -.*"POST /api/v1/auth/login HTTP/.*" 401
ignoreregex =
EOF

  # Create custom jail for Pekan API
  cat <<EOF | as_root tee /etc/fail2ban/jail.d/pekan.conf >/dev/null
[pekan-auth]
enabled = true
port = http,https,${WEB_PORT},${HTTP_PORT}
filter = pekan-auth
logpath = /var/log/nginx/access.log
maxretry = 5
findtime = 600
bantime = 3600

[pekan-api-limit]
enabled = true
port = http,https,${WEB_PORT},${HTTP_PORT}
filter = nginx-limit-req
logpath = /var/log/nginx/access.log
maxretry = 10
findtime = 600
bantime = 3600
EOF

  as_root systemctl restart fail2ban
  as_root systemctl enable fail2ban
}

install_nginx_site() {
  if [[ "$ENABLE_WEB" != "1" ]]; then
    return
  fi

  check_web_port_conflict

  log "Configuring Nginx web gateway"
  cat <<EOF | as_root tee /etc/nginx/conf.d/pekan.conf >/dev/null
server {
    listen ${WEB_PORT};
    server_name _;
    client_max_body_size 50M;

    root ${WEB_ROOT};
    index index.html;

    # Gzip Compression
    gzip on;
    gzip_vary on;
    gzip_proxied any;
    gzip_comp_level 6;
    gzip_types text/plain text/css text/xml application/json application/javascript application/xml+rss application/atom+xml image/svg+xml;

    # Security Headers
    add_header X-Frame-Options "SAMEORIGIN" always;
    add_header X-XSS-Protection "1; mode=block" always;
    add_header X-Content-Type-Options "nosniff" always;
    add_header Referrer-Policy "strict-origin-when-cross-origin" always;
    add_header Content-Security-Policy "default-src 'self' http: https: data: blob: 'unsafe-inline' 'unsafe-eval';" always;

    location /api/ {
        proxy_pass http://127.0.0.1:${HTTP_PORT};
        proxy_http_version 1.1;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
    }

    location / {
        try_files \$uri /index.html;
    }
}
EOF

  as_root nginx -t
  if ! start_or_restart_nginx; then
    warn "Nginx failed to start/restart"
    as_root systemctl status nginx.service --no-pager || true
    as_root journalctl -u nginx.service -n 120 --no-pager || true
    die "Failed to start nginx. See logs above."
  fi
}

configure_web_firewall() {
  if [[ "$ENABLE_WEB" != "1" ]]; then
    return
  fi
  if ! command -v firewall-cmd >/dev/null 2>&1; then
    return
  fi
  if ! as_root systemctl is-active --quiet firewalld; then
    return
  fi

  if [[ "$WEB_PORT" == "80" ]]; then
    as_root firewall-cmd --permanent --add-service=http >/dev/null 2>&1 || true
  else
    as_root firewall-cmd --permanent --add-port="${WEB_PORT}/tcp" >/dev/null 2>&1 || true
  fi
  as_root firewall-cmd --reload >/dev/null 2>&1 || true
}

install_systemd_services() {
  log "Installing systemd services"

  cat <<EOF | as_root tee /etc/systemd/system/pekan-api.service >/dev/null
[Unit]
Description=PEKAN API Service
After=network.target

[Service]
Type=simple
WorkingDirectory=${INSTALL_DIR}/backend
EnvironmentFile=${INSTALL_DIR}/backend/.env
ExecStart=${INSTALL_DIR}/bin/pekan-api
Restart=always
RestartSec=5
User=${APP_USER}
Group=${APP_GROUP}

[Install]
WantedBy=multi-user.target
EOF

  cat <<EOF | as_root tee /etc/systemd/system/pekan-worker.service >/dev/null
[Unit]
Description=PEKAN File Scan Worker Service
After=network.target

[Service]
Type=simple
WorkingDirectory=${INSTALL_DIR}/backend
EnvironmentFile=${INSTALL_DIR}/backend/.env
ExecStart=${INSTALL_DIR}/bin/pekan-worker
Restart=always
RestartSec=5
User=${APP_USER}
Group=${APP_GROUP}

[Install]
WantedBy=multi-user.target
EOF

  as_root systemctl daemon-reload
  if [[ "$ENABLE_SERVICES" == "1" ]]; then
    as_root systemctl enable pekan-api.service pekan-worker.service
    # Always restart to pick up fresh binaries/config after deployment.
    as_root systemctl restart pekan-api.service pekan-worker.service
  else
    warn "Services installed but not started (--no-enable-services)"
  fi
}

verify_services() {
  if [[ "$ENABLE_SERVICES" != "1" ]]; then
    return
  fi

  log "Verifying systemd services"
  for svc in pekan-api pekan-worker; do
    if ! as_root systemctl is-active --quiet "${svc}.service"; then
      warn "Service ${svc}.service is not active"
      as_root systemctl status "${svc}.service" --no-pager || true
      as_root journalctl -u "${svc}.service" -n 80 --no-pager || true
      die "Service ${svc}.service failed to start"
    fi
  done
}

verify_api_health() {
  if [[ "$ENABLE_SERVICES" != "1" ]]; then
    return
  fi

  log "Verifying API health endpoint on 127.0.0.1:${HTTP_PORT}"
  local ok=0
  for _ in $(seq 1 20); do
    if curl -fsS "http://127.0.0.1:${HTTP_PORT}/api/v1/healthz" >/dev/null 2>&1; then
      ok=1
      break
    fi
    sleep 1
  done
  if [[ "$ok" != "1" ]]; then
    warn "API health check failed on port ${HTTP_PORT}"
    as_root systemctl status pekan-api.service --no-pager || true
    as_root journalctl -u pekan-api.service -n 120 --no-pager || true
    die "API is not reachable on 127.0.0.1:${HTTP_PORT}"
  fi
}

verify_web_health() {
  if [[ "$ENABLE_WEB" != "1" ]]; then
    return
  fi

  log "Verifying web UI endpoint on 127.0.0.1:${WEB_PORT}"
  local ok=0
  for _ in $(seq 1 20); do
    if curl -fsS "http://127.0.0.1:${WEB_PORT}/" >/dev/null 2>&1; then
      ok=1
      break
    fi
    sleep 1
  done

  if [[ "$ok" != "1" ]]; then
    warn "Web UI health check failed on port ${WEB_PORT}"
    as_root systemctl status nginx --no-pager || true
    die "Web UI is not reachable on 127.0.0.1:${WEB_PORT}"
  fi
}

print_summary() {
  cat <<EOF

Installation completed (Rocky installer).

Key paths:
- Install dir: ${INSTALL_DIR}
- Backend env: ${INSTALL_DIR}/backend/.env
- API binary: ${INSTALL_DIR}/bin/pekan-api
- Worker binary: ${INSTALL_DIR}/bin/pekan-worker
- Web root: ${WEB_ROOT}

Demo credential (if seed enabled):
- email: owner@pekan.local
- password: password
- tenant_id: 11111111-1111-1111-1111-111111111111

Quick checks:
- API health: curl -i http://127.0.0.1:${HTTP_PORT}/api/v1/healthz
- Service status: systemctl status pekan-api pekan-worker --no-pager
- Web health: curl -i http://127.0.0.1:${WEB_PORT}/

EOF
}

load_existing_config() {
  local env_file="${INSTALL_DIR}/backend/.env"
  if [[ -f "$env_file" ]]; then
    log "Loading existing configuration from $env_file"
    
    # Only load if not already set via flags
    [[ -z "${DATABASE_URL:-}" ]] && DATABASE_URL=$(as_root grep "^DATABASE_URL=" "$env_file" | cut -d= -f2- || echo "")
    [[ -z "${HTTP_PORT:-}" ]] && HTTP_PORT=$(as_root grep "^HTTP_PORT=" "$env_file" | cut -d= -f2- || echo "")
    [[ -z "${JWT_SECRET:-}" ]] && JWT_SECRET=$(as_root grep "^JWT_SECRET=" "$env_file" | cut -d= -f2- || echo "")
    [[ -z "${RECEIPT_SCAN_SECRET:-}" ]] && RECEIPT_SCAN_SECRET=$(as_root grep "^RECEIPT_SCAN_SECRET=" "$env_file" | cut -d= -f2- || echo "")
    [[ -z "${CORS_ALLOWED_ORIGINS:-}" ]] && CORS_ALLOWED_ORIGINS=$(as_root grep "^CORS_ALLOWED_ORIGINS=" "$env_file" | cut -d= -f2- || echo "")
    
    # If DATABASE_URL was loaded, we don't need to reconstruct it from DB_* flags
    if [[ -n "${DATABASE_URL}" ]]; then
      INSTALL_DB=0
    fi
  fi
}

main() {
  load_existing_config
  parse_args "$@"

  if [[ -z "$DATABASE_URL" ]]; then
    DATABASE_URL="postgres://${DB_USER}:${DB_PASS}@${DB_HOST}:${DB_PORT}/${DB_NAME}?sslmode=${DB_SSLMODE}"
    if [[ "$DB_SCHEMA" != "public" ]]; then
      DATABASE_URL="${DATABASE_URL}&search_path=${DB_SCHEMA}"
    fi
  fi

  require_linux
  require_rocky_family
  normalize_cors_origins
  validate_runtime_config
  
  install_base_packages
  ensure_app_user
  sync_project

  # Infrastructure (DB + Redis)
  if [[ "$INSTALL_DB" == "1" ]] || [[ "$INSTALL_REDIS" == "1" ]]; then
    install_docker # only installs if infra_mode=docker
    start_infra
    wait_for_infra
  fi

  # Go Backend (App)
  if [[ "$INSTALL_APP" == "1" ]]; then
    install_go
    normalize_frontend_i18n_imports
    check_frontend_i18n
    ensure_auth_route_mount_fix
    prepare_env
    run_backend_bootstrap
    build_binaries
    install_systemd_services
    verify_services
    verify_api_health
  fi

  # Frontend (Web)
  if [[ "$INSTALL_WEB" == "1" ]]; then
    install_web_packages
    prepare_frontend_env
    build_frontend
    publish_frontend
    install_nginx_site
    install_fail2ban
    configure_web_firewall
    verify_web_health
  fi

  print_summary
}

main "$@"
