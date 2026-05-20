# PEKAN Frontend Scaffold

Feature-based React architecture for multi-tenant SaaS.

Implemented baseline:
- app shell + guarded routes
- access-aware sidebar
- finance transactions pages (list/create/detail)
- transaction create form uses finance master data endpoints (accounts/categories)
- finance master data page (`/finance/master-data`) for account/category management
- API client auto refresh token retry on 401
- API client targeting `/api/v1`

## Run
1. Copy `.env.example` to `.env`
2. Install dependencies: `npm install`
3. Start dev server: `npm run dev`
