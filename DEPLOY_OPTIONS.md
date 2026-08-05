# 🚀 INSTANT DEPLOYMENT OPTIONS

## ⚡ Option 1: PowerShell Script (EASIEST)
```powershell
# In PowerShell at project root:
.\INSTANT_DEPLOY.ps1

# This will:
# ✅ Compress project
# ✅ Upload to server
# ✅ Extract and deploy
# ✅ Verify all services
```

**Time:** ~5-10 minutes  
**Requires:** SSH password prompt once per command

---

## 🖥️ Option 2: Manual SSH Commands (STEP-BY-STEP)

### Step 1: SSH to Server
```bash
ssh administrator@192.168.201.18
# Enter password
```

### Step 2: Create & Extract
```bash
cd /tmp
mkdir -p /opt/pekan
echo "Waiting for file upload from your computer..."
# Keep this terminal open, upload in another terminal
```

### Step 3: Upload in NEW Terminal/PowerShell
```powershell
# NEW Terminal window:
cd c:\Users\LT470s\Documents\HOME_DATA\HONET\project\PEKAN
scp -C PEKAN-deploy.tar.gz administrator@192.168.201.18:/tmp/
```

### Step 4: Back to SSH Terminal - Extract & Deploy
```bash
tar -xzf /tmp/PEKAN-deploy.tar.gz -C /opt/pekan/ --strip-components=1
cd /opt/pekan
chmod +x deploy/install_server.sh
sudo bash deploy/install_server.sh --app-env production --http-port 8080
```

---

## 🐳 Option 3: Docker Deploy (Advanced)

If you prefer containerized deployment:

```bash
# On server:
ssh administrator@192.168.201.18

# Clone or extract project
cd /opt && git clone <repo> pekan && cd pekan

# Use docker-compose
docker-compose -f deploy/docker-compose.server-test.yml up -d

# Run migrations
docker exec pekan-api /app/bin/api migrate
```

---

## ✅ Verification After Deploy

```bash
# SSH to server:
ssh administrator@192.168.201.18

# Check services:
systemctl status pekan-api
systemctl status pekan-worker
docker ps

# Test API:
curl http://localhost:8080/api/v1/health

# View logs:
journalctl -u pekan-api -n 20

# Test web UI:
curl http://localhost  # See if Nginx is serving
```

---

## 📊 Expected Output

After successful deployment, you should see:

```
pekan-api     : active (running)
pekan-worker  : active (running) 
postgres      : up
redis         : up

Health Check: {"status":"ok"}
Web UI: 200 OK
```

---

## 🆘 Troubleshooting

### If deployment fails:

1. **Check SSH connection:**
   ```bash
   ssh -v administrator@192.168.201.18  # Verbose SSH
   ```

2. **Check disk space on server:**
   ```bash
   ssh administrator@192.168.201.18 "df -h"
   ```

3. **Check Docker is running:**
   ```bash
   ssh administrator@192.168.201.18 "docker ps"
   ```

4. **View detailed logs:**
   ```bash
   ssh administrator@192.168.201.18 "sudo journalctl -u pekan-api -n 50"
   ```

5. **Redeploy (clean):**
   ```bash
   ssh administrator@192.168.201.18 "
   sudo systemctl stop pekan-api pekan-worker || true
   sudo rm -rf /opt/pekan
   # Then re-run deployment
   "
   ```

---

## 🎯 RECOMMENDED: Use Option 1

**Just run:**
```powershell
cd c:\Users\LT470s\Documents\HOME_DATA\HONET\project\PEKAN
.\INSTANT_DEPLOY.ps1
```

Press Enter through password prompts and watch deployment happen automatically! ✨

---

**Status:** ✅ Ready to deploy  
**Version:** 1.0  
**Last Updated:** 2026-04-03
