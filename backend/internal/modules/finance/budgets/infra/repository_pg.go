package infra

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"pekan/backend/internal/modules/finance/budgets/domain"
	"pekan/backend/internal/platform/db"
)

type RepositoryPG struct {
	conn *sql.DB
}

func NewRepositoryPG(conn *sql.DB) *RepositoryPG {
	return &RepositoryPG{conn: conn}
}

func (r *RepositoryPG) Create(ctx context.Context, in domain.Budget) (domain.Budget, error) {
	var out domain.Budget
	err := db.WithTenantTx(ctx, r.conn, func(tx *sql.Tx) error {
		in.ID = uuid.NewString()
		ida := strings.ToUpper(in.ID[:8])
		
		const insertQuery = `
INSERT INTO finance_budgets (
  id, ida, name, category_id, amount_limit_minor, currency, period, start_date, end_date,
  alert_threshold_pct, notes, status, created_by, updated_by, created_at, updated_at
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16)`

		if _, err := tx.ExecContext(ctx, insertQuery,
			in.ID, ida, in.Name, in.CategoryID, in.AmountLimitMinor, in.Currency, in.Period, in.StartDate, in.EndDate,
			in.AlertThresholdPct, in.Notes, in.Status, in.CreatedBy, in.UpdatedBy, in.CreatedAt, in.UpdatedAt,
		); err != nil {
			return err
		}

		in.IDA = ida
		out = in
		return nil
	})
	return out, err
}

func (r *RepositoryPG) GetByID(ctx context.Context, tenantID, budgetID string) (domain.Budget, error) {
	var out domain.Budget
	err := db.WithTenantTx(ctx, r.conn, func(tx *sql.Tx) error {
		const q = `
SELECT b.id,
       b.name,
       b.category_id,
       (SELECT string_agg(c.name, ', ') FROM finance_categories c WHERE c.id::text = ANY(string_to_array(b.category_id, ','))) AS category_name,
       b.amount_limit_minor,
       COALESCE(spent.spent_amount_minor, 0) AS spent_amount_minor,
       CASE
           WHEN b.amount_limit_minor > 0 THEN ROUND((COALESCE(spent.spent_amount_minor, 0)::numeric / b.amount_limit_minor::numeric) * 100, 2)
           ELSE 0
       END AS progress_percent,
       b.currency,
       b.period,
       b.start_date,
       b.end_date,
       b.alert_threshold_pct,
       b.notes,
       CASE
           WHEN b.status = 'active' AND b.end_date IS NOT NULL AND b.end_date < CURRENT_DATE THEN 'ended'
           ELSE b.status
       END AS status,
       b.created_by,
       b.updated_by,
       b.created_at,
       b.updated_at,
       b.deleted_at,
       COALESCE(to_jsonb(b)->>'ida', '') AS ida
FROM finance_budgets b
LEFT JOIN LATERAL (
    SELECT COALESCE(SUM(t.amount_minor), 0) AS spent_amount_minor
    FROM finance_transactions t
    WHERE t.deleted_at IS NULL
      AND t.type = 'expense'
      AND t.transaction_date >= b.start_date
      AND (b.end_date IS NULL OR t.transaction_date <= b.end_date)
      AND (
          b.category_id IS NULL OR b.category_id = ''
          OR t.category_id::text = ANY(string_to_array(b.category_id, ','))
          OR t.category_id IN (
              WITH RECURSIVE cat_tree AS (
                  SELECT id FROM finance_categories WHERE parent_id::text = ANY(string_to_array(b.category_id, ','))
                  UNION ALL
                  SELECT c.id FROM finance_categories c JOIN cat_tree ct ON c.parent_id = ct.id
              )
              SELECT id FROM cat_tree
          )
      )
) spent ON TRUE
WHERE b.id = $1 AND b.deleted_at IS NULL`

		var (
			categoryID   sql.NullString
			categoryName sql.NullString
			alertPct     sql.NullInt64
			notes        sql.NullString
			createdBy    sql.NullString
			updatedBy    sql.NullString
			endDate      sql.NullTime
			deletedAt    sql.NullTime
			ida          sql.NullString
		)
		err := tx.QueryRowContext(ctx, q, budgetID).Scan(
			&out.ID, &out.Name, &categoryID, &categoryName, &out.AmountLimitMinor, &out.SpentAmountMinor, &out.ProgressPercent, &out.Currency, &out.Period,
			&out.StartDate, &endDate, &alertPct, &notes, &out.Status, &createdBy, &updatedBy,
			&out.CreatedAt, &out.UpdatedAt, &deletedAt, &ida,
		)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return domain.ErrBudgetNotFound
			}
			return err
		}
		out.TenantID = tenantID
		out.CategoryID = toStringPtr(categoryID)
		out.CategoryName = toStringPtr(categoryName)
		out.AlertThresholdPct = toIntPtr(alertPct)
		out.Notes = toStringPtr(notes)
		out.CreatedBy = toString(createdBy)
		out.UpdatedBy = toString(updatedBy)
		out.EndDate = toTimePtr(endDate)
		out.DeletedAt = toTimePtr(deletedAt)
		out.IDA = normalizeIDA(toString(ida), out.ID)
		return nil
	})
	return out, err
}

