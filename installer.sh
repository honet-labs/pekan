#!/usr/bin/env bash
# =============================================================================
# PEKAN Universal One-Click Installer
# https://github.com/honet-labs/pekan
#
# Supported Modes:
#   1) docker    - Production with Docker & Docker Compose (Recommended)
#   2) systemd   - Production with Native Go binaries, Nginx & Systemd
#   3) dev       - Development setup (Local DB container, .env, migrations, npm)
#
# Usage:
#   sudo bash installer.sh [options]
#   curl -sSL https://raw.githubusercontent.com/honet-labs/pekan/main/installer.sh | sudo bash
# =============================================================================

if [ -z "${BASH_VERSION:-}" ]; then
  exec /usr/bin/env bash "$0" "$@"
fi

set -Eeuo pipefail

# ANSI Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m' # No Color

# Helper log functions
log()     { printf "${BLUE}[INFO]${NC} %s\n" "$*"; }
success() { printf "${GREEN}[SUCCESS]${NC} %s\n" "$*"; }
warn()    { printf "${YELLOW}[WARN]${NC} %s\n" "$*"; }
error()   { printf "${RED}[ERROR]${NC} %s\n" "$*" >&2; }
die()     { error "$*"; exit 1; }

REPO_URL="https://github.com/honet-labs/pekan.git"
DEFAULT_BRANCH="main"
BRANCH="$DEFAULT_BRANCH"
INSTALL_DIR="/opt/pekan"
MODE=""
WEB_PORT=""
NON_INTERACTIVE=0

show_banner() {
  cat <<'EOF'
  ____  _____ _  __    _    _   _ 
 |  _ \| ____| |/ /   / \  | \ | |
 | |_) |  _| | ' /   / _ \ |  \| |
 |  __/| |___| . \  / ___ \| |\  |
 |_|   |_____|_|\_\/_/   \_\_| \_|
                                  
 Platform Pencatatan Keuangan Multi-Tenant
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
EOF
}

show_help() {
  show_banner
  cat <<EOF
Penggunaan:
  sudo bash installer.sh [OPSI]

Opsi yang Tersedia:
  -m, --mode <docker|systemd|dev>  Pilih mode instalasi langsung
  -p, --port <port>                Port Web/HTTP (default: 80 untuk produksi, 5173 untuk dev)
  -b, --branch <nama>              Git branch (default: main)
  -d, --dir <path>                 Direktori instalasi (default: /opt/pekan untuk produksi)
  -y, --yes, --non-interactive     Jalankan tanpa konfirmasi interaktif
  -h, --help                       Tampilkan bantuan ini

Pilihan Mode:
  docker   : [Produksi - Rekomendasi] Semua service berjalan di container Docker
             (pekan-web, pekan-api, pekan-worker, pekan-ai, postgres, redis).
  systemd  : [Produksi - Native] Install binary Go native, PostgreSQL lokal,
             Redis, Nginx reverse proxy, dan unit systemd.
  dev      : [Lokal Development] Siapkan Docker untuk DB/Redis dev, buat backend/.env,
             jalankan migrasi database, dan install dependensi frontend (npm).

Contoh:
  # Mode interaktif (tampil menu pilihan)
  sudo bash installer.sh

  # Langsung install Docker produksi pada port 80
  sudo bash installer.sh --mode docker

  # Langsung install Docker produksi pada port kustom (misal: 8080)
  sudo bash installer.sh --mode docker --port 8080

  # Setup lingkungan development lokal
  bash installer.sh --mode dev

  # One-liner dari server baru
  curl -sSL https://raw.githubusercontent.com/honet-labs/pekan/main/installer.sh | sudo bash
EOF
}

# Parse CLI arguments
while [[ $# -gt 0 ]]; do
  case "$1" in
    -m|--mode)
      MODE="$2"
      shift 2
      ;;
    --docker)
      MODE="docker"
      shift
      ;;
    --systemd)
      MODE="systemd"
      shift
      ;;
    --dev|--development)
      MODE="dev"
      shift
      ;;
    -p|--port|--web-port)
      WEB_PORT="$2"
      shift 2
      ;;
    -b|--branch)
      BRANCH="$2"
      shift 2
      ;;
    -d|--dir|--install-dir)
      INSTALL_DIR="$2"
      shift 2
      ;;
    -y|--yes|--non-interactive)
      NON_INTERACTIVE=1
      shift
      ;;
    -h|--help)
      show_help
      exit 0
      ;;
    *)
      error "Opsi tidak dikenal: $1"
      echo "Gunakan 'bash installer.sh --help' untuk melihat daftar opsi."
      exit 1
      ;;
  esac
