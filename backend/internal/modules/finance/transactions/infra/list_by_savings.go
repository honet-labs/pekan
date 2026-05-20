package infra

import (
	"context"

	"pekan/backend/internal/modules/finance/transactions/domain"
)

func (r *RepositoryPG) ListBySavingsID(ctx context.Context, tenantID, savingsID string) ([]domain.Transaction, error) {
	const q = `
SELECT DISTINCT
       t.id,
       t.tenant_id,
       t.account_id,
       a.name AS account_name,
       t.category_id,
       c.name AS category_name,
       t.type,
       t.amount_minor,
       t.currency,
       t.input_date,
       t.transaction_date,
       t.description,
       t.created_by,
       t.updated_by,
       t.created_at,
       t.updated_at
FROM finance_transactions t
JOIN finance_accounts a ON a.id = t.account_id AND a.tenant_id = t.tenant_id
LEFT JOIN finance_categories c ON c.id = t.category_id AND c.tenant_id = t.tenant_id
JOIN finance_transaction_savings_links l ON l.tenant_id = t.tenant_id AND l.transaction_id = t.id
WHERE t.tenant_id = $1
  AND l.savings_id = $2
  AND t.deleted_at IS NULL
ORDER BY t.transaction_date DESC, t.created_at DESC`

	rows, err := r.conn.QueryContext(ctx, q, tenantID, savingsID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]domain.Transaction, 0)
	for rows.Next() {
		var trx domain.Transaction
		var trxType string
		if err := rows.Scan(
			&trx.ID, &trx.TenantID, &trx.AccountID, &trx.AccountName, &trx.CategoryID, &trx.CategoryName, &trxType,
			&trx.AmountMinor, &trx.Currency, &trx.InputDate, &trx.TransactionDate, &trx.Description, &trx.CreatedBy, &trx.UpdatedBy,
			&trx.CreatedAt, &trx.UpdatedAt,
		); err != nil {
			return nil, err
		}
		trx.Type = domain.TransactionType(trxType)
		out = append(out, trx)
	}

	return out, rows.Err()
}