func (r *RepositoryPG) List(ctx context.Context, filter domain.ListFilter) ([]domain.Budget, int64, error) {
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.PageSize <= 0 || filter.PageSize > 100 {
		filter.PageSize = 20
	}

	var items []domain.Budget
	var total int64

	err := db.WithTenantTx(ctx, r.conn, func(tx *sql.Tx) error {
		var (
			clauses = []string{"b.deleted_at IS NULL"}
			args    []any
			idx     = 1
		)

		if filter.Status != nil && strings.TrimSpace(*filter.Status) != "" {
			clauses = append(clauses, fmt.Sprintf("b.status = $%d", idx))
			args = append(args, strings.ToLower(strings.TrimSpace(*filter.Status)))
			idx++
		}

		where := strings.Join(clauses, " AND ")
		countQuery := "SELECT COUNT(1) FROM finance_budgets b WHERE " + where

		if err := tx.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
			return err
		}

		offset := (filter.Page - 1) * filter.PageSize
		dataArgs := append(args, filter.PageSize, offset)
		limitIdx := idx
		offsetIdx := idx + 1

		dataQuery := fmt.Sprintf(`
SELECT b.id,
       b.name,
       b.category_id,
       (SELECT string_agg(c.name, ', ') FROM finance_categories c WHERE c.id::text = ANY(string_to_array(b.category_id, ','))) AS category_name,
       b.amount_limit_minor,
       COALESCE(spent.spent_amount_minor, 0) AS spent_amount_minor,
       CASE
           WHEN b.amount_limit_minor > 0 THEN ROUND((COALESCE(spent.spent_amount_minor, 0)::numeric / b.amount_limit_minor::numeric) * 100, 2)
           ELSE 0
       END AS progress_percent,
       b.currency,
       b.period,
       b.start_date,
       b.end_date,
       b.alert_threshold_pct,
       b.notes,
       CASE
           WHEN b.status = 'active' AND b.end_date IS NOT NULL AND b.end_date < CURRENT_DATE THEN 'ended'
           ELSE b.status
       END AS status,
       b.created_by,
       b.updated_by,
       b.created_at,
       b.updated_at,
       b.deleted_at,
       COALESCE(to_jsonb(b)->>'ida', '') AS ida
FROM finance_budgets b
LEFT JOIN LATERAL (
    SELECT COALESCE(SUM(t.amount_minor), 0) AS spent_amount_minor
    FROM finance_transactions t
    WHERE t.deleted_at IS NULL
      AND t.type = 'expense'
      AND t.transaction_date >= b.start_date
      AND (b.end_date IS NULL OR t.transaction_date <= b.end_date)
      AND (
          b.category_id IS NULL OR b.category_id = ''
          OR t.category_id::text = ANY(string_to_array(b.category_id, ','))
          OR t.category_id IN (
              WITH RECURSIVE cat_tree AS (
                  SELECT id FROM finance_categories WHERE parent_id::text = ANY(string_to_array(b.category_id, ','))
                  UNION ALL
                  SELECT c.id FROM finance_categories c JOIN cat_tree ct ON c.parent_id = ct.id
              )
              SELECT id FROM cat_tree
          )
      )
) spent ON TRUE
WHERE %s
ORDER BY b.start_date DESC, b.created_at DESC
LIMIT $%d OFFSET $%d`, where, limitIdx, offsetIdx)

		rows, err := tx.QueryContext(ctx, dataQuery, dataArgs...)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var item domain.Budget
			var (
				categoryID   sql.NullString
				categoryName sql.NullString
				alertPct     sql.NullInt64
				notes        sql.NullString
				createdBy    sql.NullString
				updatedBy    sql.NullString
				endDate      sql.NullTime
				deletedAt    sql.NullTime
				ida          sql.NullString
			)
			if err := rows.Scan(
				&item.ID, &item.Name, &categoryID, &categoryName, &item.AmountLimitMinor, &item.SpentAmountMinor, &item.ProgressPercent, &item.Currency, &item.Period,
				&item.StartDate, &endDate, &alertPct, &notes, &item.Status, &createdBy, &updatedBy,
				&item.CreatedAt, &item.UpdatedAt, &deletedAt, &ida,
			); err != nil {
				return err
			}
			item.TenantID = filter.TenantID
			item.CategoryID = toStringPtr(categoryID)
			item.CategoryName = toStringPtr(categoryName)
			item.AlertThresholdPct = toIntPtr(alertPct)
			item.Notes = toStringPtr(notes)
			item.CreatedBy = toString(createdBy)
			item.UpdatedBy = toString(updatedBy)
			item.EndDate = toTimePtr(endDate)
			item.DeletedAt = toTimePtr(deletedAt)
			item.IDA = normalizeIDA(toString(ida), item.ID)
			items = append(items, item)
		}
		return rows.Err()
	})

	return items, total, err
}

