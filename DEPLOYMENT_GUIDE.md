# Production Deployment Guide
**Target Server:** administrator@192.168.201.18  
**Environment:** Production  
**OS:** Ubuntu/Debian  
**Date:** 2026-04-03

---

## Pre-Deployment Checklist

✅ **Backend:**
- [x] All test compilation errors fixed (13 methods added)
- [x] Go code compiled successfully
- [x] Migrations ready (0022_finance_transaction_items.sql)

✅ **Frontend:**
- [x] React/Vite built successfully 
- [x] 8 UI/UX features implemented
- [x] ECharts dependency added

✅ **Server:**
- [x] SSH access verified (administrator@192.168.201.18)
- [x] Ubuntu 22.04+ detected
- [ ] SSH key setup (if using automated script)

---

## Deployment Steps (Manual Process)

### Step 1: Connect to Server
```powershell
ssh administrator@192.168.201.18
# Enter password when prompted
```

### Step 2: Create Application Directory
```bash
# On server:
sudo mkdir -p /opt/pekan
sudo chown administrator:administrator /opt/pekan
cd /opt/pekan
```

### Step 3: Upload Project (from your local machine)
```powershell
# On local Windows PowerShell:
$remoteUser = "administrator"
$remoteHost = "192.168.201.18"
$localPath = "C:\Users\LT470s\Documents\HOME_DATA\HONET\project\PEKAN"
$remotePath = "/opt/pekan"

# Using SCP (if OpenSSH installed):
scp -r "$localPath\*" "$remoteUser@$remoteHost`:$remotePath"

# OR using WinSCP GUI if you prefer

# Alternative: Use Git clone on server
ssh administrator@192.168.201.18 "cd /opt && git clone <your-repo-url> pekan"
```

### Step 4: Run Deployment Script on Server
```bash
# SSH to server:
ssh administrator@192.168.201.18

# On server - run the setup:
cd /opt/pekan
chmod +x deploy/install_server.sh
sudo bash deploy/install_server.sh \
  --app-env production \
  --http-port 8080 \
  --cors https://pekan.local

# Script will:
# ✓ Install dependencies (curl, psql, git, etc)
# ✓ Install Docker + Docker Compose
# ✓ Install Go 1.21+
# ✓ Install Node.js
# ✓ Generate backend .env file
# ✓ Start PostgreSQL + Redis in Docker
# ✓ Build backend binary
# ✓ Build frontend assets
# ✓ Apply database migrations
# ✓ Install systemd services
# ✓ Start all services
```

### Step 5: Verify Deployment
```bash
# On server - check services:
systemctl status pekan-api
systemctl status pekan-worker
docker ps  # Should show postgres + redis

# Test API health:
curl http://localhost:8080/api/v1/healthz

# Check web UI:
curl http://localhost  # Should return HTML
```

### Step 6: Configure HTTPS (Optional but Recommended)
```bash
# Install Certbot
sudo apt-get install -y certbot python3-certbot-nginx

# Get certificate for pekan.local or your domain
sudo certbot certonly --standalone -d pekan.local

# Update Nginx to use HTTPS
sudo nano /etc/nginx/sites-available/default
# Add SSL configuration
```

---

## Automated Deployment (if SSH key setup)

If you've setup SSH keys on the server, you can run:

```powershell
# From project root on your local machine:
bash DEPLOY_PRODUCTION.sh
```

This will:
- Compile backend locally
- Build frontend locally  
- Upload to server
- Run deployment on server
- Verify all services

---

## Quick Troubleshooting

### Docker permission denied
```bash
sudo usermod -aG docker administrator
# Log out and log back in
```

### Port 8080 already in use
```bash
sudo netstat -tulpn | grep 8080
sudo kill -9 <PID>
# Or change port in install script: --http-port 8081
```

### Database connection failed
```bash
docker logs <container_id>
# Check POSTGRES password in .env
```

### Services not starting
```bash
journalctl -u pekan-api -n 50  # Last 50 log lines
journalctl -u pekan-worker -n 50
```

---

## Post-Deployment

### 1. Setup Backups
```bash
# Add to crontab for daily backups:
sudo crontab -e
# Add: 0 2 * * * /opt/pekan/scripts/backup_db.sh
```

### 2. Configure Monitoring
```bash
# Optional: Setup prometheus + grafana monitoring
docker-compose -f docker-compose.monitoring.yml up -d
```

### 3. Configure Email Notifications
Update `.env`:
```
MAILER_HOST=smtp.gmail.com
MAILER_PORT=587
MAILER_USER=your-email@gmail.com
MAILER_PASSWORD=your-app-password
```

### 4. Test User Login
- Open: http://192.168.201.18
- Default credentials (from demo seed):
  - Email: `admin@example.com`
  - Password: `password123`

### 5. Apply Seed Data (Optional)
```bash
cd /opt/pekan
bash scripts/apply_demo_seed.sh
```

---

## Monitoring Commands

```bash
# Tail API logs
journalctl -u pekan-api -f

# Tail Worker logs
journalctl -u pekan-worker -f

# Monitor database
docker exec pekan-postgres psql -U postgres -d pekan -c "SELECT count(*) FROM finance_transactions;"

# Check Redis
docker exec pekan-redis redis-cli INFO
```

---

## Rollback (if needed)

```bash
# Stop services
systemctl stop pekan-api pekan-worker

# Revert to previous commit
cd /opt/pekan
git checkout <previous-commit>

# Rebuild and restart
go build -o bin/api ./cmd/api
systemctl start pekan-api pekan-worker
```

---

## Resource Requirements

**Minimum:**
- CPU: 2 cores
- RAM: 4GB
- Storage: 20GB

**Recommended (Production):**
- CPU: 4+ cores
- RAM: 8+ GB
- Storage: 100+ GB (with backups)

---

**Created:** 2026-04-03  
**Version:** 1.0  
**Status:** ✅ Ready for deployment
