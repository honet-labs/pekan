<h1 align="center">PEKAN</h1>

<p align="center">
  <strong>Platform Pencatatan Keuangan Multi-Tenant</strong>
</p>

<p align="center">
  <a href="https://github.com/honet-labs/pekan/blob/main/LICENSE">
    <img src="https://img.shields.io/badge/License-MIT-yellow.svg?style=flat-square" alt="License">
  </a>
  <img src="https://img.shields.io/badge/Go-1.23-00ADD8?style=flat-square&logo=go" alt="Go">
  <img src="https://img.shields.io/badge/React-18-61DAFB?style=flat-square&logo=react" alt="React">
  <img src="https://img.shields.io/badge/PostgreSQL-16-4169E1?style=flat-square&logo=postgresql" alt="PostgreSQL">
  <img src="https://img.shields.io/badge/Redis-7-DC382D?style=flat-square&logo=redis" alt="Redis">
</p>

<p align="center">
  <a href="#fitur">Fitur</a> •
  <a href="#instalasi-cepat">Instalasi</a> •
  <a href="#arsitektur">Arsitektur</a> •
  <a href="#tech-stack">Tech Stack</a> •
  <a href="#kontribusi">Kontribusi</a> •
  <a href="#kontak">Kontak</a>
</p>

---

PEKAN adalah platform pencatatan keuangan multi-tenant yang dibangun untuk membantu tim dan organisasi mengelola keuangan secara terpusat. Dari pencatatan transaksi harian, anggaran bulanan, tabungan, hingga pengingat tagihan — semuanya bisa diakses melalui web maupun WhatsApp.

Yang membedakan PEKAN: **asisten AI** yang bisa mencatat transaksi langsung dari chat WhatsApp, memindai struk belanja secara otomatis, dan memberikan insight finansial berdasarkan data aktual.

---

## Daftar Isi

