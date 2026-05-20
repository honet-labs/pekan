# Test Setup

## Backend tests
- Storage provider test file:
  - `backend/tests/storage_provider_test.go`
- Auth security test file:
  - `backend/tests/auth_security_test.go`
- Auth refresh reuse test file:
  - `backend/tests/auth_refresh_reuse_test.go`
- Authorization matrix test file:
  - `backend/tests/authorization_matrix_test.go`
- Repository tenant-scope test file:
  - `backend/tests/transactions_repository_tenant_test.go`
- Entitlement resolver test file:
  - `backend/tests/entitlement_resolver_test.go`
- HTTP endpoint test file (`httptest`):
  - `backend/tests/http_endpoints_test.go`
- HTTP security middleware test file:
  - `backend/tests/http_security_middleware_test.go`
- Rate limit middleware test file:
  - `backend/tests/rate_limit_middleware_test.go`
- Redis rate limit store test file:
  - `backend/tests/rate_limit_redis_store_test.go`
- Config validation security test file:
  - `backend/tests/config_validation_test.go`
- Attachment security test file:
  - `backend/tests/attachment_security_test.go`
- File scan scanner test file:
  - `backend/tests/filescan_scanner_test.go`

## Run tests (when Go installed)
```bash
cd backend
go test ./...
```

## Test focus for next steps
- Usecase unit tests for:
  - permission denied
  - feature locked
  - tenant mismatch prevention
- Auth usecase tests:
  - invalid credential
  - refresh token rotation
  - refresh token reuse detection (revoke all sessions)
  - tenant switch membership check
- Attachment tests:
  - MIME allowlist reject/accept
  - extension and MIME mismatch reject
  - file size limit
  - scan_status gate before download (pending/infected)
  - cross-tenant download denial
  - EICAR signature detection baseline
- Repository integration tests for tenant-scoped queries.
- API tests for BOLA/BFLA scenarios.
- Rate limiter tests:
  - in-memory block after limit
  - store unavailable returns `429` fail-safe
