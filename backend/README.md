# PEKAN Backend Scaffold

This backend implements the first production-minded vertical slice:
- modular monolith structure
- multi-tenant-aware request context
- RBAC/feature/module authorization checks in usecase
- audit logging to PostgreSQL
- finance transactions CRUD (`/api/v1/finance/transactions`)
- auth flow (`/api/v1/auth/login`, `/api/v1/auth/refresh`, `/api/v1/auth/logout`, `/api/v1/auth/logout-all`, `/api/v1/me/context`)
- transaction attachments upload/download with storage abstraction
- finance master data (`/api/v1/finance/accounts`, `/api/v1/finance/categories`)
- tenant-level entitlement resolver + feature override endpoints (free SaaS, no subscription billing)
- perimeter hardening middleware (CORS allowlist, security headers, body limit, auth rate limit)
- auth rate limiter supports distributed Redis backend (optional) with in-memory fallback
- async attachment scan queue + worker baseline
- refresh token reuse detection + revoke-all sessions response hardening

## Run
One-command installer for Ubuntu/Debian server:
- `../deploy/install_server.sh`
Rocky Linux installer:
- `../deploy/install_server_rocky.sh`

1. Copy `.env.example` to `.env`
   - For server test baseline, you can start from `.env.server.example`.
2. Sync dependencies:
   - `go mod tidy`
3. Set security env values:
   - `CORS_ALLOWED_ORIGINS` (comma separated)
   - `REQUEST_BODY_MAX_BYTES` (non-multipart global body limit)
   - `RATE_LIMIT_REDIS_URL` (optional, for multi-instance shared rate limiting)
   - `RATE_LIMIT_REDIS_PREFIX` (optional key prefix)
4. Apply migrations using script:
   - Linux/macOS: `./scripts/apply_migrations.sh`
   - Windows: `.\scripts\apply_migrations.ps1`
5. Start API:
   - `go run ./cmd/api`
6. Start worker (optional):
   - `go run ./cmd/worker`

## Bootstrap access data
- Use `../docs/AUTH-RBAC-BOOTSTRAP.md` to create tenant/user/membership/role/module/feature baseline.
- Optional demo seed: `seeds/001_demo_tenant.sql` (see `../docs/DEMO-SEED.md`).
  - Linux/macOS helper: `./scripts/apply_demo_seed.sh`
  - Windows helper: `.\scripts\apply_demo_seed.ps1`
- Full server test sequence: `../docs/SERVER-TEST-RUNBOOK.md`.

## Important
- Every repository query is tenant-scoped.
- Handlers are thin; business rules are in usecase/domain.
- `openapi/openapi.yaml` is the REST contract baseline.
