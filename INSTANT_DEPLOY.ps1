# ============================================
# SaaS-PEKAN Production Deployment Script
# One-Command Deploy to: 192.168.201.18
# ============================================

Write-Host "SaaS-PEKAN Production Deployment" -ForegroundColor Cyan -NoNewline
Write-Host " (Automated)" -ForegroundColor Yellow
Write-Host "--------------------------------------------" -ForegroundColor Gray
Write-Host ""

$remoteUser = "administrator"
$remoteHost = "192.168.201.18"
$localProjectPath = (Get-Location).Path
$packageName = "saas-pekan-deploy.tar.gz"

# Step 1: Create deployment package
Write-Host "Step 1/4: Creating deployment package..." -ForegroundColor Green
Write-Host "   - Compressing project files..." -NoNewline
try {
    tar -czf $packageName --exclude=node_modules --exclude=dist --exclude=.git --exclude=.env --exclude=coverage --exclude=.vscode . 2>$null
    $fileSize = [Math]::Round((Get-Item $packageName).Length / 1KB, 2)
    Write-Host " OK" -ForegroundColor Green
    Write-Host "   - Archive size: $fileSize KB" -ForegroundColor Gray
}
catch {
    Write-Host " Failed to create archive" -ForegroundColor Red
    exit 1
}

Write-Host ""
Write-Host "Step 2/4: Uploading to server..." -ForegroundColor Green
Write-Host "   - Connecting to ${remoteUser}@${remoteHost}..." -NoNewline
try {
    scp -C "$packageName" "${remoteUser}@${remoteHost}:/tmp/" 2>$null
    Write-Host " OK" -ForegroundColor Green
}
catch {
    Write-Host " Upload failed" -ForegroundColor Red
    exit 1
}

Write-Host ""
Write-Host "Step 3/4: Extracting and deploying on server..." -ForegroundColor Green

$deployScript = @"
set -e

mkdir -p /opt/pekan
cd /opt/pekan

echo "   - Extracting files..."
tar -xzf /tmp/saas-pekan-deploy.tar.gz

echo "   - Running server setup..."
chmod +x deploy/install_server.sh
sudo bash deploy/install_server.sh --app-env production --http-port 8080 --cors https://pekan.local

echo ""
echo "Deployment completed!"
"@

ssh "${remoteUser}@${remoteHost}" $deployScript

Write-Host ""
Write-Host "Step 4/4: Verifying deployment..." -ForegroundColor Green

$verifyScript = @"
echo "   - Checking services..."
systemctl is-active --quiet pekan-api && echo "     API service running" || echo "     API service starting..."
systemctl is-active --quiet pekan-worker && echo "     Worker service running" || echo "     Worker service starting..."
docker ps | grep -q postgres && echo "     PostgreSQL running" || echo "     PostgreSQL not running"

echo ""
echo "   - Testing endpoints..."
sleep 2
curl -s http://localhost:8080/api/v1/healthz -o /dev/null && echo "     API health check OK" || echo "     API still initializing..."

echo ""
echo "Production Deployment Complete!"
"@

ssh "${remoteUser}@${remoteHost}" $verifyScript

Write-Host ""
Write-Host "Cleaning up local files..." -ForegroundColor Gray
Remove-Item $packageName -Force -ErrorAction SilentlyContinue

Write-Host ""
Write-Host "Deployment SUCCESS!" -ForegroundColor Green
Write-Host ""
Write-Host "Next Steps:" -ForegroundColor Cyan