done

# Ensure root privileges for system-level installation
require_root() {
  if [ "${EUID:-$(id -u)}" -ne 0 ]; then
    warn "Operasi ini membutuhkan hak akses root / sudo."
    log "Mencoba menjalankan ulang menggunakan sudo..."
    exec sudo bash "$0" "$@"
  fi
}

# Determine current repository directory or clone if running via curl/pipe
ensure_repo_dir() {
  local current_dir
  current_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

  if [ -f "$current_dir/backend/internal/app/server.go" ] && [ -d "$current_dir/deploy" ]; then
    REPO_DIR="$current_dir"
    log "Menjalankan dari direktori lokal: $REPO_DIR"
  else
    log "Script tidak dijalankan dari dalam repositori lokal yang valid."
    log "Menyiapkan repositori PEKAN di: $INSTALL_DIR"
    
    require_root
    
    if ! command -v git &>/dev/null; then
      log "Memasang git..."
      if command -v apt-get &>/dev/null; then
        apt-get update -qq && apt-get install -y -qq git
      elif command -v dnf &>/dev/null; then
        dnf install -y -q git
      elif command -v yum &>/dev/null; then
        yum install -y -q git
      else
        die "Git tidak ditemukan dan package manager tidak didukung. Harap install git secara manual."
      fi
    fi

    if [ -d "$INSTALL_DIR/.git" ]; then
      log "Memperbarui repositori yang ada di $INSTALL_DIR..."
      git -C "$INSTALL_DIR" fetch --all --prune
      git -C "$INSTALL_DIR" checkout "$BRANCH"
      git -C "$INSTALL_DIR" pull --ff-only origin "$BRANCH" || true
    else
      log "Melakukan clone repositori dari $REPO_URL (branch: $BRANCH)..."
      mkdir -p "$(dirname "$INSTALL_DIR")"
      git clone -b "$BRANCH" "$REPO_URL" "$INSTALL_DIR"
    fi
    REPO_DIR="$INSTALL_DIR"
  fi
}

# Interactive mode menu
prompt_mode() {
  show_banner
  echo -e "${BOLD}Pilih mode instalasi yang Anda inginkan:${NC}"
  echo ""
  echo -e "  ${GREEN}1) Produksi - Docker Containers (Direkomendasikan)${NC}"
  echo -e "     • Semua service berjalan di container Docker terisolasi."
  echo -e "     • Otomatis setup: Web (Nginx), API, Background Worker, AI Queue, PostgreSQL, & Redis."
  echo -e "     • Mudah diupdate dan di-backup secara independen."
  echo ""
  echo -e "  ${CYAN}2) Produksi - Systemd Native${NC}"
  echo -e "     • Binary Go native + Nginx + PostgreSQL + Redis sistem."
  echo -e "     • Performa bare-metal maksimal, konsumsi RAM lebih hemat."
  echo ""
  echo -e "  ${YELLOW}3) Development - Lingkungan Lokal${NC}"
  echo -e "     • Setup cepat untuk developer / pengujian lokal."
  echo -e "     • Menjalankan database di Docker, migrasi schema, buat .env, dan npm install."
  echo ""
  echo -e "  ${RED}4) Batalkan / Keluar${NC}"
  echo ""

  local choice=""
  while true; do
    read -r -p "Masukkan pilihan Anda [1-4] (default: 1): " choice
    choice="${choice:-1}"
    case "$choice" in
      1)
        MODE="docker"
        break
        ;;
      2)
        MODE="systemd"
        break
        ;;
      3)
        MODE="dev"
        break
        ;;
      4|q|Q)
        log "Instalasi dibatalkan."
        exit 0
        ;;
      *)
        warn "Pilihan tidak valid. Silakan masukkan angka 1, 2, 3, atau 4."
        ;;
    esac
  done
}

