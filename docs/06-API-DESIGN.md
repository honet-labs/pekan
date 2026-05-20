# REST API Design (/api/v1)

## 1) Conventions

- Base path: `/api/v1`
- Auth: `Authorization: Bearer <access_token>`
- Content-Type: `application/json`
- Pagination: `page`, `page_size`, response `meta.pagination`
- Sorting: whitelist fields only, format `sort=transaction_date:desc`
- Error format standar:

```json
{
  "error": {
    "code": "FORBIDDEN",
    "message": "You do not have permission finance.transactions.create",
    "details": null
  },
  "meta": {
    "request_id": "req_9a8b"
  }
}
```

## 2) Authentication and Context

### `POST /api/v1/auth/login`
Request:
```json
{
  "email": "user@example.com",
  "password": "StrongPass123!"
}
```
Response:
```json
{
  "data": {
    "access_token": "jwt...",
    "refresh_token": "rt_...",
    "expires_in": 900
  }
}
```

### `POST /api/v1/auth/refresh`
Request:
```json
{
  "refresh_token": "rt_..."
}
```

### `GET /api/v1/me/context`
Response:
```json
{
  "data": {
    "user": {
      "id": "u_1",
      "email": "user@example.com",
      "full_name": "Demo User"
    },
    "active_tenant": {
      "id": "t_1",
      "name": "Tenant A",
      "status": "active"
    },
    "memberships": [],
    "modules": ["finance"],
    "features": ["finance.transactions.read", "finance.transactions.write"],
    "permissions": ["finance.transactions.create", "finance.transactions.read"]
  }
}
```

### `POST /api/v1/tenants/{tenantId}/switch`
Purpose:
- Ganti tenant aktif dan issue access token baru dengan tenant context baru.

## 3) Module and Entitlement (Free SaaS)

### `GET /api/v1/tenants/{tenantId}/modules`
- List module status `enabled/disabled`.

### `GET /api/v1/tenants/{tenantId}/features`
- List feature effective result dari tenant config + override.

### `POST /api/v1/tenants/{tenantId}/feature-overrides`
- Admin-only granular unlock/lock.

### `GET /api/v1/tenants/{tenantId}/entitlements/effective`
- Effective modules/features hasil resolver (tenant_modules + tenant_features + tenant_feature_overrides).

## 4) RBAC APIs

### `GET /api/v1/tenants/{tenantId}/roles`
### `POST /api/v1/tenants/{tenantId}/roles`
### `PUT /api/v1/tenants/{tenantId}/roles/{roleId}`
### `POST /api/v1/tenants/{tenantId}/members/{membershipId}/roles`

Rule:
- Semua endpoint RBAC wajib permission admin (misalnya `core.rbac.manage`).

## 5) Finance Transactions APIs

### `POST /api/v1/finance/transactions`
Required:
- Module: `finance`
- Feature: `finance.transactions.write`
- Permission: `finance.transactions.create`

Request:
```json
{
  "account_id": "a_1",
  "category_id": "c_1",
  "type": "expense",
  "amount": "125000.00",
  "currency": "IDR",
  "transaction_date": "2026-03-23",
  "description": "Belanja bulanan"
}
```

Response:
```json
{
  "data": {
    "id": "trx_1",
    "account_id": "a_1",
    "category_id": "c_1",
    "type": "expense",
    "amount": "125000.00",
    "currency": "IDR",
    "transaction_date": "2026-03-23",
    "description": "Belanja bulanan",
    "created_at": "2026-03-23T08:10:00Z"
  }
}
```

### `GET /api/v1/finance/transactions`
Query:
- `from`, `to`, `type`, `account_id`, `category_id`, `page`, `page_size`, `sort`

### `GET /api/v1/finance/transactions/{transactionId}`
### `PUT /api/v1/finance/transactions/{transactionId}`
### `DELETE /api/v1/finance/transactions/{transactionId}`

Object-level authorization:
- Jika `transactionId` milik tenant lain, return `404` atau `403` sesuai kebijakan.

## 6) File Attachment APIs

### `POST /api/v1/finance/transactions/{transactionId}/attachments`
- Multipart upload.
- Validate file type/size.
- Store via storage abstraction.
- Save metadata di `files` + link di `finance_transaction_attachments`.

### `GET /api/v1/finance/transactions/{transactionId}/attachments/{attachmentId}/download`
- Verify auth + tenant + permission.
- Return presigned URL (S3) atau stream (local/gdrive).

## 6b) Finance Master Data APIs

### `GET /api/v1/finance/accounts`
### `POST /api/v1/finance/accounts`
- Permission:
  - `finance.accounts.read`
  - `finance.accounts.create`

### `GET /api/v1/finance/categories`
### `POST /api/v1/finance/categories`
- Permission:
  - `finance.categories.read`
  - `finance.categories.create`

## 7) Audit APIs

### `GET /api/v1/audit-logs`
- Filter by `action`, `resource_type`, `actor_user_id`, `from`, `to`.
- Permission: `core.audit.read`.

## 8) Module Enable-Disable Flow (Request Example)

### `PUT /api/v1/tenants/{tenantId}/modules/{moduleCode}`
Request:
```json
{
  "is_enabled": false,
  "reason": "maintenance_window"
}
```

Effect:
- Guard denies module endpoints immediately after cache invalidation.

## 9) Entitlement Flow (High-Level, Free SaaS)

1. Tenant default modules/features diaktifkan via seed/config awal.
2. Admin tenant dapat override feature tertentu via endpoint override.
3. Resolver menghitung entitlements efektif.
4. API guards enforce entitlements pada setiap request.
