# Backend Implementation Guide (Golang, Modular Monolith)

## 1) Package Boundaries

```text
internal/
  platform/
    auth/
    tenancy/
    rbac/
    entitlement/
    audit/
    storage/
    db/
    httpx/
  modules/
    finance/
      transactions/
        domain/
        usecase/
        infra/
        delivery/http/
```

Rules:
- `platform/*` tidak boleh impor package domain finance.
- `usecase` boleh impor `domain` + port interface platform.
- `delivery/http` hanya panggil usecase.
- `infra/repository_pg` implement interface dari `domain`.

## 2) Domain Layer Example (Transactions)

```go
package domain

import (
    "context"
    "time"
)

type TransactionType string

const (
    TransactionTypeIncome   TransactionType = "income"
    TransactionTypeExpense  TransactionType = "expense"
    TransactionTypeTransfer TransactionType = "transfer"
)

type Transaction struct {
    ID              string
    TenantID        string
    AccountID       string
    CategoryID      *string
    Type            TransactionType
    Amount          string
    Currency        string
    TransactionDate time.Time
    Description     *string
    CreatedBy       string
    UpdatedBy       string
    CreatedAt       time.Time
    UpdatedAt       time.Time
}

type ListFilter struct {
    TenantID string
    Type     *TransactionType
    DateFrom *time.Time
    DateTo   *time.Time
    Limit    int
    Offset   int
}

type Repository interface {
    Create(ctx context.Context, trx Transaction) (Transaction, error)
    GetByID(ctx context.Context, tenantID, transactionID string) (Transaction, error)
    List(ctx context.Context, filter ListFilter) ([]Transaction, int64, error)
    Update(ctx context.Context, trx Transaction) (Transaction, error)
    SoftDelete(ctx context.Context, tenantID, transactionID, deletedBy string) error
}
```

## 3) Usecase Layer Example

```go
package usecase

import (
    "context"
    "errors"

    "pekan/internal/modules/finance/transactions/domain"
)

type AuthzChecker interface {
    EnsurePermission(ctx context.Context, permission string) error
    EnsureFeature(ctx context.Context, featureCode string) error
    EnsureModule(ctx context.Context, moduleCode string) error
}

type AuditWriter interface {
    Write(ctx context.Context, action, resourceType, resourceID string, before, after any) error
}

type CreateTransaction struct {
    Repo  domain.Repository
    Authz AuthzChecker
    Audit AuditWriter
}

func (uc *CreateTransaction) Execute(ctx context.Context, in CreateTransactionInput) (domain.Transaction, error) {
    if err := uc.Authz.EnsureModule(ctx, "finance"); err != nil { return domain.Transaction{}, err }
    if err := uc.Authz.EnsureFeature(ctx, "finance.transactions.write"); err != nil { return domain.Transaction{}, err }
    if err := uc.Authz.EnsurePermission(ctx, "finance.transactions.create"); err != nil { return domain.Transaction{}, err }

    if in.Amount <= 0 {
        return domain.Transaction{}, errors.New("amount must be greater than zero")
    }

    trx := mapInputToDomain(in)
    created, err := uc.Repo.Create(ctx, trx)
    if err != nil { return domain.Transaction{}, err }

    _ = uc.Audit.Write(ctx, "finance.transaction.create", "finance_transaction", created.ID, nil, created)
    return created, nil
}
```

## 4) Repository Layer Example (PostgreSQL)

```go
func (r *RepositoryPG) GetByID(ctx context.Context, tenantID, transactionID string) (domain.Transaction, error) {
    const q = `
SELECT id, tenant_id, account_id, category_id, type, amount, currency, transaction_date, description, created_by, updated_by, created_at, updated_at
FROM finance_transactions
WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL`
    // scan row...
}
```

Rules:
- Selalu include `tenant_id` di `WHERE`.
- Query list wajib whitelist sortable fields.
- Tidak membangun SQL raw dari input user tanpa sanitasi.

## 5) HTTP Handler Layer Example (Thin Handler)

```go
func (h *Handler) CreateTransaction(w http.ResponseWriter, r *http.Request) {
    var req CreateTransactionRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        httpx.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
        return
    }
    if err := h.validator.Struct(req); err != nil {
        httpx.WriteValidationError(w, err)
        return
    }

    ctxData := tenancy.MustContext(r.Context())
    out, err := h.createUC.Execute(r.Context(), CreateTransactionInput{
        TenantID:        ctxData.TenantID,
        ActorUserID:     ctxData.UserID,
        AccountID:       req.AccountID,
        CategoryID:      req.CategoryID,
        Type:            req.Type,
        Amount:          req.Amount,
        Currency:        req.Currency,
        TransactionDate: req.TransactionDate,
        Description:     req.Description,
    })
    if err != nil {
        httpx.WriteMappedError(w, err)
        return
    }
    httpx.WriteJSON(w, http.StatusCreated, ToTransactionResponse(out))
}
```

## 6) Middleware Chain

Recommended order:
1. `RequestID`
2. `Recovery`
3. `AccessLog`
4. `AuthJWT`
5. `TenantResolver` (membership + active tenant)
6. `ModuleFeatureGuard` (opsional per route group)
7. `PermissionGuard` (opsional per route)
8. `AuditTrail` (write events, with action from route context)

## 7) Storage Provider Config

```yaml
storage:
  default_provider: local
  local:
    base_path: /var/lib/pekan/storage
  s3:
    endpoint: ""
    region: ap-southeast-1
    bucket: ""
    force_path_style: false
    kms_key_id: ""
  gdrive:
    enabled: false
    credentials_json: ""
    root_folder_id: ""
```

## 8) Testing Best Practices

- Unit tests:
  - usecase transactions (success, validation fail, forbidden, feature locked).
- Integration tests:
  - repository CRUD + tenant boundary.
- API tests:
  - endpoint returns 403 when permission missing.
  - endpoint returns 404 for cross-tenant resource.
- Security tests:
  - upload file disallowed mime.
  - download attachment from other tenant denied.

