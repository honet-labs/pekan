# Admin API Documentation

Dokumentasi lengkap untuk API admin PEKAN. Endpoint admin digunakan untuk mengelola platform secara keseluruhan: tenant, user, database, backup, monitoring, dan konfigurasi sistem.

---

## Daftar Isi

- [Autentikasi](#autentikasi)
- [Base URL](#base-url)
- [Login](#login)
- [Tenant Management](#tenant-management)
- [User Management](#user-management)
- [Database Management](#database-management)
- [Backup & Restore](#backup--restore)
- [Monitoring & Stats](#monitoring--stats)
- [Global Settings](#global-settings)
- [WhatsApp Queue](#whatsapp-queue)
- [Testing & Diagnostics](#testing--diagnostics)
- [System Updates](#system-updates)
- [Impersonation](#impersonation)
- [Error Handling](#error-handling)

---

## Autentikasi

Admin API menggunakan **shared secret** authentication. Tidak ada JWT atau session token.

**Header yang dibutuhkan:**

```
X-Admin-Token: <ADMIN_SECRET>
```

**Cara mendapatkan token:**

1. Set environment variable `ADMIN_SECRET` di `.env` backend
2. Login via `POST /api/v1/admin/login` dengan secret tersebut
3. Response akan mengembalikan token yang sama dengan secret

**Catatan keamanan:**
- Token tidak pernah expire
- Token tidak bisa di-revoke tanpa mengubah environment variable
- Simpan token dengan aman, jangan share ke publik

---

## Base URL

```
http://localhost:8080/api/v1/admin
```

---

## Login

### POST /admin/login

Login ke admin panel.

**Request:**
```json
{
  "secret": "your-admin-secret"
}
```

**Response (200):**
```json
{
  "data": {
    "token": "your-admin-secret"
  },
  "meta": {
    "request_id": "uuid",
    "timestamp": "2026-08-19T00:00:00Z"
  }
}
```

**Error (401):**
```json
{
  "error": {
    "code": "INVALID_SECRET",
    "message": "invalid admin secret"
  }
}
```

**Rate Limit:** 100 requests/minute per IP

---

## Tenant Management

### GET /admin/tenants

List semua tenant.

**Headers:**
```
X-Admin-Token: <token>
```

**Response (200):**
```json
{
  "data": [
    {
      "id": "uuid",
      "code": "pekan",
      "name": "PEKAN Demo",
      "is_active": true,
      "created_at": "2026-01-01T00:00:00Z"
    }
  ]
}
```

### POST /admin/bootstrap-tenant

Buat tenant baru dengan admin user.

**Request:**
```json
{
  "code": "myorg",
  "name": "My Organization",
  "admin_email": "admin@myorg.com",
  "admin_password": "secure-password",
  "admin_name": "Admin Name"
}
```

**Response (201):**
```json
{
  "data": {
    "tenant_id": "uuid",
    "user_id": "uuid",
    "code": "myorg"
  }
}
```

### PUT /admin/tenants/{tenantID}

Update nama atau status tenant.

**Request:**
```json
{
  "name": "New Name",
  "is_active": true
}
```

### DELETE /admin/tenants/{tenantID}

Hapus tenant dan seluruh schema-nya. **Tindakan ini tidak bisa dibatalkan.**

### PUT /admin/tenants/{tenantID}/quotas

Update quota tenant.

**Request:**
```json
{
  "max_users": 50,
  "max_transactions": 10000
}
```

### GET /admin/tenants/{tenantID}/modules

List status modul untuk tenant.

### PUT /admin/tenants/{tenantID}/modules

Enable/disable modul untuk tenant.

**Request:**
```json
{
  "module": "transactions",
  "is_enabled": true
}
```

### GET /admin/tenants/{tenantID}/users

List user dalam tenant.

### GET /admin/tenants/{tenantID}/backups

List backup untuk tenant tertentu.

### POST /admin/tenants/{tenantID}/backups

Buat backup untuk tenant tertentu.

### POST /admin/tenants/{tenantID}/backups/restore

Restore backup tenant.

### GET /admin/tenants/{tenantID}/backups/download/{filename}

Download file backup tenant.

---

## User Management

### POST /admin/users/{userID}/reset-password

Reset password user.

**Request:**
```json
{
  "new_password": "new-secure-password"
}
```

### PUT /admin/users/{userID}/email

Ubah email user.

**Request:**
```json
{
  "email": "newemail@example.com"
}
```

### PUT /admin/users/{userID}/phone

Ubah nomor telepon user.

**Request:**
```json
{
  "phone": "+6281234567890"
}
```

---

## Database Management

### POST /admin/database/query

Eksekusi SQL read-only (SELECT saja).

**Request:**
```json
{
  "query": "SELECT COUNT(*) FROM public.tenants"
}
```

**Response (200):**
```json
{
  "data": {
    "columns": ["count"],
    "rows": [[5]],
    "row_count": 1
  }
}
```

**Error (400):** Jika query bukan SELECT.

### GET /admin/database/stats

Statistik database (ukuran tabel, jumlah row).

**Response (200):**
```json
{
  "data": {
    "tables": [
      {
        "schema": "public",
        "name": "tenants",
        "row_count": 5,
        "size": "256 kB"
      }
    ],
    "total_size": "15 MB"
  }
}
```

### GET /admin/database/growth

Pertumbuhan ukuran database dari waktu ke waktu.

---

## Backup & Restore

### GET /admin/backups

List semua backup global.

### POST /admin/backups

Buat backup global.

**Request:**
```json
{
  "type": "full",
  "include_schemas": true,
  "include_data": true
}
```

**Type options:**
- `full` - Backup seluruh database
- `schema` - Hanya schema (DDL)
- `data` - Hanya data

### POST /admin/backups/restore

Restore backup global.

**Request:**
```json
{
  "filename": "pekan_backup_20260819.sql.gz"
}
```

### GET /admin/backups/download/{filename}

Download file backup.

---

## Monitoring & Stats

### GET /admin/server/status

Status server (OS, uptime, DB, Redis, services).

**Response (200):**
```json
{
  "data": {
    "os": {
      "platform": "linux",
      "arch": "amd64",
      "uptime": "72h30m"
    },
    "database": {
      "status": "connected",
      "open_connections": 5,
      "max_connections": 25
    },
    "redis": {
      "status": "connected",
      "used_memory": "1.5MB"
    },
    "services": {
      "pekan-api": "active",
      "pekan-worker": "active",
      "pekan-ai": "active"
    }
  }
}
```

### GET /admin/stats/growth

Statistik pertumbuhan platform.

**Response (200):**
```json
{
  "data": {
    "tenants": {
      "total": 5,
      "new_this_month": 2
    },
    "users": {
      "total": 25,
      "new_this_month": 10
    },
    "transactions": {
      "total": 1500,
      "this_month": 300
    }
  }
}
```

### GET /admin/logs

List audit logs (100 terakhir).

**Query Parameters:**
- `action` - Filter berbagai aksi (create, update, delete)
- `user_id` - Filter berdasarkan user
- `tenant_id` - Filter berdasarkan tenant

### GET /admin/system-logs

Logs dari systemd journal (pekan-api, pekan-worker, pekan-ai).

**Query Parameters:**
- `service` - Nama service (pekan-api, pekan-worker, pekan-ai)
- `lines` - Jumlah baris (default: 100)

---

## Global Settings

### GET /admin/settings/{key}

Ambil global setting. Nilai sensitif akan di-mask.

**Response (200):**
```json
{
  "data": {
    "key": "receipt_api_key_gemini",
    "value": "",
    "is_masked": true
  }
}
```

### PUT /admin/settings/{key}

Set global setting. Mendukung enkripsi otomatis untuk nilai sensitif.

**Request:**
```json
{
  "value": "your-api-key"
}
```

**Daftar setting yang tersedia:**

| Key | Encrypted | Deskripsi |
|-----|-----------|-----------|
| `receipt_api_key_gemini` | Ya | API key Google Gemini |
| `receipt_model_gemini` | Tidak | Model Gemini (e.g., gemini-2.0-flash) |
| `receipt_api_key_openai` | Ya | API key OpenAI |
| `receipt_model_openai` | Tidak | Model OpenAI |
| `receipt_api_key_claude` | Ya | API key Anthropic Claude |
| `receipt_model_claude` | Tidak | Model Claude |
| `receipt_api_key_sumopod` | Ya | API key Sumopod |
| `receipt_model_sumopod` | Tidak | Model Sumopod |
| `receipt_active_ai_provider` | Tidak | Provider AI aktif |
| `wa_bot_active_ai_provider` | Tidak | Provider AI untuk WhatsApp bot |
| `wa_bot_system_instructions` | Tidak | System prompt WhatsApp bot |
| `wa_bot_phone_number` | Tidak | Nomor WhatsApp bot |
| `ai_queue_workers` | Tidak | Jumlah worker AI concurrent |
| `notification_smtp` | Ya | Konfigurasi SMTP JSON |
| `notification_telegram` | Ya | Konfigurasi Telegram JSON |
| `notification_wa` | Ya | Konfigurasi Meta WhatsApp JSON |
| `notification_wa_fonnte` | Ya | Konfigurasi Fonnte JSON |
| `notification_wa_waha` | Ya | Konfigurasi WAHA JSON |
| `notification_wa_gowa` | Ya | Konfigurasi Gowa JSON |
| `notification_wa_active_provider` | Tidak | Provider WhatsApp aktif |
| `storage_active_provider` | Tidak | Provider storage aktif (local/s3/gdrive) |
| `storage_s3_config` | Ya | Konfigurasi S3 JSON |
| `storage_gdrive_config` | Ya | Konfigurasi Google Drive JSON |
| `storage_local_config` | Tidak | Path local storage |
| `optimization_config` | Tidak | Konfigurasi optimasi JSON |
| `branding_app_name` | Tidak | Nama aplikasi |
| `branding_page_title` | Tidak | Judul halaman |
| `branding_logo` | Tidak | URL logo |
| `branding_favicon` | Tidak | URL favicon |
| `branding_public_url` | Tidak | URL publik |

**Contoh set SMTP:**
```json
{
  "value": "{\"host\":\"smtp.gmail.com\",\"port\":587,\"username\":\"you@gmail.com\",\"password\":\"app-password\",\"security\":\"starttls\"}"
}
```

---

## WhatsApp Queue

### GET /admin/whatsapp/queue/stats

Statistik queue WhatsApp bot.

**Response (200):**
```json
{
  "data": {
    "pending": 5,
    "processing": 2,
    "completed": 150,
    "failed": 3
  }
}
```

### GET /admin/whatsapp/queue/history

History queue WhatsApp bot (paginated, searchable).

**Query Parameters:**
- `page` - Halaman (default: 1)
- `limit` - Item per halaman (default: 20)
- `status` - Filter status (pending, processing, completed, failed)
- `search` - Search berdasarkan pesan

### POST /admin/whatsapp/queue/retry/{id}

Retry pesan yang gagal.

---

## Testing & Diagnostics

### POST /admin/test/notification

Test koneksi provider notifikasi.

**Request:**
```json
{
  "provider": "smtp",
  "to": "test@example.com"
}
```

**Provider options:** `smtp`, `telegram`, `whatsapp`

### POST /admin/test/ai

Test koneksi provider AI.

**Request:**
```json
{
  "provider": "gemini",
  "prompt": "Hello, test connection"
}
```

**Provider options:** `gemini`, `openai`, `claude`, `sumopod`

### POST /admin/test/database

Test koneksi database dengan konfigurasi custom.

**Request:**
```json
{
  "host": "localhost",
  "port": 5432,
  "user": "postgres",
  "password": "password",
  "dbname": "pekan"
}
```

---

## System Updates

### GET /admin/updates/check

Cek update terbaru dari GitHub.

**Response (200):**
```json
{
  "data": {
    "current_version": "abc1234",
    "latest_version": "def5678",
    "has_update": true,
    "commits_behind": 3
  }
}
```

### POST /admin/updates/apply

Trigger update sistem (git pull, build, restart).

**Response (202):**
```json
{
  "data": {
    "status": "started",
    "message": "Update process started"
  }
}
```

### GET /admin/updates/status

Poll progress update.

**Response (200):**
```json
{
  "data": {
    "status": "building",
    "progress": 60,
    "log": "Building backend binaries..."
  }
}
```

**Status values:** `idle`, `pulling`, `building`, `restarting`, `completed`, `failed`

---

## Impersonation

### POST /admin/impersonate

Buat session token untuk impersonate user.

**Request:**
```json
{
  "user_id": "uuid",
  "tenant_id": "uuid"
}
```

**Response (200):**
```json
{
  "data": {
    "access_token": "jwt-token",
    "refresh_token": "refresh-token",
    "expires_in": 900
  }
}
```

Token yang dihasilkan memiliki permission `finance.*` dan semua features.

---

## Error Handling

Semua error mengikuti format konsisten:

```json
{
  "error": {
    "code": "ERROR_CODE",
    "message": "Human readable message",
    "request_id": "uuid"
  }
}
```

**Common Error Codes:**

| Code | HTTP Status | Deskripsi |
|------|-------------|-----------|
| `INVALID_SECRET` | 401 | Secret salah |
| `UNAUTHORIZED` | 401 | Token tidak valid atau tidak ada |
| `NOT_FOUND` | 404 | Resource tidak ditemukan |
| `VALIDATION_ERROR` | 400 | Input tidak valid |
| `RATE_LIMITED` | 429 | Terlalu banyak request |
| `INTERNAL_ERROR` | 500 | Server error |

---

## Contoh Penggunaan dengan cURL

### Login
```bash
curl -X POST http://localhost:8080/api/v1/admin/login \
  -H "Content-Type: application/json" \
  -d '{"secret": "your-admin-secret"}'
```

### List Tenants
```bash
curl http://localhost:8080/api/v1/admin/tenants \
  -H "X-Admin-Token: your-admin-secret"
```

### Create Tenant
```bash
curl -X POST http://localhost:8080/api/v1/admin/bootstrap-tenant \
  -H "Content-Type: application/json" \
  -H "X-Admin-Token: your-admin-secret" \
  -d '{
    "code": "myorg",
    "name": "My Organization",
    "admin_email": "admin@myorg.com",
    "admin_password": "secure-password",
    "admin_name": "Admin"
  }'
```

### Execute SQL Query
```bash
curl -X POST http://localhost:8080/api/v1/admin/database/query \
  -H "Content-Type: application/json" \
  -H "X-Admin-Token: your-admin-secret" \
  -d '{"query": "SELECT COUNT(*) FROM public.tenants"}'
```

### Check Server Status
```bash
curl http://localhost:8080/api/v1/admin/server/status \
  -H "X-Admin-Token: your-admin-secret"
```

### Set AI Provider
```bash
curl -X PUT http://localhost:8080/api/v1/admin/settings/receipt_api_key_gemini \
  -H "Content-Type: application/json" \
  -H "X-Admin-Token: your-admin-secret" \
  -d '{"value": "your-gemini-api-key"}'
```

---

## Contoh Penggunaan dengan Postman

1. Import collection dari `docs/postman/PEKAN-API.postman_collection.json`
2. Tambahkan header `X-Admin-Token` dengan value `{{admin_token}}`
3. Jalankan request login terlebih dahulu untuk mendapatkan token
4. Simpan token ke environment variable `admin_token`

---

## Konfigurasi Environment

Pastikan environment variable berikut sudah di-set:

```env
# Admin secret (wajib, minimal 32 karakter di production)
ADMIN_SECRET=your-strong-admin-secret-min-32-characters

# Jika tidak di-set, akan fallback ke JWT_SECRET
# Jika JWT_SECRET juga tidak di-set, default ke "change-me" (TIDAK AMAN untuk production)
```

---

**Terakhir diperbarui:** Agustus 2026
