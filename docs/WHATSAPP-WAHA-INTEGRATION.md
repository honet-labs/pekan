# Integrasi PEKAN AI dengan WhatsApp Gateway (WAHA)

Dokumentasi lengkap integrasi PEKAN dengan WAHA (WhatsApp HTTP API) untuk mengaktifkan AI assistant keuangan via WhatsApp.

---

## Daftar Isi

1. [Arsitektur](#1-arsitektur)
2. [Prasyarat](#2-prasyarat)
3. [Instalasi WAHA](#3-instalasi-waha)
4. [Konfigurasi PEKAN](#4-konfigurasi-pekan)
5. [Konfigurasi Webhook](#5-konfigurasi-webhook)
6. [Sesi WhatsApp (Koneksi Akun)](#6-sesi-whatsapp-koneksi-akun)
7. [Alur Pesan](#7-alur-pesan)
8. [AI & Pemrosesan Transaksi](#8-ai--pemrosesan-transaksi)
9. [Scanning Struk via WhatsApp](#9-scanning-struk-via-whatsapp)
10. [Grup WhatsApp](#10-grup-whatsapp)
11. [Referensi API](#11-referensi-api)
12. [Skema Database](#12-skema-database)
13. [Variable Environment](#13-variable-environment)
14. [Troubleshooting](#14-troubleshooting)
15. [Pertimbangan Keamanan](#15-pertimbangan-keamanan)
16. [Contoh Percakapan](#16-contoh-percakapan)

---

## 1. Arsitektur

```
┌──────────────────────────────────────────────────────────────┐
│                        PEKAN Server                           │
│                                                               │
│  ┌─────────────┐     ┌──────────────────┐     ┌───────────┐ │
│  │  cmd/api     │     │  cmd/ai           │     │ PostgreSQL │ │
│  │  (HTTP)      │────▶│  (Queue Worker)   │────▶│ + Redis    │ │
│  │              │     │                    │     │            │ │
│  │  Menerima    │     │  Dequeue pesan,    │     │  Menyimpan │ │
│  │  webhook +   │     │  proses AI,        │     │  sesi,     │ │
│  │  API user    │     │  kirim balasan     │     │  queue,    │ │
│  │              │     │                    │     │  transaksi │ │
│  └──────┬──────┘     └────────┬───────────┘     └───────────┘ │
│         │                      │                               │
└─────────┼──────────────────────┼───────────────────────────────┘
          │                      │
          │ Webhook              │ WAHA API
          │ (WAHA -> PEKAN)      │ (PEKAN -> WAHA)
          ▼                      ▼
┌──────────────────────────────────────────────────────────────┐
│                     WAHA (WhatsApp Gateway)                   │
│                                                               │
│  ┌─────────────┐     ┌──────────────────┐                    │
│  │  HTTP API    │     │  WhatsApp Engine  │                    │
│  │  (port 3000) │────▶│  (Baileys/Web)    │──▶ WhatsApp Cloud │
│  └─────────────┘     └──────────────────┘                    │
└──────────────────────────────────────────────────────────────┘
```

### Dua Proses Terpisah

| Proses | Binary | Fungsi |
|--------|--------|--------|
| **HTTP Server** | `cmd/api` | Menerima webhook WAHA + melayani API user |
| **AI Queue Worker** | `cmd/ai` | Background worker: dequeue pesan, proses AI, kirim balasan |

Keduanya berbagi database PostgreSQL yang sama. Webhook handler memasukkan pesan ke queue, AI worker mengambil dan memproses secara asinkron.

---

## 2. Prasyarat

| Komponen | Versi Minimum | Keterangan |
|----------|---------------|------------|
| **PEKAN Backend** | - | Go 1.21+, PostgreSQL 16 |
| **WAHA** | 2024.x | WhatsApp HTTP API (open source) |
| **Node.js** | 20.x | Untuk build frontend (opsional) |
| **Domain/SSL** | - | Diperlukan untuk production (webhook harus HTTPS) |

### Port yang Dibutuhkan

| Port | Service | Akses |
|------|---------|-------|
| 80/443 | Nginx (frontend + reverse proxy) | Public |
| 8080 | PEKAN API | Internal |
| 3000 | WAHA HTTP API | Internal |
| 5432 | PostgreSQL | Internal |
| 6379 | Redis | Internal |

---

## 3. Instalasi WAHA

### 3.1 Docker (Recommended)

```bash
# WAHA Core (gratis)
docker run -d \
  --name waha \
  --restart always \
  -p 3000:3000 \
  -e WAHA_STORAGE=memory \
  -e WAHA_CORE_CONFIG='{"debug":false}' \
  devlikeapro/waha

# WAHA Plus (fitur lengkap, berbayar)
docker run -d \
  --name waha \
  --restart always \
  -p 3000:3000 \
  -e WAHA_STORAGE=memory \
  -e WAHA_CORE_CONFIG='{"debug":false}' \
  devlikeapro/waha-plus
```

### 3.2 Docker Compose (Bersama PEKAN)

Tambahkan ke `docker-compose.yml`:

```yaml
services:
  # ... existing postgres, redis ...

  waha:
    image: devlikeapro/waha
    container_name: waha
    restart: always
    ports:
      - "3000:3000"
    environment:
      - WAHA_STORAGE=memory
      - WAHA_CORE_CONFIG={"debug":false}
    networks:
      - pekan-network

  pekan-api:
    # ... existing config ...
    depends_on:
      - waha
    environment:
      - WAHA_URL=http://waha:3000
```

### 3.3 Verifikasi Instalasi

```bash
# Cek status WAHA
curl http://localhost:3000/api/server/status

# Response yang diharapkan:
{
  "status": "running",
  "version": "2024.x.x"
}
```

---

## 4. Konfigurasi PEKAN

### 4.1 Melalui Admin Dashboard

Buka **Admin Dashboard** → tab **WhatsApp** → isi konfigurasi:

| Field | Nilai | Keterangan |
|-------|-------|------------|
| **Active Provider** | `wa_waha` | Pilih WAHA sebagai gateway |
| **API URL** | `http://waha:3000` | Base URL WAHA (internal network) |
| **API Key** | *(opsional)* | API key jika diaktifkan di WAHA |
| **Session** | `default` | Nama session WAHA |

### 4.2 Melalui Database (Global Settings)

Jika perlu set manual via SQL:

```sql
-- Aktifkan provider WAHA
INSERT INTO global_settings (key, value) VALUES
  ('notification_wa_active_provider', 'wa_waha')
ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value;

-- Konfigurasi WAHA
INSERT INTO global_settings (key, value) VALUES
  ('notification_wa_waha', '{"apiUrl":"http://localhost:3000","apiKey":"","session":"default"}')
ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value;

-- Secret untuk validasi webhook (opsional)
INSERT INTO global_settings (key, value) VALUES
  ('whatsapp_webhook_secret', 'rahasia-token-webhook-anda')
ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value;
```

### 4.3 Format JSON Konfigurasi WAHA

```json
{
  "apiUrl": "http://waha-host:3000",
  "apiKey": "your-waha-api-key",
  "session": "default"
}
```

| Field | Tipe | Wajib | Keterangan |
|-------|------|-------|------------|
| `apiUrl` | string | Ya | Base URL WAHA tanpa trailing slash |
| `apiKey` | string | Tidak | API key WAHA (kosongkan jika tidak diaktifkan) |
| `session` | string | Tidak | Nama session (default: `"default"`) |

---

## 5. Konfigurasi Webhook

### 5.1 Set Webhook di WAHA

WAHA perlu dikonfigurasi untuk mengirim webhook ke PEKAN saat pesan masuk.

#### Melalui WAHA API:

```bash
curl -X POST http://localhost:3000/api/default/settings \
  -H "Content-Type: application/json" \
  -d '{
    "webhook": {
      "url": "https://your-pekan-server.com/api/v1/webhook/whatsapp",
      "events": ["message"],
      "secret": "rahasia-token-webhook-anda",
      "retriesCount": 3
    }
  }'
```

#### Melalui WAHA Web UI:

1. Buka `http://localhost:3000`
2. Pilih session `default`
3. Klik **Settings** → **Webhooks**
4. Isi:
   - **URL**: `https://your-pekan-server.com/api/v1/webhook/whatsapp`
   - **Events**: `message`
   - **Secret**: *(sama dengan `whatsapp_webhook_secret` di PEKAN)*

### 5.2 Format Webhook yang Diterima PEKAN

PEKAN mendukung **5 format webhook** berbeda:

#### Format 1: WAHA (Primary)

```json
{
  "session": "default",
  "me": {
    "id": "6281234567890@c.us",
    "pushName": "Aish | Support HONET"
  },
  "payload": {
    "fromMe": false,
    "from": "6281234567890@c.us",
    "id": "true_6281234567890@c.us_3EB0F75D61F3E5A1C4",
    "body": "beli kopi 15rb",
    "caption": "",
    "media": null,
    "mediaUrl": null,
    "participant": null
  }
}
```

#### Format 2: Generic / Fonnte

```json
{
  "sender": "6281234567890",
  "message": "beli kopi 15rb",
  "phone": "6281234567890",
  "text": "beli kopi 15rb",
  "url": null,
  "id": "msg-123"
}
```

#### Format 3: Evolution API

```json
{
  "data": {
    "key": {
      "remoteJid": "6281234567890@s.whatsapp.net",
      "id": "message-id-xxx"
    },
    "message": {
      "conversation": "beli kopi 15rb"
    },
    "participant": null
  }
}
```

#### Format 4: Form Data

```
Content-Type: application/x-www-form-urlencoded

sender=6281234567890&message=beli+kopi+15rb&id=msg-123
```

#### Format 5: WhatsApp Cloud API (Meta)

```json
{
  "entry": [{
    "changes": [{
      "value": {
        "messages": [{
          "from": "6281234567890",
          "id": "message-id",
          "text": { "body": "beli kopi 15rb" }
        }]
      }
    }]
  }]
}
```

### 5.3 Validasi Token Webhook

PEKAN memvalidasi token webhook dari 3 sumber (prioritas):

1. **Query parameter**: `?token=xxx`
2. **Header**: `X-Webhook-Token: xxx`
3. **Header**: `Authorization: Bearer xxx`

Token diambil dari:
1. Environment variable `WHATSAPP_WEBHOOK_SECRET`
2. Database setting `whatsapp_webhook_secret`

### 5.4 Filter Pesan Diri Sendiri (Self-Sent)

Pesan yang dikirim dari bot sendiri (`payload.fromMe = true` pada format WAHA) secara otomatis diabaikan dan tidak diproses.

---

## 6. Sesi WhatsApp (Koneksi Akun)

### 6.1 Metode Koneksi

PEKAN mendukung **2 metode** menghubungkan akun WhatsApp user:

| Metode | Keamanan | Keterangan |
|--------|----------|------------|
| **OTP Login** | Tinggi | Verifikasi via kode OTP, nomor HP harus cocok dengan profil |
| **Direct Connect** | Sedang | Langsung hubungkan tanpa OTP (admin only) |

### 6.2 OTP Login (Recommended)

#### Langkah 1: Generate Kode OTP

User membuka **Settings → WhatsApp** di web UI PEKAN, klik **"Generate Kode OTP"**.

```
POST /api/v1/settings/whatsapp/otp
Authorization: Bearer <access_token>

Response:
{
  "otp_code": "WA-A3F21B"
}
```

Kode berformat `WA-XXXXXX` (6 karakter hex), berlaku **10 menit**.

#### Langkah 2: Kirim ke WhatsApp

User mengirim kode ke bot PEKAN di WhatsApp:

```
!login WA-A3F21B
```

Atau klik link langsung:
```
https://wa.me/6281234567890?text=!login%20WA-A3F21B
```

#### Langkah 3: Verifikasi

PEKAN memverifikasi:
1. Kode OTP valid dan belum expired
2. Nomor HP pengirim cocok dengan nomor di profil user (`user_profiles.phone`)
3. Jika nomor terenkripsi, di-decrypt dulu

#### Langkah 4: Sesi Tersimpan

```
whatsapp_sessions:
  phone_number: "081234567890"  →  tenant_id: xxx, user_id: xxx
```

#### Flow Lengkap OTP:

```
┌─────────┐    Generate OTP     ┌─────────┐
│  User    │ ──────────────────▶ │  PEKAN  │
│  (Web)   │ ◀────────────────── │  API    │
└─────────┘   WA-A3F21B         └─────────┘
     │
     │ Kirim "!login WA-A3F21B"
     ▼
┌─────────┐    Webhook           ┌─────────┐    Validasi    ┌──────────┐
│  WAHA   │ ──────────────────▶ │  PEKAN  │ ────────────▶ │ Postgres │
└─────────┘                      └─────────┘               └──────────┘
     │                                   │
     │         Balasan                  │
     │◀──────────────────────────────────┘
     │   "Akun Berhasil Terhubung!"
     ▼
┌─────────┐
│  User   │
│  (WA)   │
└─────────┘
```

### 6.3 Direct Connect (Admin)

```
POST /api/v1/settings/whatsapp/connect
Authorization: Bearer <access_token>
Content-Type: application/json

{
  "phone_number": "081234567890"
}
```

### 6.4 Cek Status Koneksi

```
GET /api/v1/settings/whatsapp/status
Authorization: Bearer <access_token>

Response (terhubung):
{
  "connected": true,
  "phone_number": "081234567890",
  "last_active": "2026-08-06T10:30:00Z"
}

Response (tidak terhubung):
{
  "connected": false
}
```

### 6.5 Disconnect

```
DELETE /api/v1/settings/whatsapp/disconnect
Authorization: Bearer <access_token>
```

---

## 7. Alur Pesan

### 7.1 Flow Lengkap: Pesan Masuk → Balasan AI

```
1. WhatsApp User mengirim pesan
         │
         ▼
2. WAHA menerima pesan, kirim webhook ke PEKAN
   POST /api/v1/webhook/whatsapp
         │
         ▼
3. WebhookHandler.HandleIncomingMessage()
   ├── Parse payload (5 format didukung)
   ├── Filter pesan diri sendiri (fromMe=true)
   ├── Deteksi pesan grup (butuh @mention)
   ├── Deteksi perintah !login → ProcessLogin()
   │
   ▼
4. EnqueueMessage()
   ├── Generate message_id (jika kosong)
   ├── INSERT INTO whatsapp_bot_queue (status='pending')
   ├── Jika duplikat → return 200 OK (abaikan)
   └── Return 200 OK ke WAHA
         │
         ▼
5. [cmd/ai Queue Worker]
   ├── Polling setiap 2 detik
   ├── SELECT ... FOR UPDATE SKIP LOCKED
   ├── Update status → 'processing'
   │
   ▼
6. ProcessAIChat()
   ├── Lookup sesi (phone → user)
   ├── Update last_active
   ├── Cek konfirmasi pending (receipt scan)
   ├── Deteksi perintah !scan → scan struk
   │
   ▼
7. parseTransactionWithLLM()
   ├── Kirim pesan ke AI provider (Gemini/OpenAI/Anthropic)
   ├── AI mengembalikan JSON array transaksi
   ├── Deduplikasi hasil parse
   │
   ▼
8. Proses Setiap Transaksi
   ├── action="create" → CreateChatTransaction()
   ├── action="delete" → DeleteChatTransaction()
   ├── action="update" → UpdateChatTransaction()
   │
   ▼
9. Jika bukan transaksi → generateInteractiveResponseWithLLM()
   ├── Bangun system prompt dengan data keuangan
   ├── Kirim ke AI provider
   └── Kembalikan respons percakapan
         │
         ▼
10. SendWhatsAppMessage()
    ├── POST {wahaUrl}/api/sendText
    └── Kirim balasan ke user
         │
         ▼
11. Update status → 'success' + recording latency
```

### 7.2 Status Queue

| Status | Keterangan |
|--------|------------|
| `pending` | Pesan diterima, menunggu diproses |
| `processing` | Sedang diproses oleh AI worker |
| `success` | Berhasil diproses dan balasan terkirim |
| `failed` | Gagal diproses (error message dicatat) |

### 7.3 Timeout & Error Handling

- **AI Processing Timeout**: 60 detik per pesan
- **WAHA API Timeout**: 10 detik untuk kirim pesan
- **JID Resolution Timeout**: 3 detik
- **Queue Worker**: 4 goroutine concurrent (configurable)
- **Polling Interval**: 2 detik antar dequeue

---

## 8. AI & Pemrosesan Transaksi

### 8.1 Provider AI yang Didukung

| Provider | Model Default | Endpoint |
|----------|---------------|----------|
| **Gemini** | `gemini-2.0-flash` | `generativelanguage.googleapis.com` |
| **OpenAI** | `gpt-4o-mini` | `api.openai.com` |
| **Anthropic** | `claude-3-5-sonnet-20240620` | `api.anthropic.com` |
| **SumoPod** | - | `ai.sumopod.com` |

### 8.2 Parsing Transaksi (Natural Language → JSON)

User mengirim pesan natural language, AI mengubahnya menjadi struktur transaksi:

**Input:**
```
beli kopi 15rb
```

**Output AI:**
```json
[{
  "action": "create",
  "target_id": "",
  "amount": 15000,
  "type": "expense",
  "description": "beli kopi",
  "category_name": "Makanan",
  "items": [{"name": "Kopi", "qty": 1, "price": 15000}],
  "date": "2026-08-06"
}]
```

### 8.3 Format Nominal yang Dipahami

| Format | Arti |
|--------|------|
| `15rb` / `15k` | Rp 15.000 |
| `1.5jt` / `1.5jt` | Rp 1.500.000 |
| `25000` | Rp 25.000 |
| `25.000` | Rp 25.000 |

### 8.4 Multi-Transaksi dalam Satu Pesan

User bisa mencatat beberapa transaksi sekaligus:

```
catat pengeluaran:
- nasi soto 20rb
- kopi 8rb
```

AI akan mengembalikan array dengan beberapa objek transaksi.

### 8.5 Aksi yang Didukung

| Aksi | Contoh Input | Hasil |
|------|-------------|-------|
| **Create** | `beli kopi 15rb` | Buat transaksi baru |
| **Delete** | `hapus transaksi abc123` | Hapus transaksi by ID |
| **Update** | `ubah transaksi abc123 jadi 20rb` | Update nominal transaksi |

### 8.6 Percakapan Interaktif (Bukan Transaksi)

Jika pesan BUKAN perintah transaksi, AI akan merespons secara percakapan:

```
User: "berapa pengeluaran saya bulan ini?"
AI: "Berdasarkan data Anda bulan ini:
     - Total Pengeluaran: Rp 2.450.000
     - Kategori terbesar: Makan & Jajan (Rp 1.200.000)
     - Sisa anggaran: Rp 550.000

     Apakah Anda ingin melihat rincian lengkap?"
```

### 8.7 System Prompt AI

AI diberikan system prompt yang mencakup:

1. **Batasan Kuasa**: AI HANYA membahas data keuangan PEKAN
2. **Aturan Komunikasi**: Bahasa Indonesia, to the point, format bold WhatsApp
3. **Data Keuangan**: Summary bulanan, daftar anggaran, 100 transaksi terakhir

### 8.8 Custom System Prompt

Admin bisa mengkustom system prompt melalui:
- **Admin Dashboard** → tab **WhatsApp** → **System Instructions**
- Database key: `wa_bot_system_instructions`

---

## 9. Scanning Struk via WhatsApp

### 9.1 Kirim Gambar Struk

User mengirim foto struk dengan caption `!scan`:

```
!scan https://media.url/struk.jpg
```

Atau kirim gambar langsung tanpa caption (otomatis di-detect sebagai `!scan`).

### 9.2 Alur Scanning

```
1. User kirim foto struk + "!scan"
         │
         ▼
2. PEKAN download gambar dari URL
         │
         ▼
3. scanReceiptWithLLM() → AI Vision
   ├── Gemini: inline_data (base64)
   ├── Anthropic: image type
   └── OpenAI: image_url
         │
         ▼
4. AI mengembalikan parsed receipt:
   {
     "merchant_name": "Warung Kopi ABC",
     "total": 23000,
     "suggested_type": "expense",
     "suggested_category_name": "Makanan",
     "transaction_date": "2026-08-06",
     "description": "beli kopi"
   }
         │
         ▼
5. PEKAN menampilkan konfirmasi:
   "Apakah Anda ingin mencatat:
    📋 Warung Kopi ABC
    💰 Rp 23.000
    📅 2026-08-06
    Kategori: Makanan

    Balas YA untuk mencatat atau BATAL untuk membatalkan."
         │
         ▼
6. User balas "YA" → CreateChatTransaction()
   User balas "BATAL" → batalkan
```

### 9.3 Pending Confirmation

Hasil scan disimpan di memory (in-memory map) dengan TTL 5 menit. Jika user tidak merespons dalam 5 menit, konfirmasi expired.

---

## 10. Grup WhatsApp

### 10.1 Deteksi Grup

PEKAN mendeteksi pesan grup via:
- Domain `@g.us` atau `@us` di `fromJid`
- Field `participant` di payload

### 10.2 Filter Mention

Di grup, bot HANYA merespons jika di-mention:
- **Nomor bot**: `@6281234567890`
- **Push name bot**: `@Aish`
- **Keyword khusus**: `@pekan` atau `@bot`

### 10.3 Contoh Penggunaan di Grup

```
@pekan catat pengeluaran bensin 50rb
```

```
@6281234567890 berapa pengeluaran saya bulan ini?
```

---

## 11. Referensi API

### 11.1 Endpoint Publik (Tanpa Auth)

| Method | Path | Fungsi |
|--------|------|--------|
| `POST` | `/api/v1/webhook/whatsapp` | Webhook receiver dari WAHA |

### 11.2 Endpoint Terproteksi (Auth + Tenant)

| Method | Path | Fungsi |
|--------|------|--------|
| `POST` | `/api/v1/settings/whatsapp/otp` | Generate kode OTP |
| `POST` | `/api/v1/settings/whatsapp/connect` | Direct connect |
| `GET` | `/api/v1/settings/whatsapp/status` | Cek status koneksi |
| `DELETE` | `/api/v1/settings/whatsapp/disconnect` | Putuskan sesi |
| `POST` | `/api/v1/settings/whatsapp/chat` | Kirim pesan chatbot dari web |

### 11.3 WAHA API yang Dipanggil PEKAN

| Endpoint WAHA | Method | Fungsi | Timeout |
|---------------|--------|--------|---------|
| `/api/sendText` | POST | Kirim pesan teks | 10 detik |
| `/api/contacts/check-exists` | GET | Cek apakah nomor ada di WhatsApp | 3 detik |
| `/api/{session}/lids/pn/{phone}` | GET | Resolve nomor ke LID | 3 detik |

### 11.4 Request/Response Examples

#### Kirim Pesan via WAHA

```bash
curl -X POST http://localhost:3000/api/sendText \
  -H "Content-Type: application/json" \
  -d '{
    "chatId": "6281234567890@c.us",
    "text": "Halo! Ada yang bisa saya bantu?",
    "session": "default"
  }'
```

#### Cek Nomor via WAHA

```bash
curl "http://localhost:3000/api/contacts/check-exists?phone=6281234567890&session=default"

# Response:
{
  "numberExists": true,
  "chatId": "6281234567890@s.whatsapp.net"
}
```

---

## 12. Skema Database

### 12.1 `whatsapp_otp_tokens`

Menyimpan kode OTP untuk verifikasi koneksi.

```sql
CREATE TABLE whatsapp_otp_tokens (
    token       VARCHAR(20) PRIMARY KEY,        -- "WA-A3F21B"
    tenant_id   UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expires_at  TIMESTAMPTZ NOT NULL,           -- NOW() + 10 menit
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_whatsapp_otp_tokens_expires_at
  ON whatsapp_otp_tokens(expires_at);

CREATE INDEX idx_whatsapp_otp_tokens_user
  ON whatsapp_otp_tokens(tenant_id, user_id);
```

### 12.2 `whatsapp_sessions`

Mapping nomor WhatsApp → user PEKAN.

```sql
CREATE TABLE whatsapp_sessions (
    phone_number VARCHAR(20) PRIMARY KEY,       -- "081234567890"
    tenant_id    UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    user_id      UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    last_active  TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_whatsapp_sessions_user
  ON whatsapp_sessions(tenant_id, user_id);
```

### 12.3 `whatsapp_bot_queue`

Antrian pesan untuk diproses oleh AI worker.

```sql
CREATE TABLE whatsapp_bot_queue (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    phone_number       VARCHAR(20) NOT NULL,
    message            TEXT NOT NULL,
    reply_message      TEXT,
    status             VARCHAR(20) NOT NULL DEFAULT 'pending',
    error_message      TEXT,
    processing_time_ms INT,
    tenant_id          UUID REFERENCES tenants(id) ON DELETE SET NULL,
    user_id            UUID REFERENCES users(id) ON DELETE SET NULL,
    received_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    processed_at       TIMESTAMPTZ,
    message_id         VARCHAR(255)
);

-- Unique index untuk deduplikasi (hanya jika message_id tidak NULL)
CREATE UNIQUE INDEX idx_whatsapp_bot_queue_message_id
  ON whatsapp_bot_queue(message_id)
  WHERE message_id IS NOT NULL;
```

---

## 13. Variable Environment

| Variable | Default | Keterangan |
|----------|---------|------------|
| `WHATSAPP_WEBHOOK_SECRET` | *(kosong)* | Token validasi webhook (fallback dari DB setting) |
| `AI_QUEUE_WORKERS` | `4` | Jumlah concurrent AI worker goroutine |

> **Catatan**: Konfigurasi utama WAHA disimpan di database (`global_settings`), bukan di environment variable.

---

## 14. Troubleshooting

### 14.1 Pesan tidak diproses

| Cek | Solusi |
|-----|--------|
| Webhook URL benar? | Pastikan WAHA mengirim ke `/api/v1/webhook/whatsapp` |
| Token valid? | Cocokkan `WHATSAPP_WEBHOOK_SECRET` dengan yang di WAHA |
| Sesi tersambung? | Cek `whatsapp_sessions` ada data untuk nomor tersebut |
| Queue ada data? | Cek `whatsapp_bot_queue` ada baris dengan status `pending` |
| AI worker berjalan? | Pastikan `cmd/ai` dijalankan sebagai service |

### 14.2 Duplikat pesan / duplikat transaksi

| Penyebab | Solusi |
|----------|--------|
| Webhook retry | WAHA retry karena tidak dapat 200 OK. Pastikan PEKAN merespons 200. |
| Message ID berbeda | Sudah di-fix: fallback ID sekarang stabil (tanpa time bucket). |
| LLM return duplikat | Sudah di-fix: deduplikasi parsedList sebelum loop transaksi. |

### 14.3 AI tidak merespons / timeout

| Cek | Solusi |
|-----|--------|
| API key AI valid? | Pastikan `receipt_api_key_<provider>` benar di admin settings |
| Model tersedia? | Cek model name di `wa_bot_model_<provider>` |
| Network ke AI provider? | Test manual: `curl` ke endpoint AI provider |
| Worker timeout? | Default 60 detik. Cek log untuk latency tinggi. |

### 14.4 OTP tidak bisa login

| Cek | Solusi |
|-----|--------|
| Kode expired? | OTP berlaku 10 menit. Generate ulang jika expired. |
| Nomor tidak cocok? | Pastikan nomor WhatsApp cocok dengan profil di PEKAN. |
| Nomor terenkripsi? | PEKAN otomatis decrypt. Cek log jika ada error decrypt. |

### 14.5 Grup tidak merespons

| Cek | Solusi |
|-----|--------|
| Bot di-mention? | Di grup, bot hanya merespons jika di-mention (`@pekan`, `@bot`, nomor bot). |
| Participant ada? | Pastikan payload WAHA mengirim field `participant`. |

### 14.6 Log Debug

```bash
# Cek log PEKAN API
journalctl -u pekan-api -f

# Cek log PEKAN AI Worker
journalctl -u pekan-ai -f

# Cek log WAHA
docker logs waha -f
```

---

## 15. Pertimbangan Keamanan

### 15.1 Webhook Secret

**WAJIB** di production. Tanpa secret, siapa saja bisa mengirim pesan palsu ke PEKAN.

```bash
# Generate secret acak
openssl rand -base64 32
```

Set di kedua sisi:
- PEKAN: `WHATSAPP_WEBHOOK_SECRET` env var atau DB setting `whatsapp_webhook_secret`
- WAHA: `webhook.secret` di settings

### 15.2 HTTPS

Webhook HARUS menggunakan HTTPS di production. Gunakan Nginx reverse proxy dengan Let's Encrypt.

### 15.3 Rate Limiting

PEKAN menerapkan rate limiting:
- **Login endpoint**: 100 requests/menit per IP
- **Refresh token**: 200 requests/menit per IP
- **Global API**: Configurable (default 120/menit)

### 15.4 Data Sensitivity

- Nomor telepon tersimpan di database (bisa terenkripsi)
- Pesan WhatsApp diproses oleh AI provider eksternal (Gemini/OpenAI/Anthropic)
- Pastikan compliance dengan kebijakan privasi perusahaan

### 15.5 Self-Sent Filter

Pesan yang dikirim dari bot sendiri secara otomatis diabaikan (filter `fromMe`).

---

## 16. Contoh Percakapan

### 16.1 Catat Pengeluaran

```
User: beli kopi 15rb

AI: ✅ Dicatat (ID: abc12345): Pengeluaran Rp 15.000
    (beli kopi, Tanggal: 2026-08-06)
```

### 16.2 Multi-Item

```
User: catat pengeluaran beli makan siang
- nasi soto 20rb
- kopi 8rb

AI: ✅ Dicatat (ID: def56789): Pengeluaran Rp 28.000
    (beli makan siang, Tanggal: 2026-08-06)
```

### 16.3 Cek Anggaran

```
User: berapa sisa anggaran makan bulan ini?

AI: Berikut sisa anggaran *Makan & Jajan* bulan ini:

    📊 Anggaran: Rp 1.500.000
    💸 Terpakai: Rp 950.000
    ✅ Sisa: *Rp 550.000*

    Anda masih memiliki 36% sisa anggaran.
    Pertahankan pengeluaran tetap bijak! 💪
```

### 16.4 Hapus Transaksi

```
User: hapus transaksi abc123

AI: 🗑️ Dihapus (ID: abc123): beli kopi (Rp 15.000)
```

### 16.5 Scan Struk

```
User: !scan https://media.url/struk.jpg

AI: 📋 Struk terdeteksi:
    🏪 Warung Kopi ABC
    💰 Total: Rp 23.000
    📅 Tanggal: 2026-08-06
    📝 Kopi + Snack

    Apakah ingin mencatat? Balas YA atau BATAL

User: YA

AI: ✅ Dicatat (ID: ghi12345): Pengeluaran Rp 23.000
    (Warung Kopi ABC, Tanggal: 2026-08-06, Kategori: Makanan)
```

### 16.6 Topik di Luar Cakupan

```
User: buatkan script python untuk fibonacci

AI: Maaf, saya hanya bisa membantu mengenai data keuangan Anda di PEKAN.
    Silakan tanyakan tentang transaksi, anggaran, atau laporan keuangan Anda.
```

### 16.7 Login OTP

```
User (di PEKAN Web): Generate Kode OTP
PEKAN: WA-A3F21B

User (di WhatsApp): !login WA-A3F21B

AI: ✅ Akun Berhasil Terhubung!
    Nomor: 081234567890
    Sekarang Anda bisa mencatat transaksi langsung dari WhatsApp.
```

### 16.8 Sapaan

```
User: halo

AI: Hai! 👋 Ada yang bisa saya bantu?
    Saya bisa bantu catat transaksi atau cek laporan keuangan Anda.
    Contoh: "catat pengeluaran bensin 50rb"
```

---

## Ringkasan Konfigurasi Cepat

```bash
# 1. Jalankan WAHA
docker run -d --name waha -p 3000:3000 devlikeapro/waha

# 2. Set webhook di WAHA
curl -X POST http://localhost:3000/api/default/settings \
  -H "Content-Type: application/json" \
  -d '{"webhook":{"url":"https://pekan.com/api/v1/webhook/whatsapp","events":["message"],"secret":"rahasia123"}}'

# 3. Set provider di PEKAN admin
# notification_wa_active_provider = wa_waha
# notification_wa_waha = {"apiUrl":"http://waha:3000","apiKey":"","session":"default"}
# whatsapp_webhook_secret = rahasia123

# 4. Start AI worker
systemctl start pekan-ai

# 5. User buka Settings → WhatsApp → Generate OTP → kirim ke WhatsApp
```

---

*Dokumentasi ini terakhir diperbarui: Agustus 2026*
