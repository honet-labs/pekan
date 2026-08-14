# Panduan Instalasi PEKAN

Panduan lengkap installasi PEKAN dari nol, baik untuk **development** (lokal) maupun **produksi** (server).

---

## Daftar Isi

- [Prasyarat](#prasyarat)
- [1. Clone Repository](#1-clone-repository)
- [2. Setup Database](#2-setup-database)
- [3. Setup Backend](#3-setup-backend)
- [4. Setup Frontend](#4-setup-frontend)
- [5. Jalankan Aplikasi](#5-jalankan-aplikasi)
- [6. Verifikasi](#6-verifikasi)
- [7. Login Pertama](#7-login-pertama)
- [8. Menjalankan Test](#8-menjalankan-test)
- [Troubleshooting](#troubleshooting)
- [Production Deployment](#production-deployment)

---

## Prasyarat

Pastikan komputer Anda sudah terinstall software berikut:

| Software | Versi Minimal | Cara Cek | Untuk Apa |
| :--- | :--- | :--- | :--- |
| **Git** | 2.0+ | `git --version` | Clone repository |
| **Go** | 1.23+ | `go version` | Backend |
| **Node.js** | 20+ LTS | `node --version` | Frontend build |
| **npm** | 10+ | `npm --version` | Install dependencies frontend |
| **PostgreSQL** | 16+ | `psql --version` | Database |
| **Redis** | 7+ (opsional) | `redis-cli --version` | Rate limiting & session |
| **Docker** | 24+ (opsional) | `docker --version` | Jalankan PostgreSQL & Redis |

### Install Prasyarat (Linux/Ubuntu)

```bash
# Update system
sudo apt update && sudo apt upgrade -y

# Install Git
sudo apt install -y git

# Install Go 1.23+
wget https://go.dev/dl/go1.23.8.linux-amd64.tar.gz
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf go1.23.8.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
echo 'export GOPATH=$HOME/go' >> ~/.bashrc
echo 'export PATH=$PATH:$GOPATH/bin' >> ~/.bashrc
source ~/.bashrc

# Install Node.js 20 LTS
curl -fsSL https://deb.nodesource.com/setup_20.x | sudo -E bash -
sudo apt install -y nodejs

# Install Docker (opsional, untuk PostgreSQL & Redis)
sudo apt install -y docker.io docker-compose-plugin
sudo usermod -aG docker $USER
# Logout & login lagi agar group aktif
```

### Install Prasyarat (macOS)

```bash
# Install Homebrew (jika belum ada)
/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"

# Install semua dependencies
brew install git go node docker docker-compose
```

### Install Prasyarat (Windows)

1. **Git**: Download dari https://git-scm.com/download/win
2. **Go**: Download dari https://go.dev/dl/ (pilih `.msi` installer)
3. **Node.js**: Download dari https://nodejs.org (pilih LTS)
4. **Docker Desktop**: Download dari https://www.docker.com/products/docker-desktop

> **Tip**: Gunakan **PowerShell** atau **Windows Terminal** untuk menjalankan perintah.

---

## 1. Clone Repository

```bash
# Clone repository ke direktori pilihan Anda
git clone https://github.com/honet-labs/pekan.git

# Masuk ke direktori project
cd pekan

# Cek apakah clone berhasil (harus ada file backend/, frontend/, dll)
ls
```

Setelah clone, struktur direktori Anda akan seperti ini:

```
pekan/
├── backend/        # Backend Go
├── frontend/       # Frontend React
├── deploy/         # Script deployment
├── docs/           # Dokumentasi
├── README.md
└── INSTALL.md      # (dokumentasi ini)
```

---

## 2. Setup Database

### Opsi A: Docker (Recommended)

Cara ini paling mudah dan cepat. Pastikan Docker sudah terinstall.

```bash
# Jalankan PostgreSQL dan Redis via Docker
docker compose -f deploy/docker-compose.server-test.yml up -d

# Tunggu beberapa detik hingga container ready
# Cek status container
docker ps
```

Harus muncul 2 container yang running:
- `pekan-postgres` (PostgreSQL 16)
- `pekan-redis` (Redis 7)

```bash
# Verifikasi PostgreSQL bisa diakses
docker exec pekan-postgres pg_isready -U postgres

# Verifikasi Redis bisa diakses
docker exec pekan-redis redis-cli ping
# Harus return "PONG"
```

### Opsi B: PostgreSQL Native

Jika sudah install PostgreSQL di komputer:

```bash
# Buat database pekan
# Linux/macOS:
sudo -u postgres createdb pekan

# Atau via psql:
psql -U postgres -c "CREATE DATABASE pekan;"

# Pastikan password user postgres sudah di-set
psql -U postgres -c "ALTER USER postgres PASSWORD 'postgres';"
```

### Opsi C: Docker Individual

```bash
# PostgreSQL saja
docker run -d --name pekan-postgres \
  -e POSTGRES_DB=pekan \
  -e POSTGRES_USER=postgres \
  -e POSTGRES_PASSWORD=postgres \
  -p 5432:5432 \
  postgres:16-alpine

# Redis saja
docker run -d --name pekan-redis \
  -p 6379:6379 \
  redis:7-alpine
```

---

## 3. Setup Backend

```bash
# Masuk ke direktori backend
cd backend

# Copy file .env.example menjadi .env
cp .env.example .env

# Generate JWT secret yang kuat
# Linux/macOS:
JWT_SECRET=$(openssl rand -base64 48)
echo "JWT_SECRET=$JWT_SECRET"

# Windows PowerShell:
# $JWT_SECRET = [Convert]::ToBase64String([System.Security.Cryptography.RandomNumberGenerator]::GetBytes(48))
# echo "JWT_SECRET=$JWT_SECRET"

# Edit file .env dan isi JWT_SECRET dengan hasil generate di atas
# ANDA BISA MENGGUNAKAN EDITOR FAVORIT, CONTOH:
#   nano .env        (Linux/macOS)
#   notepad .env     (Windows)
#   code .env        (VS Code)
```

**Minimal yang harus diubah di `.env`:**

```env
JWT_SECRET=<paste hasil generate di sini>
DATABASE_URL=postgres://postgres:postgres@localhost:5432/pekan?sslmode=disable
```

> **Penting**: `JWT_SECRET` harus minimal 32 karakter dan unik. Jangan gunakan nilai default di production.

```bash
# Install dependencies Go
go mod tidy

# Jalankan migrasi database (membuat semua tabel)
DATABASE_URL="postgres://postgres:postgres@localhost:5432/pekan?sslmode=disable" \
  ./scripts/apply_migrations.sh

# (Opsional) Masukkan data demo untuk testing
DATABASE_URL="postgres://postgres:postgres@localhost:5432/pekan?sslmode=disable" \
  ./scripts/apply_demo_seed.sh
```

### Flags Migrasi Tambahan

```bash
# Jika menggunakan Redis:
RATE_LIMIT_REDIS_URL="redis://localhost:6379/0" ./scripts/apply_migrations.sh

# Jika PostgreSQL di port lain:
DATABASE_URL="postgres://postgres:password@other-host:5433/pekan?sslmode=disable" \
  ./scripts/apply_migrations.sh
```

---

## 4. Setup Frontend

```bash
# Buka terminal baru, masuk ke direktori frontend
cd frontend

# Install dependencies Node.js
npm install

# (Opsional) Copy .env untuk konfigurasi API URL
cp .env.example .env
```

**Isi `.env` frontend** (default sudah benar untuk development):

```env
VITE_API_BASE_URL=http://localhost:8080/api/v1
```

---

## 5. Jalankan Aplikasi

Anda membutuhkan **minimal 2 terminal** (3 jika ingin AI worker).

### Terminal 1 — Backend API Server

```bash
cd backend
go run cmd/api/main.go

# Output yang diharapkan:
# API listening on :8080
```

### Terminal 2 — Frontend Dev Server

```bash
cd frontend
npm run dev

# Output yang diharapkan:
# Local: http://localhost:5173/
```

### Terminal 3 — AI Worker (Opsional, untuk WhatsApp chatbot)

```bash
cd backend
go run cmd/ai/main.go

# Output yang diharapkan:
# [PEKAN-AI] Asynchronous AI Queue Worker Service started
# [PEKAN-AI] Running 4 concurrent worker threads!
```

### Terminal 4 — Background Worker (Opsional, untuk reminder & file scan)

```bash
cd backend
go run cmd/worker/main.go

# Output yang diharapkan:
# file scan worker started
# reminder worker started
```

---

## 6. Verifikasi

Pastikan semua komponen berjalan dengan benar:

```bash
# 1. Cek API health check
curl http://localhost:8080/api/v1/healthz
# Harus return: {"status":"ok"} atau response JSON

# 2. Cek frontend bisa diakses
# Buka browser: http://localhost:5173
# Harus muncul halaman login PEKAN

# 3. (Opsional) Cek AI worker
# Tidak ada endpoint spesifik, cukup cek log di terminal tidak ada error
```

### Via PowerShell (Windows)

```powershell
# Cek API
Invoke-WebRequest -Uri "http://localhost:8080/api/v1/healthz" -UseBasicParsing

# Buka browser
Start-Process "http://localhost:5173"
```

---

## 7. Login Pertama

Buka browser dan akses **http://localhost:5173**

### Jika Menggunakan Data Demo

| Field | Nilai |
| :--- | :--- |
| Tenant Code | `default` |
| Email | `owner@pekan.local` |
| Password | `password` |

### Membuat Akun Baru (Tanpa Demo Seed)

1. Klik **"Register"** atau **"Buat Workspace Baru"**
2. Isi nama workspace (tenant code), email, dan password
3. Cek email untuk kode OTP (jika email terkonfigurasi)
4. Masukkan OTP untuk verifikasi
5. Login dengan akun yang baru dibuat

---

## 8. Menjalankan Test

```bash
# Jalankan semua test backend
cd backend
go test ./tests/... -v

# Jalankan test tertentu
go test ./tests/... -run TestTransaction -v

# Jalankan CI pipeline lokal (backend + frontend checks)
bash local-ci.sh
```

---

## Troubleshooting

### Error: `command not found: go`

```bash
# Pastikan Go terinstall dan ada di PATH
go version

# Jika tidak ditemukan, tambahkan ke PATH:
# Linux:
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
source ~/.bashrc

# Windows: tambahkan C:\Go\bin ke System PATH
```

### Error: `command not found: psql`

```bash
# Install PostgreSQL client tools
# Ubuntu/Debian:
sudo apt install -y postgresql-client

# macOS:
brew install postgresql

# Atau gunakan Docker untuk menjalankan psql:
docker exec -it pekan-postgres psql -U postgres
```

### Error: `connection refused` (database)

```bash
# Cek apakah PostgreSQL running
# Docker:
docker ps | grep postgres

# Jika belum running:
docker compose -f deploy/docker-compose.server-test.yml up -d

# Jika native, cek service:
sudo systemctl status postgresql
```

### Error: `pekan_access_token` cookie not set

```bash
# Pastikan CORS_ALLOWED_ORIGINS di .env backend sudah benar:
# CORS_ALLOWED_ORIGINS=http://localhost:5173

# Restart backend setelah mengubah .env
```

### Error: `401 Unauthorized` setelah login

```bash
# Pastikan JWT_SECRET di .env backend sudah diisi dengan benar
# Jika mengganti JWT_SECRET, semua session lama akan invalid
# Login ulang untuk membuat session baru
```

### Error: `ECONNREFUSED localhost:5432`

```bash
# PostgreSQL belum running atau port berbeda
# Cek container Docker:
docker ps

# Atau cek port PostgreSQL:
# Linux:
sudo netstat -tulpn | grep 5432

# Jika port berbeda, update DATABASE_URL di .env
```

### Frontend error: `Failed to resolve import`

```bash
# Install ulang dependencies
cd frontend
rm -rf node_modules package-lock.json
npm install
```

### Error: `go.sum` mismatch

```bash
# Update dependencies Go
cd backend
go mod tidy
go mod download
```

### Docker permission denied

```bash
# Tambahkan user ke group docker
sudo usermod -aG docker $USER
# Logout dan login lagi
```

---

## Production Deployment

Untuk deployment di server produksi, gunakan script installer otomatis:

```bash
# SSH ke server
ssh user@your-server-ip

# Clone repository
git clone https://github.com/honet-labs/pekan.git
cd pekan

# Jalankan installer (Ubuntu/Debian)
sudo bash deploy/install_server.sh

# Atau (Rocky Linux/AlmaLinux)
sudo bash deploy/install_server_rocky.sh
```

### Flag Installer yang Tersedia

```bash
# Install semua komponen (default)
sudo bash deploy/install_server.sh

# Hanya install database
sudo bash deploy/install_server.sh --only-db

# Custom port dan domain
sudo bash deploy/install_server.sh \
  --app-env production \
  --http-port 8080 \
  --cors https://yourdomain.com \
  --jwt-secret "your-strong-secret-min-32-characters"

# Lewati test saat install
sudo bash deploy/install_server.sh --skip-tests

# Lewati demo seed
sudo bash deploy/install_server.sh --skip-seed
```

### Setelah Install Produksi

```bash
# Cek semua service
systemctl status pekan-api
systemctl status pekan-worker
docker ps

# Cek API
curl https://yourdomain.com/api/v1/healthz

# Lihat logs
journalctl -u pekan-api -f
```

### Backup & Restore

```bash
# Backup database
sudo bash deploy/backup.sh

# Restore database
sudo bash deploy/restore.sh /opt/pekan/backups/pekan_backup_YYYYMMDD.tar.gz

# Update aplikasi
sudo bash deploy/update_app.sh
```

### Spesifikasi Server

| Tier | vCPU | RAM | Storage | Cocok Untuk |
| :--- | :--- | :--- | :--- | :--- |
| **Development** | 1 Core | 2 GB | 20 GB | Testing, solo usage |
| **Production** | 2 Core | 4 GB | 50 GB | Tim kecil (5-20 user) |
| **Enterprise** | 4+ Core | 8 GB+ | 100 GB+ | Organisasi besar |

---

## Dokumentasi Tambahan

| Dokumen | Lokasi | Deskripsi |
| :--- | :--- | :--- |
| Technical Blueprint | `docs/01-TECHNICAL-BLUEPRINT.md` | Arsitektur teknis lengkap |
| Database Schema | `docs/02-DATABASE-SCHEMA.md` | Skema database detail |
| Security Review | `docs/03-SECURITY-ARCHITECTURE-REVIEW.md` | Review keamanan |
| API Design | `docs/06-API-DESIGN.md` | Desain API |
| WhatsApp Integration | `docs/WHATSAPP-WAHA-INTEGRATION.md` | Integrasi WhatsApp |
| Server Installer | `docs/SERVER-INSTALLER.md` | Panduan server installer |

---

**Terakhir diperbarui:** Agustus 2026
