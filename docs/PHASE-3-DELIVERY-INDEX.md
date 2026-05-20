# Phase 3 Delivery Index

This file tracks deliverables after core + transaction vertical slice.

## Planned Phase 3 Scope
- Finance savings goals
- Finance budgets/goals
- Payment reminders + worker scheduler
- Notification delivery hooks

## Dependencies from current scaffold
- Core auth/tenant context middleware
- RBAC + entitlement guard in usecase
- Audit logger
- Storage abstraction interface
- Async file scan job table + worker baseline
- Refresh token reuse detection + logout-all session invalidation
- Redis-backed auth rate limiting (optional, with in-memory fallback)

## Recommended implementation order
1. Savings module skeleton
2. Budgets module with period validation
3. Reminder scheduler worker
4. Notification channels
5. E2E tests per feature
