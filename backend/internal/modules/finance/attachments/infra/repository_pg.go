package infra

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"pekan/backend/internal/modules/finance/attachments/domain"
	"pekan/backend/internal/platform/db"
)

type RepositoryPG struct {
	conn *sql.DB
}

func NewRepositoryPG(conn *sql.DB) *RepositoryPG {
	return &RepositoryPG{conn: conn}
}

func (r *RepositoryPG) EnsureOwnerExists(ctx context.Context, tenantID string, ownerType domain.OwnerType, ownerID string) error {
	return db.WithTenantTx(ctx, r.conn, func(tx *sql.Tx) error {
		query, err := ownerValidationQuery(ownerType)
		if err != nil {
			return err
		}

		var marker int
		if err := tx.QueryRowContext(ctx, query, ownerID).Scan(&marker); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return domain.ErrOwnerNotFound
			}
			return err
		}
		return nil
	})
}

func (r *RepositoryPG) CreateAttachmentRecord(ctx context.Context, in domain.CreateAttachmentRecordInput) (domain.Attachment, error) {
	var out domain.Attachment
	err := db.WithTenantTx(ctx, r.conn, func(tx *sql.Tx) error {
		attachmentID := uuid.NewString()
		now := time.Now().UTC()
		moduleCode := "finance." + string(in.OwnerType)

		const insertFileQuery = `
INSERT INTO public.files (
  id, tenant_id, module_code, owner_type, owner_id, provider, object_key,
  original_filename, stored_filename, mime_type, size_bytes, uploaded_by, created_at, deleted_at
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,NULL)`

		const insertAttachmentQuery = `
INSERT INTO finance_entity_attachments (
  id, owner_type, owner_id, file_id, created_by, created_at
) VALUES ($1,$2,$3,$4,$5,$6)`

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

		_, err := tx.ExecContext(ctx, insertFileQuery,
			in.FileID,
			in.TenantID,
			moduleCode,
			string(in.OwnerType),
			in.OwnerID,
			in.Provider,
			in.ObjectKey,
			in.OriginalFilename,
			in.StoredFilename,
			in.MimeType,
			in.SizeBytes,
			in.UploadedBy,
			now,
		)
		if err != nil {
			return err
		}

		_, err = tx.ExecContext(ctx, insertAttachmentQuery,
			attachmentID,
			string(in.OwnerType),
			in.OwnerID,
			in.FileID,
			in.UploadedBy,
			now,
		)

		if err != nil {
			return err
		}

		_, err = tx.ExecContext(ctx, upsertScanJobQuery, uuid.NewString(), in.TenantID, in.FileID)
		if err != nil {
			return err
		}

		out = domain.Attachment{
			ID:               attachmentID,
			TenantID:         in.TenantID,
			OwnerType:        in.OwnerType,
			OwnerID:          in.OwnerID,
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
	return out, err
}

func (r *RepositoryPG) GetAttachmentByID(ctx context.Context, tenantID string, ownerType domain.OwnerType, ownerID, attachmentID string) (domain.Attachment, error) {
	var out domain.Attachment
	err := db.WithTenantTx(ctx, r.conn, func(tx *sql.Tx) error {
		const q = `
SELECT
  a.id,
  a.owner_type,
  a.owner_id,
  a.file_id,
  f.provider,
  f.object_key,
  f.original_filename,
  f.stored_filename,
  f.mime_type,
  f.scan_status,
  f.size_bytes,
  a.created_at
FROM finance_entity_attachments a
JOIN public.files f ON f.id = a.file_id
WHERE a.owner_type = $1
  AND a.owner_id = $2
  AND a.id = $3
  AND f.deleted_at IS NULL`


		var ownerTypeRaw string
		err := tx.QueryRowContext(ctx, q, string(ownerType), ownerID, attachmentID).Scan(
			&out.ID,
			&ownerTypeRaw,
			&out.OwnerID,
			&out.FileID,
			&out.Provider,
			&out.ObjectKey,
			&out.OriginalFilename,
			&out.StoredFilename,
			&out.MimeType,
			&out.ScanStatus,
			&out.SizeBytes,
			&out.CreatedAt,
		)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return domain.ErrAttachmentNotFound
			}
			return err
		}
		out.TenantID = tenantID
		out.OwnerType = domain.OwnerType(ownerTypeRaw)
		return nil
	})
	return out, err
}

func (r *RepositoryPG) ListByOwner(ctx context.Context, tenantID string, ownerType domain.OwnerType, ownerID string) ([]domain.Attachment, error) {
	var items []domain.Attachment
	err := db.WithTenantTx(ctx, r.conn, func(tx *sql.Tx) error {
		const q = `
SELECT
  a.id,
  a.owner_type,
  a.owner_id,
  a.file_id,
  f.provider,
  f.object_key,
  f.original_filename,
  f.stored_filename,
  f.mime_type,
  f.scan_status,
  f.size_bytes,
  a.created_at
FROM finance_entity_attachments a
JOIN public.files f ON f.id = a.file_id
WHERE a.owner_type = $1
  AND a.owner_id = $2
  AND f.deleted_at IS NULL
ORDER BY a.created_at DESC`


		rows, err := tx.QueryContext(ctx, q, string(ownerType), ownerID)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var (
				item         domain.Attachment
				ownerTypeRaw string
			)
			if err := rows.Scan(
				&item.ID,
				&ownerTypeRaw,
				&item.OwnerID,
				&item.FileID,
				&item.Provider,
				&item.ObjectKey,
				&item.OriginalFilename,
				&item.StoredFilename,
				&item.MimeType,
				&item.ScanStatus,
				&item.SizeBytes,
				&item.CreatedAt,
			); err != nil {
				return err
			}
			item.TenantID = tenantID
			item.OwnerType = domain.OwnerType(ownerTypeRaw)
			items = append(items, item)
		}
		return rows.Err()
	})
	return items, err
}

