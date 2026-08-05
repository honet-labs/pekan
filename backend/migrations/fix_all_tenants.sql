-- MASTER REPAIR SCRIPT FOR ALL TENANTS
-- This script synchronizes all tenant schemas with the latest v1.0.4 structure.
-- It fixes missing tables, missing columns, and inconsistent constraints.

DO $$ 
DECLARE 
    schema_record RECORD;
BEGIN 
    FOR schema_record IN 
        SELECT schema_name 
        FROM information_schema.schemata 
        WHERE schema_name LIKE 'wkspid_pekan_%' 
    LOOP
        RAISE NOTICE 'Fixing schema: %', schema_record.schema_name;

        -- 1. FINANCE TABLES (CORE)
        EXECUTE format('
            CREATE TABLE IF NOT EXISTS %I.finance_accounts (
                id UUID PRIMARY KEY,
                name VARCHAR(200) NOT NULL,
                account_type VARCHAR(50) NOT NULL,
                currency CHAR(3) NOT NULL DEFAULT ''IDR'',
                opening_balance_minor BIGINT NOT NULL DEFAULT 0,
                is_active BOOLEAN NOT NULL DEFAULT TRUE,
                created_by UUID NOT NULL,
                created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
                updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
            );
        ', schema_record.schema_name);

        EXECUTE format('
            CREATE TABLE IF NOT EXISTS %I.finance_categories (
                id UUID PRIMARY KEY,
                name VARCHAR(200) NOT NULL,
                category_type VARCHAR(20) NOT NULL CHECK (category_type IN (''income'', ''expense'')),
                parent_id UUID NULL REFERENCES %I.finance_categories(id),
                is_active BOOLEAN NOT NULL DEFAULT TRUE,
                created_by UUID NOT NULL,
                created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
                updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
            );
        ', schema_record.schema_name, schema_record.schema_name);

        EXECUTE format('
            CREATE TABLE IF NOT EXISTS %I.finance_transactions (
                id UUID PRIMARY KEY,
                account_id UUID NOT NULL REFERENCES %I.finance_accounts(id),
                category_id UUID NULL REFERENCES %I.finance_categories(id),
                type VARCHAR(20) NOT NULL CHECK (type IN (''income'', ''expense'', ''transfer'', ''savings'')),
                amount_minor BIGINT NOT NULL CHECK (amount_minor > 0),
                currency CHAR(3) NOT NULL,
                input_date TIMESTAMPTZ NOT NULL DEFAULT now(),
                transaction_date DATE NOT NULL,
                description TEXT NULL,
                merchant_name TEXT NULL,
                receipt_number TEXT NULL,
                payment_method TEXT NULL,
                subtotal_minor BIGINT NOT NULL DEFAULT 0,
                tax_minor BIGINT NOT NULL DEFAULT 0,
                service_charge_minor BIGINT NOT NULL DEFAULT 0,
                receipt_discount_minor BIGINT NOT NULL DEFAULT 0,
                created_by UUID NOT NULL,
                updated_by UUID NOT NULL,
                deleted_at TIMESTAMPTZ NULL,
                created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
                updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
            );
        ', schema_record.schema_name, schema_record.schema_name, schema_record.schema_name);

        -- 2. ADD MISSING COLUMNS TO TRANSACTIONS (v1.0.3+ Upgrade)
        EXECUTE format('
            ALTER TABLE %I.finance_transactions ADD COLUMN IF NOT EXISTS subtotal_minor BIGINT NOT NULL DEFAULT 0;
            ALTER TABLE %I.finance_transactions ADD COLUMN IF NOT EXISTS tax_minor BIGINT NOT NULL DEFAULT 0;
            ALTER TABLE %I.finance_transactions ADD COLUMN IF NOT EXISTS service_charge_minor BIGINT NOT NULL DEFAULT 0;
            ALTER TABLE %I.finance_transactions ADD COLUMN IF NOT EXISTS receipt_discount_minor BIGINT NOT NULL DEFAULT 0;
            ALTER TABLE %I.finance_transactions ADD COLUMN IF NOT EXISTS notes TEXT;
        ', schema_record.schema_name, schema_record.schema_name, schema_record.schema_name, schema_record.schema_name, schema_record.schema_name);

        -- 3. SAVINGS & BUDGETS
        EXECUTE format('
            CREATE TABLE IF NOT EXISTS %I.finance_savings (
                id UUID PRIMARY KEY,
                sid VARCHAR(20) NOT NULL,
                name VARCHAR(200) NOT NULL,
                target_amount_minor BIGINT NOT NULL,
                current_amount_minor BIGINT NOT NULL DEFAULT 0,
                currency CHAR(3) NOT NULL DEFAULT ''IDR'',
                start_date DATE NULL,
                target_date DATE NULL,
                status VARCHAR(20) NOT NULL DEFAULT ''active'',
                notes TEXT NULL,
                created_by UUID NOT NULL,
                updated_by UUID NOT NULL,
                created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
                updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
                deleted_at TIMESTAMPTZ NULL
            );
        ', schema_record.schema_name);

        EXECUTE format('
            CREATE TABLE IF NOT EXISTS %I.finance_budgets (
                id UUID PRIMARY KEY,
                ida VARCHAR(20) NOT NULL,
                name VARCHAR(200) NOT NULL,
                category_id UUID NOT NULL REFERENCES %I.finance_categories(id),
                amount_limit_minor BIGINT NOT NULL,
                currency CHAR(3) NOT NULL DEFAULT ''IDR'',
                period VARCHAR(20) NOT NULL CHECK (period IN (''monthly'', ''weekly'', ''yearly'')),
                start_date DATE NOT NULL,
                end_date DATE NULL,
                alert_threshold_pct INT NULL,
                status VARCHAR(20) NOT NULL DEFAULT ''active'',
                notes TEXT NULL,
                created_by UUID NOT NULL,
                updated_by UUID NOT NULL,
                created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
                updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
                deleted_at TIMESTAMPTZ NULL
            );
        ', schema_record.schema_name, schema_record.schema_name);

        -- 4. REMINDERS
        EXECUTE format('
            CREATE TABLE IF NOT EXISTS %I.finance_reminders (
                id UUID PRIMARY KEY,
                title VARCHAR(255) NOT NULL,
                description TEXT NULL,
                amount_minor BIGINT NOT NULL,
                currency CHAR(3) NOT NULL DEFAULT ''IDR'',
                due_date DATE NOT NULL,
                repeat_interval VARCHAR(20) NOT NULL DEFAULT ''none'',
                status VARCHAR(20) NOT NULL DEFAULT ''pending'',
                total_tenor INTEGER DEFAULT 0,
                current_tenor INTEGER DEFAULT 0,
                last_triggered_at TIMESTAMPTZ NULL,
                created_by UUID NOT NULL,
                updated_by UUID NOT NULL,
                created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
                updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
                deleted_at TIMESTAMPTZ NULL
            );
        ', schema_record.schema_name);

        EXECUTE format('
            CREATE TABLE IF NOT EXISTS %I.finance_reminder_payments (
                id UUID PRIMARY KEY,
                reminder_id UUID NOT NULL REFERENCES %I.finance_reminders(id) ON DELETE CASCADE,
                paid_at TIMESTAMPTZ NOT NULL DEFAULT now(),
                amount_minor BIGINT NOT NULL,
                status VARCHAR(20) NOT NULL DEFAULT ''completed'',
                notes TEXT NULL,
                proof_image_url TEXT NULL,
                created_by UUID NOT NULL,
                updated_by UUID NOT NULL,
                created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
                updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
            );
        ', schema_record.schema_name, schema_record.schema_name);

        -- 5. ATTACHMENTS & SCAN JOBS
        EXECUTE format('
            CREATE TABLE IF NOT EXISTS %I.finance_entity_attachments (
                id UUID PRIMARY KEY,
                owner_type VARCHAR(50) NOT NULL,
                owner_id UUID NOT NULL,
                file_id UUID NOT NULL,
                created_by UUID NOT NULL,
                created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
                UNIQUE (owner_type, owner_id, file_id)
            );
        ', schema_record.schema_name);

        EXECUTE format('
            CREATE TABLE IF NOT EXISTS %I.finance_receipt_scan_jobs (
                id UUID PRIMARY KEY,
                file_id UUID NOT NULL,
                status VARCHAR(20) NOT NULL,
                result_json JSONB NULL,
                error_message TEXT NULL,
                created_by UUID NOT NULL,
                created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
                updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
            );
        ', schema_record.schema_name);

        -- 6. LINKS & RELATIONSHIPS
        EXECUTE format('
            CREATE TABLE IF NOT EXISTS %I.finance_transaction_savings_links (
                id UUID PRIMARY KEY,
                transaction_id UUID NOT NULL REFERENCES %I.finance_transactions(id) ON DELETE CASCADE,
                savings_id UUID NOT NULL REFERENCES %I.finance_savings(id),
                allocated_amount_minor BIGINT NOT NULL DEFAULT 0,
                created_by UUID NOT NULL,
                created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
                UNIQUE (transaction_id, savings_id)
            );
        ', schema_record.schema_name, schema_record.schema_name, schema_record.schema_name);

        -- 7. WHATSAPP & SECURITY
        EXECUTE format('
            CREATE TABLE IF NOT EXISTS %I.whatsapp_otp_tokens (
                token VARCHAR(10) PRIMARY KEY,
                user_id UUID NOT NULL,
                expires_at TIMESTAMPTZ NOT NULL,
                created_at TIMESTAMPTZ NOT NULL DEFAULT now()
            );
        ', schema_record.schema_name);

        EXECUTE format('
            CREATE TABLE IF NOT EXISTS %I.whatsapp_sessions (
                phone_number VARCHAR(20) PRIMARY KEY,
                user_id UUID NOT NULL,
                last_active TIMESTAMPTZ NOT NULL DEFAULT now(),
                created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
                updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
            );
        ', schema_record.schema_name);

        -- 8. SETTINGS
        EXECUTE format('
            CREATE TABLE IF NOT EXISTS %I.finance_settings (
                default_currency CHAR(3) NOT NULL DEFAULT ''IDR'',
                fiscal_year_start_month INTEGER NOT NULL DEFAULT 1,
                multi_currency_enabled BOOLEAN NOT NULL DEFAULT FALSE,
                updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
                id_fixed INTEGER PRIMARY KEY DEFAULT 1 CHECK (id_fixed = 1)
            );
        ', schema_record.schema_name);

        -- 9. TRANSACTION TYPE CONSTRAINT SYNC
        EXECUTE format('
            ALTER TABLE %I.finance_transactions DROP CONSTRAINT IF EXISTS finance_transactions_type_check;
            ALTER TABLE %I.finance_transactions ADD CONSTRAINT finance_transactions_type_check 
                CHECK (type IN (''income'', ''expense'', ''transfer'', ''savings''));
        ', schema_record.schema_name, schema_record.schema_name);

    END LOOP;
END $$;
