INSERT INTO permissions (id, code, name, module_code, action, created_at)
VALUES
    (uuid_generate_v4(), 'finance.transactions.scan.manage', 'Manage finance attachment scan status', 'finance.transactions', 'scan.manage', now())
ON CONFLICT (code) DO NOTHING;