func (r *RepositoryPG) SoftDeleteAttachment(ctx context.Context, tenantID string, ownerType domain.OwnerType, ownerID, attachmentID, actorUserID string) (domain.Attachment, error) {
	var attachment domain.Attachment
	err := db.WithTenantTx(ctx, r.conn, func(tx *sql.Tx) error {
		var err error
		attachment, err = r.GetAttachmentByID(ctx, tenantID, ownerType, ownerID, attachmentID)
		if err != nil {
			return err
		}

		now := time.Now().UTC()
		const deleteAttachmentQuery = `
DELETE FROM finance_entity_attachments
WHERE owner_type = $1
  AND owner_id = $2
  AND id = $3`


		res, err := tx.ExecContext(ctx, deleteAttachmentQuery, string(ownerType), ownerID, attachmentID)
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

		const deleteFileQuery = `
UPDATE public.files
SET deleted_at = $1
WHERE id = $2
  AND deleted_at IS NULL`

		_, err = tx.ExecContext(ctx, deleteFileQuery, now, attachment.FileID)
		if err != nil {
			return err
		}
		return nil
	})
	return attachment, err
}

func ownerValidationQuery(ownerType domain.OwnerType) (string, error) {
	switch ownerType {
	case domain.OwnerTypeSavings:
		return `SELECT 1 FROM finance_savings WHERE id = $1 AND deleted_at IS NULL`, nil
	case domain.OwnerTypeBudgets:
		return `SELECT 1 FROM finance_budgets WHERE id = $1 AND deleted_at IS NULL`, nil
	case domain.OwnerTypeReminders:
		return `SELECT 1 FROM finance_reminders WHERE id = $1 AND deleted_at IS NULL`, nil
	default:
		return "", fmt.Errorf("%w: %s", domain.ErrInvalidOwnerType, ownerType)
	}
}

