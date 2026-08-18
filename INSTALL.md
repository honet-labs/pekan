# Panduan Instalasi PEKAN

Panduan lengkap installasi PEKAN dari nol, baik untuk **development** (lokal) maupun **produksi** (server).

---

## Daftar Isi

- [Prasyarat](#prasyarat)
- [Instalasi Development (Lokal)](#instalasi-development-lokal)
- [Instalasi Produksi](#instalasi-produksi)
  - [Opsi A: Docker (Recommended)](#opsi-a-docker-recommended)
  - [Opsi B: Systemd (Native)](#opsi-b-systemd-native)
- [Update Versi](#update-versi)
- [Konfigurasi](#konfigurasi)
- [Login Pertama](#login-pertama)
- [Troubleshooting](#troubleshooting)
- [Backup & Restore](#backup--restore)

---

## Prasyarat

### Development (Lokal)

| Software | Versi Minimal | Cara Cek | Untuk Apa |
| :--- | :--- | :--- | :--- |
| **Git** | 2.0+ | `git --version` | Clone repository |
| **Go** | 1.23+ | `go version` | Backend |
| **Node.js** | 20+ LTS | `node --version` | Frontend build |
| **npm** | 10+ | `npm --version` | Install dependencies frontend |
| **PostgreSQL** | 16+ | `psql --version` | Database |
| **Redis** | 7+ (opsional) | `redis-cli --version` | Rate limiting & session |
| **Docker** | 24+ (opsional) | `docker --version` | Jalankan PostgreSQL & Redis |

### Produksi — Docker

| Software | Versi Minimal | Cara Cek |
| :--- | :--- | :--- |
| **Docker** | 24+ | `docker --version` |
| **Docker Compose** | 2.20+ | `docker compose version` |
| **Git** | 2.0+ | `git --version` |

> **Catatan**: Docker Compose plugin sudah termasuk dalam installer. Tidak perlu install Go, Node.js, PostgreSQL, atau Redis secara terpisah — semuanya berjalan di dalam container.

### Produksi — Systemd

| Software | Versi Minimal | Cara Cek |
| :--- | :--- | :--- |
| **Git** | 2.0+ | `git --version` |
| **Go** | 1.23+ | `go version` |
| **Node.js** | 20+ LTS | `node --version` |
| **PostgreSQL** | 16+ | `psql --version` |
| **Redis** | 7+ | `redis-cli --version` |
| **Nginx** | 1.24+ | `nginx -v` |

> **Catatan**: Installer akan menginstall semua dependency secara otomatis.

### Install Prasyarat Manual (Jika Dibutuhkan)

<details>
<summary><strong>Ubuntu/Debian</strong></summary>

```bash
sudo apt update && sudo apt upgrade -y
sudo apt install -y git

# Go 1.23+
wget https://go.dev/dl/go1.23.8.linux-amd64.tar.gz
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf go1.23.8.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
source ~/.bashrc

# Node.js 20 LTS
curl -fsSL https://deb.nodesource.com/setup_20.x | sudo -E bash -
sudo apt install -y nodejs
```
</details>

<details>
<summary><strong>macOS</strong></summary>

```bash
/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
brew install git go node docker docker-compose
```
</details>

<details>
<summary><strong>Windows</strong></summary>

1. **Git**: https://git-scm.com/download/win
2. **Go**: https://go.dev/dl/ (pilih `.msi` installer)
3. **Node.js**: https://nodejs.org (pilih LTS)
4. **Docker Desktop**: https://www.docker.com/products/docker-desktop
</details>

---

## Instalasi Development (Lokal)

Untuk development di komputer lokal. Menggunakan `go run` dan `npm run dev`.

```bash
# 1. Clone repository
git clone https://github.com/honet-labs/pekan.git
cd pekan

# 2. Jalankan PostgreSQL & Redis via Docker
docker compose -f deploy/docker-compose.server-test.yml up -d

# 3. Setup backend
cd backend
cp .env.example .env
# Edit .env — generate JWT_SECRET:
#   openssl rand -base64 48

# 4. Jalankan migrasi database
DATABASE_URL="postgres://postgres:postgres@localhost:5432/pekan?sslmode=disable" \
  ./scripts/apply_migrations.sh

# 5. Jalankan backend (terminal 1)
go run cmd/api/main.go

# 6. Setup & jalankan frontend (terminal 2)
cd ../frontend
npm install
npm run dev
```

Buka `http://localhost:5173` → login dengan:
- **Tenant**: `default`
- **Email**: `owner@pekan.local`
- **Password**: `password`

### Services Tambahan (Opsional)

```bash
# Terminal 3 — Background Worker (reminder & file scan)
cd backend && go run cmd/worker/main.go

# Terminal 4 — AI Queue Worker (WhatsApp chatbot)
cd backend && go run cmd/ai/main.go
```

---

## Instalasi Produksi

### Opsi A: Docker (Recommended)

Cara ini paling mudah dan cepat. Semua komponen berjalan di container Docker.

**Kelebihan:**
- Tidak perlu install Go, Node.js, PostgreSQL, atau Redis secara terpisah
- Isolasi sempurna antar komponen
- Mudah diupdate dan di-backup
- Cocok untuk server dengan resource terbatas

```bash
# 1. SSH ke server
ssh user@your-server-ip

# 2. Clone repository
git clone https://github.com/honet-labs/pekan.git
cd pekan

# 3. Jalankan installer Docker
sudo bash deploy/installer-docker.sh
```

**Opsi installer yang tersedia:**

```bash
# Install dengan default (port 80)
sudo bash deploy/installer-docker.sh

# Custom port
sudo bash deploy/installer-docker.sh --web-port 8080

# Custom install directory
sudo bash deploy/installer-docker.sh --install-dir /var/www/pekan

# Custom PostgreSQL password
sudo bash deploy/installer-docker.sh --postgres-password "your-strong-password"
```

**Setelah install:**

```bash
# Cek status containers
docker compose -f /opt/pekan/docker-compose.yml ps

# Lihat logs
docker compose -f /opt/pekan/docker-compose.yml logs -f pekan-api

# Restart semua
docker compose -f /opt/pekan/docker-compose.yml restart

# Stop semua
docker compose -f /opt/pekan/docker-compose.yml down
```

**Container yang berjalan:**

| Container | Fungsi | Port |
| :--- | :--- | :--- |
| `pekan-postgres` | PostgreSQL 16 | 5432 (internal) |
| `pekan-redis` | Redis 7 | 6379 (internal) |
| `pekan-api` | API Server | 8080 (internal) |
| `pekan-worker` | Background Worker | - |
| `pekan-ai` | AI Queue Worker | - |
| `pekan-web` | Frontend Nginx | 80 (external) |

---

### Opsi B: Systemd (Native)

Cara ini menggunakan binary Go native dan systemd services. Performa lebih baik untuk server dengan resource besar.

**Kelebihan:**
- Performa lebih baik (tidak ada overhead container)
- Akses langsung ke sistem file
- Cocok untuk server production dengan traffic tinggi
- Lebih mudah di-debug dengan journalctl

```bash
# 1. SSH ke server
ssh user@your-server-ip

# 2. Clone repository
git clone https://github.com/honet-labs/pekan.git
cd pekan

# 3. Jalankan installer Systemd
sudo bash deploy/installer-systemd.sh
```

**Opsi installer yang tersedia:**

```bash
# Install dengan default
sudo bash deploy/installer-systemd.sh

# Custom port
sudo bash deploy/installer-systemd.sh --http-port 8080 --web-port 80

# Custom database
sudo bash deploy/installer-systemd.sh --db-host 192.168.1.100 --db-port 5432

# Custom install directory
sudo bash deploy/installer-systemd.sh --install-dir /var/www/pekan
```

**Setelah install:**

```bash
# Cek status services
systemctl status pekan-api
systemctl status pekan-worker
systemctl status pekan-ai

# Lihat logs
journalctl -u pekan-api -f
journalctl -u pekan-worker -f

# Restart service
sudo systemctl restart pekan-api
sudo systemctl restart pekan-worker
sudo systemctl restart pekan-ai
```

**Services yang berjalan:**

| Service | Fungsi |
| :--- | :--- |
| `pekan-api.service` | API Server (Go binary) |
| `pekan-worker.service` | Background Worker (Go binary) |
| `pekan-ai.service` | AI Queue Worker (Go binary) |
| `nginx` | Reverse proxy + Frontend |
| `postgresql` | Database |
| `redis` | Cache & Session |

---

## Perbandingan Opsi Deploy

| Aspek | Docker | Systemd |
| :--- | :--- | :--- |
| **Kemudahan Install** | Sangat Mudah | Mudah |
| **Performa** | Baik | Lebih Baik |
| **Resource Usage** | Lebih tinggi (container overhead) | Lebih rendah |
| **Isolasi** | Sempurna | Biasa |
| **Update** | `docker compose build` | Rebuild binary |
| **Debug** | `docker compose logs` | `journalctl` |
| **Backup** | `docker compose exec pg_dump` | `pg_dump` langsung |
| **Cocok Untuk** | Server kecil-menengah, mudah dipindah | Server besar, traffic tinggi |

---

## Update Versi

Script update mendukung kedua mode deployment (systemd dan docker) secara otomatis.

```bash
cd pekan

# Update ke versi terbaru (auto-detect mode)
sudo bash deploy/update-versi.sh

# Skip backup sebelum update
sudo bash deploy/update-versi.sh --no-backup

# Force mode deployment
sudo bash deploy/update-versi.sh --mode docker
sudo bash deploy/update-versi.sh --mode systemd

# Custom install directory
sudo bash deploy/update-versi.sh --install-dir /var/www/pekan
```

**Apa yang dilakukan update:**
1. Pull kode terbaru dari repository
2. Backup database (kecuali `--no-backup`)
3. Rebuild backend binaries (systemd) atau Docker images (docker)
4. Rebuild frontend
5. Jalankan migrasi database
6. Restart services/containers
7. Verifikasi health check

---

## Konfigurasi

### File Konfigurasi

| File | Lokasi | Fungsi |
| :--- | :--- | :--- |
| Backend `.env` | `/opt/pekan/backend/.env` | Konfigurasi API, database, JWT, dll |
| Docker `.env` | `/opt/pekan/.env` | Password PostgreSQL (docker mode) |
| Nginx | `/etc/nginx/sites-available/pekan` | Konfigurasi reverse proxy (systemd mode) |

### Environment Variables Penting

```env
# Wajib diubah di production
JWT_SECRET=<minimal-32-karakter-random>
RECEIPT_SCAN_SECRET=<minimal-32-karakter-random>
ADMIN_SECRET=<minimal-32-karakter-random>

# Database
DATABASE_URL=postgres://user:pass@host:5432/pekan?sslmode=require

# CORS (isi dengan domain production)
CORS_ALLOWED_ORIGINS=https://pekan.yourdomain.com

# Redis (opsional, untuk rate limiting dan session)
RATE_LIMIT_REDIS_URL=redis://127.0.0.1:6379/0
```

### Konfigurasi SSL (HTTPS)

```bash
# Install Certbot
sudo apt install -y certbot python3-certbot-nginx

# Generate SSL certificate
sudo certbot --nginx -d pekan.yourdomain.com

# Auto-renewal
sudo certbot renew --dry-run
```

---

## Login Pertama

Buka browser dan akses aplikasi Anda.

### Jika Menggunakan Data Demo

| Field | Nilai |
| :--- | :--- |
| Tenant Code | `default` |
| Email | `owner@pekan.local` |
| Password | `password` |

### Membuat Akun Baru (Tanpa Demo Seed)

1. Buka aplikasi di browser
2. Klik **"Register"** atau **"Buat Workspace Baru"**
3. Isi nama workspace (tenant code), email, dan password
4. Cek email untuk kode OTP (jika email terkonfigurasi)
5. Masukkan OTP untuk verifikasi
6. Login dengan akun yang baru dibuat

---

## Troubleshooting

### Docker Mode

```bash
# Cek status containers
docker compose -f /opt/pekan/docker-compose.yml ps

# Lihat logs container tertentu
docker compose -f /opt/pekan/docker-compose.yml logs -f pekan-api
docker compose -f /opt/pekan/docker-compose.yml logs -f pekan-postgres

# Restart container
docker compose -f /opt/pekan/docker-compose.yml restart pekan-api

# Masuk ke container
docker compose -f /opt/pekan/docker-compose.yml exec pekan-api sh

# Cek database
docker compose -f /opt/pekan/docker-compose.yml exec pekan-postgres psql -U postgres pekan
```

### Systemd Mode

```bash
# Cek status service
systemctl status pekan-api
systemctl status pekan-worker
systemctl status pekan-ai

# Lihat logs
journalctl -u pekan-api -f
journalctl -u pekan-api -n 100 --no-pager

# Restart service
sudo systemctl restart pekan-api

# Cek database
sudo -u postgres psql pekan

# Cek Redis
redis-cli ping

# Cek Nginx
sudo nginx -t
sudo systemctl status nginx
```

### Error Umum

| Error | Penyebab | Solusi |
| :--- | :--- | :--- |
| `connection refused` | Database belum running | Cek PostgreSQL: `systemctl status postgresql` atau `docker ps` |
| `401 Unauthorized` | JWT_SECRET berubah | Login ulang untuk session baru |
| `CORS error` | CORS_ALLOWED_ORIGINS salah | Update `.env` dan restart API |
| `502 Bad Gateway` | Nginx tidak bisa ke API | Cek API running: `curl localhost:8080/api/v1/healthz` |
| `permission denied` | File permission salah | `sudo chown -R pekan:pekan /opt/pekan` |

---

## Backup & Restore

### Docker Mode

```bash
# Backup manual
docker compose -f /opt/pekan/docker-compose.yml exec pekan-postgres \
  pg_dump -U postgres pekan | gzip > backup_$(date +%Y%m%d).sql.gz

# Restore
gunzip < backup_20260819.sql.gz | \
  docker compose -f /opt/pekan/docker-compose.yml exec -T pekan-postgres \
  psql -U postgres pekan
```

### Systemd Mode

```bash
# Backup manual
pg_dump -U postgres pekan | gzip > backup_$(date +%Y%m%d).sql.gz

# Restore
gunzip < backup_20260819.sql.gz | psql -U postgres pekan
```

### Backup Otomatis

Docker mode sudah mengkonfigurasi backup otomatis setiap jam 02:00. Backup tersimpan di `/opt/pekan/backups/`.

Untuk systemd mode, tambahkan cron:

```bash
# Edit crontab
crontab -e

# Tambahkan baris berikut (backup setiap jam 02:00)
0 2 * * * pg_dump -U postgres pekan | gzip > /opt/pekan/backups/pekan_$(date +\%Y\%m\%d).sql.gz
```

---

## Spesifikasi Server

| Tier | vCPU | RAM | Storage | Cocok Untuk |
| :--- | :--- | :--- | :--- | :--- |
| **Development** | 1 Core | 2 GB | 20 GB | Testing, solo usage |
| **Production** | 2 Core | 4 GB | 50 GB | Tim kecil (5-20 user) |
| **Enterprise** | 4+ Core | 8 GB+ | 100 GB+ | Organisasi besar |

> **Rekomendasi**: Gunakan **Docker** untuk server dengan RAM < 4 GB. Gunakan **Systemd** untuk server dengan RAM >= 4 GB dan traffic tinggi.

---

## Dokumentasi Tambahan

| Dokumen | Lokasi | Deskripsi |
| :--- | :--- | :--- |
| Technical Blueprint | `docs/01-TECHNICAL-BLUEPRINT.md` | Arsitektur teknis lengkap |
| Database Schema | `docs/02-DATABASE-SCHEMA.md` | Skema database detail |
| Security Review | `docs/03-SECURITY-ARCHITECTURE-REVIEW.md` | Review keamanan |
| API Design | `docs/06-API-DESIGN.md` | Desain API |
| WhatsApp Integration | `docs/WHATSAPP-WAHA-INTEGRATION.md` | Integrasi WhatsApp |

---

**Terakhir diperbarui:** Agustus 2026
