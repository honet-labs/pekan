# Setup Index

## Backend
- Path: `backend`
- Copy env: `cp .env.example .env` (or manual on Windows)
- Sync Go dependencies: `cd backend && go mod tidy`
- Server test runbook:
  - `docs/SERVER-TEST-RUNBOOK.md`
- Server env template:
  - `backend/.env.server.example`
- Server infra compose (PostgreSQL + Redis):
  - `deploy/docker-compose.server-test.yml`
- Automated server installer:
  - `deploy/install_server.sh`
  - `deploy/install_server_rocky.sh`
  - `deploy/uninstall_server_rocky.sh`
  - `docs/SERVER-INSTALLER.md`
  - Rocky installer juga build + publish frontend WebUI via Nginx (default `:80`)
- Optional systemd templates:
  - `deploy/systemd/pekan-api.service`
  - `deploy/systemd/pekan-worker.service`
- Security env baseline:
  - `CORS_ALLOWED_ORIGINS` (comma separated)
  - `REQUEST_BODY_MAX_BYTES` (default `1048576`)
  - `RATE_LIMIT_REDIS_URL` (optional for distributed auth rate limiting)
  - `RATE_LIMIT_REDIS_PREFIX` (optional, default `pekan:ratelimit`)
- Apply migrations:
  - `backend/migrations/0001_init.sql`
  - `backend/migrations/0002_access_control_and_sessions.sql`
  - `backend/migrations/0003_seed_permissions.sql`
  - `backend/migrations/0004_subscription_catalog.sql`
  - `backend/migrations/0005_finance_master_and_fk.sql`
  - `backend/migrations/0006_seed_modules_features_plans.sql`
  - `backend/migrations/0007_remove_subscription_tables.sql`
  - `backend/migrations/0008_tenant_scoped_foreign_keys.sql`
  - `backend/migrations/0009_file_scan_jobs.sql`
  - `backend/migrations/0010_seed_attachment_scan_permission.sql`
  - `backend/migrations/0011_auth_refresh_token_rotation_hardening.sql`
- Bootstrap auth/RBAC data:
  - `docs/AUTH-RBAC-BOOTSTRAP.md`
- Optional quick demo data:
  - `backend/seeds/001_demo_tenant.sql`
  - `docs/DEMO-SEED.md`
  - Linux/macOS: `backend/scripts/apply_demo_seed.sh`
  - Windows: `backend/scripts/apply_demo_seed.ps1`
- Run API (after Go installed): `go run ./cmd/api`
- Run worker (optional, attachment scan async): `go run ./cmd/worker`
- Migration scripts:
  - Linux/macOS: `backend/scripts/apply_migrations.sh`
  - Windows: `backend/scripts/apply_migrations.ps1`

## Frontend
- Path: `frontend`
- Copy env: `cp .env.example .env`
- Install deps: `npm install`
- Run dev server: `npm run dev`
- Untuk server build tanpa `package-lock.json`, gunakan `npm install --include=dev` (bukan `npm ci`)

## API Contract
- OpenAPI spec: `backend/openapi/openapi.yaml`
- Postman collection: `docs/postman/PEKAN-API.postman_collection.json`
- Postman environment: `docs/postman/PEKAN-LOCAL.postman_environment.json`
- Postman usage: `docs/POSTMAN-USAGE.md`

## First Vertical Slice
- Module: Finance Transactions
- Endpoint group: `/api/v1/finance/transactions`
- Auth endpoints:
  - `POST /api/v1/auth/login`
  - `POST /api/v1/auth/refresh`
  - `POST /api/v1/auth/logout-all`
  - `GET /api/v1/me/context`
- Finance master endpoints:
  - `GET /api/v1/finance/accounts`
  - `POST /api/v1/finance/accounts`
  - `GET /api/v1/finance/categories`
  - `POST /api/v1/finance/categories`
- Entitlement endpoints:
  - `GET /api/v1/tenants/{tenantID}/entitlements/effective`
  - `POST /api/v1/tenants/{tenantID}/feature-overrides`
