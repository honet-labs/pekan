# PEKAN — Platform Pencatatan Keuangan Enterprise

[![Go](https://img.shields.io/badge/Go-1.23-00ADD8?style=flat-square&logo=go)](https://golang.org/)
[![React](https://img.shields.io/badge/React-18-61DAFB?style=flat-square&logo=react)](https://react.dev/)
[![Vite](https://img.shields.io/badge/Vite-5-646CFF?style=flat-square&logo=vite)](https://vitejs.dev/)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-15%2B-4169E1?style=flat-square&logo=postgresql)](https://www.postgresql.org/)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg?style=flat-square)](LICENSE)

**PEKAN** adalah platform pengelolaan keuangan berskala enterprise/SaaS yang dibangun dengan arsitektur **Clean Architecture & Domain-Driven Design (DDD)**. Memadukan kekuatan **Go (Golang)** di backend dengan **React + Vite** di frontend, serta dilengkapi asisten keuangan cerdas berbasis **AI** melalui integrasi **WhatsApp Chat Bot** dan **OCR Receipt Scanner**.

---

## Daftar Isi

- [Fitur Lengkap](#fitur-lengkap)
- [Arsitektur Sistem](#arsitektur-sistem)
- [Arsitektur Database](#arsitektur-database)
- [Workflow Aplikasi](#workflow-aplikasi)
- [Stack Teknologi](#stack-teknologi)
- [Struktur Proyek](#struktur-proyek)
- [Panduan Instalasi](#panduan-instalasi)
- [Deployment Produksi](#deployment-produksi)
- [Lisensi](#lisensi)
- [Kontribusi](#kontribusi)

---

## Fitur Lengkap

### Modul Keuangan Inti

| Modul | Deskripsi |
| :--- | :--- |
| **Transaksi** | Pencatatan income, expense, transfer & savings dengan dukungan line-items, pajak, diskon, service charge, metode pembayaran, dan merchant name. |
| **Tabungan (Savings)** | Manajemen target tabungan dengan tracking progress real-time, tanggal target, dan status aktif/tercapai. |
| **Anggaran (Budgets)** | Pengelolaan batas anggaran per kategori/periode dengan alert threshold dan tracking pengeluaran otomatis. |
| **Pengingat (Reminders)** | Pengingat tagihan berkala (none/daily/weekly/monthly/yearly) dengan sistem tenor, pembayaran cicilan, dan bukti bayar. |
| **Laporan & Ekspor** | Ekspor laporan keuangan ke format PDF/CSV/Excel untuk transaksi, tabungan, anggaran, dan pengingat. |
| **Lampiran (Attachments)** | Upload file (JPEG/PNG/WebP, max 10MB) ke transaksi, tabungan, anggaran, dan pengingat dengan virus scan queue. |
| **Master Data** | Manajemen akun keuangan dan kategori transaksi kustom per workspace. |
| **Notifikasi** | Sistem notifikasi in-app real-time dengan status read/unread dan metadata kontekstual. |

### Kecerdasan Buatan (AI)

| Fitur | Deskripsi |
| :--- | :--- |
| **OCR Receipt Scanner** | Upload foto struk belanja, AI mengekstrak merchant, item, harga, pajak, diskon, dan kategori secara otomatis. Mendukung multi-provider: **Google Gemini**, **OpenAI GPT**, dan **Anthropic Claude**. |
| **WhatsApp Chat Bot** | Asisten keuangan interaktif via WhatsApp — catat transaksi, cek saldo, lihat anggaran, hapus/edit transaksi via percakapan natural language. Dilengkapi background queue worker dengan monitoring latensi real-time. |

### Dashboard & Visualisasi

| Fitur | Deskripsi |
| :--- | :--- |
| **Dashboard Keuangan** | Visualisasi arus kas, pendapatan, pengeluaran, dan transfer menggunakan **Apache ECharts** dengan filter rentang tanggal dinamis. |
| **Dashboard Admin** | Monitoring global: statistik workspace, user, transaksi, pertumbuhan data, WhatsApp queue performance, dan database health. |

### Multi-Tenant & Keamanan

| Fitur | Deskripsi |
| :--- | :--- |
| **Multi-Workspace** | Isolasi data penuh antar workspace menggunakan PostgreSQL schema-per-tenant. |
| **RBAC** | Role-Based Access Control dengan sistem roles, permissions, features, dan modules yang granular. |
| **Autentikasi** | JWT access + refresh token dengan rotasi otomatis, deteksi reuse, dan session management. |
| **Registrasi OTP** | Registrasi workspace baru dengan verifikasi OTP via Email atau WhatsApp. |
| **Audit Log** | Pencatatan lengkap setiap aksi pengguna (create, update, delete) dengan before/after snapshot. |
| **File Scanning** | Antrian pemindaian virus otomatis untuk setiap file yang diupload. |
| **Rate Limiting** | Pembatasan request berbasis Redis (terdistribusi) atau in-memory. |

### Notifikasi Multi-Channel

| Channel | Provider |
| :--- | :--- |
| **Email** | SMTP Server (SSL/TLS & STARTTLS, Port 465/587) |
| **Telegram** | Telegram Bot API |
| **WhatsApp** | Meta Official API, WAHA, Fonnte, GOWA |

### Panel Admin (Super Admin)

| Fitur | Deskripsi |
| :--- | :--- |
| **Manajemen Workspace** | CRUD workspace/tenant, kuota user, limit transaksi, status aktif/suspended. |
| **Manajemen User** | Manajemen akun global, force password change, aktivasi/deaktivasi. |
| **Database Tools** | SQL dump backup langsung dari panel admin, statistik ukuran tabel dan pertumbuhan data. |
| **Konfigurasi AI** | Pengaturan provider OCR, model AI, system prompt, dan API key per workspace. |
| **Konfigurasi Global** | Rate limiting, request timeout, max payload size — disimpan di database. |
| **Audit Log Global** | Riwayat seluruh aksi sistem lintas workspace. |

### Pengaturan Workspace

| Fitur | Deskripsi |
| :--- | :--- |
| **Manajemen Anggota** | Undang, kelola status, dan atur role anggota workspace. |
| **Manajemen Role** | Buat role kustom dengan permission granular per modul. |
| **Channel Notifikasi** | Konfigurasi SMTP, Telegram, dan WhatsApp provider per workspace. |
| **Template Pesan** | Kustomisasi template notifikasi per channel dan bahasa. |
| **Audit Log** | Riwayat aksi dalam lingkup workspace. |

### Fitur Tambahan

- **Internationalisasi (i18n)**: Dukungan penuh Bahasa Indonesia dan English.
- **Responsive Design**: Optimasi tampilan desktop, tablet, dan mobile.
- **Dark Mode Ready**: Desain dengan CSS Custom Properties yang mendukung tema gelap.
- **Paginasi**: Navigasi halaman untuk semua daftar data.

---

## Arsitektur Sistem

### High-Level Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                         CLIENTS                                     │
│  ┌──────────┐  ┌──────────────┐  ┌──────────────────────────────┐  │
│  │ Browser  │  │ WhatsApp App │  │ External API (Telegram/SMTP) │  │
│  └────┬─────┘  └──────┬───────┘  └──────────────┬───────────────┘  │
└───────┼───────────────┼──────────────────────────┼──────────────────┘
        │               │                          │
        ▼               ▼                          ▼
┌───────────────────────────────────────────────────────────────────┐
│                      NGINX REVERSE PROXY                          │
│              (SSL Termination, Static Files, Gzip)                │
└────────────────────────────┬──────────────────────────────────────┘
                             │
            ┌────────────────┼────────────────┐
            ▼                                 ▼
┌───────────────────────┐          ┌───────────────────────┐
│   FRONTEND (Vite)     │          │   BACKEND (Go Chi)    │
│                       │          │                       │
│  React 18 + TypeScript│          │  REST API Server      │
│  Apache ECharts       │          │  (:8080)              │
│  CSS Vanilla Modern   │          │                       │
│  i18n (ID/EN)         │          │  ┌─────────────────┐  │
│                       │          │  │ HTTP Handlers    │  │
│  Modules:             │          │  │ (Delivery Layer) │  │
│  - Auth & Profile     │          │  ├─────────────────┤  │
│  - Finance Dashboard  │          │  │ Use Cases        │  │
│  - Transactions       │          │  │ (Business Logic) │  │
│  - Savings & Budgets  │          │  ├─────────────────┤  │
│  - Reminders          │          │  │ Repositories     │  │
│  - Reports & Receipts │          │  │ (Data Access)    │  │
│  - Settings & Admin   │          │  ├─────────────────┤  │
│  - Chatbot AI         │          │  │ Domain Entities  │  │
│                       │          │  │ (Core Models)    │  │
│                       │          │  └─────────────────┘  │
└───────────────────────┘          └───────────┬───────────┘
                                               │
                          ┌────────────────────┼────────────────────┐
                          ▼                    ▼                    ▼
                ┌──────────────┐    ┌──────────────┐    ┌──────────────┐
                │ PostgreSQL   │    │    Redis      │    │  Background  │
                │              │    │              │    │   Worker     │
                │ Schema-per-  │    │ Rate Limit   │    │              │
                │ Tenant       │    │ Session Cache │    │ WhatsApp AI  │
                │ Isolation    │    │              │    │ Queue        │
                └──────────────┘    └──────────────┘    │ File Scan    │
                                                        │ Notifications│
                                                        └──────────────┘
```

### Clean Architecture (Per Module)

```
┌──────────────────────────────────────────────────────┐
│                   Delivery Layer                      │
│              (HTTP Handlers / Routes)                 │
│         Menerima request, validasi input,             │
│         mengembalikan response JSON                   │
├──────────────────────────────────────────────────────┤
│                   Use Case Layer                      │
│              (Business Logic / Service)               │
│         Orkestrasi bisnis, autorisasi,                │
│         validasi domain, audit logging                │
├──────────────────────────────────────────────────────┤
│                 Infrastructure Layer                  │
│            (Repository / Data Access)                 │
│         Query SQL, integrasi storage,                 │
│         koneksi external API                          │
├──────────────────────────────────────────────────────┤
│                   Domain Layer                        │
│              (Entities & Errors)                      │
│         Model data murni, domain errors,              │
│         interface kontrak repository                  │
└──────────────────────────────────────────────────────┘
```

### Middleware Pipeline

```
Request ──▶ Logger ──▶ Recovery ──▶ CORS ──▶ Rate Limiter
    ──▶ Request ID ──▶ Auth (JWT) ──▶ Tenant Context
    ──▶ RBAC (Module/Feature/Permission) ──▶ Handler
```

---

## Arsitektur Database

### Multi-Tenant Schema Isolation

PEKAN menggunakan strategi **Schema-per-Tenant** di PostgreSQL untuk isolasi data yang aman:

```
PostgreSQL Database: pekan
│
├── Schema: public (Global/Shared)
│   ├── tenants              — Daftar workspace
│   ├── users                — Akun pengguna global
│   ├── tenant_memberships   — Relasi user ↔ workspace
│   ├── roles                — Definisi role per workspace
│   ├── permissions          — Daftar permission sistem
│   ├── membership_roles     — Relasi membership ↔ role
│   ├── role_permissions     — Relasi role ↔ permission
│   ├── modules              — Modul aplikasi (finance.transactions, etc.)
│   ├── features             — Fitur per modul (read/write)
│   ├── products             — Produk langganan
│   ├── sessions             — Session JWT (refresh token rotation)
│   ├── files                — Metadata file upload global
│   ├── file_scan_jobs       — Antrian pemindaian virus
│   ├── audit_logs           — Log audit global
│   ├── global_settings      — Konfigurasi sistem global
│   ├── registration_otps    — OTP pendaftaran workspace
│   ├── whatsapp_sessions    — Sesi WhatsApp ↔ User mapping
│   ├── whatsapp_otp_tokens  — Token OTP untuk pairing WA
│   └── whatsapp_message_queue — Antrian pesan chatbot
│
├── Schema: tenant_<code> (Per-Workspace — Isolated)
│   ├── finance_transactions           — Transaksi keuangan
│   ├── finance_transaction_items      — Line items per transaksi
│   ├── finance_transaction_attachments — Lampiran transaksi
│   ├── finance_categories             — Kategori kustom
│   ├── finance_accounts               — Akun keuangan
│   ├── finance_savings                — Target tabungan
│   ├── finance_budgets                — Anggaran
│   ├── finance_reminders              — Pengingat tagihan
│   ├── finance_reminder_payments      — Pembayaran cicilan
│   ├── finance_entity_attachments     — Lampiran entitas (savings/budgets/reminders)
│   ├── finance_notifications          — Notifikasi in-app
│   ├── finance_reports                — Laporan yang digenerate
│   ├── finance_notification_channels  — Konfigurasi channel notifikasi
│   ├── finance_message_templates      — Template pesan
│   └── finance_receipt_scan_configs   — Konfigurasi provider OCR AI
│
└── Transaction Isolation: SET LOCAL search_path TO tenant_<code>, public
```

### Entity Relationship Diagram (Simplified)

```
┌─────────┐    ┌───────────────────┐    ┌───────────┐
│ tenants │◄───│ tenant_memberships │───►│   users   │
└────┬────┘    └───────────────────┘    └─────┬─────┘
     │                                        │
     │  ┌──────────────────────────────┐      │
     └──│    Tenant-Scoped Schema      │      │
        │                              │      │
        │  transactions ◄── items      │      │
        │       │                      │      │
        │       ├── attachments ──► files (public)
        │       │                      │
        │  categories ◄── budgets      │
        │                              │
        │  savings                     │
        │                              │
        │  reminders ◄── payments      │
        │       │                      │
        │       └── entity_attachments │
        │                              │
        │  notifications               │
        │  reports                     │
        │  notification_channels       │
        │  message_templates           │
        │  receipt_scan_configs        │
        └──────────────────────────────┘
```

---

## Workflow Aplikasi

### 1. Registrasi Workspace Baru

```
User mengisi form ──▶ Backend validasi ──▶ Generate OTP (6 digit)
     ──▶ Kirim OTP via Email/WhatsApp ──▶ User verifikasi OTP
     ──▶ Buat Tenant + Schema + User + Membership + Default Roles
     ──▶ Kirim Welcome Email ──▶ Redirect ke Login
```

### 2. Alur Autentikasi (Login)

```
Email + Password ──▶ Verifikasi bcrypt hash
     ──▶ Generate JWT Access Token (15 min)
     ──▶ Generate Refresh Token (7 hari, hashed di DB)
     ──▶ Set HTTP-Only Cookie ──▶ Redirect ke Dashboard

Token Expired ──▶ POST /auth/refresh (cookie)
     ──▶ Validasi + Rotasi Refresh Token
     ──▶ Deteksi Token Reuse (revoke semua session)
     ──▶ Generate token pair baru
```

### 3. Alur Pencatatan Transaksi

```
User input form ──▶ Validasi (amount, date, account, category)
     ──▶ Insert finance_transactions + finance_transaction_items
     ──▶ Upload attachments (jika ada) ──▶ Queue file scan
     ──▶ Write audit log ──▶ Response JSON
```

### 4. Alur OCR Receipt Scanner

```
Upload foto struk ──▶ Detect MIME type + validasi
     ──▶ Encode base64 ──▶ Kirim ke AI Provider (Gemini/OpenAI/Claude)
     ──▶ Parse JSON response (merchant, items, total, tax, dll.)
     ──▶ Tampilkan preview hasil ekstraksi
     ──▶ User review & edit ──▶ Simpan sebagai transaksi resmi
```

### 5. Alur WhatsApp Chat Bot

```
User kirim pesan WA ──▶ Webhook terima pesan
     ──▶ Enqueue ke whatsapp_message_queue
     ──▶ Background Worker ambil dari queue
     ──▶ Lookup session (phone ↔ user mapping)
     ──▶ Load financial context (saldo, budget, transaksi terakhir)
     ──▶ Kirim ke AI (system prompt + context + user message)
     ──▶ Parse AI response (action: create_tx, list_tx, delete_tx, etc.)
     ──▶ Execute action di tenant schema
     ──▶ Kirim reply ke user via WhatsApp provider
     ──▶ Update queue status + latency metrics
```

### 6. Alur Notifikasi Multi-Channel

```
Trigger event (reminder due, OTP, etc.)
     ──▶ Load channel configs per workspace
     ──▶ Render message template (with variables)
     ──▶ Dispatch ke driver: SMTP / Telegram / WhatsApp
     ──▶ Log hasil pengiriman
```

---

## Stack Teknologi

### Backend

| Komponen | Teknologi |
| :--- | :--- |
| Bahasa | Go (Golang) 1.23+ |
| HTTP Router | `go-chi/chi/v5` |
| Database Driver | `jackc/pgx/v5` (PostgreSQL) |
| Cache & Session | `go-redis/redis/v9` |
| Auth & Security | `golang-jwt/jwt/v5`, `bcrypt` |
| PDF Generation | `phpdave11/gofpdf` |
| File Storage | Local FS, Amazon S3, Google Drive |

### Frontend

| Komponen | Teknologi |
| :--- | :--- |
| UI Library | React 18+ (TypeScript) |
| Build Tool | Vite 5+ |
| Routing | React Router DOM v6 |
| Charts | Apache ECharts (vanilla JS integration) |
| Styling | CSS Vanilla Modern (Glassmorphism, Custom Properties, Responsive Grid) |
| i18n | Custom React Context (ID/EN) |

### Infrastructure

| Komponen | Teknologi |
| :--- | :--- |
| Database | PostgreSQL 15+ |
| Cache | Redis 7+ (opsional) |
| Web Server | Nginx (reverse proxy, SSL, gzip) |
| Process Manager | Systemd |
| OS Support | Ubuntu/Debian, Rocky Linux |

---

## Struktur Proyek

```
PEKAN/
├── backend/
│   ├── cmd/
│   │   ├── api/             # Entry point REST API Server
│   │   ├── worker/          # Background Queue Worker (AI, WA, Scan)
│   │   └── ai/              # AI model testing utilities
│   ├── internal/
│   │   ├── modules/
│   │   │   ├── core/
│   │   │   │   ├── auth/    # Login, register, session, OTP, profile
│   │   │   │   ├── admin/   # Super admin panel (tenants, users, DB tools)
│   │   │   │   └── subscription/
│   │   │   └── finance/
│   │   │       ├── transactions/  # CRUD transaksi + line items
│   │   │       ├── savings/       # Target tabungan
│   │   │       ├── budgets/       # Anggaran per kategori
│   │   │       ├── reminders/     # Pengingat tagihan + cicilan
│   │   │       ├── reports/       # Ekspor PDF/CSV/Excel
│   │   │       ├── receipts/      # OCR receipt scanner AI
│   │   │       ├── attachments/   # File upload + virus scan
│   │   │       ├── dashboard/     # Statistik & chart data
│   │   │       ├── master/        # Akun & kategori
│   │   │       ├── notifications/ # In-app notifications
│   │   │       ├── settings/      # Roles, channels, templates, audit
│   │   │       └── whatsapp/      # Chat bot AI + queue worker
│   │   ├── platform/             # Infrastructure drivers
│   │   │   ├── access/           # RBAC engine
│   │   │   ├── audit/            # Audit log writer
│   │   │   ├── auth/             # JWT token management
│   │   │   ├── config/           # Environment & global settings
│   │   │   ├── db/               # DB connection + tenant TX
│   │   │   ├── httpx/            # HTTP helpers & error mapper
│   │   │   ├── middleware/       # Logger, CORS, rate limit, auth
│   │   │   ├── notification/     # Multi-channel drivers
│   │   │   ├── security/         # Encryption helpers
│   │   │   ├── session/          # Session store
│   │   │   ├── storage/          # Object storage (Local/S3/GDrive)
│   │   │   └── tenancy/          # Tenant context propagation
│   │   └── app/                  # Application bootstrap
│   ├── migrations/               # SQL DDL migration files (0001–0040+)
│   ├── seeds/                    # Demo data seeding
│   └── tests/                    # Unit & integration tests
├── frontend/
│   └── src/
│       ├── core/                 # Global: components, auth, i18n, hooks, styles
│       └── features/
│           ├── core/             # Admin panel, auth pages, profile
│           └── finance/          # All finance feature modules
├── deploy/                       # Server installer scripts & systemd configs
└── docs/                         # Additional documentation
```

---

## Panduan Instalasi

### Prasyarat Sistem

| Software | Versi | Kegunaan |
| :--- | :--- | :--- |
| **Go** | 1.23+ | Backend API server & worker |
| **Node.js** | 18+ LTS | Frontend build (Vite + React) |
| **PostgreSQL** | 15+ | Database utama (multi-tenant) |
| **Redis** | 7+ | *Opsional* — rate limiting & queue |

### Langkah 1: Setup Database

```bash
# Buat database
psql -c "CREATE DATABASE pekan;"

# Jalankan migrasi
cd backend
./scripts/apply_migrations.sh        # Linux/macOS
# atau
.\scripts\apply_migrations.ps1       # Windows PowerShell

# (Opsional) Masukkan data demo
./scripts/apply_demo_seed.sh
```

### Langkah 2: Jalankan Backend

```bash
cd backend
cp .env.example .env
# Edit .env sesuai konfigurasi Anda (DATABASE_URL, JWT_SECRET, dll.)

go mod tidy
go run cmd/api/main.go               # API Server → http://<IP_SERVER>:8080
go run cmd/worker/main.go            # Background Worker (terminal terpisah)
```

### Langkah 3: Jalankan Frontend

```bash
cd frontend
cp .env.example .env
npm install
npm run dev                           # Dev Server → http://<IP_SERVER>:5173
```

### Login Demo

| Field | Value |
| :--- | :--- |
| URL | `http://<IP_SERVER>:5173` (dev) atau `https://<DOMAIN>` (production) |
| Tenant Code | `default` |
| Email | `owner@pekan.local` |
| Password | `password` |
| Role | Owner (full access) |

> **Catatan:** Ganti `<IP_SERVER>` dengan IP address server Anda (contoh: `192.168.1.100`).

---

## Deployment Produksi

PEKAN menyediakan skrip instalasi otomatis untuk server produksi yang mengkonfigurasi Systemd, Nginx, PostgreSQL, dan Redis:

| OS | Script |
| :--- | :--- |
| Ubuntu / Debian | `deploy/install_server.sh` |
| Rocky Linux | `deploy/install_server_rocky.sh` |
| Uninstall | `deploy/uninstall_server.sh` |
| Update | `deploy/update_app.sh` |
| Backup DB | `deploy/backup.sh` |
| Restore DB | `deploy/restore.sh` |

Panduan lengkap: [docs/SERVER-INSTALLER.md](docs/SERVER-INSTALLER.md)

### Akses Aplikasi via IP Server

Setelah deployment, aplikasi dapat diakses melalui IP address server Anda:

| Mode | URL |
| :--- | :--- |
| Development | `http://<IP_SERVER>:5173` (frontend) / `http://<IP_SERVER>:8080` (API) |
| Production (Nginx) | `http://<IP_SERVER>` (port 80) atau `https://<IP_SERVER>` (port 443 dengan SSL) |

### Menggunakan Cloudflare Tunnel (Untuk Server Tanpa IP Publik)

Jika server Anda hanya memiliki **IP private** (contoh: `192.168.x.x`, `10.x.x.x`) dan tidak memiliki IP publik, Anda dapat menggunakan **Cloudflare Tunnel (`cloudflared`)** untuk membungkus aplikasi dengan domain yang dapat diakses dari internet secara aman tanpa perlu membuka port atau konfigurasi port forwarding.

**Langkah-langkah:**

1. **Install `cloudflared`** di server Anda:
   ```bash
   # Debian/Ubuntu
   curl -L --output cloudflared.deb https://github.com/cloudflare/cloudflared/releases/latest/download/cloudflared-linux-amd64.deb
   sudo dpkg -i cloudflared.deb
   ```

2. **Login ke akun Cloudflare:**
   ```bash
   cloudflared tunnel login
   ```

3. **Buat tunnel baru:**
   ```bash
   cloudflared tunnel create pekan
   ```

4. **Konfigurasi tunnel** (`~/.cloudflared/config.yml`):
   ```yaml
   tunnel: <TUNNEL_ID>
   credentials-file: /root/.cloudflared/<TUNNEL_ID>.json

   ingress:
     - hostname: pekan.yourdomain.com
       service: http://localhost:80    # Arahkan ke Nginx
     - service: http_status:404
   ```

5. **Tambahkan DNS record** di Cloudflare Dashboard:
   ```bash
   cloudflared tunnel route dns pekan pekan.yourdomain.com
   ```

6. **Jalankan tunnel sebagai service:**
   ```bash
   sudo cloudflared service install
   sudo systemctl start cloudflared
   sudo systemctl enable cloudflared
   ```

Setelah konfigurasi selesai, aplikasi PEKAN Anda dapat diakses melalui `https://pekan.yourdomain.com` dari mana saja — meskipun server hanya memiliki IP private. Cloudflare secara otomatis menyediakan sertifikat SSL gratis.

### Menjalankan Pengujian

```bash
cd backend
go test ./tests/... -v
```

---

## Lisensi

Proyek ini dirilis di bawah lisensi **[Apache License 2.0](LICENSE)**. Anda bebas menggunakan, memodifikasi, dan mendistribusikan kode ini baik untuk keperluan komersial maupun non-komersial.

---

## Kontribusi

1. **Fork** repositori ini.
2. Buat branch fitur baru (`git checkout -b feature/FiturBaru`).
3. Commit perubahan (`git commit -m 'Menambahkan fitur baru'`).
4. Push ke branch Anda (`git push origin feature/FiturBaru`).
5. Buat **Pull Request** ke branch `main`.

---

<p align="center">
  <strong>PEKAN</strong> — Dibangun dengan ❤️ oleh <a href="https://github.com/honet-labs">HONET Labs</a>
</p>
