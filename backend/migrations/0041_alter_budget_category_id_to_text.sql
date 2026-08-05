-- Migration 0041: Alter finance_budgets category_id column to TEXT to support multiple categories
-- Date: 2026-06-07

-- 1. Drop foreign key constraint and alter category_id column in active schema
ALTER TABLE finance_budgets DROP CONSTRAINT IF EXISTS finance_budgets_category_id_fkey;
ALTER TABLE finance_budgets ALTER COLUMN category_id TYPE TEXT USING category_id::text;

-- 2. Loop through all tenant schemas to apply the same change
DO $$
DECLARE
    tenant_record RECORD;
    v_schema_name TEXT;
BEGIN
    FOR tenant_record IN 
        SELECT id, code FROM public.tenants
    LOOP
        -- Skip system/public tenants if they don't have schemas
        IF tenant_record.code = 'public' THEN
            CONTINUE;
        END IF;

        IF LOWER(tenant_record.code) = 'pekan' OR LOWER(tenant_record.code) = 'pekanhonet' THEN
            v_schema_name := 'wkspid_pekan_pekanhonet';
        ELSE
            v_schema_name := 'wkspid_pekan_' || LOWER(tenant_record.code);
        END IF;
        
        -- Check if schema exists and contains the table
        IF EXISTS (
            SELECT 1 
            FROM information_schema.tables 
            WHERE table_schema = LOWER(v_schema_name) 
              AND table_name = 'finance_budgets'
        ) THEN
            RAISE NOTICE '=== Altering finance_budgets in schema: % ===', v_schema_name;

            EXECUTE format('
                ALTER TABLE %I.finance_budgets DROP CONSTRAINT IF EXISTS finance_budgets_category_id_fkey;
                ALTER TABLE %I.finance_budgets ALTER COLUMN category_id TYPE TEXT USING category_id::text;
            ', v_schema_name, v_schema_name);
        END IF;
    END LOOP;
END $$;
