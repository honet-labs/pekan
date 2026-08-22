# Panduan Instalasi PEKAN

Panduan lengkap instalasi PEKAN dari nol, baik untuk **development** (lokal) maupun **produksi** (server).

---

## Daftar Isi

- [Prasyarat](#prasyarat)
- [Instalasi Development (Lokal)](#instalasi-development-lokal)
- [Instalasi Produksi](#instalasi-produksi)
  - [Opsi A: Docker](#opsi-a-docker)
  - [Opsi B: Systemd](#opsi-b-systemd)
- [Pilihan Branch](#pilihan-branch)
- [Update Versi](#update-versi)
- [Konfigurasi](#konfigurasi)
- [Login Pertama](#login-pertama)
- [Troubleshooting](#troubleshooting)
- [Backup & Restore](#backup--restore)

---

## Prasyarat

### Development (Lokal)

| Software | Versi | Cara Cek |
| :--- | :--- | :--- |
| Git | 2.0+ | `git --version` |
| Go | 1.23+ | `go version` |
| Node.js | 20+ | `node --version` |
| npm | 10+ | `npm --version` |
| Docker | 24+ | `docker --version` |

### Produksi - Docker

| Software | Versi | Cara Cek |
| :--- | :--- | :--- |
| Docker | 24+ | `docker --version` |
| Docker Compose | 2.20+ | `docker compose version` |
| Git | 2.0+ | `git --version` |

> Tidak perlu install Go, Node.js, PostgreSQL, atau Redis secara terpisah.

### Produksi - Systemd

| Software | Versi | Cara Cek |
| :--- | :--- | :--- |
| Git | 2.0+ | `git --version` |

> Semua dependency (Go, Node.js, PostgreSQL, Redis, Nginx) akan diinstall otomatis oleh installer.

---

## Instalasi Development (Lokal)

```bash
# 1. Clone repository
git clone https://github.com/honet-labs/pekan.git
cd pekan

# 2. Jalankan PostgreSQL & Redis via Docker
docker compose -f deploy/docker-compose.server-test.yml up -d

# 3. Setup backend
cd backend
cp .env.example .env
# Edit .env, generate JWT_SECRET:
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

Buka `http://localhost:5173`

Login:
- Tenant: `default`
- Email: `owner@pekan.local`
- Password: `password`

---

## Instalasi Produksi

### Opsi A: Docker

Semua komponen berjalan di container Docker.

**Langkah 1: Clone repository**

```bash
git clone https://github.com/honet-labs/pekan.git
cd pekan
```

**Langkah 2: Jalankan installer**

```bash
# Install dari branch main (default, stabil)
sudo bash deploy/installer-docker.sh

# Atau install dari branch dev (fitur terbaru)
sudo bash deploy/installer-docker.sh --branch dev
```

**Langkah 3: Verifikasi**

```bash
# Cek status containers
docker compose -f /opt/pekan/docker-compose.yml ps

# Cek health
curl http://localhost:8080/api/v1/healthz

# Buka browser
# http://localhost
```

**Langkah 4: Login**

Buka `http://localhost` dan login dengan credentials yang muncul di akhir instalasi.

**Opsi installer yang tersedia:**

```bash
sudo bash deploy/installer-docker.sh [options]

Options:
  --branch <name>           Branch yang diinstall (default: main)
  --install-dir <path>      Direktori instalasi (default: /opt/pekan)
  --web-port <port>         Port web (default: 80)
  --postgres-pass <pass>    Password PostgreSQL (auto-generate jika kosong)
  --skip-build              Skip build Docker image
  --help                    Tampilkan bantuan
```

**Container yang berjalan:**

| Container | Fungsi | Port |
| :--- | :--- | :--- |
| pekan-postgres | PostgreSQL 16 | 5432 (internal) |
| pekan-redis | Redis 7 | 6379 (internal) |
| pekan-api | API Server | 8080 (internal) |
| pekan-worker | Background Worker | - |
| pekan-ai | AI Queue Worker | - |
| pekan-web | Frontend Nginx | 80 (external) |

**Perintah Docker yang berguna:**

```bash
cd /opt/pekan

# Status containers
docker compose ps

# Logs
docker compose logs -f pekan-api
docker compose logs -f pekan-worker
docker compose logs -f pekan-ai

# Restart
docker compose restart pekan-api

# Stop semua
docker compose down

# Start semua
docker compose up -d

# Rebuild dan restart
docker compose up -d --build
```

---

### Opsi B: Systemd

Binary Go native dengan systemd services. Performa lebih baik.

**Langkah 1: Clone repository**

```bash
git clone https://github.com/honet-labs/pekan.git
cd pekan
```

**Langkah 2: Jalankan installer**

```bash
# Install dari branch main (default, stabil)
sudo bash deploy/installer-systemd.sh

# Atau install dari branch dev (fitur terbaru)
sudo bash deploy/installer-systemd.sh --branch dev
```

**Langkah 3: Verifikasi**

```bash
# Cek status services
systemctl status pekan-api
systemctl status pekan-worker
systemctl status pekan-ai

# Cek health
curl http://localhost:8080/api/v1/healthz

# Buka browser
# http://localhost
```

**Langkah 4: Login**

Buka `http://localhost` dan login dengan credentials yang muncul di akhir instalasi.

**Opsi installer yang tersedia:**

```bash
sudo bash deploy/installer-systemd.sh [options]

Options:
  --branch <name>         Branch yang diinstall (default: main)
  --install-dir <path>    Direktori instalasi (default: /opt/pekan)
  --http-port <port>      Port API server (default: 8080)
  --web-port <port>       Port Nginx (default: 80)
  --db-pass <password>    Password PostgreSQL (auto-generate jika kosong)
  --jwt-secret <secret>   JWT secret (auto-generate jika kosong)
  --skip-deps             Skip install dependency
  --skip-migrate          Skip migrasi database
  --help                  Tampilkan bantuan
```

**Services yang berjalan:**

| Service | Fungsi |
| :--- | :--- |
| pekan-api.service | API Server |
| pekan-worker.service | Background Worker |
| pekan-ai.service | AI Queue Worker |
| nginx | Reverse Proxy + Frontend |
| postgresql | Database |
| redis | Cache & Session |

**Perintah Systemd yang berguna:**

```bash
# Status
systemctl status pekan-api
systemctl status pekan-worker
systemctl status pekan-ai

# Logs
journalctl -u pekan-api -f
journalctl -u pekan-worker -f
journalctl -u pekan-ai -f
journalctl -u pekan-api -n 100

# Restart
sudo systemctl restart pekan-api
sudo systemctl restart pekan-worker
sudo systemctl restart pekan-ai

# Stop
sudo systemctl stop pekan-api

# Disable (prevent auto-start)
sudo systemctl disable pekan-api
```

---

## Pilihan Branch

PEKAN memiliki 2 branch utama:

| Branch | Deskripsi | Kapan Digunakan |
| :--- | :--- | :--- |
| `main` | Branch stabil, sudah di-test | Produksi, penggunaan harian |
| `dev` | Branch development, fitur terbaru | Testing, kontribusi development |

**Contoh penggunaan:**

```bash
# Install versi stabil (produksi)
sudo bash deploy/installer-docker.sh --branch main

# Install versi development (testing)
sudo bash deploy/installer-docker.sh --branch dev

# Update ke branch dev
sudo bash deploy/update-versi.sh --branch dev

# Kembali ke branch main
sudo bash deploy/update-versi.sh --branch main
```

---

## Update Versi

Script update mendukung kedua mode deployment (systemd dan docker) secara otomatis.

**Langkah 1: Masuk ke direktori repository**

```bash
cd /opt/pekan
# Atau dimana saja repository PEKAN di-clone
```

**Langkah 2: Jalankan update**

```bash
# Update branch saat ini (auto-detect mode)
sudo bash deploy/update-versi.sh

# Update ke branch tertentu
sudo bash deploy/update-versi.sh --branch dev
sudo bash deploy/update-versi.sh --branch main

# Update tanpa backup
sudo bash deploy/update-versi.sh --no-backup

# Force mode deployment
sudo bash deploy/update-versi.sh --mode docker
sudo bash deploy/update-versi.sh --mode systemd
```

**Apa yang dilakukan update:**

1. Fetch kode terbaru dari repository
2. Switch branch (jika --branch digunakan)
3. Backup database (kecuali --no-backup)
4. Sync kode ke direktori instalasi
5. Rebuild backend binaries (systemd) atau Docker images (docker)
6. Rebuild frontend
7. Jalankan migrasi database
8. Restart services/containers
9. Verifikasi health check

---

## Konfigurasi

### File Konfigurasi

| File | Lokasi | Fungsi |
| :--- | :--- | :--- |
| Backend .env | `/opt/pekan/backend/.env` | Konfigurasi API, database, JWT |
| Docker .env | `/opt/pekan/.env` | Password PostgreSQL (docker mode) |
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

# Redis (opsional)
RATE_LIMIT_REDIS_URL=redis://127.0.0.1:6379/0
```

### Konfigurasi SSL (HTTPS)

```bash
# Install Certbot
sudo apt install -y certbot python3-certbot-nginx

# Generate SSL certificate
sudo certbot --nginx -d pekan.yourdomain.com

# Auto-renewal test
sudo certbot renew --dry-run
```

---

## Login Pertama

Buka browser dan akses aplikasi.

### Jika Menggunakan Data Demo

| Field | Nilai |
| :--- | :--- |
| Tenant Code | `default` |
| Email | `owner@pekan.local` |
| Password | `password` |

### Membuat Akun Baru

1. Buka aplikasi di browser
2. Klik "Register" atau "Buat Workspace Baru"
3. Isi nama workspace, email, dan password
4. Cek email untuk kode OTP
5. Masukkan OTP untuk verifikasi
6. Login dengan akun yang baru dibuat

---

## Troubleshooting

### Docker Mode

```bash
# Cek status containers
docker compose -f /opt/pekan/docker-compose.yml ps

# Lihat logs
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
| `connection refused` | Database belum running | Cek PostgreSQL |
| `401 Unauthorized` | JWT_SECRET berubah | Login ulang |
| `CORS error` | CORS_ALLOWED_ORIGINS salah | Update .env, restart API |
| `502 Bad Gateway` | API tidak running | Cek: `curl localhost:8080/api/v1/healthz` |
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

Docker mode sudah mengkonfigurasi backup otomatis setiap jam 02:00.

---

## Spesifikasi Server

| Tier | vCPU | RAM | Storage | Cocok Untuk |
| :--- | :--- | :--- | :--- | :--- |
| Development | 1 Core | 2 GB | 20 GB | Testing, solo |
| Production | 2 Core | 4 GB | 50 GB | Tim kecil (5-20 user) |
| Enterprise | 4+ Core | 8 GB+ | 100 GB+ | Organisasi besar |

Rekomendasi: Gunakan **Docker** untuk RAM < 4 GB. Gunakan **Systemd** untuk RAM >= 4 GB.

---

## Dokumentasi Tambahan

| Dokumen | Lokasi |
| :--- | :--- |
| Technical Blueprint | `docs/01-TECHNICAL-BLUEPRINT.md` |
| Database Schema | `docs/02-DATABASE-SCHEMA.md` |
| Security Review | `docs/03-SECURITY-ARCHITECTURE-REVIEW.md` |
| API Design | `docs/06-API-DESIGN.md` |
| Admin API | `docs/ADMIN-API.md` |
| WhatsApp Integration | `docs/WHATSAPP-WAHA-INTEGRATION.md` |

---

**Terakhir diperbarui:** Agustus 2026
