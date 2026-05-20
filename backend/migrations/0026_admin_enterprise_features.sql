-- Add quota fields to tenants
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS quota_users INTEGER DEFAULT 10;
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS quota_transactions INTEGER DEFAULT 1000;

-- Create audit_logs table for platform-wide auditing
CREATE TABLE IF NOT EXISTS audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID, -- Can be NULL for system-level actions
    actor_user_id UUID,
    action VARCHAR(100) NOT NULL,
    resource_type VARCHAR(100),
    resource_id VARCHAR(100),
    ip_address VARCHAR(45),
    metadata JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_audit_logs_tenant ON audit_logs(tenant_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_created ON audit_logs(created_at DESC);

-- Create tenant_modules table for feature toggling
CREATE TABLE IF NOT EXISTS tenant_modules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    module_code VARCHAR(50) NOT NULL,
    is_enabled BOOLEAN DEFAULT FALSE,
    source VARCHAR(20) DEFAULT 'system', -- 'system', 'manual', 'subscription'
    created_at TIMESTAMP WITH TIME ZONE DEFAULT now(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT now(),
    UNIQUE(tenant_id, module_code)
);

CREATE INDEX IF NOT EXISTS idx_tenant_modules_tenant ON tenant_modules(tenant_id);
