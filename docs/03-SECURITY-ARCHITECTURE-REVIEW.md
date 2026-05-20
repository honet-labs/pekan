# Security Architecture Review - PEKAN SaaS

## 1) Ancaman Utama dan Prioritas

### High
- Cross-tenant data leakage (missing `tenant_id` filter).
- Broken object level authorization (BOLA) pada endpoint detail/update/delete.
- Broken function level authorization (BFLA) pada endpoint admin/entitlement/RBAC.
- Token theft/replay (refresh token reuse, long-lived token).
- Unsafe file upload/download (malicious file, path traversal, unauthorized download).
- Secrets exposure (hardcoded secret, leaked env in logs).
- SQL injection dan insecure query composition.

### Medium
- Abuse/rate-limit bypass.
- Audit log tampering atau tidak lengkap.
- Insufficient backup encryption/restore control.
- Misconfigured object storage policy (public bucket/object ACL).
- CSRF/CORS misconfiguration (terutama jika pakai cookie-based session).

### Low
- Information disclosure via verbose error.
- Timing/profile leak dari endpoint enumerasi.
- Non-critical dependency vulnerabilities.

## 2) Mitigasi Teknis

### Tenant Isolation
- Wajib `tenant_id` pada semua query domain.
- Query helper yang selalu menerima `TenantContext`.
- Negative tests cross-tenant di CI.
- Optional RLS untuk defense in depth.

### Auth/Session/Token
- Access token 10-15 menit.
- Refresh token rotation + token family tracking.
- Simpan hash refresh token di DB, bukan plaintext.
- Revoke all sessions on password reset/suspicious login.

### RBAC + Entitlement Enforcement
- Backend guard chain: `auth -> tenant -> module -> feature -> permission -> object ownership`.
- Jangan rely hide UI saja.
- Permission matrix tests untuk endpoint sensitif.

### API Security
- Input validation ketat (schema-based).
- Rate limiting per IP + per user + per tenant.
- Idempotency key untuk endpoint write penting.
- Centralized error mapping (no stack traces to client).

### Database Security
- Least privilege DB user (no superuser).
- SSL/TLS DB connection.
- Migration role terpisah dari app role.
- Encrypted backups + restore access control.

### File Security
- Allowlist MIME + extension + max size.
- Rename stored file (no trust original filename/path).
- Malware scan async, status `pending/clean/infected`.
- Download wajib auth + tenant scope + permission check.
- Storage private by default.

### Audit & Monitoring
- Append-only audit logs.
- Log auth failures, RBAC denial, entitlement denial.
- SIEM/alerting untuk anomaly (brute force, high download rate, repeated denied access).
- Protect logs from PII leakage.

### Secrets Management
- Gunakan secret manager atau environment injection dari platform.
- Rotasi secret berkala.
- Larang commit secret via pre-commit + scanner.

## 3) Checklist Implementasi

### High Priority Checklist
- [ ] Semua tabel domain punya `tenant_id`.
- [ ] Semua repository method menerima `tenantID`.
- [ ] Semua endpoint sensitif punya object-level authorization.
- [x] Auth token rotation dan revocation aktif.
- [x] Rate limiting aktif.
- [ ] Upload/download file dengan authorization check backend.
- [ ] Audit log untuk create/update/delete + login events.
- [ ] Secrets tidak disimpan di repo.

### Medium Priority Checklist
- [ ] RLS pilot di tabel finance kritikal.
- [ ] Malware scanning pipeline untuk file.
- [ ] Backup restore drill per kuartal.
- [ ] Threat model review tiap module baru.

### Low Priority Checklist
- [ ] Security headers hardening lengkap.
- [ ] Dependency scanning + SBOM.
- [ ] Chaos/security game day.

## 4) Rekomendasi Best Practice Relevan Stack

- React:
  - Simpan token dengan strategi yang meminimalkan XSS impact.
  - Escape output UI, no unsafe HTML injection.
- Golang:
  - Context propagation untuk request ID, actor, tenant.
  - Gunakan typed errors + centralized middleware.
  - Hindari reflection-heavy generic helper untuk logic kritikal.
- PostgreSQL:
  - Composite index diawali `tenant_id`.
  - Perhatikan query plan untuk list transactions high-volume.
- S3/Storage:
  - Private bucket, SSE enabled, short-lived presigned URL.
  - Folder/object key by tenant scope.

## 5) Security Gate per Fase

- Phase 1 gate:
  - Auth + tenant isolation tests pass.
  - RBAC minimal path pass.
- Phase 2 gate:
  - Finance transaction BOLA/BFLA tests pass.
  - File upload/download security pass.
- Phase 3 gate:
  - Reminder/report worker secure processing pass.
  - Audit completeness check pass.
