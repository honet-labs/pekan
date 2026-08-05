# Frontend Architecture (ReactJS, Multi-Tenant, Module-Aware)

## 1) Folder Structure (Feature-Based)

```text
src/
  app/
    App.tsx
    bootstrap.tsx
    router/
      index.tsx
      route-config.ts
    layout/
      AppShell.tsx
      Sidebar.tsx
      Header.tsx
    guards/
      ProtectedRoute.tsx
      TenantRoute.tsx
      FeatureGuard.tsx
      PermissionGuard.tsx
  core/
    api/
      client.ts
      auth-interceptor.ts
      error-mapper.ts
    auth/
      auth-store.ts
      useAuth.ts
    tenant/
      tenant-store.ts
      useTenant.ts
    access/
      access-store.ts
      access-selectors.ts
    ui/
      components/
      theme/
  features/
    finance/
      transactions/
        pages/
          TransactionListPage.tsx
          TransactionCreatePage.tsx
          TransactionDetailPage.tsx
        components/
          TransactionForm.tsx
          TransactionTable.tsx
          TransactionFilters.tsx
        hooks/
          useTransactionList.ts
          useCreateTransaction.ts
        api/
          transaction.api.ts
          transaction.types.ts
        state/
          transaction-query-keys.ts
        schemas/
          transaction.schema.ts
  shared/
    types/
    constants/
    lib/
```

## 2) App Shell Architecture

- `AppShell`:
  - Header: tenant switcher, profile, notifications.
  - Sidebar: generated from access profile (`enabledModules`, `enabledFeatures`, `permissions`).
  - Main content routed by `react-router`.

- UI visibility and action control:
  - Menu hidden jika module disabled.
  - Button/action disabled jika permission/feature missing.
  - Backend tetap menjadi source of truth untuk authorization.

## 3) Routing Strategy

Example routes:
- `/login`
- `/app/:tenantCode/dashboard`
- `/app/:tenantCode/finance/transactions`
- `/app/:tenantCode/finance/transactions/new`
- `/app/:tenantCode/finance/transactions/:id`

Guards order:
1. `ProtectedRoute`
2. `TenantRoute` (valid tenant membership)
3. `FeatureGuard("finance.transactions.read")`
4. page render

## 4) Auth + Tenant-Aware Flow

1. Login success -> save tokens securely.
2. Fetch `/api/v1/me/context`.
3. Set active tenant (default or selected last tenant).
4. Render app shell with tenant context.
5. On tenant switch -> call `/api/v1/tenants/{tenantId}/switch`, refresh context.

## 5) Dynamic Sidebar/Menu

Source menu from backend payload:
- `modules`: `finance`, `finance.savings`, etc.
- `features`: `finance.transactions.read`, `finance.reports.export`.
- `permissions`: action-level.

Do not hardcode entitlements on frontend only. Frontend reads access profile and adapts UI.

## 6) Permission Guard Pattern

```tsx
<PermissionGuard permission="finance.transactions.create">
  <CreateTransactionButton />
</PermissionGuard>
```

```tsx
<FeatureGuard feature="finance.transactions.write">
  <Route element={<TransactionCreatePage />} />
</FeatureGuard>
```

## 7) API Client Strategy

- Single API client wrapper (Axios/fetch wrapper).
- Inject bearer token automatically.
- Retry once on 401 with refresh flow.
- Standardized error mapping by `error.code`.
- Include `X-Request-ID` when available.

Current implementation notes:
- `frontend/src/core/api/client.ts` already supports automatic refresh token retry.
- `frontend/src/app/App.tsx` bootstraps `/me/context` on app load when token exists.

## 8) State Management Recommendation

- **Server state**: TanStack Query.
- **Client state**: Zustand/Redux Toolkit (tenant context, auth summary, UI preferences).
- Keep form state local (`react-hook-form` + schema validation zod/yup).

## 9) Finance Transactions Module Example

### API functions
- `createTransaction(payload)`
- `getTransactionList(params)`
- `getTransactionDetail(id)`
- `updateTransaction(id, payload)`
- `deleteTransaction(id)`
- `uploadAttachment(transactionId, file)`

### UI Composition
- `TransactionListPage`:
  - uses `TransactionFilters`, `TransactionTable`, pagination.
- `TransactionCreatePage`:
  - uses reusable `TransactionForm`.
  - account/category loaded from finance master data endpoints.
- `TransactionDetailPage`:
  - shows detail + attachment section + audit summary snippet.

### Finance Master Data Page
- Route: `/app/:tenantCode/finance/master-data`
- Supports list/create for accounts and categories.
- Guarded by:
  - feature `finance.masterdata.read`
  - permission `finance.accounts.read` (route access)
  - create actions also check create permissions in UI.

## 10) Frontend Clean Code Rules

- Business rule ringan di hooks/service, bukan komponen tampilan besar.
- Component presentasional tidak memanggil API langsung.
- DTO mapping di `features/*/api` atau `mappers`.
- Hindari folder `utils` generik untuk logic domain.
- Setiap fitur punya boundary jelas dan testable.
