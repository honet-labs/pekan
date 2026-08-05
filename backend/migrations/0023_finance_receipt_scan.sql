CREATE TABLE IF NOT EXISTS finance_receipt_provider_configs (
  id UUID PRIMARY KEY,
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  provider_code VARCHAR(64) NOT NULL,
  display_name VARCHAR(120) NOT NULL,
  base_url TEXT NULL,
  model_name VARCHAR(255) NOT NULL,
  is_enabled BOOLEAN NOT NULL DEFAULT FALSE,
  api_key_ciphertext TEXT NULL,
  created_by UUID NOT NULL REFERENCES users(id),
  updated_by UUID NOT NULL REFERENCES users(id),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (tenant_id, provider_code)
);

CREATE INDEX IF NOT EXISTS idx_finance_receipt_provider_configs_tenant_enabled
ON finance_receipt_provider_configs (tenant_id, is_enabled, provider_code);

CREATE TABLE IF NOT EXISTS finance_receipt_scans (
  id UUID PRIMARY KEY,
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  provider_code VARCHAR(64) NOT NULL,
  model_name VARCHAR(255) NOT NULL,
  status VARCHAR(32) NOT NULL,
  original_filename TEXT NOT NULL,
  mime_type VARCHAR(128) NOT NULL,
  extracted_json JSONB NULL,
  error_message TEXT NULL,
  created_by UUID NOT NULL REFERENCES users(id),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_finance_receipt_scans_tenant_created
ON finance_receipt_scans (tenant_id, created_at DESC);
