# PEKAN Technical Blueprint (Modular Multi-Tenant)

## 1) System Architecture (End-to-End)

### Architectural Style
- **Backend**: modular monolith (single deployable service) with strict module boundaries.
- **Frontend**: React feature-based architecture, API-first, permission-aware UI.
- **Data**: PostgreSQL shared database, shared schema, strict `tenant_id` scoping.
- **Storage**: abstraction layer with providers: local (default), S3 (recommended production), Google Drive (optional connector).

### Logical Components
- **Web App (ReactJS)**: app shell, auth, tenant switcher, module-aware navigation, feature guards.
- **API Service (Golang)**:
  - Platform core modules: identity, tenant, entitlement, RBAC, audit, storage.
  - Product modules: finance (transactions, savings, budgets, reminders, reports, attachments).
- **PostgreSQL**: tenant-aware core and domain tables.
- **Object Storage**: pluggable provider via storage port.
- **Async Worker (same codebase/binary profile optional)**: notification dispatch, reminder jobs, report generation, file virus scan queue.

### Request Pipeline (API-First)
1. `HTTP -> RequestID middleware -> Auth middleware -> Tenant context middleware -> Permission/Feature guard -> Handler`
2. Handler maps request DTO -> usecase.
3. Usecase enforces business policy and tenant boundary.
4. Repository executes tenant-scoped query.
5. Audit middleware and/or usecase logger writes audit event.
6. Response DTO returned (never expose DB entity directly).

### Deployment Baseline
- 1 service (API) + 1 PostgreSQL + 1 storage provider.
- Horizontal scale API stateless nodes behind load balancer.
- Use Redis later for cache/rate-limit/session blacklist if needed.
- Start modular monolith first; extract service only when module team/load justifies.

## 2) Core Platform vs Product Modules

### Core Platform (must exist first)
- Identity & Access: users, sessions, JWT/refresh token, password policy, MFA-ready hooks.
- Tenant management: tenant, memberships, tenant settings, tenant switch.
- RBAC: roles, permissions, role-permission mapping, user-role assignment per tenant.
- Module catalog: products/modules/features.
- Entitlements: tenant modules/features baseline + feature overrides.
- Audit logging: actor, action, target, before/after summary, IP, user agent.
- Storage service: upload/download metadata, signed URL/stream, provider abstraction.

### Product Module (Finance)
- Transactions
- Savings
- Budgets/Goals
- Payment reminders
- Reports (PDF/CSV generation)
- Receipts/attachments
- Dashboard analytics

## 3) Multi-Tenant Model

### Isolation Strategy
- Single DB, shared schema, **mandatory `tenant_id`** on all tenant-owned tables.
- Every repository query must include `tenant_id = :tenant_id`.
- API context must always contain `actor_id`, `tenant_id`, `role set`, `permission set`, `entitlements`.
- Optional hardening: PostgreSQL RLS as second line of defense (recommended phase 2+).

### Tenant Context Rules
- User can belong to multiple tenants.
- Access token stores: `sub`, `active_tenant_id`, `membership_id`, `roles_version`.
- Tenant switch endpoint issues new access token with selected tenant context.
- Forbidden when tenant is suspended/deleted.

## 4) Backend Golang Structure (Modular Monolith, Clean Architecture)

```text
backend/
  cmd/
    api/
      main.go
    worker/
      main.go
  internal/
    platform/
      config/
      logger/
      httpx/
      auth/
      tenancy/
      rbac/
      entitlement/
      audit/
      storage/
      db/
      errors/
      observability/
    modules/
      finance/
        transactions/
          domain/
            entity.go
            repository.go
            policy.go
          usecase/
            create_transaction.go
            list_transactions.go
            update_transaction.go
            delete_transaction.go
          infra/
            repository_pg.go
            mapper.go
          delivery/
            http/
              handler.go
              request.go
              response.go
              routes.go
          tests/
            usecase_test.go
            repository_test.go
        savings/
        budgets/
        reminders/
        reports/
    app/
      bootstrap.go
      router.go
      module_registry.go
  migrations/
  docs/
  go.mod
```

### Package Boundaries
- `domain`: business entities, repository port interfaces, domain rules.
- `usecase`: orchestrates domain logic + policy enforcement.
- `infra`: implementation details (PostgreSQL, external clients).
- `delivery/http`: transport adapter (DTO validation, mapping, status code).
- `platform/*`: reusable platform capabilities only (no finance logic).

