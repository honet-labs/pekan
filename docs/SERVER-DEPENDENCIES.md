# Server Dependencies

## Required
- Go 1.23+ (recommended 1.25.8)
- PostgreSQL 14+

## Recommended
- `migrate` CLI or equivalent DB migration runner
- `golangci-lint`
- PostgreSQL client `psql` (for `backend/scripts/apply_migrations.*`)
- Docker + Docker Compose plugin (for `deploy/docker-compose.server-test.yml`)

## Optional for production
- Redis (rate limit/distributed cache)
- S3-compatible object storage
- Malware scan engine integration
- Background worker process (`go run ./cmd/worker`)

## Security notes
- Auth rate limit middleware supports Redis backend via `RATE_LIMIT_REDIS_URL`.
- If Redis is unavailable, limiter falls back to in-memory store.
- Configure CORS allowlist via `CORS_ALLOWED_ORIGINS` (comma separated, no `*` in production).
- Configure global non-multipart body limit via `REQUEST_BODY_MAX_BYTES`.
- Refresh token rotation now tracks token reuse and revokes all user sessions on reuse detection.

## Go modules used
- `github.com/go-chi/chi/v5` (router)
- `github.com/golang-jwt/jwt/v5` (JWT parsing)
- `github.com/jackc/pgx/v5/stdlib` (PostgreSQL driver)
- `github.com/google/uuid` (ID generation)
- `github.com/redis/go-redis/v9` (distributed rate limiting backend)
- `golang.org/x/crypto/bcrypt` (password hashing verification)
- `github.com/DATA-DOG/go-sqlmock` (repository and entitlement query tests)
