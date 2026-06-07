package infra

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"pekan/backend/internal/modules/finance/reports/domain"
	"pekan/backend/internal/platform/db"
)

type TransactionExportPG struct {
	db *sql.DB
}

func NewTransactionExportPG(db *sql.DB) *TransactionExportPG {
	return &TransactionExportPG{db: db}
}

func (r *TransactionExportPG) ListTransactions(ctx context.Context, tenantID string, dateFrom, dateTo, categoryID, transactionType *string) ([]domain.TransactionRow, error) {
	clauses := []string{"t.tenant_id = $1", "t.deleted_at IS NULL"}
	args := []any{tenantID}
	idx := 2

	if dateFrom != nil && strings.TrimSpace(*dateFrom) != "" {
		clauses = append(clauses, fmt.Sprintf("t.transaction_date >= $%d", idx))
		args = append(args, *dateFrom)
		idx++
	}
	if dateTo != nil && strings.TrimSpace(*dateTo) != "" {
		clauses = append(clauses, fmt.Sprintf("t.transaction_date <= $%d", idx))
		args = append(args, *dateTo)
		idx++
	}
	if categoryID != nil && strings.TrimSpace(*categoryID) != "" {
		clauses = append(clauses, fmt.Sprintf("t.category_id = $%d", idx))
		args = append(args, *categoryID)
		idx++
	}
	if transactionType != nil && strings.TrimSpace(*transactionType) != "" {
		clauses = append(clauses, fmt.Sprintf("t.type = $%d", idx))
		args = append(args, strings.ToLower(strings.TrimSpace(*transactionType)))
		idx++
	}

	where := strings.Join(clauses, " AND ")
	q := fmt.Sprintf(`
SELECT t.id,
       t.input_date,
       t.account_id,
       a.name AS account_name,
       t.category_id,
       c.name AS category_name,
       t.type,
       t.amount_minor,
       t.currency,
       t.transaction_date,
       t.description,
       t.merchant_name,
       t.payment_method
FROM finance_transactions t
JOIN finance_accounts a ON a.id = t.account_id AND a.tenant_id = t.tenant_id
LEFT JOIN finance_categories c ON c.id = t.category_id AND c.tenant_id = t.tenant_id
WHERE %s
ORDER BY t.transaction_date ASC, t.created_at ASC`, where)

	items := make([]domain.TransactionRow, 0)
	err := db.WithTenantTx(ctx, r.db, func(tx *sql.Tx) error {
		rows, err := tx.QueryContext(ctx, q, args...)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var row domain.TransactionRow
			if err := rows.Scan(
				&row.ID, &row.InputDate, &row.AccountID, &row.AccountName, &row.CategoryID, &row.CategoryName, &row.Type,
				&row.AmountMinor, &row.Currency, &row.TransactionDate, &row.Description, &row.MerchantName, &row.PaymentMethod,
			); err != nil {
				return err
			}
			items = append(items, row)
		}
		return rows.Err()
	})
	if err != nil {
		return nil, err
	}
	return items, nil
}

func (r *TransactionExportPG) ListSavings(ctx context.Context, tenantID string, dateFrom, dateTo, status *string) ([]domain.SavingsRow, error) {
	clauses := []string{"tenant_id = $1", "deleted_at IS NULL"}
	args := []any{tenantID}
	idx := 2

	if dateFrom != nil && strings.TrimSpace(*dateFrom) != "" {
		clauses = append(clauses, fmt.Sprintf("COALESCE(target_date, created_at::date) >= $%d", idx))
		args = append(args, *dateFrom)
		idx++
	}
	if dateTo != nil && strings.TrimSpace(*dateTo) != "" {
		clauses = append(clauses, fmt.Sprintf("COALESCE(target_date, created_at::date) <= $%d", idx))
		args = append(args, *dateTo)
		idx++
	}
	if status != nil && strings.TrimSpace(*status) != "" {
		clauses = append(clauses, fmt.Sprintf("status = $%d", idx))
		args = append(args, strings.ToLower(strings.TrimSpace(*status)))
		idx++
	}

	q := fmt.Sprintf(`
SELECT id, name, target_amount_minor, current_amount_minor, progress_percent, currency, start_date, target_date, status, updated_at
FROM finance_savings
WHERE %s
ORDER BY updated_at DESC`, strings.Join(clauses, " AND "))

	items := make([]domain.SavingsRow, 0)
	err := db.WithTenantTx(ctx, r.db, func(tx *sql.Tx) error {
		rows, err := tx.QueryContext(ctx, q, args...)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var row domain.SavingsRow
			if err := rows.Scan(
				&row.ID,
				&row.Name,
				&row.TargetAmountMinor,
				&row.CurrentAmountMinor,
				&row.ProgressPercent,
				&row.Currency,
				&row.StartDate,
				&row.TargetDate,
				&row.Status,
				&row.UpdatedAt,
			); err != nil {
				return err
			}
			items = append(items, row)
		}
		return rows.Err()
	})
	if err != nil {
		return nil, err
	}
	return items, nil
}