### Rules
- Handler thin (parse/validate/map only).
- No SQL in usecase/handler.
- Repository only persistence concern.
- Usecase must receive `TenantContext` and enforce authorization + entitlement.

## 5) Frontend ReactJS Structure (Feature-Based, Scalable)

```text
frontend/
  src/
    app/
      App.tsx
      providers/
      router/
      guards/
      layout/
    core/
      api/
        client.ts
        interceptors.ts
      auth/
      tenant/
      rbac/
      entitlement/
      storage/
      ui/
      utils/
    features/
      finance/
        transactions/
          pages/
          components/
          hooks/
          api/
          schemas/
          mappers/
        savings/
        budgets/
        reminders/
        reports/
    shared/
      components/
      hooks/
      types/
      constants/
```

### App Shell & Routing
- `ProtectedRoute`: requires authenticated user.
- `TenantScopedRoute`: requires active tenant context.
- `FeatureRouteGuard`: checks module + feature entitlement + permission from server profile payload.
- Sidebar/menu built from server-driven module entitlement payload, not hardcoded.

## 6) API Design `/api/v1` (Versioned REST)

### Core Endpoints
- `POST /api/v1/auth/login`
- `POST /api/v1/auth/refresh`
- `POST /api/v1/auth/logout`
- `GET /api/v1/me/context` (user profile + memberships + active tenant + permissions + features)
- `POST /api/v1/tenants/{tenantId}/switch`
- `GET /api/v1/tenants/{tenantId}/modules`
- `GET /api/v1/tenants/{tenantId}/features`

### Finance Transactions Endpoints (Vertical Slice 1)
- `POST /api/v1/finance/transactions`
- `GET /api/v1/finance/transactions?from=...&to=...&type=expense&page=1&page_size=20`
- `GET /api/v1/finance/transactions/{transactionId}`
- `PUT /api/v1/finance/transactions/{transactionId}`
- `DELETE /api/v1/finance/transactions/{transactionId}`
- `POST /api/v1/finance/transactions/{transactionId}/attachments`
- `GET /api/v1/finance/transactions/{transactionId}/attachments/{attachmentId}/download`

### API Response Pattern
- Success:
```json
{
  "data": {},
  "meta": {
    "request_id": "req_123",
    "timestamp": "2026-03-23T10:00:00Z"
  }
}
```
- Error:
```json
{
  "error": {
    "code": "FORBIDDEN_FEATURE_LOCKED",
    "message": "Feature finance.transactions.write is not enabled for tenant",
    "details": null
  },
  "meta": {
    "request_id": "req_123"
  }
}
```

## 7) Auth, RBAC, Feature Entitlement, Tenant Isolation Flow

### Auth Flow
1. User login by email/password.
2. Verify credential and membership.
3. Issue access token (short TTL) + refresh token (rotating, hashed in DB).
4. Client calls `/me/context` to load permissions, modules, features.
5. Every request includes bearer token.

### Authorization Order (Backend Enforced)
1. Authentication valid.
2. Active tenant valid and membership active.
3. Module enabled for tenant.
4. Feature entitlement unlocked for tenant config/override.
5. Permission check (RBAC action-level).
6. Object-level check (resource belongs to same tenant and accessible scope).

### Example Policy for Create Transaction
- Required module: `finance`.
- Required feature: `finance.transactions.write`.
- Required permission: `finance.transactions.create`.
- Required tenant scope: payload `tenant_id` ignored from client, injected from token context.

## 8) Storage Abstraction Design (Local/S3/GDrive)

### Storage Port
```go
type ObjectStorage interface {
    Put(ctx context.Context, in PutObjectInput) (PutObjectResult, error)
    GetDownloadURL(ctx context.Context, in GetObjectInput) (string, error)
    OpenStream(ctx context.Context, in GetObjectInput) (io.ReadCloser, ObjectMeta, error)
    Delete(ctx context.Context, in DeleteObjectInput) error
}
```

### Provider Strategy
- `local`: default dev/small deployments (`/var/app-data/{tenant_id}/{module}/...`).
- `s3`: production default (private bucket, SSE, lifecycle policy, pre-signed URL).
- `gdrive`: optional connector, never primary for critical production path.

### File Metadata
- Always save metadata in DB (`files` table) including tenant, owner, checksum, MIME, size, provider, object key, scan status.
- Download authorization checks tenant + permission before issuing URL/stream.

## 9) Module Enable-Disable & Entitlement Flow

