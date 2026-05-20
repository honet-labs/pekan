package infra

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"pekan/backend/internal/modules/finance/reports/domain"
	"pekan/backend/internal/platform/db"
)

type RepositoryPG struct {
	conn *sql.DB
}

func NewRepositoryPG(conn *sql.DB) *RepositoryPG {
	return &RepositoryPG{conn: conn}
}

func (r *RepositoryPG) Create(ctx context.Context, in domain.Report) (domain.Report, error) {
	var out domain.Report
	err := db.WithTenantTx(ctx, r.conn, func(tx *sql.Tx) error {
		in.ID = uuid.NewString()
		const q = `
INSERT INTO finance_reports (
  id, report_type, format, status, params, storage_provider, storage_key, created_by, created_at, updated_at
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`

		_, err := tx.ExecContext(ctx, q,
			in.ID, in.ReportType, in.Format, in.Status, in.Params, in.StorageProvider, in.StorageKey,
			in.CreatedBy, in.CreatedAt, in.UpdatedAt,
		)
		if err != nil {
			return err
		}
		out = in
		return nil
	})
	return out, err
}

func (r *RepositoryPG) UpdateStatus(ctx context.Context, in domain.Report) (domain.Report, error) {
	var out domain.Report
	err := db.WithTenantTx(ctx, r.conn, func(tx *sql.Tx) error {
		const q = `
UPDATE finance_reports
SET status = $1, storage_provider = $2, storage_key = $3, updated_at = $4
WHERE id = $5
RETURNING id, report_type, format, status, params, storage_provider, storage_key, created_by, created_at, updated_at`

		err := tx.QueryRowContext(ctx, q,
			in.Status, in.StorageProvider, in.StorageKey, in.UpdatedAt, in.ID,
		).Scan(
			&out.ID, &out.ReportType, &out.Format, &out.Status, &out.Params, &out.StorageProvider,
			&out.StorageKey, &out.CreatedBy, &out.CreatedAt, &out.UpdatedAt,
		)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return domain.ErrReportNotFound
			}
			return err
		}
		out.TenantID = in.TenantID
		return nil
	})
	return out, err
}

func (r *RepositoryPG) GetByID(ctx context.Context, tenantID, reportID string) (domain.Report, error) {
	var out domain.Report
	err := db.WithTenantTx(ctx, r.conn, func(tx *sql.Tx) error {
		const q = `
SELECT id, report_type, format, status, params, storage_provider, storage_key, created_by, created_at, updated_at
FROM finance_reports
WHERE id = $1`

		err := tx.QueryRowContext(ctx, q, reportID).Scan(
			&out.ID, &out.ReportType, &out.Format, &out.Status, &out.Params, &out.StorageProvider,
			&out.StorageKey, &out.CreatedBy, &out.CreatedAt, &out.UpdatedAt,
		)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return domain.ErrReportNotFound
			}
			return err
		}
		out.TenantID = tenantID
		return nil
	})
	return out, err
}

func (r *RepositoryPG) List(ctx context.Context, filter domain.ListFilter) ([]domain.Report, int64, error) {
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.PageSize <= 0 || filter.PageSize > 100 {
		filter.PageSize = 20
	}

	var items []domain.Report
	var total int64

	err := db.WithTenantTx(ctx, r.conn, func(tx *sql.Tx) error {
		where := "1=1"
		countQuery := "SELECT COUNT(1) FROM finance_reports"

		if err := tx.QueryRowContext(ctx, countQuery).Scan(&total); err != nil {
			return err
		}

		offset := (filter.Page - 1) * filter.PageSize
		dataQuery := fmt.Sprintf(`
SELECT id, report_type, format, status, params, storage_provider, storage_key, created_by, created_at, updated_at
FROM finance_reports
WHERE %s
ORDER BY created_at DESC
LIMIT $1 OFFSET $2`, where)

		rows, err := tx.QueryContext(ctx, dataQuery, filter.PageSize, offset)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var item domain.Report
			if err := rows.Scan(
				&item.ID, &item.ReportType, &item.Format, &item.Status, &item.Params,
				&item.StorageProvider, &item.StorageKey, &item.CreatedBy, &item.CreatedAt, &item.UpdatedAt,
			); err != nil {
				return err
			}
			item.TenantID = filter.TenantID
			items = append(items, item)
		}
		return rows.Err()
	})
	return items, total, err
}

func (r *RepositoryPG) Delete(ctx context.Context, tenantID, reportID string) error {
	return db.WithTenantTx(ctx, r.conn, func(tx *sql.Tx) error {
		const q = `
DELETE FROM finance_reports
WHERE id = $1`

		res, err := tx.ExecContext(ctx, q, reportID)
		if err != nil {
			return err
		}
		affected, err := res.RowsAffected()
		if err != nil {
			return err
		}
		if affected == 0 {
			return domain.ErrReportNotFound
		}
		return nil
	})
}
