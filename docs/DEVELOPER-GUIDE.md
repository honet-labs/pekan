# Developer Guide

Panduan untuk developer yang ingin mengembangkan atau berkontribusi pada PEKAN.

---

## Daftar Isi

- [Glossary](#glossary)
- [Arsitektur](#arsitektur)
- [Multi-Tenant System](#multi-tenant-system)
- [Authentication Flow](#authentication-flow)
- [Database Schema](#database-schema)
- [API Reference](#api-reference)
- [Environment Variables](#environment-variables)
- [Development Setup](#development-setup)
- [Testing](#testing)
- [Deployment](#deployment)

---

## Glossary

| Istilah Teknis | Istilah User | Penjelasan |
|----------------|--------------|------------|
| `tenant_code` | Kode Akses | Identifier unik untuk setiap workspace/organisasi. Digunakan saat login. Contoh: `pekan`, `myorg` |
| `tenant_id` | - | UUID internal untuk tenant di database |
| `schema_name` | - | Nama schema PostgreSQL untuk tenant. Format: `wkspid_pekan_{code}` |
| `membership_id` | - | Relasi antara user dan tenant. Satu user bisa punya membership di banyak tenant |
| `access_profile` | - | Kumpulan permissions, modules, dan features yang dimiliki user di tenant tertentu |
| `role` | Role | Kumpulan permissions yang bisa diberikan ke user. Contoh: `owner`, `admin`, `member` |
| `permission` | Permission | Hak akses spesifik. Contoh: `transactions.create`, `budgets.read` |
| `module` | Modul | Fitur utama aplikasi. Contoh: `transactions`, `budgets`, `savings` |
| `feature` | Fitur | Sub-fitur dalam module. Contoh: `transactions.create`, `transactions.export` |
| `global_setting` | Pengaturan | Konfigurasi sistem yang disimpan di database. Contoh: `receipt_api_key_gemini` |
| `ADMIN_SECRET` | - | Secret untuk akses admin panel. Bukan JWT, tapi shared secret |

---

## Arsitektur

### High-Level

```
┌─────────────────────────────────────────────────────────┐
│                      Clients                             │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐              │
│  │ Browser  │  │ WhatsApp │  │ API Call │              │
│  │ (React)  │  │  (WAHA)  │  │ (cURL)   │              │
│  └────┬─────┘  └────┬─────┘  └────┬─────┘              │
│       │              │              │                    │
└───────┼──────────────┼──────────────┼────────────────────┘
        │              │              │
        ▼              ▼              ▼
┌─────────────────────────────────────────────────────────┐
│                    Nginx (port 80)                       │
│              Reverse Proxy + Static Files                │
└───────────────────────┬─────────────────────────────────┘
                        │
        ┌───────────────┼───────────────┐
        ▼               ▼               ▼
┌──────────────┐ ┌──────────────┐ ┌──────────────┐
│  pekan-api   │ │ pekan-worker │ │  pekan-ai    │
│  (REST API)  │ │ (Background) │ │ (AI Queue)   │
│  port 8080   │ │              │ │              │
└──────┬───────┘ └──────┬───────┘ └──────┬───────┘
       │                │                │
       ▼                ▼                ▼
┌─────────────────────────────────────────────────────────┐
│                    PostgreSQL 16                         │
│  ┌─────────────┐  ┌─────────────────────────────────┐  │
│  │   public    │  │  tenant_pekan (schema per tenant) │  │
│  │   schema    │  │  tenant_myorg                     │  │
│  │  (global)   │  │  tenant_xxx                       │  │
│  └─────────────┘  └─────────────────────────────────┘  │
└─────────────────────────────────────────────────────────┘
                        │
                        ▼
                ┌──────────────┐
                │    Redis 7   │
                │ (Optional)   │
                └──────────────┘
```

### Backend Modules

```
backend/
├── cmd/
│   ├── api/main.go        → Entry point: REST API server
│   ├── worker/main.go     → Entry point: file scan + reminder worker
│   └── ai/main.go         → Entry point: WhatsApp AI queue worker
│
├── internal/
│   ├── app/server.go      → Bootstrap, DI wiring, route registration
│   │
│   ├── modules/
│   │   ├── core/
│   │   │   ├── auth/      → Login, register, session, OTP
│   │   │   ├── admin/     → Super admin panel
│   │   │   └── subscription/ → Entitlement management
│   │   │
│   │   └── finance/
│   │       ├── transactions/  → CRUD transaksi + items
│   │       ├── savings/       → Target tabungan
│   │       ├── budgets/       → Anggaran per kategori
│   │       ├── reminders/     → Pengingat + cicilan
│   │       ├── reports/       → Laporan + export
│   │       ├── receipts/      → OCR receipt scanner
│   │       ├── attachments/   → File upload + scan
│   │       ├── dashboard/     → Statistik & chart
│   │       ├── master/        → Akun & kategori
│   │       ├── notifications/ → In-app notifikasi
│   │       ├── settings/      → Roles, channels, templates
│   │       └── whatsapp/      → Chatbot AI + queue
│   │
│   └── platform/              → Shared infrastructure
│       ├── access/            → RBAC engine
│       ├── auth/              → JWT management
│       ├── config/            → Environment config
│       ├── db/                → DB connection + tenant transaction
│       ├── middleware/        → Logger, CORS, rate limit, auth
│       ├── security/          → Encryption (AES-256-GCM)
│       ├── storage/           → File storage (Local/S3/GDrive)
│       ├── tenancy/           → Tenant context propagation
│       ├── audit/             → Audit log writer
│       ├── session/           → Session store
│       └── httpx/             → HTTP helpers & error mapper
```

---

## Multi-Tenant System

### Schema-per-Tenant

Setiap tenant punya schema PostgreSQL sendiri:

```
PostgreSQL: pekan
│
├── Schema: public (Global)
│   ├── tenants                  → Daftar workspace
│   ├── users                    → Akun pengguna
│   ├── tenant_memberships       → Relasi user ↔ workspace
│   ├── roles                    → Role per workspace
│   ├── permissions              → Daftar permission
│   ├── role_permissions         → Relasi role ↔ permission
│   ├── membership_roles         → Relasi membership ↔ role
│   ├── auth_sessions            → JWT session tracking
│   ├── auth_refresh_tokens      → Refresh token (hashed)
│   ├── audit_logs               → Log audit global
│   ├── global_settings          → Konfigurasi sistem
│   └── tenant_modules           → Module per tenant
│   └── tenant_features          → Feature per tenant
│
├── Schema: tenant_pekan (Per-Workspace)
│   ├── finance_transactions
│   ├── finance_transaction_items
│   ├── finance_categories
│   ├── finance_accounts
│   ├── finance_savings
│   ├── finance_budgets
│   ├── finance_reminders
│   └── ...
```

### Tenant Context Flow

```
Request → JWT Auth → Extract TenantID → SET search_path → Query
```

```go
// Set search_path untuk isolasi tenant
SET LOCAL search_path TO tenant_pekan, public
```

### Schema Name Mapping

| tenant_code | schema_name |
|-------------|-------------|
| `pekan` | `wkspid_pekan_pekanhonet` |
| `myorg` | `wkspid_pekan_myorg` |
| `test` | `wkspid_pekan_test` |

---

## Authentication Flow

### Login

```
1. User input: kode_aksess + email + password
2. Cari tenant berdasarkan tenant_code
3. Cari user berdasarkan email di tenant tersebut
4. Verify password (Argon2id)
5. Cek account lockout (Redis/in-memory)
6. Fetch membership + access profile
7. Generate JWT (access + refresh)
8. Set HttpOnly cookie
```

### JWT Token Structure

```json
{
  "user_id": "uuid",
  "tenant_id": "uuid",
  "tenant_code": "pekan",
  "email": "user@example.com",
  "permissions": ["transactions.create", "budgets.read"],
  "features": ["transactions.create", "transactions.read"],
  "modules": ["transactions", "budgets"],
  "session_id": "uuid",
  "exp": 1234567890
}
```

### Access Profile

Access profile diambil dari:
1. **tenant_modules** - module yang di-enable untuk tenant
2. **tenant_features** - feature yang di-enable untuk tenant
3. **role_permissions** - permission yang dimiliki user melalui role

```go
type AccessProfile struct {
    Permissions []string  // ["transactions.create", "budgets.read"]
    Features    []string  // ["transactions.create", "transactions.read"]
    Modules     []string  // ["transactions", "budgets"]
}
```

---

## Database Schema

### Key Tables (public schema)

| Table | Fungsi |
|-------|--------|
| `tenants` | Daftar workspace/organisasi |
| `users` | Akun pengguna (global) |
| `tenant_memberships` | Relasi user ↔ tenant |
| `roles` | Role per tenant |
| `permissions` | Daftar permission |
| `role_permissions` | Mapping role → permission |
| `membership_roles` | Mapping membership → role |
| `auth_sessions` | JWT session tracking |
| `auth_refresh_tokens` | Refresh token (hashed) |
| `audit_logs` | Log audit global |
| `global_settings` | Konfigurasi sistem |
| `tenant_modules` | Module per tenant |
| `tenant_features` | Feature per tenant |

### Key Tables (tenant schema)

| Table | Fungsi |
|-------|--------|
| `finance_transactions` | Transaksi keuangan |
| `finance_transaction_items` | Line items transaksi |
| `finance_categories` | Kategori (income/expense) |
| `finance_accounts` | Akun keuangan |
| `finance_savings` | Target tabungan |
| `finance_budgets` | Anggaran per kategori |
| `finance_reminders` | Pengingat tagihan |
| `finance_reports` | Laporan tersimpan |
| `finance_notifications` | Notifikasi in-app |

---

## API Reference

### Base URL

```
http://localhost:8080/api/v1
```

### Authentication

Semua endpoint (kecuali public) membutuhkan JWT token:

```
Authorization: Bearer <access_token>
```

Atau cookie:
```
Cookie: pekan_access_token=<access_token>
```

### Public Endpoints

| Method | Endpoint | Fungsi |
|--------|----------|--------|
| POST | `/auth/login` | Login |
| POST | `/auth/refresh` | Refresh token |
| POST | `/auth/register/init` | Register (init) |
| POST | `/auth/register/verify` | Register (verify OTP) |
| POST | `/auth/forgot-password` | Lupa password |
| GET | `/healthz` | Health check |
| GET | `/branding` | Public branding |

### Protected Endpoints

| Method | Endpoint | Fungsi | Permission |
|--------|----------|--------|------------|
| GET | `/finance/transactions` | List transaksi | `transactions.read` |
| POST | `/finance/transactions` | Buat transaksi | `transactions.create` |
| GET | `/finance/budgets` | List anggaran | `budgets.read` |
| POST | `/finance/budgets` | Buat anggaran | `budgets.create` |
| GET | `/finance/savings` | List tabungan | `savings.read` |
| GET | `/finance/reports` | List laporan | `reports.read` |
| POST | `/finance/reports` | Generate laporan | `reports.create` |

### Admin Endpoints

| Method | Endpoint | Fungsi | Auth |
|--------|----------|--------|------|
| POST | `/admin/login` | Admin login | Secret |
| GET | `/admin/tenants` | List tenant | X-Admin-Token |
| POST | `/admin/backup` | Buat backup | X-Admin-Token |
| POST | `/admin/restore` | Restore backup | X-Admin-Token |

---

## Environment Variables

### Required

| Variable | Contoh | Fungsi |
|----------|--------|--------|
| `DATABASE_URL` | `postgres://user:pass@host:5432/pekan?sslmode=prefer` | Koneksi database |
| `JWT_SECRET` | `random-32-chars-minimum` | Signing JWT token |
| `ADMIN_SECRET` | `random-32-chars-minimum` | Admin panel access |

### Optional

| Variable | Default | Fungsi |
|----------|---------|--------|
| `APP_ENV` | `development` | Environment (development/production) |
| `HTTP_PORT` | `8080` | Port API server |
| `RATE_LIMIT_REDIS_URL` | - | Redis URL untuk rate limiting |
| `STORAGE_PROVIDER` | `local` | Storage provider (local/s3/gdrive) |
| `RECEIPT_SCAN_SECRET` | - | Secret untuk OCR receipt |

---

## Development Setup

### Prerequisites

- Go 1.23+
- Node.js 20+
- PostgreSQL 16+
- Redis 7+ (optional)

### Quick Start

```bash
# Clone
git clone https://github.com/honet-labs/pekan.git
cd pekan

# Start dependencies
docker compose -f deploy/docker-compose.server-test.yml up -d

# Backend
cd backend
cp .env.example .env
# Edit .env
go mod tidy
go run cmd/api/main.go

# Frontend (new terminal)
cd frontend
npm install
npm run dev
```

---

## Testing

```bash
# Run all tests
cd backend
go test ./...

# Run specific test
go test ./internal/modules/core/auth/usecase/... -v

# Run with coverage
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

---

## Deployment

### Docker

```bash
sudo bash deploy/installer-docker.sh --branch dev
```

### Systemd

```bash
sudo bash deploy/installer-systemd.sh --branch dev
```

### Update

```bash
sudo bash deploy/update-versi.sh
```

---

**Terakhir diperbarui:** Agustus 2026