- [Fitur](#fitur)
- [Arsitektur](#arsitektur)
- [Tech Stack](#tech-stack)
- [Struktur Proyek](#struktur-proyek)
- [Database](#database)
- [Instalasi](#instalasi)
- [Deployment Produksi](#deployment-produksi)
- [Konfigurasi](#konfigurasi)
- [API Endpoints](#api-endpoints)
- [WhatsApp Integration](#whatsapp-integration)
- [Kontribusi](#kontribusi)
- [Lisensi](#lisensi)
- [Kontak](#kontak)

---

## Fitur

### Pencatatan Keuangan

| Modul | Apa yang Dilakukan |
| :--- | :--- |
| **Transaksi** | Catat pemasukan, pengeluaran, transfer, dan tabungan. Mendukung line items, pajak, diskon, metode pembayaran, dan merchant. |
| **Tabungan** | Atur target tabungan dengan deadline dan pantau progresnya secara real-time. |
| **Anggaran** | Tetapkan batas pengeluaran per kategori per periode. Sistem otomatis memberi tahu saat anggaran mulai menipis. |
| **Pengingat** | Pengingat tagihan berkala (harian/mingguan/bulanan/tahunan) dengan sistem cicilan dan riwayat pembayaran. |
| **Laporan** | Generate laporan keuangan dan ekspor ke PDF/CSV/Excel. |
| **Lampiran** | Upload foto/bukti transaksi dengan virus scanning otomatis. |
| **Dashboard** | Visualisasi arus kas, tren pengeluaran, dan perbandingan anggaran via chart interaktif. |

### Kecerdasan Buatan

| Fitur | Penjelasan |
| :--- | :--- |
| **Chatbot WhatsApp** | Ketik "beli kopi 15rb" di WhatsApp, AI otomatis mencatat transaksi. Bisa juga cek saldo, anggaran, atau hapus transaksi — semua lewat chat. |
| **OCR Struk** | Foto struk belanja → AI ekstrak merchant, item, harga, pajak → langsung jadi transaksi. |
| **Web Chat** | Asisten AI yang bisa diakses langsung dari browser, fokus menjawab pertanyaan seputar data keuangan pengguna. |

### Multi-Tenant & Keamanan

| Fitur | Penjelasan |
| :--- | :--- |
| **Schema-per-Tenant** | Setiap workspace punya schema database sendiri — data benar-benar terisolasi. |
| **RBAC** | Role-Based Access Control dengan permission granular per modul. |
| **JWT + Refresh Rotation** | Autentikasi dengan access token (15 menit) + refresh token (30 hari) yang dirotasi otomatis. |
| **Audit Log** | Setiap aksi (create, update, delete) tercatat lengkap dengan siapa, kapan, dan perubahan apa yang dilakukan. |
| **Rate Limiting** | Proteksi dari brute-force berbasis Redis atau in-memory. |
| **File Scanning** | Antrian pemindaian virus untuk setiap file upload. |

### Notifikasi

Email (SMTP), Telegram, dan WhatsApp — bisa dikonfigurasi per workspace.

### Panel Admin

Monitoring global: statistik workspace, user, transaksi, database health, konfigurasi AI, dan tools untuk SQL dump backup.

### Tambahan

- **i18n**: Bahasa Indonesia & English
- **Responsive**: Desktop, tablet, mobile
- **Dark Mode**: Mendukung tema gelap via CSS Custom Properties

---

## Arsitektur

### High-Level

```
                          ┌─────────────────────┐
                          │       Clients        │
                          │                       │
                          │  Browser  WhatsApp    │
                          │  (React)  (WAHA GW)  │
                          └───────────┬───────────┘
                                      │
                                      ▼
                          ┌─────────────────────┐
                          │   Nginx (port 80)    │
                          │   SSL + Static Files  │
                          │   + Reverse Proxy     │
                          └──────┬───────────────┘
                                 │
                    ┌────────────┼────────────────┐
                    ▼                             ▼
          ┌──────────────────┐         ┌──────────────────┐
          │  Frontend (SPA)  │         │  Backend (Go)    │
          │  React + Vite    │         │                  │
          │  port 5173 (dev) │         │  ┌────────────┐  │
          │                  │         │  │ REST API   │──┼──▶ PostgreSQL
          │                  │         │  │ port 8080  │  │    (port 5432)
          └──────────────────┘         │  └────────────┘  │
                                       │  ┌────────────┐  │
                                       │  │ AI Worker  │──┼──▶ Redis
                                       │  │ (queue)    │  │    (port 6379)
                                       │  └────────────┘  │
                                       └──────────────────┘
```

### Backend Architecture (per module)

PEKAN mengikuti prinsip **Clean Architecture** — setiap modul punya 4 layer yang terpisah:

```
┌─────────────────────────────────────────────────┐
│              Delivery (HTTP Handler)              │
│   Menerima request, validasi input, response      │
├─────────────────────────────────────────────────┤
│              Use Case (Business Logic)            │
│   Aturan bisnis, orkestrasi, autorisasi           │
├─────────────────────────────────────────────────┤
│              Infrastructure (Repository)          │
│   Query database, integrasi external service      │
├─────────────────────────────────────────────────┤
│              Domain (Entity & Error)              │
│   Model data murni, kontrak interface             │
└─────────────────────────────────────────────────┘
```

### Middleware Pipeline

```
Request
  │
  ▼
Logger → Recovery → CORS → Rate Limiter → Request ID
  │
  ▼
JWT Auth → Tenant Context → RBAC Check → Handler
```

### Request Lifecycle

```
1. Client kirim request
2. Nginx terima → proxy ke Go API (:8080)
3. Middleware pipeline proses (auth, tenant, rate limit)
4. Router cocokkan ke handler yang tepat
5. Handler validasi input → panggil use case
6. Use case proses bisnis → panggil repository
7. Repository query database → return data
8. Response JSON dikirim kembali ke client
```

---

## Tech Stack

### Backend

| Komponen | Teknologi | Keterangan |
| :--- | :--- | :--- |
| Bahasa | **Go 1.23+** | Performa tinggi, compiled binary |
| Router | **go-chi/chi/v5** | Lightweight, stdlib-compatible |
| Database | **PostgreSQL 16** | Schema-per-tenant untuk isolasi data |
| Cache | **Redis 7** | Rate limiting & session cache |
| Auth | **JWT (golang-jwt)** | Access + refresh token rotation |
| Password | **Argon2id** (primary), **bcrypt** (fallback) | Hashing modern & aman |
| PDF | **phpdave11/gofpdf** | Generate laporan PDF |
| Storage | **Local FS / S3 / Google Drive** | Pluggable file storage |

### Frontend

| Komponen | Teknologi | Keterangan |
| :--- | :--- | :--- |
| UI | **React 18 + TypeScript** | Type-safe, component-based |
| Build | **Vite 5** | HMR cepat, bundling efisien |
| Routing | **React Router DOM v6** | Client-side routing |
| Charts | **Apache ECharts** | Dashboard visualisasi |
| Styling | **Vanilla CSS** | Custom properties, glassmorphism |
| State | **useSyncExternalStore** | Tanpa library eksternal |
| HTTP | **Native fetch** | Tanpa axios |
| i18n | **Custom Context** | ID/EN |

### Infrastructure

| Komponen | Teknologi | Keterangan |
| :--- | :--- | :--- |
| Database | **PostgreSQL 16** | Multi-schema tenant isolation |
| Cache | **Redis 7** (opsional) | Bisa fallback ke in-memory |
| Web Server | **Nginx** | Reverse proxy + static files |
| Process | **Systemd** | Service management di Linux |
| WhatsApp | **WAHA** (WhatsApp HTTP API) | Gateway untuk chat bot |
| AI | **Gemini / OpenAI / Anthropic** | OCR & chatbot AI |
| OS | **Ubuntu/Debian**, **Rocky Linux** | Dukungan distro populer |

---

## Struktur Proyek

```
pekan/
│
├── backend/                          # ─── Backend (Go) ───
│   ├── cmd/
│   │   ├── api/main.go               # Entry point: REST API server
│   │   ├── worker/main.go            # Entry point: background worker
│   │   └── ai/main.go                # Entry point: AI queue worker
│   │
│   ├── internal/
│   │   ├── app/server.go             # Bootstrap & wiring semua komponen
│   │   │
│   │   ├── modules/                  # ── Business Modules ──
│   │   │   ├── core/
│   │   │   │   ├── auth/             # Login, register, session, OTP
│   │   │   │   ├── admin/            # Super admin panel
│   │   │   │   └── subscription/     # Manajemen langganan
│   │   │   │
│   │   │   └── finance/
│   │   │       ├── transactions/     # CRUD transaksi + items
│   │   │       ├── savings/          # Target tabungan
│   │   │       ├── budgets/          # Anggaran per kategori
│   │   │       ├── reminders/        # Pengingat + cicilan
│   │   │       ├── reports/          # Ekspor PDF/CSV/Excel
│   │   │       ├── receipts/         # OCR receipt scanner
│   │   │       ├── attachments/      # File upload + scan
│   │   │       ├── dashboard/        # Statistik & chart
│   │   │       ├── master/           # Akun & kategori
│   │   │       ├── notifications/    # In-app notifikasi
│   │   │       ├── settings/         # Roles, channels, templates
│   │   │       └── whatsapp/         # Chat bot AI + queue
│   │   │
│   │   └── platform/                 # ── Shared Infrastructure ──
│   │       ├── access/               # RBAC engine
│   │       ├── auth/                 # JWT management
│   │       ├── config/               # Environment & global settings
│   │       ├── db/                   # DB connection + tenant transaction
│   │       ├── middleware/           # Logger, CORS, rate limit, auth
│   │       ├── security/             # Encryption (AES-256-GCM)
│   │       ├── storage/              # File storage (Local/S3/GDrive)
│   │       ├── tenancy/              # Tenant context propagation
│   │       ├── audit/                # Audit log writer
│   │       ├── session/              # Session store
│   │       ├── notification/         # Multi-channel notification driver
│   │       └── httpx/                # HTTP helpers & error mapper
│   │
│   ├── migrations/                   # SQL DDL (0001–0042+)
│   ├── scripts/                      # Migration & seed scripts
│   ├── tests/                        # Unit & integration tests
│   ├── openapi/                      # API specification (OpenAPI)
│   └── go.mod
│
├── frontend/                         # ─── Frontend (React) ───
│   └── src/
│       ├── core/                     # Shared: components, auth, i18n, hooks
│       │   ├── api/client.ts         # Central HTTP client
│       │   ├── auth/                 # Auth store & API
│       │   ├── access/               # RBAC permission store
│       │   ├── tenant/               # Tenant state management
│       │   └── components/           # Shared UI components
│       │
│       ├── features/
│       │   ├── core/
│       │   │   ├── auth/             # Login, register, forgot password
│       │   │   ├── admin/            # Admin dashboard
│       │   │   └── profile/          # User profile
│       │   │
│       │   └── finance/
│       │       ├── transactions/     # Transaksi CRUD + items
│       │       ├── savings/          # Tabungan
│       │       ├── budgets/          # Anggaran
│       │       ├── reminders/        # Pengingat
│       │       ├── reports/          # Laporan & ekspor
│       │       ├── receipts/         # OCR scan
│       │       ├── dashboard/        # Dashboard charts
│       │       ├── masterdata/       # Akun & kategori
│       │       ├── notifications/    # Notifikasi
│       │       ├── settings/         # Pengaturan workspace
│       │       ├── attachments/      # Lampiran
│       │       └── chatbot/          # AI chat web UI
│       │
│       └── styles/app.css
│
├── deploy/                           # ─── Deployment ───
│   ├── installer-systemd.sh          # Installer Systemd (native)
│   ├── installer-docker.sh           # Installer Docker (containers)
│   ├── update-versi.sh               # Update versi (auto-detect mode)
│   ├── install_server.sh             # Installer Ubuntu/Debian (legacy)
│   ├── install_server_rocky.sh       # Installer Rocky Linux (legacy)
│   ├── update_app.sh                 # Script update (legacy)
│   ├── backup.sh / restore.sh        # Backup & restore
│   └── systemd/                      # Service unit files
│       ├── pekan-api.service
│       ├── pekan-worker.service
│       └── pekan-ai.service
│
├── docker-compose.yml                # ─── Docker Full Stack ───
│
├── docs/                             # ─── Dokumentasi ───
│   ├── 01-TECHNICAL-BLUEPRINT.md
│   ├── 02-DATABASE-SCHEMA.md
│   ├── 03-SECURITY-ARCHITECTURE-REVIEW.md
│   ├── 06-API-DESIGN.md
│   ├── WHATSAPP-WAHA-INTEGRATION.md
│   └── ...
│
└── README.md
```

---

## Database

### Multi-Tenant Isolation

PEKAN menggunakan **Schema-per-Tenant** — setiap workspace (tenant) punya schema database sendiri. Ini memastikan data benar-benar terisolasi secara物理, bukan hanya logical.

```
PostgreSQL: pekan
│
├── Schema: public (Global)
│   ├── tenants                  — Daftar workspace
│   ├── users                    — Akun pengguna
│   ├── tenant_memberships       — Relasi user ↔ workspace
│   ├── roles                    — Role per workspace
│   ├── permissions              — Daftar permission
│   ├── role_permissions         — Relasi role ↔ permission
│   ├── membership_roles         — Relasi membership ↔ role
│   ├── auth_sessions            — JWT session tracking
│   ├── auth_refresh_tokens      — Refresh token (hashed)
│   ├── audit_logs               — Log audit global
│   ├── global_settings          — Konfigurasi sistem
│   ├── registration_otps        — OTP pendaftaran
│   ├── whatsapp_sessions        — Sesi WA ↔ user
│   ├── whatsapp_otp_tokens      — Token pairing WA
│   └── whatsapp_bot_queue       — Antrian pesan chatbot
│
├── Schema: tenant_<code> (Per-Workspace — Isolated)
│   ├── finance_transactions           — Transaksi
│   ├── finance_transaction_items      — Line items
│   ├── finance_transaction_attachments — Lampiran transaksi
│   ├── finance_categories             — Kategori kustom
│   ├── finance_accounts               — Akun keuangan
│   ├── finance_savings                — Tabungan
│   ├── finance_budgets                — Anggaran
│   ├── finance_reminders              — Pengingat
│   ├── finance_reminder_payments      — Cicilan
│   ├── finance_entity_attachments     — Lampiran entitas
│   ├── finance_notifications          — Notifikasi
│   ├── finance_reports                — Laporan tersimpan
│   ├── finance_channels               — Channel notifikasi
│   ├── finance_templates              — Template pesan
│   └── finance_receipt_scan_configs   — Konfigurasi OCR
│
└── Isolation: SET LOCAL search_path TO tenant_<code>, public
```

### Entity Relationship (Simplified)

```
┌──────────┐     ┌──────────────────┐     ┌──────────┐
│ tenants  │◄────│ tenant_memberships │────►│  users   │
└────┬─────┘     └──────────────────┘     └────┬─────┘
     │                                         │
     │  ┌───────────────────────────────────┐  │
     └──│      Tenant Schema (isolated)     │  │
        │                                   │  │
        │  transactions ──► items           │  │
        │      │                            │  │
        │      └──► attachments ──► files   │  │
        │                                   │  │
        │  categories ──► budgets           │  │
        │                                   │  │
        │  savings                          │  │
        │                                   │  │
        │  reminders ──► payments           │  │
        │      │                            │  │
        │      └──► entity_attachments      │  │
        │                                   │  │
        │  notifications                    │  │
        │  reports                          │  │
        └───────────────────────────────────┘  │
                                               │
        ┌──────────────────────────────────────┘
        │
        │  public schema:
        │  sessions ──► users
        │  refresh_tokens ──► sessions
        │  roles ──► role_permissions ──► permissions
        └──────────────────────────────────────
```

### Migrasi Database

```bash
# Global migration (public schema)
cd backend
./scripts/apply_migrations.sh

# Tenant migration (semua tenant schema)
DATABASE_URL="postgres://..." go run ./scripts/migrate_tenants.go
```

---

## Instalasi

### Instalasi Cepat (~5 menit)

Butuh Docker, Go, dan Node.js sudah terinstall.

```bash
# 1. Clone repository
git clone https://github.com/honet-labs/pekan.git
cd pekan

# 2. Jalankan PostgreSQL & Redis via Docker
docker compose -f deploy/docker-compose.server-test.yml up -d

# 3. Setup backend
cd backend
cp .env.example .env
# Edit .env — isi JWT_SECRET dengan random string yang kuat:
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

### Panduan Lengkap

Untuk panduan instalasi **step-by-step** yang lebih detail (termasuk install prasyarat, troubleshooting, production deployment), lihat:

> **[INSTALL.md](INSTALL.md)** — Panduan lengkap instalasi PEKAN

---

## Deployment Produksi

PEKAN menyediakan **2 opsi deployment** produksi:

### Opsi A: Docker (Recommended)

Semua komponen berjalan di container. Mudah diinstall, diupdate, dan di-backup.

```bash
git clone https://github.com/honet-labs/pekan.git
cd pekan
sudo bash deploy/installer-docker.sh
```

**Container yang berjalan:**
- `pekan-postgres` (PostgreSQL 16)
- `pekan-redis` (Redis 7)
- `pekan-api` (API Server)
- `pekan-worker` (Background Worker)
- `pekan-ai` (AI Queue Worker)
- `pekan-web` (Frontend Nginx)

### Opsi B: Systemd (Native)

Binary Go native dengan systemd services. Performa lebih baik untuk traffic tinggi.

```bash
git clone https://github.com/honet-labs/pekan.git
cd pekan
sudo bash deploy/installer-systemd.sh
```

**Services yang berjalan:**
- `pekan-api.service` (API Server)
- `pekan-worker.service` (Background Worker)
- `pekan-ai.service` (AI Queue Worker)
- `nginx` (Reverse Proxy + Frontend)
- `postgresql` (Database)
- `redis` (Cache)

### Perbandingan Opsi

| Aspek | Docker | Systemd |
| :--- | :--- | :--- |
| **Kemudahan** | Sangat Mudah | Mudah |
| **Performa** | Baik | Lebih Baik |
| **Resource** | Lebih tinggi | Lebih rendah |
| **Isolasi** | Sempurna | Biasa |
| **Cocok Untuk** | Server kecil-menengah | Server besar, traffic tinggi |

### Update Versi

```bash
# Auto-detect mode (systemd/docker)
sudo bash deploy/update-versi.sh

# Force mode
sudo bash deploy/update-versi.sh --mode docker
sudo bash deploy/update-versi.sh --mode systemd
```

### Backup & Restore

```bash
# Backup (Docker)
docker compose -f /opt/pekan/docker-compose.yml exec pekan-postgres \
  pg_dump -U postgres pekan | gzip > backup.sql.gz

# Backup (Systemd)
sudo bash deploy/backup.sh

# Restore
sudo bash deploy/restore.sh /opt/pekan/backups/pekan_backup_YYYYMMDD.tar.gz
```

### Cloudflare Tunnel (Server Tanpa IP Publik)

Kalau server cuma punya IP private (`192.168.x.x`), bisa pakai Cloudflare Tunnel:

```bash
# Install cloudflared
curl -L https://github.com/cloudflare/cloudflared/releases/latest/download/cloudflared-linux-amd64.deb -o cloudflared.deb
sudo dpkg -i cloudflared.deb

# Login & buat tunnel
cloudflared tunnel login
cloudflared tunnel create pekan

# Konfigurasi (~/.cloudflared/config.yml)
cat > ~/.cloudflared/config.yml << EOF
tunnel: <TUNNEL_ID>
credentials-file: /root/.cloudflared/<TUNNEL_ID>.json
ingress:
  - hostname: pekan.yourdomain.com
    service: http://localhost:80
  - service: http_status:404
EOF

# Jalankan
cloudflared tunnel route dns pekan pekan.yourdomain.com
sudo cloudflared service install
sudo systemctl start cloudflared
```

### Spesifikasi Server

| Tier | vCPU | RAM | Storage | Cocok Untuk |
| :--- | :--- | :--- | :--- | :--- |
| **Development** | 1 Core | 2 GB | 20 GB | Testing, solo usage |
| **Production** | 2 Core | 4 GB | 50 GB | Tim kecil (5-20 user) |
| **Enterprise** | 4+ Core | 8 GB+ | 100 GB+ | Organisasi besar |

---

## Konfigurasi

### Environment Variables (backend/.env)

| Variable | Default | Deskripsi |
| :--- | :--- | :--- |
| `APP_ENV` | `development` | `development` atau `production` |
| `HTTP_PORT` | `8080` | Port API server |
| `DATABASE_URL` | - | Koneksi PostgreSQL |
| `JWT_SECRET` | - | **Wajib diisi!** Secret untuk JWT signing |
| `JWT_ACCESS_TTL_MINUTES` | `15` | Masa aktif access token |
| `JWT_REFRESH_TTL_HOURS` | `720` | Masa aktif refresh token (30 hari) |
| `CORS_ALLOWED_ORIGINS` | `http://localhost:5173` | Domain yang diizinkan CORS |
| `API_RATE_LIMIT_PER_MINUTE` | `120` | Rate limit per menit |
| `STORAGE_PROVIDER` | `local` | `local`, `s3`, atau `gdrive` |
| `STORAGE_LOCAL_PATH` | `./data/storage` | Path penyimpanan file lokal |
| `RATE_LIMIT_REDIS_URL` | *(kosong)* | Redis URL (kosong = in-memory) |

> **Penting di production**: Ganti `JWT_SECRET` dengan random string yang kuat, set `APP_ENV=production`, dan gunakan `sslmode=require` untuk database.

### Database Settings (global_settings)

Konfigurasi yang disimpan di database dan bisa diubah via admin dashboard:

| Key | Deskripsi |
| :--- | :--- |
| `notification_wa_active_provider` | Provider WhatsApp aktif (`wa_waha`, `wa_fonnte`, atau `wa`) |
| `notification_wa_waha` | Konfigurasi WAHA: `{"apiUrl":"...","apiKey":"...","session":"default"}` |
| `wa_bot_active_ai_provider` | AI provider untuk chatbot |
| `wa_bot_system_instructions` | Custom system prompt untuk AI chatbot |
| `receipt_active_ai_provider` | AI provider untuk OCR |
| `ai_queue_workers` | Jumlah concurrent AI worker (default: 4) |

---

## API Endpoints

### Publik (Tanpa Auth)

| Method | Path | Deskripsi |
| :--- | :--- | :--- |
| `GET` | `/api/v1/healthz` | Health check |
| `POST` | `/api/v1/auth/login` | Login |
| `POST` | `/api/v1/auth/register/init` | Register workspace baru |
| `POST` | `/api/v1/auth/register/verify` | Verifikasi OTP registrasi |
| `POST` | `/api/v1/auth/forgot-password` | Lupa password |
| `POST` | `/api/v1/auth/reset-password` | Reset password |
| `POST` | `/api/v1/auth/refresh` | Refresh access token |
| `POST` | `/api/v1/webhook/whatsapp` | Webhook dari WAHA |

### Terproteksi (Auth + Tenant Required)

| Module | Method & Path | Deskripsi |
| :--- | :--- | :--- |
| **Profile** | `GET /me/profile` | Lihat profil |
| | `PUT /me/profile` | Update profil |
| | `POST /me/change-password` | Ganti password |
| **Transactions** | `GET /finance/transactions` | List transaksi |
| | `POST /finance/transactions` | Buat transaksi |
| | `GET/PUT/DELETE /finance/transactions/:id` | Detail/update/hapus |
| | `POST /finance/transactions/:id/attachments` | Upload lampiran |
| **Savings** | `GET/POST /finance/savings` | List & buat tabungan |
| | `GET/PUT/DELETE /finance/savings/:id` | Detail/update/hapus |
| **Budgets** | `GET/POST /finance/budgets` | List & buat anggaran |
| | `GET/PUT/DELETE /finance/budgets/:id` | Detail/update/hapus |
| **Reminders** | `GET/POST /finance/reminders` | List & buat pengingat |
| | `POST /finance/reminders/:id/payments` | Catat cicilan |
| **Reports** | `POST /finance/reports/transactions` | Generate laporan |
| | `GET /finance/reports/:id/download` | Download laporan |
| **Dashboard** | `GET /finance/dashboard/summary` | Ringkasan keuangan |
| | `GET /finance/dashboard/series` | Data untuk chart |
| **Settings** | `GET/PUT /finance/settings/channels` | Channel notifikasi |
| | `GET/POST /finance/settings/roles` | Manajemen role |
| | `GET/POST /finance/settings/users` | Manajemen user |
| **WhatsApp** | `POST /settings/whatsapp/otp` | Generate OTP pairing |
| | `GET /settings/whatsapp/status` | Status koneksi WA |
| | `POST /settings/whatsapp/chat` | Kirim pesan chatbot |
| **Receipts** | `POST /finance/receipt-scan/scan` | Scan struk |
| | `GET /finance/receipt-scan/history` | Riwayat scan |

### Admin (X-Admin-Token Header)

| Method | Path | Deskripsi |
| :--- | :--- | :--- |
| `POST` | `/admin/login` | Login admin |
| `GET` | `/admin/tenants` | List semua workspace |
| `POST` | `/admin/tenants` | Buat workspace baru |
| `GET` | `/admin/stats` | Statistik global |
| `GET` | `/admin/system-logs` | Log sistem |
| `GET` | `/admin/database/stats` | Statistik database |

Spec lengkap: [backend/openapi/openapi.yaml](backend/openapi/openapi.yaml)

---

## WhatsApp Integration

PEKAN bisa diintegrasikan dengan WhatsApp via WAHA (WhatsApp HTTP API). Pengguna bisa mencatat transaksi, cek anggaran, dan berinteraksi dengan AI langsung dari WhatsApp.

### Setup Cepat

```bash
# 1. Jalankan WAHA
docker run -d --name waha -p 3000:3000 devlikeapro/waha

# 2. Set webhook di WAHA
curl -X POST http://localhost:3000/api/default/settings \
  -H "Content-Type: application/json" \
  -d '{
    "webhook": {
      "url": "https://your-server.com/api/v1/webhook/whatsapp",
      "events": ["message"],
      "secret": "rahasia-webhook-anda"
    }
  }'

# 3. Konfigurasi di PEKAN Admin Dashboard
#    → WhatsApp → Active Provider: wa_waha
#    → API URL: http://waha:3000
#    → Webhook Secret: rahasia-webhook-anda
```

### Flow Penggunaan

```
User (WhatsApp)           PEKAN                    WAHA
     │                      │                       │
     │── "beli kopi 15rb" ──▶                       │
     │                      │◀── webhook ───────────│
     │                      │                       │
     │                      │── enqueue message      │
     │                      │   (async)              │
     │                      │                       │
     │                      │── AI proses pesan      │
     │                      │   → parse transaksi    │
     │                      │   → simpan ke DB       │
     │                      │                       │
     │◀── "✅ Dicatat" ─────│── sendText ───────────▶│
```

### Fitur WhatsApp

- **Catat transaksi**: "beli kopi 15rb", "gaji 5jt"
- **Multi-item**: "catat: nasi 20rb, kopi 8rb"
- **Cek anggaran**: "berapa sisa anggaran makan?"
- **Hapus transaksi**: "hapus transaksi abc123"
- **Scan struk**: Kirim foto + "!scan"
- **Grup**: Bot merespons saat di-mention (@pekan)

Dokumentasi lengkap: [docs/WHATSAPP-WAHA-INTEGRATION.md](docs/WHATSAPP-WAHA-INTEGRATION.md)

---

## Kontribusi

Kontribusi sangat dipersilakan! Berikut cara berkontribusi:

### Branch Strategy

| Branch | Fungsi |
| :--- | :--- |
| `main` | Code production yang stabil |
| `dev` | Development branch — PR merge ke sini |
| `bug` | Branch untuk perbaikan bug |
| `feature/<nama>` | Branch untuk fitur baru |

### Workflow

1. **Fork** repositori ini
2. **Clone** fork Anda: `git clone https://github.com/<username>/pekan.git`
3. Buat branch baru dari `dev`: `git checkout -b feature/fitur-baru dev`
4. Buat perubahan Anda
5. Pastikan test passing: `cd backend && go test ./tests/... -v`
6. **Commit** dengan pesan yang jelas: `git commit -m "feat: tambah fitur X"`
7. **Push**: `git push origin feature/fitur-baru`
8. Buka **Pull Request** ke branch `dev`

### Coding Conventions

- **Backend**: Ikuti clean architecture pattern yang sudah ada (delivery → usecase → repository → domain)
- **Frontend**: Ikuti feature module pattern (api → hooks → components → pages)
- **Naming**: Gunakan bahasa Inggris untuk kode, bahasa Indonesia untuk UI text
- **Commit**: Gunakan format conventional commits (`feat:`, `fix:`, `docs:`, `refactor:`, dll.)

### Menambah Modul Baru

1. Buat directory di `backend/internal/modules/<category>/<module>/`
2. Buat 4 layer: `delivery/http/`, `usecase/`, `infra/`, `domain/`
3. Register route di `backend/internal/app/server.go`
4. Buat migration di `backend/migrations/`
5. Buat frontend feature di `frontend/src/features/<module>/`

---

## Dokumentasi Tambahan

| Dokumen | Deskripsi |
| :--- | :--- |
| [**Instalasi (INSTALL.md)**](INSTALL.md) | **Panduan lengkap instalasi dari nol** |
| [Technical Blueprint](docs/01-TECHNICAL-BLUEPRINT.md) | Arsitektur teknis lengkap |
| [Database Schema](docs/02-DATABASE-SCHEMA.md) | Skema database detail |
| [Security Review](docs/03-SECURITY-ARCHITECTURE-REVIEW.md) | Review keamanan |
| [Backend Guide](docs/04-BACKEND-IMPLEMENTATION-GUIDE.md) | Panduan implementasi backend |
| [Frontend Architecture](docs/05-FRONTEND-ARCHITECTURE.md) | Arsitektur frontend |
| [API Design](docs/06-API-DESIGN.md) | Desain API |
| [WhatsApp Integration](docs/WHATSAPP-WAHA-INTEGRATION.md) | Integrasi WhatsApp + WAHA |
| [Server Installer](docs/SERVER-INSTALLER.md) | Panduan server installer |

---

## Lisensi

Proyek ini dirilis di bawah **[MIT License](LICENSE)**. Bebas digunakan untuk keperluan komersial maupun non-komersial.

---

## Kontak

Punya pertanyaan, masukan, atau butuh bantuan?

| Channel | Link |
| :--- | :--- |
| **Email** | [info@honet.web.id](mailto:info@honet.web.id) |
| **GitHub Issues** | [github.com/honet-labs/pekan/issues](https://github.com/honet-labs/pekan/issues) |
| **GitHub Discussions** | [github.com/honet-labs/pekan/discussions](https://github.com/honet-labs/pekan/discussions) |

---

<p align="center">
  <strong>PEKAN</strong> — Dibangun dengan ❤️ oleh <a href="https://github.com/honet-labs">HONET Labs</a>
</p>
