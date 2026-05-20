-- Cleanup migration for old builds that used subscription/billing tables.
-- Safe to run even when tables do not exist.

DROP TABLE IF EXISTS plan_features;
DROP TABLE IF EXISTS plan_modules;
DROP TABLE IF EXISTS tenant_subscriptions;
DROP TABLE IF EXISTS plans;

