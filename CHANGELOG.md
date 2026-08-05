# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased] - 2026-05-02

### Added
- **Finance Dashboard Privacy**: Implementasi fitur "Hide/View" nominal saldo (Total Income, Expense, Savings) untuk menjaga privasi data. Nominal disembunyikan secara default.
- **Advanced Transaction Search**: Dukungan pencarian transaksi berdasarkan nama item di dalam struk dan nominal harga (amount).
- **Transaction Visibility**: Integrasi tombol "Lihat Transaksi" pada modul Tabungan dan Anggaran untuk mempermudah audit transaksi tanpa masuk ke mode edit.
- **Reminder Payment Management**: Fitur manajemen riwayat pembayaran (CRUD) pada modul Pengingat Pembayaran, memungkinkan user mengoreksi atau menghapus catatan pembayaran.
- **Receipt Preview**: Peningkatan antarmuka pratinjau bukti pembayaran (PDF/Image) pada riwayat pengingat.

### Changed
- **UI/UX Standardization**: Standarisasi tombol aksi di seluruh modul keuangan (menggunakan teks "Lihat Transaksi" daripada ikon mata tunggal).
- **Transaction Filtering**: Optimasi backend untuk mendukung filtering transaksi berdasarkan Category ID dan Savings ID secara lebih efisien.

### Fixed
- **Admin Portal Translation**: Perbaikan label translasi yang masih muncul sebagai key mentah (`admin.server.service_name`, dll) pada halaman Status Server.
- **Dashboard Privacy**: Mengganti tombol global hide nominal dengan tombol individual pada setiap kartu (Pemasukan, Pengeluaran, Tabungan) untuk kontrol yang lebih fleksibel.
- **Reminders Attachment**: Implementasi penuh pengunggahan dan tampilan bukti pembayaran (gambar) pada riwayat pembayaran pengingat.
- **Admin AI Configuration**: Penambahan fitur fetch models otomatis dan autocomplete untuk provider Sumopod (Custom).
- **TypeScript Build Fixes**: Perbaikan error redeclarasi variabel pada `AdminDashboardPage.tsx` dan masalah destructuring/missing imports pada modul Pengingat.
- **SMTP Connection Diagnostics**: Identifikasi isu timeout koneksi SMTP pada port 465 terkait penggunaan proxy Cloudflare.