func (r *RepositoryPG) Update(ctx context.Context, in domain.Budget) (domain.Budget, error) {
	err := db.WithTenantTx(ctx, r.conn, func(tx *sql.Tx) error {
		const q = `
UPDATE finance_budgets
SET name = $1, category_id = $2, amount_limit_minor = $3, currency = $4, period = $5, start_date = $6,
    end_date = $7, alert_threshold_pct = $8, notes = $9, status = $10, updated_by = $11, updated_at = $12
WHERE id = $13 AND deleted_at IS NULL`

		res, err := tx.ExecContext(ctx, q,
			in.Name, in.CategoryID, in.AmountLimitMinor, in.Currency, in.Period, in.StartDate, in.EndDate,
			in.AlertThresholdPct, in.Notes, in.Status, in.UpdatedBy, in.UpdatedAt, in.ID,
		)
		if err != nil {
			return err
		}
		affected, err := res.RowsAffected()
		if err != nil {
			return err
		}
		if affected == 0 {
			return domain.ErrBudgetNotFound
		}
		
		// Internal call to GetByID (within tx) would be better but we'll use outside for simplicity or refactor GetByID to take tx
		// For now we'll just return 'in' or re-fetch. Let's re-fetch manually.
		return nil
	})
	if err != nil {
		return domain.Budget{}, err
	}
	return r.GetByID(ctx, in.TenantID, in.ID)
}

func (r *RepositoryPG) SoftDelete(ctx context.Context, tenantID, budgetID, actorUserID string) error {
	return db.WithTenantTx(ctx, r.conn, func(tx *sql.Tx) error {
		const q = `
UPDATE finance_budgets
SET deleted_at = $1, updated_by = $2, updated_at = $1
WHERE id = $3 AND deleted_at IS NULL`

		now := time.Now().UTC()
		res, err := tx.ExecContext(ctx, q, now, actorUserID, budgetID)
		if err != nil {
			return err
		}
		affected, err := res.RowsAffected()
		if err != nil {
			return err
		}
		if affected == 0 {
			return domain.ErrBudgetNotFound
		}
		return nil
	})
}