### Enable/Disable Flow
1. Default tenant config activates products/modules/features.
2. Overrides can unlock/lock specific feature.
3. `entitlement_resolver` composes final effective entitlements (module baseline + feature baseline + override + expiry).
5. Backend guard uses effective entitlements for every request.

## 10) Audit Log Design

### Logged Events
- Login success/failure, token refresh, tenant switch.
- Role/permission changes.
- Entitlement/feature changes.
- CRUD operations for sensitive domain data (transactions, budgets, savings).
- File upload/download/delete.

### Audit Fields
- `tenant_id`, `actor_user_id`, `action`, `resource_type`, `resource_id`,
  `before_json`, `after_json`, `ip_address`, `user_agent`, `request_id`, `created_at`.

### Principles
- Immutable append-only.
- Redact sensitive fields (password, token, secrets).
- Centralized helper in platform audit package.

## 11) Vertical Slice 1: Finance Transactions

### Domain Entity (high-level)
- `Transaction`: id, tenant_id, account_id, category_id, amount, currency, type(income/expense/transfer), transaction_date, note, created_by, updated_by.

### Usecases
- `CreateTransaction`
- `ListTransactions`
- `GetTransactionDetail`
- `UpdateTransaction`
- `DeleteTransaction` (soft delete recommended for auditability)
- `AttachReceipt`

### Handler Responsibilities
- Decode request, validate DTO.
- Build usecase input with tenant context.
- Call usecase.
- Map output to response DTO.

### Usecase Responsibilities
- Check entitlement + permission.
- Validate domain invariants (non-negative amount rules, account ownership, date boundaries).
- Call repository transaction-safe operations.
- Emit audit event.

### Repository Responsibilities
- Execute SQL with explicit tenant filter.
- Never return cross-tenant rows.
- Support pagination, filtering, sorting with whitelist.

## 12) Security Baseline (Production-Minded)

### High Priority Controls
- Access token short TTL (10-15 min), refresh rotation with revocation.
- Password hashing Argon2id/Bcrypt strong params.
- Strict CORS allowlist.
- Global and per-tenant rate limit.
- Object-level authorization on every resource endpoint.
- SQL parameterized queries only.
- File upload allowlist MIME + extension + size + malware scan queue.
- Private object storage by default (never public bucket for receipts).
- DB least privilege account per environment.
- Secrets from secret manager/env injection, never hardcoded.

### Additional Hardening
- CSP, HSTS, X-Content-Type-Options, X-Frame-Options.
- Idempotency key for sensitive write endpoints.
- Optional PostgreSQL RLS for defense in depth.
- Backup encryption + restore drills + tenant-scoped restore strategy.

## 13) Clean Code & Testing Conventions

### Conventions
- Package names explicit, no ambiguous `utils` for business logic.
- DTO in delivery layer, entities in domain layer.
- Interfaces owned by consumer package (usecase owns repository port).
- Error taxonomy with typed domain/application errors.
- No shared mutable global state in business code.

### Testing Strategy
- Unit tests for usecase (mock repository + policy checker).
- Repository integration tests with PostgreSQL test container.
- API contract tests for `/api/v1`.
- Authorization matrix tests (role x feature x module x tenant state).
- Security tests: BOLA/BFLA cases, token replay, file upload abuse.

## 14) Development Roadmap (Safe and Realistic)

### Phase 0: Foundations
- Repo setup, CI, lint, formatting, migrations framework, config management.
- Basic observability: structured logs, metrics, request ID.

### Phase 1: Core Platform MVP
- Auth + tenant membership + tenant switch.
- RBAC engine.
- Module catalog + entitlement resolver.
- Audit logging framework.
- Storage abstraction + local provider.

### Phase 2: Vertical Slice Finance Transactions
- Transaction CRUD + list filters.
- Transaction attachment upload/download (local first).
- Frontend transaction list/form/detail with guards.
- API integration tests and authorization matrix.

### Phase 3: Finance Expansion
- Savings, budgets/goals, reminders.
- Notifications worker.
- Reports CSV then PDF.

### Phase 4: Production Hardening
- S3 provider, background scan, rate limiting distributed.
- Security review closure, backup/restore drill, SLO dashboards.
- Performance tuning, indexing refinement, query observability.

### Phase 5: Scale and Extensibility
- Introduce module SDK conventions for future products.
- Feature rollout strategy (tenant cohort, kill switch).
- Optional split hot modules into separate services if needed.
