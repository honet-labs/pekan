-- Re-seed tenant modules and features after restore
-- This script ensures all tenants have the required modules and features

DO $$
DECLARE
    tenant_rec RECORD;
    module_codes TEXT[] := ARRAY['transactions', 'savings', 'budgets', 'reminders', 'reports', 'receipts', 'attachments', 'dashboard', 'master', 'notifications', 'settings', 'whatsapp'];
    feature_codes TEXT[] := ARRAY['transactions.create', 'transactions.read', 'transactions.update', 'transactions.delete', 
                                  'savings.create', 'savings.read', 'savings.update', 'savings.delete',
                                  'budgets.create', 'budgets.read', 'budgets.update', 'budgets.delete',
                                  'reminders.create', 'reminders.read', 'reminders.update', 'reminders.delete',
                                  'reports.create', 'reports.read', 'reports.export',
                                  'receipts.create', 'receipts.read',
                                  'attachments.create', 'attachments.read', 'attachments.delete',
                                  'dashboard.read',
                                  'master.create', 'master.read', 'master.update', 'master.delete',
                                  'notifications.read', 'notifications.update',
                                  'settings.read', 'settings.update',
                                  'whatsapp.read', 'whatsapp.update'];
    mc TEXT;
    fc TEXT;
BEGIN
    FOR tenant_rec IN SELECT id, code FROM public.tenants LOOP
        -- Insert modules if not exists
        FOREACH mc IN ARRAY module_codes LOOP
            INSERT INTO public.tenant_modules (id, tenant_id, module_code, is_enabled, source, created_at, updated_at)
            VALUES (gen_random_uuid(), tenant_rec.id, mc, true, 'system', now(), now())
            ON CONFLICT (tenant_id, module_code) DO NOTHING;
        END LOOP;

        -- Insert features if not exists
        FOREACH fc IN ARRAY feature_codes LOOP
            INSERT INTO public.tenant_features (id, tenant_id, feature_code, is_enabled, source, created_at, updated_at)
            VALUES (gen_random_uuid(), tenant_rec.id, fc, true, 'system', now(), now())
            ON CONFLICT (tenant_id, feature_code) DO NOTHING;
        END LOOP;

        RAISE NOTICE 'Seeded modules and features for tenant: %', tenant_rec.code;
    END LOOP;
END $$;