func (r *RepositoryPG) ValidateCategory(ctx context.Context, tenantID, categoryID string) error {
	return db.WithTenantTx(ctx, r.conn, func(tx *sql.Tx) error {
		const q = `
SELECT 1
FROM finance_categories
WHERE id = $1 AND category_type = 'expense' AND is_active = TRUE`

		var exists int
		if err := tx.QueryRowContext(ctx, q, categoryID).Scan(&exists); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return domain.ErrCategoryNotFound
			}
			return err
		}
		return nil
	})
}

func (r *RepositoryPG) ResolveCategoryID(ctx context.Context, tenantID, actorUserID string, categoryID, categoryName *string) (*string, error) {
	var out *string
	err := db.WithTenantTx(ctx, r.conn, func(tx *sql.Tx) error {
		if categoryID != nil && strings.TrimSpace(*categoryID) != "" {
			cleanID := strings.TrimSpace(*categoryID)
			ids := strings.Split(cleanID, ",")
			var validIDs []string
			for _, idStr := range ids {
				id := strings.TrimSpace(idStr)
				if id == "" {
					continue
				}
				const q = `SELECT id FROM finance_categories WHERE id = $1 AND category_type = 'expense' AND is_active = TRUE`
				var exists string
				if err := tx.QueryRowContext(ctx, q, id).Scan(&exists); err != nil {
					if errors.Is(err, sql.ErrNoRows) {
						return domain.ErrCategoryNotFound
					}
					return err
				}
				validIDs = append(validIDs, exists)
			}
			joined := strings.Join(validIDs, ",")
			out = &joined
			return nil
		}

		if categoryName == nil || strings.TrimSpace(*categoryName) == "" {
			return nil
		}

		normalizedName := strings.TrimSpace(*categoryName)
		names := strings.Split(normalizedName, ",")
		var resolvedIDs []string
		for _, nameStr := range names {
			name := strings.TrimSpace(nameStr)
			if name == "" {
				continue
			}

			const findQuery = `
SELECT id
FROM finance_categories
WHERE category_type = 'expense'
  AND LOWER(name) = LOWER($1)
  AND is_active = TRUE
LIMIT 1`

			var existingID string
			if err := tx.QueryRowContext(ctx, findQuery, name).Scan(&existingID); err == nil {
				resolvedIDs = append(resolvedIDs, existingID)
				continue
			} else if !errors.Is(err, sql.ErrNoRows) {
				return err
			}

			newID := uuid.NewString()
			now := time.Now().UTC()
			const insertQuery = `
INSERT INTO finance_categories (
  id, name, category_type, parent_id, is_active, created_by, created_at, updated_at
) VALUES ($1,$2,'expense',NULL,TRUE,$3,$4,$4)`

			if _, err := tx.ExecContext(ctx, insertQuery, newID, name, actorUserID, now); err != nil {
				if err := tx.QueryRowContext(ctx, findQuery, name).Scan(&existingID); err == nil {
					resolvedIDs = append(resolvedIDs, existingID)
					continue
				}
				return err
			}
			resolvedIDs = append(resolvedIDs, newID)
		}
		joined := strings.Join(resolvedIDs, ",")
		out = &joined
		return nil
	})
	return out, err
}

func toString(value sql.NullString) string {
	if !value.Valid {
		return ""
	}
	return value.String
}

func toStringPtr(value sql.NullString) *string {
	if !value.Valid {
		return nil
	}
	out := value.String
	return &out
}

func toIntPtr(value sql.NullInt64) *int {
	if !value.Valid {
		return nil
	}
	converted := int(value.Int64)
	return &converted
}

func toTimePtr(value sql.NullTime) *time.Time {
	if !value.Valid {
		return nil
	}
	out := value.Time.UTC()
	return &out
}

