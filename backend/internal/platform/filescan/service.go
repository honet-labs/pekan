package filescan

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"time"

	"pekan/backend/internal/platform/storage"
)

const (
	ScanStatusPending  = "pending"
	ScanStatusClean    = "clean"
	ScanStatusInfected = "infected"
	ScanStatusFailed   = "failed"
)

type Service struct {
	db           *sql.DB
	storage      storage.ObjectStorage
	scanner      Scanner
	pollInterval time.Duration
	maxAttempts  int
	retryDelay   time.Duration
}

func NewService(db *sql.DB, objectStorage storage.ObjectStorage, scanner Scanner, pollInterval time.Duration, maxAttempts int, retryDelay time.Duration) *Service {
	if scanner == nil {
		scanner = NewSignatureScanner()
	}
	return &Service{
		db:           db,
		storage:      objectStorage,
		scanner:      scanner,
		pollInterval: pollInterval,
		maxAttempts:  maxAttempts,
		retryDelay:   retryDelay,
	}
}

func (s *Service) Run(ctx context.Context) error {
	ticker := time.NewTicker(s.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			_, err := s.ProcessNext(ctx)
			if err != nil {
				fmt.Printf("file-scan worker error: %v\n", err)
			}
		}
	}
}

type scanJob struct {
	ID        string
	TenantID  string
	FileID    string
	Attempts  int
}

type fileMeta struct {
	ObjectKey string
	MimeType  string
	SizeBytes int64
}

func (s *Service) ProcessNext(ctx context.Context) (bool, error) {
	job, ok, err := s.claimNextJob(ctx)
	if err != nil || !ok {
		return false, err
	}

	meta, err := s.getFileMeta(ctx, job.TenantID, job.FileID)
	if err != nil {
		_ = s.markFailure(ctx, job, err)
		return true, err
	}

	reader, err := s.storage.Open(ctx, storage.GetObjectInput{ObjectKey: meta.ObjectKey})
	if err != nil {
		_ = s.markFailure(ctx, job, err)
		return true, err
	}
	defer reader.Close()

	outcome, err := s.scanner.Scan(ctx, ScanInput{
		TenantID:  job.TenantID,
		FileID:    job.FileID,
		ObjectKey: meta.ObjectKey,
		MimeType:  meta.MimeType,
		SizeBytes: meta.SizeBytes,
		Reader:    reader,
	})
	if err != nil {
		_ = s.markFailure(ctx, job, err)
		return true, err
	}
	if outcome != ScanStatusClean && outcome != ScanStatusInfected {
		_ = s.markFailure(ctx, job, errors.New("invalid scanner outcome"))
		return true, errors.New("invalid scanner outcome")
	}

	if err := s.markDone(ctx, job, outcome); err != nil {
		return true, err
	}
	return true, nil
}

func (s *Service) claimNextJob(ctx context.Context) (scanJob, bool, error) {
	const q = `
WITH picked AS (
    SELECT id, tenant_id, file_id, attempts
    FROM file_scan_jobs
    WHERE status IN ('queued','failed')
      AND scheduled_at <= now()
      AND attempts < $1
    ORDER BY scheduled_at ASC, created_at ASC
    LIMIT 1
    FOR UPDATE SKIP LOCKED
)
UPDATE file_scan_jobs j
SET status = 'processing',
    attempts = picked.attempts + 1,
    started_at = now(),
    updated_at = now(),
    last_error = NULL
FROM picked
WHERE j.id = picked.id
RETURNING j.id, j.tenant_id, j.file_id, j.attempts`

	var out scanJob
	err := s.db.QueryRowContext(ctx, q, s.maxAttempts).Scan(
		&out.ID, &out.TenantID, &out.FileID, &out.Attempts,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return scanJob{}, false, nil
		}
		return scanJob{}, false, err
	}
	return out, true, nil
}

func (s *Service) getFileMeta(ctx context.Context, tenantID, fileID string) (fileMeta, error) {
	const q = `
SELECT object_key, mime_type, size_bytes
FROM files
WHERE tenant_id = $1 AND id = $2`

	var out fileMeta
	err := s.db.QueryRowContext(ctx, q, tenantID, fileID).Scan(
		&out.ObjectKey, &out.MimeType, &out.SizeBytes,
	)
	return out, err
}

func (s *Service) markDone(ctx context.Context, job scanJob, scanStatus string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	const updateFile = `
UPDATE files
SET scan_status = $1
WHERE tenant_id = $2 AND id = $3`

	if _, err := tx.ExecContext(ctx, updateFile, scanStatus, job.TenantID, job.FileID); err != nil {
		return err
	}

	const updateJob = `
UPDATE file_scan_jobs
SET status = 'done',
    finished_at = now(),
    updated_at = now(),
    last_error = NULL
WHERE id = $1`

	if _, err := tx.ExecContext(ctx, updateJob, job.ID); err != nil {
		return err
	}

	return tx.Commit()
}

func (s *Service) markFailure(ctx context.Context, job scanJob, inErr error) error {
	errText := truncateError(inErr, 1000)

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	const updateFile = `
UPDATE files
SET scan_status = $1
WHERE tenant_id = $2 AND id = $3`

	if _, err := tx.ExecContext(ctx, updateFile, ScanStatusFailed, job.TenantID, job.FileID); err != nil {
		return err
	}

	if job.Attempts >= s.maxAttempts {
		const markFinalFail = `
UPDATE file_scan_jobs
SET status = 'failed',
    finished_at = now(),
    updated_at = now(),
    last_error = $2
WHERE id = $1`
		if _, err := tx.ExecContext(ctx, markFinalFail, job.ID, errText); err != nil {
			return err
		}
		return tx.Commit()
	}

	const requeue = `
UPDATE file_scan_jobs
SET status = 'queued',
    updated_at = now(),
    last_error = $2,
    scheduled_at = now() + $3::interval
WHERE id = $1`

	retryInterval := fmt.Sprintf("%d seconds", int(s.retryDelay.Seconds()))
	if _, err := tx.ExecContext(ctx, requeue, job.ID, errText, retryInterval); err != nil {
		return err
	}

	return tx.Commit()
}

func truncateError(err error, max int) string {
	if err == nil {
		return ""
	}
	raw := err.Error()
	if len(raw) <= max {
		return raw
	}
	return raw[:max]
}

type ScanInput struct {
	TenantID  string
	FileID    string
	ObjectKey string
	MimeType  string
	SizeBytes int64
	Reader    io.Reader
}

type Scanner interface {
	Scan(ctx context.Context, in ScanInput) (string, error)
}
