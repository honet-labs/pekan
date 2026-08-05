# Server Installer

Installer otomatis tersedia untuk:
- Ubuntu/Debian: `deploy/install_server.sh`
- Rocky Linux family: `deploy/install_server_rocky.sh`

Fitur installer:
- install dependency (`curl`, `psql`, dll)
- install Docker + Compose
- install Go
- install Node.js + Nginx (Rocky installer)
- sync project ke server path
- generate `.env` backend
- start PostgreSQL + Redis (docker compose)
- `go mod tidy`, optional `go test`
- apply migration + optional demo seed
- build binary API/worker
- build frontend React (fallback `npm install` if lockfile tidak ada)
- publish web ke Nginx (`/var/www/pekan-web`, proxy `/api` ke backend)
- install + start systemd service

## Rocky Linux (VM Anda)
```bash
chmod +x deploy/install_server_rocky.sh
./deploy/install_server_rocky.sh
```

Contoh custom Rocky:
```bash
./deploy/install_server_rocky.sh \
  --app-env development \
  --http-port 8080 \
  --cors https://app.example.com
```

Uninstall Rocky:
```bash
chmod +x deploy/uninstall_server_rocky.sh
./deploy/uninstall_server_rocky.sh
```

Uninstall Rocky (bersih total):
```bash
./deploy/uninstall_server_rocky.sh \
  --remove-install-dir \
  --remove-data \
  --remove-user-group \
  --remove-docker-images
```

## Ubuntu/Debian
```bash
chmod +x deploy/install_server.sh
./deploy/install_server.sh
```

Contoh custom Ubuntu/Debian:
```bash
./deploy/install_server.sh \
  --install-dir /opt/pekan \
  --app-env production \
  --http-port 8080 \
  --cors https://app.example.com \
  --jwt-secret "replace-with-strong-secret-min-32-char" \
  --database-url "postgres://postgres:postgres@127.0.0.1:5432/pekan?sslmode=disable" \
  --redis-url "redis://127.0.0.1:6379/0"
```

## Flag penting
- `--skip-tests`: lewati `go test ./...`
- `--skip-seed`: lewati demo seed
- `--no-enable-services`: hanya install unit systemd, tidak start service
- `--skip-web`: lewati frontend build + nginx publish
- `--web-port <port>`: set port web Nginx (default `80`)
- `--web-root <path>`: set folder publish frontend (default `/var/www/pekan-web`)
- `--frontend-api-base-url <url>`: set `VITE_API_BASE_URL` saat build frontend (default `/api/v1`)

## Catatan
- Default installer memakai `APP_ENV=development` agar cocok dengan setup PostgreSQL docker lokal (`sslmode=disable`).
- Jika ingin `APP_ENV=production`, pastikan `DATABASE_URL` dan security env sesuai validasi backend.