func normalizeIDA(rawIDA, budgetID string) string {
	cleanIDA := strings.TrimSpace(rawIDA)
	if cleanIDA != "" {
		return cleanIDA
	}

	cleanID := strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(budgetID), "-", ""))
	if cleanID == "" {
		return "BGT-UNKNOWN"
	}
	if len(cleanID) > 8 {
		cleanID = cleanID[:8]
	}
	return "BGT-" + cleanID
}
func (r *RepositoryPG) FindActiveByCategory(ctx context.Context, tenantID, categoryID string) ([]domain.Budget, error) {
	var items []domain.Budget
	err := db.WithTenantTx(ctx, r.conn, func(tx *sql.Tx) error {
		const q = `
SELECT b.id,
       b.name,
       b.category_id,
       b.amount_limit_minor,
       COALESCE(spent.spent_amount_minor, 0) AS spent_amount_minor,
       CASE
           WHEN b.amount_limit_minor > 0 THEN ROUND((COALESCE(spent.spent_amount_minor, 0)::numeric / b.amount_limit_minor::numeric) * 100, 2)
           ELSE 0
       END AS progress_percent,
       b.currency,
       b.period,
       b.start_date,
       b.end_date,
       b.alert_threshold_pct,
       b.notes,
       CASE
           WHEN b.status = 'active' AND b.end_date IS NOT NULL AND b.end_date < CURRENT_DATE THEN 'ended'
           ELSE b.status
       END AS status,
       COALESCE(to_jsonb(b)->>'ida', '') AS ida
FROM finance_budgets b
LEFT JOIN LATERAL (
    SELECT COALESCE(SUM(t.amount_minor), 0) AS spent_amount_minor
    FROM finance_transactions t
    WHERE t.deleted_at IS NULL
      AND t.type = 'expense'
      AND t.transaction_date >= b.start_date
      AND (b.end_date IS NULL OR t.transaction_date <= b.end_date)
      AND (
          b.category_id IS NULL OR b.category_id = ''
          OR t.category_id::text = ANY(string_to_array(b.category_id, ','))
          OR t.category_id IN (
              WITH RECURSIVE cat_tree AS (
                  SELECT id FROM finance_categories WHERE parent_id::text = ANY(string_to_array(b.category_id, ','))
                  UNION ALL
                  SELECT c.id FROM finance_categories c JOIN cat_tree ct ON c.parent_id = ct.id
              )
              SELECT id FROM cat_tree
          )
      )
) spent ON TRUE
WHERE b.deleted_at IS NULL 
  AND b.status = 'active'
  AND (b.end_date IS NULL OR b.end_date >= CURRENT_DATE)
  AND (b.category_id IS NULL OR b.category_id = '' OR $1::text = ANY(string_to_array(b.category_id, ',')) OR EXISTS (
      WITH RECURSIVE cat_tree AS (
          SELECT parent_id FROM finance_categories WHERE id = $1
          UNION ALL
          SELECT c.parent_id FROM finance_categories c JOIN cat_tree ct ON c.id = ct.parent_id
          WHERE c.parent_id IS NOT NULL
      )
      SELECT 1 FROM cat_tree ct WHERE ct.parent_id::text = ANY(string_to_array(b.category_id, ','))
  ))`

		rows, err := tx.QueryContext(ctx, q, categoryID)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var item domain.Budget
			var (
				catID    sql.NullString
				alertPct sql.NullInt64
				notes    sql.NullString
				endDate  sql.NullTime
				ida      sql.NullString
			)
			if err := rows.Scan(
				&item.ID, &item.Name, &catID, &item.AmountLimitMinor, &item.SpentAmountMinor, &item.ProgressPercent, &item.Currency, &item.Period,
				&item.StartDate, &endDate, &alertPct, &notes, &item.Status, &ida,
			); err != nil {
				return err
			}
			item.TenantID = tenantID
			item.CategoryID = toStringPtr(catID)
			item.AlertThresholdPct = toIntPtr(alertPct)
			item.EndDate = toTimePtr(endDate)
			item.Notes = toStringPtr(notes)
			item.IDA = normalizeIDA(toString(ida), item.ID)
			items = append(items, item)
		}
		return rows.Err()
	})
	return items, err
}