func (r *TransactionExportPG) ListBudgets(ctx context.Context, tenantID string, dateFrom, dateTo, status *string) ([]domain.BudgetRow, error) {
	clauses := []string{"b.tenant_id = $1", "b.deleted_at IS NULL"}
	args := []any{tenantID}
	idx := 2

	if dateFrom != nil && strings.TrimSpace(*dateFrom) != "" {
		clauses = append(clauses, fmt.Sprintf("b.start_date >= $%d", idx))
		args = append(args, *dateFrom)
		idx++
	}
	if dateTo != nil && strings.TrimSpace(*dateTo) != "" {
		clauses = append(clauses, fmt.Sprintf("COALESCE(b.end_date, b.start_date) <= $%d", idx))
		args = append(args, *dateTo)
		idx++
	}
	if status != nil && strings.TrimSpace(*status) != "" {
		clauses = append(clauses, fmt.Sprintf("b.status = $%d", idx))
		args = append(args, strings.ToLower(strings.TrimSpace(*status)))
		idx++
	}

	q := fmt.Sprintf(`
SELECT
  b.id,
  b.name,
  b.category_id,
  COALESCE((SELECT string_agg(c.name, ', ') FROM finance_categories c WHERE c.id::text = ANY(string_to_array(b.category_id, ',')) AND c.tenant_id = b.tenant_id), 'Semua Kategori') AS category_name,
  b.amount_limit_minor,
  b.currency,
  b.period,
  b.start_date,
  b.end_date,
  b.alert_threshold_pct,
  CASE
      WHEN b.status = 'active' AND b.end_date IS NOT NULL AND b.end_date < CURRENT_DATE THEN 'ended'
      ELSE b.status
  END AS status,
  b.updated_at
FROM finance_budgets b
WHERE %s
ORDER BY b.updated_at DESC`, strings.Join(clauses, " AND "))

	items := make([]domain.BudgetRow, 0)
	err := db.WithTenantTx(ctx, r.db, func(tx *sql.Tx) error {
		rows, err := tx.QueryContext(ctx, q, args...)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var row domain.BudgetRow
			if err := rows.Scan(
				&row.ID,
				&row.Name,
				&row.CategoryID,
				&row.CategoryName,
				&row.AmountLimitMinor,
				&row.Currency,
				&row.Period,
				&row.StartDate,
				&row.EndDate,
				&row.AlertThresholdPct,
				&row.Status,
				&row.UpdatedAt,
			); err != nil {
				return err
			}
			items = append(items, row)
		}
		return rows.Err()
	})
	if err != nil {
		return nil, err
	}
	return items, nil
}

func (r *TransactionExportPG) ListReminders(ctx context.Context, tenantID string, dateFrom, dateTo, status *string) ([]domain.ReminderRow, error) {
	clauses := []string{"tenant_id = $1", "deleted_at IS NULL"}
	args := []any{tenantID}
	idx := 2

	if dateFrom != nil && strings.TrimSpace(*dateFrom) != "" {
		clauses = append(clauses, fmt.Sprintf("due_date >= $%d", idx))
		args = append(args, *dateFrom)
		idx++
	}
	if dateTo != nil && strings.TrimSpace(*dateTo) != "" {
		clauses = append(clauses, fmt.Sprintf("due_date <= $%d", idx))
		args = append(args, *dateTo)
		idx++
	}
	if status != nil && strings.TrimSpace(*status) != "" {
		clauses = append(clauses, fmt.Sprintf("status = $%d", idx))
		args = append(args, strings.ToLower(strings.TrimSpace(*status)))
		idx++
	}

	q := fmt.Sprintf(`
SELECT
  id,
  title,
  description,
  amount_minor,
  currency,
  due_date,
  repeat_interval,
  status,
  last_triggered_at,
  updated_at
FROM finance_reminders
WHERE %s
ORDER BY due_date ASC, updated_at DESC`, strings.Join(clauses, " AND "))

	items := make([]domain.ReminderRow, 0)
	err := db.WithTenantTx(ctx, r.db, func(tx *sql.Tx) error {
		rows, err := tx.QueryContext(ctx, q, args...)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var row domain.ReminderRow
			if err := rows.Scan(
				&row.ID,
				&row.Title,
				&row.Description,
				&row.AmountMinor,
				&row.Currency,
				&row.DueDate,
				&row.RepeatInterval,
				&row.Status,
				&row.LastTriggeredAt,
				&row.UpdatedAt,
			); err != nil {
				return err
			}
			items = append(items, row)
		}
		return rows.Err()
	})
	if err != nil {
		return nil, err
	}
	return items, nil
}
