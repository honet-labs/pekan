package infra

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"pekan/backend/internal/modules/finance/dashboard/domain"
	"pekan/backend/internal/platform/db"
)

type RepositoryPG struct {
	conn *sql.DB
}

func NewRepositoryPG(conn *sql.DB) *RepositoryPG {
	return &RepositoryPG{conn: conn}
}

func (r *RepositoryPG) GetSummary(ctx context.Context, tenantID string, dateFrom, dateTo *string) (domain.Summary, error) {
	var out domain.Summary
	err := db.WithTenantTx(ctx, r.conn, func(tx *sql.Tx) error {
		clauses := []string{"deleted_at IS NULL"}
		var args []any
		idx := 1

		if dateFrom != nil && strings.TrimSpace(*dateFrom) != "" {
			clauses = append(clauses, fmt.Sprintf("transaction_date::date >= $%d::date", idx))
			args = append(args, *dateFrom)
			idx++
		}
		if dateTo != nil && strings.TrimSpace(*dateTo) != "" {
			clauses = append(clauses, fmt.Sprintf("transaction_date::date <= $%d::date", idx))
			args = append(args, *dateTo)
			idx++
		}
		where := strings.Join(clauses, " AND ")
		q := fmt.Sprintf(`
SELECT
  COALESCE(SUM(CASE WHEN type = 'income' THEN amount_minor ELSE 0 END), 0) AS total_income,
  COALESCE(SUM(CASE WHEN type = 'expense' THEN amount_minor ELSE 0 END), 0) AS total_expense,
  COALESCE(SUM(CASE WHEN type = 'transfer' THEN amount_minor ELSE 0 END), 0) AS total_transfer,
  COALESCE(SUM(CASE WHEN type = 'savings' THEN amount_minor ELSE 0 END), 0) AS total_savings,
  COUNT(1) AS total_count,
  COALESCE(SUM(CASE WHEN type = 'income' THEN 1 ELSE 0 END), 0) AS income_count,
  COALESCE(SUM(CASE WHEN type = 'expense' THEN 1 ELSE 0 END), 0) AS expense_count,
  COALESCE(SUM(CASE WHEN type = 'transfer' THEN 1 ELSE 0 END), 0) AS transfer_count,
  COALESCE(SUM(CASE WHEN type = 'savings' THEN 1 ELSE 0 END), 0) AS savings_count
FROM finance_transactions
WHERE %s`, where)

		var income, expense, transfer, totalSavings, count, incomeCount, expenseCount, transferCount, savingsCount int64
		if err := tx.QueryRowContext(ctx, q, args...).Scan(&income, &expense, &transfer, &totalSavings, &count, &incomeCount, &expenseCount, &transferCount, &savingsCount); err != nil {
			return err
		}
		out = domain.Summary{
			TotalIncomeMinor:   income,
			TotalExpenseMinor:  expense,
			TotalTransferMinor: transfer,
			NetAmountMinor:     income - expense,
			TotalSavingsMinor:  totalSavings,
			TransactionCount:   count,
			IncomeCount:        incomeCount,
			ExpenseCount:       expenseCount,
			TransferCount:      transferCount,
			SavingsCount:       savingsCount,
		}
		return nil
	})
	return out, err
}

func (r *RepositoryPG) GetDailySeries(ctx context.Context, tenantID string, dateFrom, dateTo *string) ([]domain.SeriesPoint, error) {
	var items []domain.SeriesPoint
	err := db.WithTenantTx(ctx, r.conn, func(tx *sql.Tx) error {
		clauses := []string{"deleted_at IS NULL"}
		var args []any
		idx := 1

		if dateFrom != nil && strings.TrimSpace(*dateFrom) != "" {
			clauses = append(clauses, fmt.Sprintf("transaction_date::date >= $%d::date", idx))
			args = append(args, *dateFrom)
			idx++
		}
		if dateTo != nil && strings.TrimSpace(*dateTo) != "" {
			clauses = append(clauses, fmt.Sprintf("transaction_date::date <= $%d::date", idx))
			args = append(args, *dateTo)
			idx++
		}
		where := strings.Join(clauses, " AND ")

		q := fmt.Sprintf(`
SELECT date_trunc('day', transaction_date) AS day,
       COALESCE(SUM(CASE WHEN type = 'income' THEN amount_minor ELSE 0 END), 0) AS income_minor,
       COALESCE(SUM(CASE WHEN type = 'expense' THEN amount_minor ELSE 0 END), 0) AS expense_minor
FROM finance_transactions
WHERE %s
GROUP BY day
ORDER BY day ASC`, where)

		rows, err := tx.QueryContext(ctx, q, args...)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var point domain.SeriesPoint
			if err := rows.Scan(&point.Date, &point.IncomeMinor, &point.ExpenseMinor); err != nil {
				return err
			}
			items = append(items, point)
		}
		return rows.Err()
	})
	return items, err
}

func (r *RepositoryPG) GetTopCategories(ctx context.Context, tenantID string, dateFrom, dateTo *string, limit int) ([]domain.CategoryTotal, error) {
	var items []domain.CategoryTotal
	err := db.WithTenantTx(ctx, r.conn, func(tx *sql.Tx) error {
		clauses := []string{"t.deleted_at IS NULL"}
		var args []any
		idx := 1

		if dateFrom != nil && strings.TrimSpace(*dateFrom) != "" {
			clauses = append(clauses, fmt.Sprintf("t.transaction_date::date >= $%d::date", idx))
			args = append(args, *dateFrom)
			idx++
		}
		if dateTo != nil && strings.TrimSpace(*dateTo) != "" {
			clauses = append(clauses, fmt.Sprintf("t.transaction_date::date <= $%d::date", idx))
			args = append(args, *dateTo)
			idx++
		}
		where := strings.Join(clauses, " AND ")

		args = append(args, limit)
		q := fmt.Sprintf(`
SELECT c.id, 
       COALESCE(c.name, CASE WHEN t.type = 'savings' THEN 'Tabungan' WHEN t.type = 'transfer' THEN 'Transfer' ELSE 'Tanpa Kategori' END) AS category_label,
       t.type,
       COALESCE(SUM(t.amount_minor), 0) AS total_minor,
       COUNT(1) AS total_count
FROM finance_transactions t
LEFT JOIN finance_categories c ON c.id = t.category_id
WHERE %s
GROUP BY c.id, category_label, t.type
ORDER BY total_count DESC, total_minor DESC
LIMIT $%d`, where, idx)

		rows, err := tx.QueryContext(ctx, q, args...)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var item domain.CategoryTotal
			if err := rows.Scan(&item.CategoryID, &item.CategoryName, &item.TransactionType, &item.TotalMinor, &item.Count); err != nil {
				return err
			}
			items = append(items, item)
		}
		return rows.Err()
	})
	return items, err
}
