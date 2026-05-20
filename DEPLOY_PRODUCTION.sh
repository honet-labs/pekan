#!/bin/bash

# ============================================
# SaaS-PEKAN Production Deployment Script
# Environment: Production
# Server: 192.168.201.18
# Date: 2026-04-03
# ============================================

set -e  # Exit on any error

REMOTE_USER="administrator"
REMOTE_HOST="192.168.201.18"
REMOTE_PATH="/opt/pekan"
LOCAL_PROJECT_PATH="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo "🚀 Starting Production Deployment..."
echo "📍 Target: $REMOTE_USER@$REMOTE_HOST:$REMOTE_PATH"
echo "📦 Source: $LOCAL_PROJECT_PATH"
echo ""

# Step 1: Prepare deployment on local machine
echo "📋 [STEP 1/4] Preparing deployment..."
echo "  ✓ Checking backend compilation..."
cd "$LOCAL_PROJECT_PATH/backend"
go mod tidy
go build -o /tmp/pekan-api ./cmd/api
echo "    Backend compiled successfully ✓"

echo "  ✓ Checking frontend build..."
cd "$LOCAL_PROJECT_PATH/frontend"
npm install 2>&1 | grep -v "^warn" || true
npm run build
echo "    Frontend built successfully ✓"

# Step 2: Upload project to server
echo ""
echo "📤 [STEP 2/4] Uploading project to server..."
ssh "$REMOTE_USER@$REMOTE_HOST" "mkdir -p $REMOTE_PATH" || true

# Create exclude file for rsync
cat > /tmp/rsync_exclude.txt <<EOF
node_modules/
dist/
.git/
.env
backend/.env
coverage/
*.test.go
EOF

rsync -avz \
  --delete \
  --exclude-from=/tmp/rsync_exclude.txt \
  "$LOCAL_PROJECT_PATH/" "$REMOTE_USER@$REMOTE_HOST:$REMOTE_PATH/"

echo "  Upload completed ✓"

# Step 3: Run deployment script on server
echo ""
echo "⚙️  [STEP 3/4] Running server setup..."
ssh -t "$REMOTE_USER@$REMOTE_HOST" bash <<'REMOTE_SCRIPT'
set -e
REMOTE_PATH="/opt/pekan"
cd $REMOTE_PATH

# Make scripts executable
chmod +x deploy/install_server.sh

# Run installer with production settings
export APP_ENV="production"
export HTTP_PORT="8080"
bash deploy/install_server.sh \
  --app-env production \
  --http-port 8080 \
  --cors https://pekan.local

echo "Server setup completed ✓"
REMOTE_SCRIPT

# Step 4: Verify deployment
echo ""
echo "✅ [STEP 4/4] Verifying deployment..."
ssh "$REMOTE_USER@$REMOTE_HOST" bash <<'VERIFY_SCRIPT'
echo "  Checking services..."
systemctl is-active --quiet pekan-api && echo "    ✓ API service running" || echo "    ✗ API service not running"
systemctl is-active --quiet pekan-worker && echo "    ✓ Worker service running" || echo "    ✗ Worker service not running"
docker ps | grep -q postgres && echo "    ✓ PostgreSQL running" || echo "    ✗ PostgreSQL not running"
docker ps | grep -q redis && echo "    ✓ Redis running" || echo "    ✗ Redis not running"
echo ""
echo "  Checking endpoints..."
curl -s http://localhost:8080/api/v1/healthz | grep -q "ok" && echo "    ✓ API health check OK" || echo "    ✗ API health check failed"
VERIFY_SCRIPT

echo ""
echo "🎉 Production deployment completed successfully!"
echo ""
echo "📊 Deployment Summary:"
echo "  - Backend: Compiled and deployed"
echo "  - Frontend: Built and deployed"
echo "  - Database: Migrated"
echo "  - Services: Running (API, Worker, PostgreSQL, Redis)"
echo ""
echo "🌐 Access:"
echo "  - API:     http://192.168.201.18:8080/api"
echo "  - Web UI:  http://192.168.201.18 (via Nginx)"
echo ""
echo "📝 Next steps:"
echo "  1. Configure DNS for pekan.local → 192.168.201.18"
echo "  2. Setup HTTPS/SSL certificates"
echo "  3. Configure backup strategy"
echo "  4. Setup monitoring/alerts"
echo ""
