-- Migration: 0034_whatsapp_integration.sql
-- Description: Create tables for WhatsApp OTP pairing and sessions

CREATE TABLE IF NOT EXISTS whatsapp_otp_tokens (
    token VARCHAR(20) PRIMARY KEY,
    tenant_id UUID NOT NULL,
    user_id UUID NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_whatsapp_otp_tokens_expires_at ON whatsapp_otp_tokens(expires_at);
CREATE INDEX IF NOT EXISTS idx_whatsapp_otp_tokens_user ON whatsapp_otp_tokens(tenant_id, user_id);

CREATE TABLE IF NOT EXISTS whatsapp_sessions (
    phone_number VARCHAR(20) PRIMARY KEY,
    tenant_id UUID NOT NULL,
    user_id UUID NOT NULL,
    last_active TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_whatsapp_sessions_user ON whatsapp_sessions(tenant_id, user_id);
