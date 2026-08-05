# PEKAN Deployment & Operations Guide

Dokumentasi ini menjelaskan penggunaan skrip otomatisasi di folder `deploy/` untuk instalasi, pembaruan, pencadangan, dan pemeliharaan platform PEKAN.

---

## 1. Instalasi Awal (`install_server.sh`)

Gunakan skrip ini untuk setup pertama kali atau saat ingin mengubah konfigurasi utama.

### Mode Docker (Default)
Menggunakan kontainer Docker untuk PostgreSQL dan Redis.
```bash
sudo bash deploy/install_server.sh --app-env production --cors "https://domain.anda"
```

### 3. Update Existing Installation

To update the application code and rebuild without changing your current configuration:

```bash
sudo bash deploy/update.sh
```

This script will sync the latest code, rebuild the backend and frontend, apply new migrations, and restart services while preserving your `.env` settings and database data.

### Mode Standalone (Native)
Menginstal PostgreSQL dan Redis langsung di sistem operasi host.
```bash
sudo bash deploy/install_server.sh --infra-mode standalone --app-env production
```

**Parameter Penting:**
*   `--infra-mode <docker|standalone>`: Pilih arsitektur infrastruktur.
*   `--app-env <production|development>`: Mode aplikasi.
*   `--database-url "<url>"`: URL koneksi PostgreSQL.
*   `--cors "<origins>"`: Daftar domain yang diizinkan (dipisahkan koma).

### Instalasi Modular (Multi-Server)
Anda dapat memisahkan komponen platform ke beberapa server berbeda menggunakan flag komponen:

*   **Setup Database Server Saja**:
    ```bash
    sudo bash deploy/install_server.sh --database --infra-mode standalone
    ```
*   **Setup Web & App Server Saja** (Menghubungkan ke DB remote):
    ```bash
    sudo bash deploy/install_server.sh --app --web --database-url "postgres://user:pass@remote-db-ip:5432/pekan?sslmode=require"
    ```
*   **Flag Tersedia**:
    *   `--database` atau `--only-db`: Instalasi PostgreSQL saja.
    *   `--redis` atau `--only-redis`: Instalasi Redis saja.
    *   `--app` atau `--only-app`: Instalasi Go API & Worker saja.
    *   `--web` atau `--only-web`: Instalasi Nginx & Frontend saja.
    *   `--all-services`: Instalasi seluruh komponen (default).

*   **Parameter Database Detil**:
    *   `--db-user <user>`: Nama pengguna database (default: `postgres`).
    *   `--db-pass <pass>`: Kata sandi database (default: `postgres`).
    *   `--db-name <name>`: Nama database (default: `pekan`).
    *   `--db-host <host>`: Host database (default: `127.0.0.1`).
    *   `--db-port <port>`: Port database (default: `5432`).
    *   `--db-schema <schema>`: Schema database (default: `public`).
    *   *Catatan: `--database-url` tetap memiliki prioritas tertinggi jika diberikan.*

---

## 2. Pembaruan Aplikasi (`update_app.sh`)

Gunakan skrip ini untuk memperbarui fitur atau bug fix tanpa mengubah konfigurasi yang sudah ada.

```bash
sudo bash deploy/update_app.sh
```

**Fungsi:**
*   Sinkronisasi kode terbaru.
*   **Menjaga file `.env` tetap aman** (tidak overwrite).
*   Rebuild Backend & Frontend.
*   Menjalankan migrasi database otomatis.
*   Restart service `pekan-api` dan `pekan-worker`.

---

## 3. Pencadangan Data (`backup.sh`)

Lakukan backup secara rutin untuk mengamankan data Anda.

```bash
sudo bash deploy/backup.sh
```

**Output:**
File backup berupa `.tar.gz` akan disimpan di `/opt/pekan/backups/` dengan format nama `pekan_backup_YYYYMMDD_HHMMSS.tar.gz`.

---

## 4. Pemulihan Data (`restore.sh`)

Mengembalikan seluruh data dari file backup tertentu.

```bash
sudo bash deploy/restore.sh /opt/pekan/backups/pekan_backup_20260501_120000.tar.gz
```

**Peringatan:** Proses ini akan menimpa database dan file storage yang ada saat ini dengan data dari backup.

---

## 5. Penghapusan Sistem (`uninstall_server.sh`)

Digunakan untuk menghapus platform dari server.

### Hapus Total (Full Clean)
Menghapus seluruh file, database, kontainer, dan konfigurasi.
```bash
sudo bash deploy/uninstall_server.sh
```

### Hapus Service Saja (Keep Data)
Hanya menghapus systemd unit dan config Nginx. Data database dan file upload tetap aman.
```bash
sudo bash deploy/uninstall_server.sh --only-services
```

---

## Best Practices

1.  **Sebelum Update**: Selalu jalankan `backup.sh` sebelum menjalankan `update_app.sh`.
2.  **Keamanan**: Simpan file `.env` di tempat yang aman sebagai cadangan manual.
3.  **Logs**: Jika terjadi masalah pada service, gunakan perintah:
    ```bash
    journalctl -u pekan-api -f
    journalctl -u pekan-worker -f
    ```
4.  **Akses**: Pastikan Anda menjalankan skrip ini dengan hak akses `root` atau menggunakan `sudo`.
