package infra

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"

	"pekan/backend/internal/modules/finance/transactions/domain"
	"pekan/backend/internal/platform/db"
)

func (r *RepositoryPG) CreateAttachmentRecord(ctx context.Context, in domain.CreateAttachmentRecordInput) (attachment domain.Attachment, err error) {
	attachmentID := uuid.NewString()
	now := time.Now().UTC()

	const insertFileQuery = `
INSERT INTO public.files (
  id, tenant_id, module_code, owner_type, owner_id, provider, object_key, original_filename, stored_filename, mime_type, size_bytes, uploaded_by, created_at
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)`

	const insertAttachmentQuery = `
INSERT INTO finance_transaction_attachments (
  id, transaction_id, file_id, created_by, created_at
) VALUES ($1,$2,$3,$4,$5)`

	const upsertScanJobQuery = `
INSERT INTO public.file_scan_jobs (
  id, tenant_id, file_id, status, attempts, scheduled_at, created_at, updated_at
) VALUES ($1,$2,$3,'queued',0,now(),now(),now())
ON CONFLICT (file_id)
DO UPDATE SET
  status = 'queued',
  attempts = 0,
  last_error = NULL,
  scheduled_at = now(),
  started_at = NULL,
  finished_at = NULL,
  updated_at = now()`

	err = db.WithTenantTx(ctx, r.conn, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, insertFileQuery,
			in.FileID, in.TenantID, "finance.transactions", "transaction", in.TransactionID,
			in.Provider, in.ObjectKey, in.OriginalFilename, in.StoredFilename, in.MimeType, in.SizeBytes, in.UploadedBy, now,
		)
		if err != nil {
			return err
		}

		_, err = tx.ExecContext(ctx, insertAttachmentQuery,
			attachmentID, in.TransactionID, in.FileID, in.UploadedBy, now,
		)
		if err != nil {
			return err
		}

		_, err = tx.ExecContext(ctx, upsertScanJobQuery, uuid.NewString(), in.TenantID, in.FileID)
		if err != nil {
			return err
		}

		attachment = domain.Attachment{
			ID:               attachmentID,
			TenantID:         in.TenantID,
			TransactionID:    in.TransactionID,
			FileID:           in.FileID,
			Provider:         in.Provider,
			ObjectKey:        in.ObjectKey,
			OriginalFilename: in.OriginalFilename,
			StoredFilename:   in.StoredFilename,
			MimeType:         in.MimeType,
			ScanStatus:       "pending",
			SizeBytes:        in.SizeBytes,
			CreatedAt:        now,
		}
		return nil
	})
	return attachment, err
}

func (r *RepositoryPG) GetAttachmentByID(ctx context.Context, tenantID, transactionID, attachmentID string) (out domain.Attachment, err error) {
	err = db.WithTenantTx(ctx, r.conn, func(tx *sql.Tx) error {
		const q = `
SELECT
  a.id, $1 AS tenant_id, a.transaction_id, a.file_id, f.provider, f.object_key,
  f.original_filename, f.stored_filename, f.mime_type, f.scan_status, f.size_bytes, a.created_at
FROM finance_transaction_attachments a
JOIN public.files f ON f.id = a.file_id
JOIN finance_transactions t ON t.id = a.transaction_id
WHERE a.transaction_id = $2 AND a.id = $3 AND t.deleted_at IS NULL AND f.deleted_at IS NULL`

		return tx.QueryRowContext(ctx, q, tenantID, transactionID, attachmentID).Scan(
			&out.ID, &out.TenantID, &out.TransactionID, &out.FileID, &out.Provider, &out.ObjectKey,
			&out.OriginalFilename, &out.StoredFilename, &out.MimeType, &out.ScanStatus, &out.SizeBytes, &out.CreatedAt,
		)
	})
	return out, err
}

func (r *RepositoryPG) ListAttachmentsByTransaction(ctx context.Context, tenantID, transactionID string) (items []domain.Attachment, err error) {
	err = db.WithTenantTx(ctx, r.conn, func(tx *sql.Tx) error {
		const q = `
SELECT
  a.id, $1 AS tenant_id, a.transaction_id, a.file_id, f.provider, f.object_key,
  f.original_filename, f.stored_filename, f.mime_type, f.scan_status, f.size_bytes, a.created_at
FROM finance_transaction_attachments a
JOIN public.files f ON f.id = a.file_id
JOIN finance_transactions t ON t.id = a.transaction_id
WHERE a.transaction_id = $2
  AND t.deleted_at IS NULL
  AND f.deleted_at IS NULL
ORDER BY a.created_at DESC`

		rows, err := tx.QueryContext(ctx, q, tenantID, transactionID)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var item domain.Attachment
			if err := rows.Scan(
				&item.ID, &item.TenantID, &item.TransactionID, &item.FileID, &item.Provider, &item.ObjectKey,
				&item.OriginalFilename, &item.StoredFilename, &item.MimeType, &item.ScanStatus, &item.SizeBytes, &item.CreatedAt,
			); err != nil {
				return err
			}
			items = append(items, item)
		}
		return rows.Err()
	})
	return items, err
}

func (r *RepositoryPG) SetAttachmentScanStatus(ctx context.Context, tenantID, transactionID, attachmentID, scanStatus string) error {
	return db.WithTenantTx(ctx, r.conn, func(tx *sql.Tx) error {
		const q = `
UPDATE public.files f
SET scan_status = $1
FROM finance_transaction_attachments a
JOIN finance_transactions t ON t.id = a.transaction_id
WHERE f.id = a.file_id
  AND a.transaction_id = $2
  AND a.id = $3
  AND t.deleted_at IS NULL
  AND f.deleted_at IS NULL`

		res, err := tx.ExecContext(ctx, q, scanStatus, transactionID, attachmentID)
		if err != nil {
			return err
		}

		affected, err := res.RowsAffected()
		if err != nil {
			return err
		}
		if affected == 0 {
			return domain.ErrAttachmentNotFound
		}
		return nil
	})
}
