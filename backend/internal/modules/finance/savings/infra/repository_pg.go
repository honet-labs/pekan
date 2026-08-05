package infra

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"pekan/backend/internal/modules/finance/savings/domain"
	"pekan/backend/internal/platform/db"
	"strings"
	"time"

	"github.com/google/uuid"
)

type RepositoryPG struct {
	conn *sql.DB
}

func NewRepositoryPG(conn *sql.DB) *RepositoryPG {
	return &RepositoryPG{conn: conn}
}

func (r *RepositoryPG) Create(ctx context.Context, in domain.Savings) (domain.Savings, error) {
	var out domain.Savings
	err := db.WithTenantTx(ctx, r.conn, func(tx *sql.Tx) error {
		in.ID = uuid.NewString()
		// Generate a short ID from UUID if not provided (typically for internal display)
		sid := strings.ToUpper(in.ID[:8])
		
		const q = `
INSERT INTO finance_savings (
  id, sid, name, target_amount_minor, current_amount_minor, currency, start_date, target_date, notes, status,
  created_by, updated_by, created_at, updated_at
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)`

		if _, err := tx.ExecContext(ctx, q,
			in.ID, sid, in.Name, in.TargetAmountMinor, in.CurrentAmountMinor, in.Currency, in.StartDate, in.TargetDate, in.Notes, in.Status,
			in.CreatedBy, in.UpdatedBy, in.CreatedAt, in.UpdatedAt,
		); err != nil {
			return err
		}

		return r.getByID(ctx, tx, in.ID, &out)
	})
	if err != nil {
		return domain.Savings{}, err
	}
	return out, nil
}

func (r *RepositoryPG) GetByID(ctx context.Context, tenantID, savingsID string) (domain.Savings, error) {
	var out domain.Savings
	err := db.WithTenantTx(ctx, r.conn, func(tx *sql.Tx) error {
		return r.getByID(ctx, tx, savingsID, &out)
	})
	if err != nil {
		return domain.Savings{}, err
	}
	out.TenantID = tenantID
	return out, nil
}

func (r *RepositoryPG) getByID(ctx context.Context, tx *sql.Tx, savingsID string, out *domain.Savings) error {
	const q = `
SELECT s.id,
       s.sid,
       s.name,
       s.target_amount_minor,
       s.current_amount_minor,
       CASE
           WHEN s.target_amount_minor > 0 THEN ROUND((s.current_amount_minor::numeric / s.target_amount_minor::numeric) * 100, 2)
           WHEN s.target_amount_minor = 0 AND s.current_amount_minor > 0 THEN 100.00
           ELSE 0
       END AS progress_percent,
       s.currency,
       s.start_date,
       s.target_date,
       s.notes,
       s.status,
       s.created_by,
       s.updated_by,
       s.created_at,
       s.updated_at,
       s.deleted_at
FROM finance_savings s
WHERE s.id = $1 AND s.deleted_at IS NULL`

	var (
		startDate  sql.NullTime
		targetDate sql.NullTime
		notes      sql.NullString
		deletedAt  sql.NullTime
	)
	err := tx.QueryRowContext(ctx, q, savingsID).Scan(
		&out.ID, &out.SID, &out.Name, &out.TargetAmountMinor, &out.CurrentAmountMinor, &out.ProgressPercent, &out.Currency, &startDate,
		&targetDate, &notes, &out.Status, &out.CreatedBy, &out.UpdatedBy, &out.CreatedAt, &out.UpdatedAt, &deletedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.ErrSavingsNotFound
		}
		return err
	}
	out.StartDate = toTimePtr(startDate)
	out.TargetDate = toTimePtr(targetDate)
	out.Notes = toStringPtr(notes)
	out.DeletedAt = toTimePtr(deletedAt)
	return nil
}

