CREATE TABLE IF NOT EXISTS auth_refresh_tokens (
    id BIGSERIAL PRIMARY KEY,
    session_id UUID NOT NULL REFERENCES auth_sessions(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    refresh_token_hash VARCHAR(255) NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    consumed_at TIMESTAMPTZ NULL,
    revoked_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

INSERT INTO auth_refresh_tokens (
    session_id,
    user_id,
    tenant_id,
    refresh_token_hash,
    expires_at,
    consumed_at,
    revoked_at,
    created_at,
    updated_at
)
SELECT
    s.id,
    s.user_id,
    s.tenant_id,
    s.refresh_token_hash,
    s.expires_at,
    NULL,
    s.revoked_at,
    s.created_at,
    s.updated_at
FROM auth_sessions s
ON CONFLICT (refresh_token_hash) DO NOTHING;

CREATE INDEX IF NOT EXISTS idx_auth_refresh_tokens_hash ON auth_refresh_tokens (refresh_token_hash);
CREATE INDEX IF NOT EXISTS idx_auth_refresh_tokens_user_status ON auth_refresh_tokens (user_id, revoked_at, consumed_at, expires_at);
CREATE INDEX IF NOT EXISTS idx_auth_refresh_tokens_session ON auth_refresh_tokens (session_id);
