# 🏢 PEKAN (Web Aplikasi Pencatatan Keuangan)

[![Go Version](https://img.shields.io/badge/Go-1.23-00ADD8?style=flat-square&logo=go)](https://golang.org/)
[![React Version](https://img.shields.io/badge/React-18-61DAFB?style=flat-square&logo=react)](https://react.dev/)
[![Vite](https://img.shields.io/badge/Vite-5-646CFF?style=flat-square&logo=vite)](https://vitejs.dev/)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-15%2B-4169E1?style=flat-square&logo=postgresql)](https://www.postgresql.org/)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg?style=flat-square)](LICENSE)

**PEKAN** (Web Aplikasi Pencatatan Keuangan) adalah platform pengelolaan keuangan berskala korporat/SaaS modern yang dirancang dengan arsitektur tangguh, terukur, dan aman. Platform ini memadukan kekuatan **Go (Golang)** di bagian backend dengan **React + Vite** di bagian frontend, serta dilengkapi dengan fitur asisten keuangan cerdas berbasis **Kecerdasan Buatan (AI)** melalui integrasi **WhatsApp** dan **OCR Receipt Scanner**.

---

## ✨ Fitur Utama

### 1. 💼 Manajemen Multi-Workspace (Multi-Tenant)
* Isolasi data keuangan penuh antar workspace yang aman.
* Pengaturan kuota pengguna, limit transaksi, dan status aktif.
* Dashboard audit log global dan riwayat sistem real-time untuk administrator utama.

### 2. 📊 Dashboard Keuangan Modern & Interaktif
* Visualisasi grafik performa arus kas (Cashflow), pendapatan, pengeluaran, transfer, dan pertumbuhan workspace menggunakan **Apache ECharts**.
* Filter tangguh berdasarkan rentang tanggal dinamis dan opsi penyesuaian visualisasi visual beresolusi tinggi.

### 3. 🧠 OCR Receipt Scanner Cerdas (Scan AI)
* Memungkinkan pengguna mengunggah foto struk belanja/nota transaksi fisik.
* Memanfaatkan kekuatan LLM (Google Gemini, OpenAI, Claude) untuk membaca, mengekstrak rincian belanja, mendeteksi kategori pengeluaran, jumlah pajak, nama toko, hingga total nominal transaksi secara otomatis.
* Pengguna dapat memeriksa hasil ekstraksi terlebih dahulu sebelum menyimpan transaksi secara resmi ke basis data keuangan.

### 4. 💬 Asisten Keuangan WhatsApp & Chat Bot AI
* Mengintegrasikan asisten keuangan interaktif melalui chat WhatsApp.
* Mendukung deteksi perintah percakapan sehari-hari (seperti mencatat transaksi baru, mengecek sisa anggaran bulanan, memindahkan dana via transfer).
* Memiliki antrean pemrosesan pesan berbasis background worker yang andal, dilengkapi dengan visualisasi latensi real-time, tingkat keberhasilan, dan grafik tren pemrosesan pesan di Dashboard Admin.
* Dukungan multi-provider: WhatsApp Business Platform (Meta), Fonnte, WAHA, dan GOWA.

### 5. 🔔 Sistem Notifikasi Multi-Saluran
* Pengiriman OTP (pendaftaran & pemulihan akun) serta pengingat tagihan otomatis (billing reminders).
* Mendukung integrasi dengan SMTP Server (Secure SSL/TLS & STARTTLS), Telegram Bot API, dan WhatsApp Provider.

### 6. 🛠️ Utilitas Admin & Pemeliharaan Database
* Cadangkan database secara otomatis langsung dari panel admin (*SQL Dump Tool*).
* Statistik performa ukuran tabel database PostgreSQL dan pertumbuhan historical data secara real-time.
* Konfigurasi optimasi backend (Rate Limiting, Request Timeout, Max Payload Size) yang disimpan di database secara global.

---

## 🛠️ Stack Teknologi

### 🟢 Backend
* **Bahasa Pemrograman:** Go (Golang) 1.23+
* **HTTP Router / Framework:** `go-chi/chi/v5`
* **Driver Basis Data:** `jackc/pgx/v5` (PostgreSQL)
* **Penyimpanan Cache & Session:** `go-redis/go-redis/v9`
* **Keamanan & Autentikasi:** `golang-jwt/jwt/v5` & `bcrypt`
* **Pembuatan PDF:** `phpdave11/gofpdf`

### 🔵 Frontend
* **UI Library:** React 18+ (dengan TypeScript)
* **Build Tool:** Vite 5+
* **Routing:** React Router DOM v6
* **Visualisasi Grafik:** Apache ECharts (melalui integrasi vanilla JS berkinerja tinggi)
* **Desain & Styling:** CSS Vanilla modern (menerapkan Glassmorphism, CSS Custom Properties, dan Responsive Grid Layout)

---

## 🏗️ Struktur Arsitektur (Clean Architecture)

Proyek ini dirancang menggunakan prinsip **Clean Architecture & Domain-Driven Design (DDD)** untuk pemisahan fungsionalitas (*separation of concerns*) yang jelas dan kemudahan pengujian (*testability*):

```text
PEKAN/
├── backend/
│   ├── cmd/
│   │   ├── api/            # Titik masuk (Entry point) HTTP REST API Server
│   │   ├── worker/         # Titik masuk untuk Background Queue Worker (Chatbot & AI Tasks)
│   │   └── ai/             # Utilitas pengujian model kecerdasan buatan
│   ├── internal/
│   │   ├── modules/        # Modul domain bisnis (finance, core, auth, etc.)
│   │   │   └── finance/
│   │   │       └── whatsapp/ # Penerimaan pesan WA, Queue, & Integrasi Chat Bot AI
│   │   ├── platform/       # Driver infrastruktur (config, database, notification, redis, httpx)
│   │   └── app/            # Setup inisialisasi aplikasi
│   ├── migrations/         # Berkas SQL DDL untuk migrasi basis data
│   ├── seeds/              # Berkas SQL DML untuk data awal (seeding)
│   └── tests/              # Berkas pengujian unit dan integrasi
└── frontend/
    ├── src/
    │   ├── core/           # Komponen UI global, i18n, router, & utils
    │   ├── features/       # Modul fungsional (auth, core/admin, finance, dashboard)
    │   └── main.tsx        # File inisiasi frontend
```

---

## 🚀 Panduan Instalasi & Menjalankan Aplikasi

Ikuti panduan di bawah ini untuk menyiapkan dan menjalankan platform **PEKAN** di lingkungan pengembangan lokal (*local development*) Anda.

---

### 📋 Prasyarat Sistem

Sebelum melakukan pemasangan, pastikan sistem komputer Anda memenuhi prasyarat berikut:

| Perangkat Lunak | Versi Minimal | Kegunaan dalam Sistem |
| :--- | :--- | :--- |
| **Go (Golang)** | `1.23+` | Bahasa pemrograman utama untuk REST API server dan background queue worker. |
| **Node.js** | `18+` (LTS direkomendasikan) | Lingkungan runtime untuk menjalankan compiler Vite dan bundler React frontend. |
| **PostgreSQL** | `15+` | Basis data utama relasional untuk menyimpan data multi-tenant, transaksi keuangan, audit logs, dsb. |
| **Redis Server** | `7+` | *Opsional (sangat direkomendasikan)* untuk Rate Limiting terdistribusi dan manajemen antrean background worker. |

---

### 🗄️ Langkah 1: Setup Basis Data (PostgreSQL)

1. **Buat Database Baru**:
   Masuk ke PostgreSQL CLI (`psql`) atau gunakan GUI editor (seperti DBeaver/pgAdmin), lalu buat database baru bernama `pekan`:
   ```sql
   CREATE DATABASE pekan;
   ```

2. **Inisialisasi Skema (Migrasi)**:
   Kami menyediakan skrip otomatis untuk menjalankan seluruh file migrasi SQL DDL secara berurutan sesuai urutan versinya.
   * Masuk ke folder backend:
     ```bash
     cd backend
     ```
   * Jalankan skrip migrasi berdasarkan Sistem Operasi Anda:
     * **Linux / macOS**:
       ```bash
       chmod +x scripts/apply_migrations.sh
       ./scripts/apply_migrations.sh
       ```
     * **Windows (PowerShell)**:
       ```powershell
       .\scripts\apply_migrations.ps1
       ```

3. **Memasukkan Data Uji Coba (Seeding Demo - Opsional)**:
   Untuk mempermudah pengujian awal tanpa harus mendaftarkan tenant dari nol, Anda dapat memasukkan data demo bawaan (terdiri atas tenant default, user owner, kategori dasar, dan transaksi awal):
   * Jalankan skrip demo seed dari folder `backend`:
     * **Linux / macOS**:
       ```bash
       chmod +x scripts/apply_demo_seed.sh
       ./scripts/apply_demo_seed.sh
       ```
     * **Windows (PowerShell)**:
       ```powershell
       .\scripts\apply_demo_seed.ps1
       ```

---

### 🟢 Langkah 2: Konfigurasi & Jalankan Backend (Go)

1. **Salin File Environment**:
   Salin berkas template konfigurasi lingkungan di dalam folder `backend`:
   ```bash
   cp .env.example .env
   ```

2. **Sesuaikan Konfigurasi `.env`**:
   Buka file `.env` yang baru dibuat dengan teks editor Anda, lalu sesuaikan nilai variabelnya:
   ```env
   # Pengaturan Dasar Aplikasi
   APP_ENV=development
   HTTP_PORT=8080

   # String URL Koneksi PostgreSQL (Sesuaikan username, password, host, port, dan dbname Anda)
   DATABASE_URL=postgres://postgres:postgres@localhost:5432/pekan?sslmode=disable

   # Keamanan Kunci Sesi (Gunakan string acak, panjang, dan rahasia untuk lingkungan produksi)
   JWT_SECRET=rahasia-jwt-32-karakter-atau-lebih
   RECEIPT_SCAN_SECRET=rahasia-ocr-yang-kuat
   
   # Konfigurasi Opsional Redis (Jika Redis Server terpasang)
   # RATE_LIMIT_REDIS_URL=redis://localhost:6379
   ```

3. **Unduh Dependensi Backend**:
   Pastikan dependensi pustaka Go telah sinkron dan terunduh:
   ```bash
   go mod tidy
   ```

4. **Jalankan API Server Utama**:
   ```bash
   go run cmd/api/main.go
   ```
   *API Server backend akan berjalan secara lokal di alamat `http://localhost:8080`.*

5. **Jalankan Background Worker (Penting untuk Fitur AI / WhatsApp)**:
   Buka terminal baru, masuk ke direktori `backend`, lalu jalankan:
   ```bash
   go run cmd/worker/main.go
   ```
   *Worker antrean ini bertanggung jawab memproses integrasi chat WhatsApp, pengiriman notifikasi, dan pemrosesan pemindaian struk berbasis AI.*

---

### 🔵 Langkah 3: Konfigurasi & Jalankan Frontend (React + Vite)

1. **Navigasi ke Direktori Frontend**:
   Buka terminal baru, lalu masuk ke folder `frontend`:
   ```bash
   cd frontend
   ```

2. **Salin File Environment Frontend**:
   Salin berkas template lingkungan frontend:
   ```bash
   cp .env.example .env
   ```
   *Variabel default di dalamnya (`VITE_API_BASE_URL=http://localhost:8080/api/v1`) sudah diarahkan secara otomatis ke port default API server.*

3. **Pemasangan Dependensi Node.js**:
   Instal semua paket dependensi yang dibutuhkan oleh React + Vite:
   ```bash
   npm install
   ```

4. **Jalankan Server Pengembangan (Dev Mode)**:
   ```bash
   npm run dev
   ```
   *Aplikasi frontend akan terkompilasi secara instan dan dapat Anda akses melalui peramban (browser) di alamat `http://localhost:5173`.*

---

### 🔑 Informasi Login Uji Coba (Demo Credentials)

Jika Anda menjalankan langkah **Seeding Demo (Langkah 1.3)** di atas, Anda dapat langsung melakukan login ke sistem menggunakan kredensial pengujian berikut:

* **URL Dashboard**: `http://localhost:5173`
* **Workspace / Tenant Code**: `default`
* **Alamat Email**: `owner@pekan.local`
* **Kata Sandi (Password)**: `password`
* **Hak Akses**: `Owner` (Memiliki otoritas penuh atas Workspace default)

---

## 🧪 Menjalankan Pengujian (Testing)

Proyek ini dilengkapi dengan skenario pengujian unit (*unit test*) untuk memverifikasi fungsionalitas kritis backend seperti otentikasi sesi, autorisasi RBAC, validasi token penyegaran (*refresh token rotation*), dan audit log.

Untuk menjalankan seluruh rangkaian pengujian backend secara otomatis:
```bash
cd backend
go test ./tests/... -v
```

---

## 🚀 Panduan Deployment ke Server Produksi

Bagi Anda yang ingin mendeploy sistem **PEKAN** ke server *staging* atau *production* (Ubuntu/Debian atau Rocky Linux), kami telah menyediakan skrip automasi penginstalan yang tangguh di direktori `deploy/`. Skrip tersebut akan mengonfigurasi Systemd service, Nginx reverse proxy, PostgreSQL, dan Redis secara otomatis:

* **Ubuntu / Debian Server**: `deploy/install_server.sh`
* **Rocky Linux Server**: `deploy/install_server_rocky.sh`
* **Panduan Penginstalan Server**: Lihat panduan lengkap di [docs/SERVER-INSTALLER.md](docs/SERVER-INSTALLER.md).

---

## 📄 Lisensi

Proyek ini dirilis secara publik sebagai perangkat lunak open-source di bawah naungan lisensi **[Apache License 2.0](LICENSE)**. Anda bebas menggunakan, memodifikasi, mendistribusikan, dan menerapkan kode ini dalam proyek Anda, baik secara komersial maupun non-komersial, asalkan mematuhi seluruh syarat dan ketentuan lisensi Apache 2.0.

---

## 👥 Kontribusi

Kontribusi dari komunitas sangat kami hargai! Jika Anda menemukan bug, ingin mengajukan fitur baru, atau meningkatkan kualitas dokumentasi ini, silakan ikuti alur berikut:
1. **Fork** repositori ini.
2. Buat branch fitur baru (`git checkout -b feature/FiturKeren`).
3. Lakukan commit perubahan Anda (`git commit -m 'Menambahkan fitur keren yang bermanfaat'`).
4. Push ke branch Anda (`git push origin feature/FiturKeren`).
5. Buat sebuah **Pull Request** baru ke branch utama (`main`) proyek.
