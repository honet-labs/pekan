CREATE TABLE IF NOT EXISTS file_scan_jobs (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    file_id UUID NOT NULL REFERENCES files(id) UNIQUE,
    status VARCHAR(20) NOT NULL CHECK (status IN ('queued', 'processing', 'done', 'failed')),
    attempts INT NOT NULL DEFAULT 0,
    last_error TEXT NULL,
    scheduled_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    started_at TIMESTAMPTZ NULL,
    finished_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_file_scan_jobs_status_schedule ON file_scan_jobs (status, scheduled_at, created_at);
CREATE INDEX IF NOT EXISTS idx_file_scan_jobs_tenant_status ON file_scan_jobs (tenant_id, status, scheduled_at);
