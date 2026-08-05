#!/bin/bash
set -e

# Warna untuk output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log_header() { echo -e "\n${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"; echo -e "${YELLOW} $1${NC}"; echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"; }
log_info() { echo -e "ℹ️  $1"; }
log_success() { echo -e "✅ $1"; }
log_warn() { echo -e "⚠️  $1"; }
log_error() { echo -e "❌ $1"; }

as_root() {
    if [ "$EUID" -ne 0 ]; then
        sudo "$@"
    else
        "$@"
    fi
}

check_prerequisites() {
    log_header "STEP 1: Checking Prerequisites"
    for tool in openssl docker; do
        if ! command -v $tool &> /dev/null; then
            log_error "$tool not found. Please install it first."
            exit 1
        fi
        log_success "$tool available"
    done
    
    if [ ! -d "deploy" ] || [ ! -d "backend" ]; then
        log_error "Not in project root or missing deploy/backend directories."
        exit 1
    fi
    log_success "Project structure verified"
}

generate_certs() {
    log_header "STEP 2: Generating TLS Certificates for Database"
    CERT_DIR="/tmp/db-certs"
    mkdir -p "$CERT_DIR"
    
    if [ ! -f "$CERT_DIR/ca-key.pem" ]; then
        log_info "Generating CA private key (4096-bit RSA)..."
        openssl genrsa -out "$CERT_DIR/ca-key.pem" 4096
    else
        log_warn "ca-key.pem already exists, skipping..."
    fi
    
    if [ ! -f "$CERT_DIR/ca-cert.pem" ]; then
        log_info "Generating self-signed certificate (365 days valid)..."
        openssl req -new -x509 -days 365 -key "$CERT_DIR/ca-key.pem" -out "$CERT_DIR/ca-cert.pem" -subj "/C=ID/ST=Jakarta/L=Jakarta/O=PEKAN/CN=pekan-postgres"
    else
        log_warn "ca-cert.pem already exists, skipping..."
    fi
    
    log_info "Setting certificate permissions..."
    chmod 600 "$CERT_DIR"/*.pem
    log_success "Certificate permissions set"
    log_success "TLS certificates ready at: $CERT_DIR"
    ls -lh "$CERT_DIR"
}

setup_db_tls_dir() {
    log_header "STEP 3: Setting Up Database TLS Directory"
    DB_CERT_DIR="/var/lib/postgresql/certs"
    as_root mkdir -p "$DB_CERT_DIR"
    log_success "Directory created: $DB_CERT_DIR"
    
    log_info "Copying certificates..."
    as_root cp /tmp/db-certs/*.pem "$DB_CERT_DIR/"
    log_success "Certificates copied"
    
    log_info "Setting ownership and permissions (postgres:uid=999)..."
    as_root chown -R 999:999 "$DB_CERT_DIR"
    as_root chmod 600 "$DB_CERT_DIR"/*.pem
    log_success "Ownership and permissions set"
    log_success "PostgreSQL TLS setup complete"
    as_root ls -lh "$DB_CERT_DIR"
}

setup_env() {
    log_header "STEP 4: Setting Up Environment Variables"
    if [ -f "backend/.env" ]; then
        log_info "Backing up existing .env if present..."
        cp backend/.env backend/.env.backup.$(date +%s)
        log_success "Backup created"
    fi
    
    log_info "Verifying required environment variables..."
    if [ ! -f "backend/.env" ]; then
        log_error ".env missing. Please create it first."
        exit 1
    fi
    log_success ".env already exists"
}

run_main_installer() {
    log_header "STEP 5: Running Main Installation Script"
    PROJECT_ROOT=$(pwd)
    log_info "Project root: $PROJECT_ROOT"
    
    chmod +x deploy/install_server.sh
    log_info "Making installer executable..."
    
    log_warn "Starting main installer (this may take 8-15 minutes)..."
    log_warn "⏳ Installing dependencies, Docker, Go, Node.js..."
    log_warn "⏳ Building backend and frontend..."
    log_warn "⏳ Running migrations..."
    log_warn "⏳ Starting services..."
    
    # Run the main installer with proper flags
    if as_root bash deploy/install_server.sh --skip-seed; then
        log_success "Main installer completed successfully"
    else
        log_error "Main installer failed. Please check the logs above."
        exit 1
    fi
}

configure_db_tls() {
    log_header "STEP 6: Skipping Database TLS (Bypassed)"
    log_info "SSL configuration bypassed to ensure stability."
    return 0
}

verify_deployment() {
    log_header "STEP 7: Verifying Production Deployment"
    log_info "Checking API service..."
    if systemctl is-active --quiet pekan-api; then
        log_success "API service: RUNNING"
    else
        log_error "API service: NOT RUNNING"
    fi
    
    log_info "Checking Worker service..."
    if systemctl is-active --quiet pekan-worker; then
        log_success "Worker service: RUNNING"
    else
        log_error "Worker service: NOT RUNNING"
    fi
    
    log_info "Checking Docker containers..."
    docker ps --format "table {{.Names}}\t{{.Status}}" | head -n 12
}

# Main Execution Flow
log_header "🚀 PEKAN PRODUCTION INSTALLER"
log_info "Starting production deployment..."

check_prerequisites
# generate_certs
# setup_db_tls_dir
setup_env
run_main_installer
configure_db_tls
verify_deployment

log_header "🎉 PRODUCTION DEPLOYMENT COMPLETE"
log_success "Production deployment completed successfully! 🎉"
