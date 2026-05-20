# 🚀 Quick Deploy to Production

**Server:** administrator@192.168.201.18  
**Command:** Copy-paste the script below in PowerShell

---

## Option 1: Using Git (Recommended - Fastest)

If your project is in Git repository:

```powershell
# 1. SSH to server and clone
ssh administrator@192.168.201.18

# On server:
cd /tmp
git clone <your-repo-url> saas-pekan
cd saas-pekan

# 2. Run deployment
sudo chmod +x deploy/install_server.sh
sudo bash deploy/install_server.sh --app-env production --http-port 8080

# 3. Done! Services will auto-start
systemctl status pekan-api
```

---

## Option 2: Using SCP/Upload (if no Git access)

### From Windows PowerShell:

```powershell
# 1. Create remote directory
ssh administrator@192.168.201.18 'mkdir -p /tmp/saas-pekan'

# 2. Upload entire project (Windows - using tar):
$projectPath = "C:\Users\LT470s\Documents\HOME_DATA\HONET\project\saas-pekan"
$remoteUser = "administrator"
$remoteHost = "192.168.201.18"

# Create tar archive
Push-Location $projectPath
tar -czf saas-pekan.tar.gz --exclude=node_modules --exclude=dist --exclude=.git --exclude=.env *
Pop-Location

# Upload
scp -C "$projectPath\saas-pekan.tar.gz" "$remoteUser@$remoteHost:/tmp/"

# 3. Extract and deploy on server
ssh -t $remoteUser@$remoteHost "
  cd /tmp
  tar -xzf saas-pekan.tar.gz -C saas-pekan/ --strip-components=1 || tar -xzf saas-pekan.tar.gz
  cd saas-pekan
  sudo chmod +x deploy/install_server.sh
  sudo bash deploy/install_server.sh --app-env production --http-port 8080
"

# 4. Monitor
ssh $remoteUser@$remoteHost 'systemctl status pekan-api && curl http://localhost:8080/api/v1/healthz'
```

---

## Option 3: Manual Quick Deploy

```bash
# 1. SSH to server
ssh administrator@192.168.201.18

# Enter password

# 2. Prepare
mkdir -p /opt/pekan
cd /opt/pekan

# 3. Copy files manually (drag-drop via WinSCP or use rsync from WSL)

# 4. Deploy
chmod +x deploy/install_server.sh
sudo bash deploy/install_server.sh --app-env production --http-port 8080

# 5. Wait for completion (5-10 minutes)
systemctl status pekan-api
curl http://localhost:8080/api/v1/healthz
```

---

## Post-Deploy Verification

```bash
# Connect to server
ssh administrator@192.168.201.18

# Check all services
systemctl status pekan-api pekan-worker
docker ps

# Test endpoints
curl http://localhost:8080/api/v1/healthz
curl http://localhost:8080/  # Web UI

# Check logs
journalctl -u pekan-api -n 20
```

---

## Access Your Deployment

- **Web UI:** http://192.168.201.18
- **API:** http://192.168.201.18:8080/api
- **Default Login:**
  - Email: admin@example.com
  - Password: password123
  - (After applying demo seed)

---

**Choose Option 1, 2, or 3 above and run in your terminal**