# --- MODE 1: DOCKER INSTALLATION ---
run_docker_install() {
  require_root
  log "Memulai instalasi Produksi - Docker..."

  local docker_script="$REPO_DIR/deploy/installer-docker.sh"
  if [ ! -f "$docker_script" ]; then
    die "Script installer Docker tidak ditemukan di: $docker_script"
  fi

  chmod +x "$docker_script"

  local cmd=(bash "$docker_script" --branch "$BRANCH" --install-dir "$INSTALL_DIR")
  if [ -n "$WEB_PORT" ]; then
    cmd+=(--web-port "$WEB_PORT")
  fi

  log "Mengeksekusi: ${cmd[*]}"
  "${cmd[@]}"
}

# --- MODE 2: SYSTEMD INSTALLATION ---
run_systemd_install() {
  require_root
  log "Memulai instalasi Produksi - Systemd..."

  local systemd_script="$REPO_DIR/deploy/installer-systemd.sh"
  if [ ! -f "$systemd_script" ]; then
    die "Script installer Systemd tidak ditemukan di: $systemd_script"
  fi

  chmod +x "$systemd_script"

  local cmd=(bash "$systemd_script" --branch "$BRANCH" --install-dir "$INSTALL_DIR")
  if [ -n "$WEB_PORT" ]; then
    cmd+=(--web-port "$WEB_PORT")
  fi

  log "Mengeksekusi: ${cmd[*]}"
  "${cmd[@]}"
}

