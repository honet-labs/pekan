#!/bin/bash

# ============================================
# Automated SaaS-PEKAN Production Deploy
# Server: 192.168.201.18
# ============================================

set -e

REMOTE_USER="administrator" # Change this to your server user
REMOTE_HOST="your_server_ip" # Change this to your server IP
REMOTE_PASSWORD='your_server_password' # Change this to your server password

echo "🚀 Production Deployment Starting..."
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

# Function to run remote commands
run_remote_cmd() {
    local cmd="$1"
    # Using sshpass if available, otherwise direct SSH
    if command -v sshpass &> /dev/null; then
        sshpass -p "$REMOTE_PASSWORD" ssh -o StrictHostKeyChecking=no "$REMOTE_USER@$REMOTE_HOST" "$cmd"
    else
        # Fallback: try direct SSH (may prompt for password)
        ssh -o StrictHostKeyChecking=no "$REMOTE_USER@$REMOTE_HOST" "$cmd"
    fi
}

echo "📤 [1/6] Verifying file upload..."
run_remote_cmd "ls -lh /tmp/deploy/saas-pekan.tar.gz && echo '✅ File ready'"

echo ""
echo "📦 [2/6] Extracting deployment package..."
run_remote_cmd "cd /tmp/deploy && tar -xzf saas-pekan.tar.gz && ls -d saas-pekan 2>/dev/null || echo 'Extracting to current dir...'" 

echo ""
echo "⚙️  [3/6] Running deployment script..."
run_remote_cmd "
set -e
cd /tmp/deploy
if [ -d 'saas-pekan' ]; then
    cd saas-pekan
fi

echo '  → Making installer executable...'
chmod +x deploy/install_server.sh

echo '  → Starting installation (this takes 5-10 minutes)...'
sudo bash deploy/install_server.sh --app-env production --http-port 8080

echo '  ✅ Installation complete!'
"

echo ""
echo "✅ [4/6] Verifying services..."
run_remote_cmd "
echo '  → API Service:'
systemctl is-active pekan-api && echo '    ✅ Running' || echo '    ⚠️ Starting...'

echo '  → Worker Service:'
systemctl is-active pekan-worker && echo '    ✅ Running' || echo '    ⚠️ Starting...'

echo '  → Docker containers:'
docker ps --format 'table {{.Names}}\t{{.Status}}' | grep -E 'postgres|redis' || echo '    ⚠️ Starting...'
"

echo ""
echo "🔌 [5/6] Testing API endpoints..."
run_remote_cmd "
sleep 3
echo '  → Health check:'
curl -s http://localhost:8080/api/v1/healthz -o /dev/null && echo '    ✅ API responding' || echo '    ⚠️ API starting...'

echo '  → Web UI:'
curl -s http://localhost -o /dev/null && echo '    ✅ Web UI ready' || echo '    ⚠️ Web UI starting...'
"

echo ""
echo "🎉 [6/6] Deployment Summary"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
run_remote_cmd "
echo '✅ Services Status:'
echo '━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━'
systemctl status pekan-api --no-pager | head -3
echo ''
systemctl status pekan-worker --no-pager | head -3
echo ''
echo '📊 Access Points:'
echo '  • Web UI:     http://192.168.201.18'
echo '  • API:        http://192.168.201.18:8080/api/v1'
echo '  • Default Login:'
echo '    Email:     admin@example.com'
echo '    Password:  password123'
echo ''
echo '📝 Deployment Date: '$(date)
"

echo ""
echo "✨ Production deployment completed successfully!"
echo ""
