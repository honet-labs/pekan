# Server Test Runbook (API + Worker)

Runbook ini untuk menyiapkan backend PEKAN sampai siap dites di server (single node) dengan PostgreSQL + Redis.

Alternatif otomatis (installer all-in-one):

```bash
chmod +x deploy/install_server.sh
./deploy/install_server.sh
```

Untuk Rocky Linux:

```bash
chmod +x deploy/install_server_rocky.sh
./deploy/install_server_rocky.sh
```

Catatan Rocky installer terbaru:
- Deploy backend API + worker.
- Build frontend React dan publish ke Nginx (default `http://<server-ip>`).
- Proxy `/api/*` dari Nginx ke backend `127.0.0.1:8080`.

Detail opsi installer:
- `docs/SERVER-INSTALLER.md`

## 1) Prasyarat server
- Go `1.23+` (direkomendasikan `1.25.8`)
- PostgreSQL client (`psql`)
- Docker + Docker Compose plugin (opsional tapi direkomendasikan untuk infra cepat)

## 2) Jalankan infra (PostgreSQL + Redis)
Di root project:

```bash
docker compose -f deploy/docker-compose.server-test.yml up -d
```

Pastikan service `pekan-postgres` dan `pekan-redis` healthy.

## 3) Siapkan environment backend
Di folder `backend`:

```bash
cp .env.server.example .env
```

Minimal cek variabel berikut:
- `DATABASE_URL`
- `JWT_SECRET` (wajib random kuat, min 32 chars untuk production)
- `CORS_ALLOWED_ORIGINS`
- `RATE_LIMIT_REDIS_URL`

## 4) Install dependency Go
Di folder `backend`:

```bash
go mod tidy
```

Opsional tapi direkomendasikan (pre-flight check):

```bash
go test ./...
```

## 5) Apply migration
Catatan:
- Untuk local Docker Compose test, `sslmode=disable` bisa dipakai.
- Untuk production DB dengan TLS aktif, gunakan `sslmode=require` atau setara.

Linux/macOS:

```bash
cd backend
chmod +x ./scripts/apply_migrations.sh
DATABASE_URL="postgres://postgres:postgres@127.0.0.1:5432/pekan?sslmode=disable" \
  ./scripts/apply_migrations.sh
```

Windows PowerShell:

```powershell
cd backend
$env:DATABASE_URL="postgres://postgres:postgres@127.0.0.1:5432/pekan?sslmode=disable"
.\scripts\apply_migrations.ps1
```

## 6) Bootstrap data auth/RBAC minimum
Cara cepat (direkomendasikan untuk server test):

Linux/macOS:

```bash
cd backend
chmod +x ./scripts/apply_demo_seed.sh
DATABASE_URL="postgres://postgres:postgres@127.0.0.1:5432/pekan?sslmode=disable" \
  ./scripts/apply_demo_seed.sh
```

Windows PowerShell:

```powershell
cd backend
$env:DATABASE_URL="postgres://postgres:postgres@127.0.0.1:5432/pekan?sslmode=disable"
.\scripts\apply_demo_seed.ps1
```

Credential demo default:
- email: `owner@pekan.local`
- password: `password`
- tenant_id: `11111111-1111-1111-1111-111111111111`

Jika ingin bootstrap manual, gunakan:
- `docs/AUTH-RBAC-BOOTSTRAP.md`

## 7) Jalankan API dan worker
Terminal 1:

```bash
cd backend
go run ./cmd/api
```

Terminal 2:

```bash
cd backend
go run ./cmd/worker
```

## 8) Smoke test cepat
Health check:

```bash
curl -i http://localhost:8080/api/v1/healthz
```

Login:

```bash
curl -s -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"owner@pekan.local","password":"password","tenant_id":"11111111-1111-1111-1111-111111111111"}'
```

Jika login sukses, uji endpoint protected dengan access token:

```bash
curl -i http://localhost:8080/api/v1/me/context \
  -H "Authorization: Bearer <ACCESS_TOKEN>"
```

## 9) Checklist siap test
- [ ] `healthz` mengembalikan `200`
- [ ] login berhasil dan token keluar
- [ ] `/me/context` berhasil dengan token valid
- [ ] finance master endpoint (`/finance/accounts`) mengembalikan `200`
- [ ] upload attachment berhasil (`scan_status` awal `pending`)
- [ ] worker berjalan dan scan status terproses
- [ ] refresh token reuse menghasilkan `401 REFRESH_TOKEN_REUSED`
- [ ] rate limiting aktif (uji burst login -> `429`)

## 10) Shutdown
Infra docker:

```bash
docker compose -f deploy/docker-compose.server-test.yml down
```

Rocky quick uninstall:

```bash
chmod +x deploy/uninstall_server_rocky.sh
./deploy/uninstall_server_rocky.sh
```

## 11) Opsional: jalankan via systemd
Template unit tersedia:
- `deploy/systemd/pekan-api.service`
- `deploy/systemd/pekan-worker.service`

Sesuaikan `WorkingDirectory`, `EnvironmentFile`, `ExecStart`, dan user/group sesuai server Anda sebelum enable service.

## 12) Jika pakai Cloudflared Tunnel
- Pastikan backend bisa diakses lokal dulu:
  - `curl -i http://127.0.0.1:8080/api/v1/healthz`
- Pastikan `CORS_ALLOWED_ORIGINS` di `/opt/pekan/backend/.env` berisi domain tunnel frontend Anda.
- Restart service setelah ubah `.env`:
  - `systemctl restart pekan-api pekan-worker`
- Contoh `ingress` cloudflared untuk API:
  - `hostname: api.domainanda.com`
  - `service: http://localhost:8080`

## Catatan Frontend Build Manual
- Jika repo belum punya `package-lock.json`, `npm ci` akan gagal.
- Gunakan:
  - `npm install --include=dev`
  - `npm run build`
