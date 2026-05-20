-- Registration OTPs for self-registration flow
CREATE TABLE IF NOT EXISTS registration_otps (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_token VARCHAR(64) UNIQUE NOT NULL,
    otp_code VARCHAR(6) NOT NULL,
    tenant_code VARCHAR(50) NOT NULL,
    tenant_name VARCHAR(255) NOT NULL,
    admin_email VARCHAR(255) NOT NULL,
    admin_name VARCHAR(255) NOT NULL,
    password_hash TEXT NOT NULL,
    password_plain TEXT,            -- Temporary plain password for welcome email
    phone VARCHAR(30),
    otp_method VARCHAR(10) NOT NULL DEFAULT 'email',
    attempts INT NOT NULL DEFAULT 0,
    verified BOOLEAN NOT NULL DEFAULT FALSE,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_registration_otps_session ON registration_otps(session_token);
CREATE INDEX IF NOT EXISTS idx_registration_otps_expires ON registration_otps(expires_at);
CREATE INDEX IF NOT EXISTS idx_registration_otps_email ON registration_otps(admin_email);