func (r *RepositoryPG) List(ctx context.Context, filter domain.ListFilter) ([]domain.Savings, int64, error) {
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.PageSize <= 0 || filter.PageSize > 100 {
		filter.PageSize = 20
	}

	var items []domain.Savings
	var total int64

	err := db.WithTenantTx(ctx, r.conn, func(tx *sql.Tx) error {
		var (
			clauses = []string{"s.deleted_at IS NULL"}
			args    []any
			idx     = 1
		)

		if filter.Status != nil && strings.TrimSpace(*filter.Status) != "" {
			clauses = append(clauses, fmt.Sprintf("s.status = $%d", idx))
			args = append(args, strings.ToLower(strings.TrimSpace(*filter.Status)))
			idx++
		}
		// Resetting clause logic for clarity
		clauses = []string{"s.deleted_at IS NULL"}
		args = []any{}
		idx = 1

		if filter.Status != nil && strings.TrimSpace(*filter.Status) != "" {
			clauses = append(clauses, "s.status = $1")
			args = append(args, strings.ToLower(strings.TrimSpace(*filter.Status)))
			idx = 2
		}

		where := strings.Join(clauses, " AND ")
		countQuery := "SELECT COUNT(1) FROM finance_savings s WHERE " + where

		if err := tx.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
			return err
		}

		offset := (filter.Page - 1) * filter.PageSize
		dataArgs := append(args, filter.PageSize, offset)
		limitIdx := idx
		offsetIdx := idx + 1

		// Using fmt.Sprintf is safer
		dataQuery := fmt.Sprintf(`
SELECT s.id, s.sid, s.name, s.target_amount_minor, s.current_amount_minor,
       CASE
           WHEN s.target_amount_minor > 0 THEN ROUND((s.current_amount_minor::numeric / s.target_amount_minor::numeric) * 100, 2)
           WHEN s.target_amount_minor = 0 AND s.current_amount_minor > 0 THEN 100.00
           ELSE 0
       END AS progress_percent,
       s.currency, s.start_date, s.target_date, s.notes, s.status, s.created_by, s.updated_by, s.created_at, s.updated_at, s.deleted_at
FROM finance_savings s
WHERE %s
ORDER BY s.created_at DESC
LIMIT $%d OFFSET $%d`, where, limitIdx, offsetIdx)

		rows, err := tx.QueryContext(ctx, dataQuery, dataArgs...)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var item domain.Savings
			var (
				startDate  sql.NullTime
				targetDate sql.NullTime
				notes      sql.NullString
				deletedAt  sql.NullTime
			)
			if err := rows.Scan(
				&item.ID, &item.SID, &item.Name, &item.TargetAmountMinor, &item.CurrentAmountMinor, &item.ProgressPercent, &item.Currency,
				&startDate, &targetDate, &notes, &item.Status, &item.CreatedBy, &item.UpdatedBy, &item.CreatedAt, &item.UpdatedAt, &deletedAt,
			); err != nil {
				return err
			}
			item.TenantID = filter.TenantID
			item.StartDate = toTimePtr(startDate)
			item.TargetDate = toTimePtr(targetDate)
			item.Notes = toStringPtr(notes)
			item.DeletedAt = toTimePtr(deletedAt)
			items = append(items, item)
		}
		return rows.Err()
	})

	return items, total, err
}

func (r *RepositoryPG) Update(ctx context.Context, in domain.Savings) (domain.Savings, error) {
	var out domain.Savings
	err := db.WithTenantTx(ctx, r.conn, func(tx *sql.Tx) error {
		const q = `
UPDATE finance_savings
SET name = $1, target_amount_minor = $2, current_amount_minor = $3, currency = $4, start_date = $5,
    target_date = $6, notes = $7, status = $8, updated_by = $9, updated_at = $10
WHERE id = $11 AND deleted_at IS NULL`

		res, err := tx.ExecContext(ctx, q,
			in.Name, in.TargetAmountMinor, in.CurrentAmountMinor, in.Currency, in.StartDate, in.TargetDate, in.Notes, in.Status,
			in.UpdatedBy, in.UpdatedAt, in.ID,
		)
		if err != nil {
			return err
		}
		affected, err := res.RowsAffected()
		if err != nil {
			return err
		}
		if affected == 0 {
			return domain.ErrSavingsNotFound
		}
		return r.getByID(ctx, tx, in.ID, &out)
	})
	if err != nil {
		return domain.Savings{}, err
	}
	out.TenantID = in.TenantID
	return out, nil
}

func (r *RepositoryPG) SoftDelete(ctx context.Context, tenantID, savingsID, actorUserID string) error {
	return db.WithTenantTx(ctx, r.conn, func(tx *sql.Tx) error {
		const q = `
UPDATE finance_savings
SET deleted_at = $1, updated_by = $2, updated_at = $1
WHERE id = $3 AND deleted_at IS NULL`

		now := time.Now().UTC()
		res, err := tx.ExecContext(ctx, q, now, actorUserID, savingsID)
		if err != nil {
			return err
		}
		affected, err := res.RowsAffected()
		if err != nil {
			return err
		}
		if affected == 0 {
			return domain.ErrSavingsNotFound
		}
		return nil
	})
}

func toStringPtr(value sql.NullString) *string {
	if !value.Valid {
		return nil
	}
	out := value.String
	return &out
}

func toTimePtr(value sql.NullTime) *time.Time {
	if !value.Valid {
		return nil
	}
	out := value.Time.UTC()
	return &out
}
