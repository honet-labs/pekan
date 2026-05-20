# 🏢 PEKAN SaaS (Pusat Kendali & Keuangan SaaS)

[![Go Version](https://img.shields.io/badge/Go-1.23-00ADD8?style=flat-square&logo=go)](https://golang.org/)
[![React Version](https://img.shields.io/badge/React-18-61DAFB?style=flat-square&logo=react)](https://react.dev/)
[![Vite](https://img.shields.io/badge/Vite-5-646CFF?style=flat-square&logo=vite)](https://vitejs.dev/)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-15%2B-4169E1?style=flat-square&logo=postgresql)](https://www.postgresql.org/)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg?style=flat-square)](LICENSE)

**PEKAN SaaS** (Pusat Kendali & Keuangan SaaS) adalah platform pengelolaan keuangan berskala korporat/SaaS modern yang dirancang dengan arsitektur tangguh, terukur, dan aman. Platform ini memadukan kekuatan **Go (Golang)** di bagian backend dengan **React + Vite** di bagian frontend, serta dilengkapi dengan fitur asisten keuangan cerdas berbasis **Kecerdasan Buatan (AI)** melalui integrasi **WhatsApp** dan **OCR Receipt Scanner**.

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
saas-pekan/
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

### 📋 Prasyarat Sistem
Sebelum memulai instalasi, pastikan komputer Anda telah terpasang perangkat lunak berikut:
* **Go (Golang)** versi 1.23 atau lebih tinggi.
* **Node.js** versi 18 atau lebih tinggi (disertai NPM/PNPM).
* **PostgreSQL** versi 15 atau lebih tinggi.
* **Redis Server** versi 7 atau lebih tinggi (opsional, digunakan untuk optimasi Rate Limiting & caching).

---

### 🟢 Langkah 1: Pengaturan Backend (Go)

1. Masuk ke direktori backend:
   ```bash
   cd backend
   ```

2. Salin berkas template lingkungan `.env.example` ke `.env`:
   ```bash
   cp .env.example .env
   ```

3. Buka berkas `.env` dan sesuaikan konfigurasinya:
   ```env
   # Pengaturan Dasar Server
   APP_ENV=development
   HTTP_PORT=8080

   # Koneksi PostgreSQL Database
   DATABASE_URL=postgres://username:password@localhost:5432/pekan?sslmode=disable

   # Kunci Keamanan JWT (Gunakan string acak panjang di production)
   JWT_SECRET=rahasia-jwt-32-karakter-atau-lebih
   RECEIPT_SCAN_SECRET=rahasia-ocr-yang-kuat
   ```

4. Jalankan Migrasi Database dan Seed Awal:
   Buat database bernama `pekan` di server PostgreSQL lokal Anda, lalu jalankan migration patch yang tersedia di direktori `backend/migrations/` ke database Anda untuk membentuk seluruh struktur tabel secara otomatis. 
   *(Opsional: masukkan seed data dari `backend/seeds/` untuk memuat data demo awal).*

5. Jalankan API Server Utama:
   ```bash
   go run cmd/api/main.go
   ```
   *Server backend akan mulai berjalan di `http://localhost:8080`.*

6. Jalankan Background AI/WhatsApp Worker (di terminal baru):
   ```bash
   cd backend
   ```
   ```bash
   go run cmd/worker/main.go
   ```
   *Worker antrean ini bertugas memproses antrean pesan masuk dari WhatsApp dan memanggil API AI secara asinkron.*

---

### 🔵 Langkah 2: Pengaturan Frontend (React + Vite)

1. Masuk ke direktori frontend:
   ```bash
   cd ../frontend
   ```

2. Install seluruh dependensi yang diperlukan:
   ```bash
   npm install
   ```

3. Jalankan server pengembangan lokal (development mode):
   ```bash
   npm run dev
   ```
   *Aplikasi frontend akan berjalan dan dapat diakses melalui browser Anda di alamat `http://localhost:5173`.*

---

## 🧪 Menjalankan Pengujian (Testing)

Untuk memastikan seluruh logika otentikasi, validasi keamanan kata sandi, dan fungsionalitas platform berjalan dengan baik, Anda dapat menjalankan test suite Go backend:

```bash
cd backend
go test ./tests/... -v
```

---

## 📄 Lisensi

Proyek ini dirilis secara publik sebagai perangkat lunak open-source di bawah naungan **[Apache License 2.0](LICENSE)**. Anda bebas menggunakan, memodifikasi, mendistribusikan, dan menerapkan kode ini dalam proyek Anda, baik komersial maupun non-komersial, asalkan tetap mematuhi seluruh syarat dan ketentuan lisensi Apache 2.0.

---

## 👥 Kontribusi

Kontribusi dari komunitas sangat dihargai! Jika Anda menemukan bug, ingin mengajukan fitur baru, atau ingin meningkatkan kualitas dokumentasi, silakan:
1. Fork repositori ini.
2. Buat branch fitur baru (`git checkout -b feature/FiturKeren`).
3. Lakukan commit perubahan Anda (`git commit -m 'Menambahkan fitur keren'`).
4. Push ke branch Anda (`git push origin feature/FiturKeren`).
5. Buat sebuah **Pull Request** ke branch utama proyek.