# --- MODE 3: LOCAL DEV INSTALLATION ---
run_dev_install() {
  log "Memulai setup lingkungan Development Lokal..."

  # 1. Prerequisite checks
  log "Langkah 1/5: Memeriksa dependensi lokal (docker, go, node, npm)..."
  local missing=()
  for bin in docker go node npm; do
    if ! command -v "$bin" &>/dev/null; then
      missing+=("$bin")
    else
      log "  ✔ $bin terdeteksi: $($bin --version 2>&1 | head -n 1)"
    fi
  done

  if [ ${#missing[@]} -gt 0 ]; then
    error "Alat berikut wajib terinstall di komputer Anda: ${missing[*]}"
    echo "Panduan instalasi prasyarat tersedia di file INSTALL.md."
    exit 1
  fi

  # 2. Start PostgreSQL & Redis via Docker
  log "Langkah 2/5: Menjalankan PostgreSQL & Redis via Docker..."
  local compose_file="$REPO_DIR/deploy/docker-compose.server-test.yml"
  if [ ! -f "$compose_file" ]; then
    compose_file="$REPO_DIR/docker-compose.yml"
  fi

  if command -v docker &>/dev/null; then
    docker compose -f "$compose_file" up -d
  else
    die "Docker compose gagal dijalankan."
  fi

  # Wait for PostgreSQL to be ready
  log "Menunggu database PostgreSQL siap menerima koneksi..."
  local attempts=0
  local max_attempts=30
  local pg_ready=0
  while [ $attempts -lt $max_attempts ]; do
    if docker exec pekan-postgres pg_isready -U postgres -d pekan &>/dev/null; then
      pg_ready=1
      break
    fi
    attempts=$((attempts + 1))
    sleep 1
  done

  if [ $pg_ready -eq 1 ]; then
    success "PostgreSQL siap digunakan!"
  else
    warn "PostgreSQL belum merespons dalam 30 detik, melanjutkan migrasi..."
  fi

  # 3. Setup backend/.env
  log "Langkah 3/5: Mengonfigurasi backend/.env..."
  local env_file="$REPO_DIR/backend/.env"
  if [ ! -f "$env_file" ]; then
    cp "$REPO_DIR/backend/.env.example" "$env_file"
    log "File backend/.env berhasil dibuat dari template."
    
    # Generate random JWT_SECRET
    local secret=""
    if command -v openssl &>/dev/null; then
      secret="$(openssl rand -base64 48 | tr -d '\n\r')"
    else
      secret="$(head -c 32 /dev/urandom | base64 | tr -d '\n\r')"
    fi
    
    # Update JWT_SECRET
    if grep -q "^JWT_SECRET=" "$env_file"; then
      sed -i "s|^JWT_SECRET=.*|JWT_SECRET=${secret}|g" "$env_file" 2>/dev/null || \
      sed -i '' "s|^JWT_SECRET=.*|JWT_SECRET=${secret}|g" "$env_file" 2>/dev/null || true
    else
      echo "JWT_SECRET=${secret}" >> "$env_file"
    fi
    success "JWT_SECRET acak yang aman berhasil digenerate."
  else
    log "File backend/.env sudah ada, mempertahankan konfigurasi saat ini."
  fi

  # 4. Run Migrations
  log "Langkah 4/5: Menjalankan migrasi database..."
  local db_url="postgres://postgres:postgres@localhost:5432/pekan?sslmode=disable"
  
  if [ -f "$REPO_DIR/backend/scripts/apply_migrations.sh" ]; then
    log "Menjalankan migrasi schema global..."
    DATABASE_URL="$db_url" bash "$REPO_DIR/backend/scripts/apply_migrations.sh" || {
      warn "Gagal via psql lokal, mencoba via container PostgreSQL..."
      for sql_file in "$REPO_DIR"/backend/migrations/*.sql; do
        if [ -f "$sql_file" ]; then
          docker exec -i pekan-postgres psql -U postgres -d pekan < "$sql_file"
        fi
      done
    }
    success "Migrasi global berhasil diaplikasikan."
  fi

  if [ -f "$REPO_DIR/backend/scripts/migrate_tenants.go" ]; then
    log "Menjalankan migrasi schema tenant..."
    (cd "$REPO_DIR/backend" && DATABASE_URL="$db_url" go run ./scripts/migrate_tenants.go) || warn "Migrasi tenant dapat dijalankan nanti jika belum ada data workspace."
  fi

  # 5. Install Frontend Dependencies
  log "Langkah 5/5: Memasang dependensi frontend (npm install)..."
  if [ -d "$REPO_DIR/frontend" ]; then
    (cd "$REPO_DIR/frontend" && npm install)
    success "Dependensi frontend berhasil dipasang."
  fi

  # Summary
  echo ""
  echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
  echo -e "${BOLD}${GREEN}🎉 SETUP DEVELOPMENT SELESAI DENGAN SUKSES!${NC}"
  echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
  echo ""
  echo -e "${BOLD}Cara Menjalankan PEKAN:${NC}"
  echo ""
  echo -e "  ${CYAN}Terminal 1 (REST API):${NC}"
  echo -e "    cd backend && go run cmd/api/main.go"
  echo ""
  echo -e "  ${CYAN}Terminal 2 (Frontend React):${NC}"
  echo -e "    cd frontend && npm run dev"
  echo ""
  echo -e "  ${CYAN}Terminal 3 (Opsional - Background Worker):${NC}"
  echo -e "    cd backend && go run cmd/worker/main.go"
  echo ""
  echo -e "  ${CYAN}Terminal 4 (Opsional - AI Chatbot Worker):${NC}"
  echo -e "    cd backend && go run cmd/ai/main.go"
  echo ""
  echo -e "${BOLD}Informasi Akses:${NC}"
  echo -e "  • Web UI    : ${CYAN}http://localhost:5173${NC}"
  echo -e "  • API Server: ${CYAN}http://localhost:8080/api/v1/healthz${NC}"
  echo -e "  • Tenant    : ${BOLD}default${NC}"
  echo -e "  • Email     : ${BOLD}owner@pekan.local${NC}"
  echo -e "  • Password  : ${BOLD}password${NC}"
  echo ""
  echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
}

# Main Execution Flow
main() {
  ensure_repo_dir

  if [ -z "$MODE" ]; then
    if [ "$NON_INTERACTIVE" -eq 1 ]; then
      MODE="docker" # Default non-interactive mode
    else
      prompt_mode
    fi
  fi

  case "$MODE" in
    docker)
      run_docker_install
      ;;
    systemd)
      run_systemd_install
      ;;
    dev|development|local)
      run_dev_install
      ;;
    *)
      error "Mode '$MODE' tidak valid. Pilih: docker, systemd, atau dev."
      exit 1
      ;;
  esac
}

main "$@"
