INSERT INTO permissions (id, code, name, module_code, action, created_at)
VALUES
    (uuid_generate_v4(), 'finance.transactions.create', 'Create finance transaction', 'finance.transactions', 'create', now()),
    (uuid_generate_v4(), 'finance.transactions.read', 'Read finance transaction', 'finance.transactions', 'read', now()),
    (uuid_generate_v4(), 'finance.transactions.update', 'Update finance transaction', 'finance.transactions', 'update', now()),
    (uuid_generate_v4(), 'finance.transactions.delete', 'Delete finance transaction', 'finance.transactions', 'delete', now()),
    (uuid_generate_v4(), 'finance.transactions.attach', 'Upload finance transaction attachment', 'finance.transactions', 'attach', now()),
    (uuid_generate_v4(), 'finance.transactions.attachment.read', 'Download finance transaction attachment', 'finance.transactions', 'attachment.read', now())
ON CONFLICT (code) DO NOTHING;
