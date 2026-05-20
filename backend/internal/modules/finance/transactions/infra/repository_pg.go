package infra

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"pekan/backend/internal/modules/finance/transactions/domain"
	"pekan/backend/internal/platform/db"
)

type RepositoryPG struct {
	conn *sql.DB
}

func NewRepositoryPG(conn *sql.DB) *RepositoryPG {
	return &RepositoryPG{conn: conn}
}

func (r *RepositoryPG) Create(ctx context.Context, trx domain.Transaction) (domain.Transaction, error) {
	err := db.WithTenantTx(ctx, r.conn, func(tx *sql.Tx) error {
		trx.ID = uuid.NewString()
		if trx.InputDate.IsZero() {
			if !trx.CreatedAt.IsZero() {
				trx.InputDate = trx.CreatedAt
			} else {
				trx.InputDate = time.Now().UTC()
			}
		}
		const q = `
INSERT INTO finance_transactions (
  id, account_id, category_id, type, amount_minor, currency, input_date, transaction_date, description, merchant_name, receipt_number, payment_method, subtotal_minor, tax_minor, service_charge_minor, receipt_discount_minor, notes, created_by, updated_by, created_at, updated_at
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21)`

		_, err := tx.ExecContext(ctx, q,
			trx.ID, trx.AccountID, trx.CategoryID, trx.Type, trx.AmountMinor, trx.Currency,
			trx.InputDate, trx.TransactionDate, trx.Description, trx.MerchantName, trx.ReceiptNumber, trx.PaymentMethod, trx.SubtotalMinor, trx.TaxMinor, trx.ServiceChargeMinor, trx.ReceiptDiscountMinor, trx.Notes, trx.CreatedBy, trx.UpdatedBy, trx.CreatedAt, trx.UpdatedAt,
		)
		return err
	})
	if err != nil {
		return domain.Transaction{}, err
	}

	return trx, nil
}

func (r *RepositoryPG) GetByID(ctx context.Context, tenantID, transactionID string) (domain.Transaction, error) {
	var trx domain.Transaction
	err := db.WithTenantTx(ctx, r.conn, func(tx *sql.Tx) error {
		const q = `
SELECT t.id,
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
       t.merchant_name,
       t.receipt_number,
       t.payment_method,
       t.subtotal_minor,
       t.tax_minor,
       t.service_charge_minor,
       t.receipt_discount_minor,
       t.notes,
       COALESCE(t.created_by::text, '') AS created_by,
       COALESCE(NULLIF(u.full_name, ''), NULLIF(up.username, ''), NULLIF(u.email, ''), t.created_by::text, '') AS created_by_name,
       COALESCE(t.updated_by::text, '') AS updated_by,
       t.created_at,
       t.updated_at
FROM finance_transactions t
JOIN finance_accounts a ON a.id = t.account_id
LEFT JOIN finance_categories c ON c.id = t.category_id
LEFT JOIN public.users u ON u.id = t.created_by
LEFT JOIN public.user_profiles up ON up.user_id = u.id
WHERE t.id = $1 AND t.deleted_at IS NULL`

		var trxType string
		err := tx.QueryRowContext(ctx, q, transactionID).Scan(
			&trx.ID, &trx.AccountID, &trx.AccountName, &trx.CategoryID, &trx.CategoryName, &trxType, &trx.AmountMinor, &trx.Currency,
			&trx.InputDate, &trx.TransactionDate, &trx.Description, &trx.MerchantName, &trx.ReceiptNumber, &trx.PaymentMethod, &trx.SubtotalMinor, &trx.TaxMinor, &trx.ServiceChargeMinor, &trx.ReceiptDiscountMinor, &trx.Notes, &trx.CreatedBy, &trx.CreatedByName, &trx.UpdatedBy, &trx.CreatedAt, &trx.UpdatedAt,
		)
		if err != nil {
			return err
		}
		trx.TenantID = tenantID
		trx.Type = domain.TransactionType(trxType)
		return nil
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Transaction{}, domain.ErrTransactionNotFound
		}
		return domain.Transaction{}, err
	}
	return trx, nil
}

func (r *RepositoryPG) List(ctx context.Context, filter domain.ListFilter) ([]domain.Transaction, int64, error) {
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.PageSize <= 0 || filter.PageSize > 100 {
		filter.PageSize = 20
	}

	var out []domain.Transaction
	var total int64

	err := db.WithTenantTx(ctx, r.conn, func(tx *sql.Tx) error {
		var (
			clauses = []string{"t.deleted_at IS NULL"}
			args    = []any{}
			idx     = 1
		)

		if filter.Type != nil {
			clauses = append(clauses, fmt.Sprintf("t.type = $%d", idx))
			args = append(args, string(*filter.Type))
			idx++
		}
		if filter.DateFrom != nil {
			clauses = append(clauses, fmt.Sprintf("t.transaction_date >= $%d", idx))
			args = append(args, *filter.DateFrom)
			idx++
		}
		if filter.DateTo != nil {
			clauses = append(clauses, fmt.Sprintf("t.transaction_date <= $%d", idx))
			args = append(args, *filter.DateTo)
			idx++
		}
		if filter.CategoryID != nil && *filter.CategoryID != "" {
			clauses = append(clauses, fmt.Sprintf("t.category_id = $%d", idx))
			args = append(args, *filter.CategoryID)
			idx++
		}
		if query := strings.ToLower(strings.TrimSpace(filter.Query)); query != "" {
			clauses = append(clauses, fmt.Sprintf(`(
LOWER(COALESCE(t.description, '')) LIKE $%d
OR LOWER(COALESCE(t.merchant_name, '')) LIKE $%d
OR LOWER(COALESCE(a.name, '')) LIKE $%d
OR LOWER(COALESCE(c.name, '')) LIKE $%d
OR LOWER(COALESCE(NULLIF(u.full_name, ''), NULLIF(up.username, ''), NULLIF(u.email, ''), t.created_by::text)) LIKE $%d
OR LOWER(t.id::text) LIKE $%d
OR EXISTS (SELECT 1 FROM finance_transaction_items fti WHERE fti.transaction_id = t.id AND LOWER(fti.item_name) LIKE $%d)
OR CAST(t.amount_minor AS TEXT) LIKE $%d
)`, idx, idx, idx, idx, idx, idx, idx, idx))
			args = append(args, "%"+query+"%")
			idx++
		}

		where := strings.Join(clauses, " AND ")

		countQuery := `SELECT COUNT(1)
FROM finance_transactions t
JOIN finance_accounts a ON a.id = t.account_id
LEFT JOIN finance_categories c ON c.id = t.category_id
LEFT JOIN public.users u ON u.id = t.created_by
LEFT JOIN public.user_profiles up ON up.user_id = u.id
WHERE ` + where
		if err := tx.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
			return err
		}

		offset := (filter.Page - 1) * filter.PageSize
		dataArgs := append([]any{}, args...)
		dataArgs = append(dataArgs, filter.PageSize, offset)
		dataQuery := fmt.Sprintf(`
SELECT t.id,
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
       t.merchant_name,
       t.receipt_number,
       t.payment_method,
       t.subtotal_minor,
       t.tax_minor,
       t.service_charge_minor,
       t.receipt_discount_minor,
       t.notes,
       COALESCE(t.created_by::text, '') AS created_by,
       COALESCE(NULLIF(u.full_name, ''), NULLIF(up.username, ''), NULLIF(u.email, ''), t.created_by::text, '') AS created_by_name,
       COALESCE(t.updated_by::text, '') AS updated_by,
       t.created_at,
       t.updated_at
FROM finance_transactions t
JOIN finance_accounts a ON a.id = t.account_id
LEFT JOIN finance_categories c ON c.id = t.category_id
LEFT JOIN public.users u ON u.id = t.created_by
LEFT JOIN public.user_profiles up ON up.user_id = u.id
WHERE %s
ORDER BY t.transaction_date DESC, t.created_at DESC
LIMIT $%d OFFSET $%d`, where, idx, idx+1)

		rows, err := tx.QueryContext(ctx, dataQuery, dataArgs...)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var trx domain.Transaction
			var trxType string
			if err := rows.Scan(
				&trx.ID, &trx.AccountID, &trx.AccountName, &trx.CategoryID, &trx.CategoryName, &trxType,
				&trx.AmountMinor, &trx.Currency, &trx.InputDate, &trx.TransactionDate, &trx.Description, &trx.MerchantName, &trx.ReceiptNumber, &trx.PaymentMethod, &trx.SubtotalMinor, &trx.TaxMinor, &trx.ServiceChargeMinor, &trx.ReceiptDiscountMinor, &trx.Notes, &trx.CreatedBy, &trx.CreatedByName, &trx.UpdatedBy,
				&trx.CreatedAt, &trx.UpdatedAt,
			); err != nil {
				return err
			}
			trx.TenantID = filter.TenantID
			trx.Type = domain.TransactionType(trxType)
			out = append(out, trx)
		}
		return rows.Err()
	})

	return out, total, err
}

func (r *RepositoryPG) Update(ctx context.Context, trx domain.Transaction) (domain.Transaction, error) {
	err := db.WithTenantTx(ctx, r.conn, func(tx *sql.Tx) error {
		const q = `
UPDATE finance_transactions
SET account_id = $1, category_id = $2, type = $3, amount_minor = $4, currency = $5, input_date = $6, transaction_date = $7, description = $8, merchant_name = $9, receipt_number = $10, payment_method = $11, subtotal_minor = $12, tax_minor = $13, service_charge_minor = $14, receipt_discount_minor = $15, notes = $16, updated_by = $17, updated_at = $18
WHERE id = $19 AND deleted_at IS NULL`

		res, err := tx.ExecContext(ctx, q,
			trx.AccountID, trx.CategoryID, trx.Type, trx.AmountMinor, trx.Currency, trx.InputDate, trx.TransactionDate,
			trx.Description, trx.MerchantName, trx.ReceiptNumber, trx.PaymentMethod, trx.SubtotalMinor, trx.TaxMinor, trx.ServiceChargeMinor, trx.ReceiptDiscountMinor, trx.Notes, trx.UpdatedBy, trx.UpdatedAt, trx.ID,
		)
		if err != nil {
			return err
		}

		affected, err := res.RowsAffected()
		if err != nil {
			return err
		}
		if affected == 0 {
			return domain.ErrTransactionNotFound
		}
		return nil
	})

	if err != nil {
		return domain.Transaction{}, err
	}
	return trx, nil
}

func (r *RepositoryPG) SoftDelete(ctx context.Context, tenantID, transactionID, deletedBy string) error {
	return db.WithTenantTx(ctx, r.conn, func(tx *sql.Tx) error {
		const q = `
UPDATE finance_transactions
SET deleted_at = now(), updated_by = $1, updated_at = now()
WHERE id = $2 AND deleted_at IS NULL`

		res, err := tx.ExecContext(ctx, q, deletedBy, transactionID)
		if err != nil {
			return err
		}
		affected, err := res.RowsAffected()
		if err != nil {
			return err
		}
		if affected == 0 {
			return domain.ErrTransactionNotFound
		}
		return nil
	})
}

func (r *RepositoryPG) ValidateReferences(ctx context.Context, tenantID, accountID string, categoryID *string, trxType domain.TransactionType) error {
	err := db.WithTenantTx(ctx, r.conn, func(tx *sql.Tx) error {
		const accountQuery = `
SELECT 1
FROM finance_accounts
WHERE id = $1 AND is_active = TRUE`

		var exists int
		if err := tx.QueryRowContext(ctx, accountQuery, accountID).Scan(&exists); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return domain.ErrAccountNotFound
			}
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}

	if categoryID == nil || strings.TrimSpace(*categoryID) == "" {
		return nil
	}

	const categoryQuery = `
SELECT category_type
FROM finance_categories
WHERE id = $1 AND is_active = TRUE`

	return db.WithTenantTx(ctx, r.conn, func(tx *sql.Tx) error {
		var categoryType string
		if err := tx.QueryRowContext(ctx, categoryQuery, *categoryID).Scan(&categoryType); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return domain.ErrCategoryNotFound
			}
			return err
		}

		switch trxType {
		case domain.TransactionTypeIncome:
			if categoryType != "income" {
				return domain.ErrCategoryTypeMismatch
			}
		case domain.TransactionTypeExpense:
			if categoryType != "expense" {
				return domain.ErrCategoryTypeMismatch
			}
		case domain.TransactionTypeTransfer:
			// Transfer should not use income/expense category.
			return domain.ErrCategoryTypeMismatch
		case domain.TransactionTypeSavings:
			// Savings transaction should not use category.
			return domain.ErrCategoryTypeMismatch
		}
		return nil
	})
}

func (r *RepositoryPG) ResolveCategoryID(ctx context.Context, tenantID, actorUserID string, categoryID, categoryName *string, trxType domain.TransactionType) (*string, error) {
	if trxType == domain.TransactionTypeTransfer || trxType == domain.TransactionTypeSavings {
		return nil, nil
	}

	if categoryID != nil && strings.TrimSpace(*categoryID) != "" {
		id := strings.TrimSpace(*categoryID)
		return &id, nil
	}

	if categoryName == nil || strings.TrimSpace(*categoryName) == "" {
		return nil, nil
	}

	normalizedName := strings.TrimSpace(*categoryName)
	typeCode := string(trxType)
	var finalID string

	err := db.WithTenantTx(ctx, r.conn, func(tx *sql.Tx) error {
		const findQuery = `
SELECT id
FROM finance_categories
WHERE category_type = $1
  AND LOWER(name) = LOWER($2)
  AND is_active = TRUE
LIMIT 1`

		if err := tx.QueryRowContext(ctx, findQuery, typeCode, normalizedName).Scan(&finalID); err == nil {
			return nil
		} else if !errors.Is(err, sql.ErrNoRows) {
			return err
		}

		newID := uuid.NewString()
		now := time.Now().UTC()
		const insertQuery = `
INSERT INTO finance_categories (
  id, name, category_type, parent_id, is_active, created_by, created_at, updated_at
) VALUES ($1,$2,$3,NULL,TRUE,$4,$5,$5)`

		if _, err := tx.ExecContext(ctx, insertQuery, newID, normalizedName, typeCode, actorUserID, now); err != nil {
			// Handle race
			if err := tx.QueryRowContext(ctx, findQuery, typeCode, normalizedName).Scan(&finalID); err == nil {
				return nil
			}
			return err
		}
		finalID = newID
		return nil
	})

	if err != nil {
		return nil, err
	}
	return &finalID, nil
}

func (r *RepositoryPG) EnsureTransactionExists(ctx context.Context, tenantID, transactionID string) error {
	return db.WithTenantTx(ctx, r.conn, func(tx *sql.Tx) error {
		const q = `
SELECT 1
FROM finance_transactions
WHERE id = $1 AND deleted_at IS NULL`

		var exists int
		if err := tx.QueryRowContext(ctx, q, transactionID).Scan(&exists); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return domain.ErrTransactionNotFound
			}
			return err
		}
		return nil
	})
}

func (r *RepositoryPG) ValidateSavingsGoals(ctx context.Context, tenantID string, savingsIDs []string) error {
	if len(savingsIDs) == 0 {
		return nil
	}

	return db.WithTenantTx(ctx, r.conn, func(tx *sql.Tx) error {
		const q = `
SELECT id
FROM finance_savings
WHERE deleted_at IS NULL
  AND id = ANY($1::uuid[])`

		rows, err := tx.QueryContext(ctx, q, pqStringArray(savingsIDs))
		if err != nil {
			return err
		}
		defer rows.Close()

		found := make(map[string]struct{}, len(savingsIDs))
		for rows.Next() {
			var id string
			if err := rows.Scan(&id); err != nil {
				return err
			}
			found[id] = struct{}{}
		}
		if err := rows.Err(); err != nil {
			return err
		}

		for _, id := range savingsIDs {
			if _, ok := found[id]; !ok {
				return domain.ErrSavingsNotFound
			}
		}
		return nil
	})
}

func (r *RepositoryPG) ReplaceSavingsLinks(ctx context.Context, tenantID, transactionID, actorUserID string, amountMinor int64, savingsIDs []string) error {
	return db.WithTenantTx(ctx, r.conn, func(tx *sql.Tx) error {
		const deleteQuery = `
DELETE FROM finance_transaction_savings_links
WHERE transaction_id = $1`
		if _, err := tx.ExecContext(ctx, deleteQuery, transactionID); err != nil {
			return err
		}

		if len(savingsIDs) > 0 {
			allocations := allocateSavingsAmounts(amountMinor, savingsIDs)
			const insertQuery = `
INSERT INTO finance_transaction_savings_links (
  id, transaction_id, savings_id, allocated_amount_minor, created_by, created_at
) VALUES ($1,$2,$3,$4,$5,$6)`
			now := time.Now().UTC()
			for _, savingsID := range savingsIDs {
				if _, err := tx.ExecContext(
					ctx,
					insertQuery,
					uuid.NewString(),
					transactionID,
					savingsID,
					allocations[savingsID],
					actorUserID,
					now,
				); err != nil {
					return err
				}
			}
		}
		return nil
	})
}

func (r *RepositoryPG) ListSavingsLinks(ctx context.Context, tenantID string, transactionIDs []string) (map[string][]string, map[string][]string, error) {
	idMap := make(map[string][]string)
	nameMap := make(map[string][]string)
	if len(transactionIDs) == 0 {
		return idMap, nameMap, nil
	}

	err := db.WithTenantTx(ctx, r.conn, func(tx *sql.Tx) error {
		const q = `
SELECT l.transaction_id, s.id, s.name
FROM finance_transaction_savings_links l
JOIN finance_savings s ON s.id = l.savings_id AND s.deleted_at IS NULL
WHERE l.transaction_id = ANY($1::uuid[])
ORDER BY l.created_at ASC`

		rows, err := tx.QueryContext(ctx, q, pqStringArray(transactionIDs))
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var (
				transactionID string
				savingsID     string
				savingsName   string
			)
			if err := rows.Scan(&transactionID, &savingsID, &savingsName); err != nil {
				return err
			}
			idMap[transactionID] = append(idMap[transactionID], savingsID)
			nameMap[transactionID] = append(nameMap[transactionID], savingsName)
		}
		return rows.Err()
	})

	return idMap, nameMap, err
}

func (r *RepositoryPG) ListSavingsAllocationsByTransaction(ctx context.Context, tenantID, transactionID string) (map[string]int64, error) {
	out := make(map[string]int64)
	err := db.WithTenantTx(ctx, r.conn, func(tx *sql.Tx) error {
		const q = `
SELECT savings_id, allocated_amount_minor
FROM finance_transaction_savings_links
WHERE transaction_id = $1`

		rows, err := tx.QueryContext(ctx, q, transactionID)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var sid string
			var amt int64
			if err := rows.Scan(&sid, &amt); err != nil {
				return err
			}
			out[sid] = amt
		}
		return rows.Err()
	})
	return out, err
}

func (r *RepositoryPG) AdjustSavingsCurrentAmounts(ctx context.Context, tenantID, actorUserID string, adjustments map[string]int64) error {
	if len(adjustments) == 0 {
		return nil
	}

	return db.WithTenantTx(ctx, r.conn, func(tx *sql.Tx) error {
		const q = `
UPDATE finance_savings
SET current_amount_minor = current_amount_minor + $1,
    updated_at = now()
WHERE id = $2`

		for sid, adj := range adjustments {
			if _, err := tx.ExecContext(ctx, q, adj, sid); err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *RepositoryPG) ReconcileSavingsCurrentAmounts(ctx context.Context, tenantID, actorUserID string, savingsIDs []string) error {
	normalizedIDs := make([]string, 0, len(savingsIDs))
	unique := make(map[string]struct{})
	for _, rawID := range savingsIDs {
		clean := strings.TrimSpace(rawID)
		if clean == "" {
			continue
		}
		if _, exists := unique[clean]; exists {
			continue
		}
		unique[clean] = struct{}{}
		normalizedIDs = append(normalizedIDs, clean)
	}
	if len(normalizedIDs) == 0 {
		return nil
	}

	return db.WithTenantTx(ctx, r.conn, func(tx *sql.Tx) error {
		const q = `
SELECT
  l.transaction_id,
  l.savings_id,
  l.allocated_amount_minor,
  t.amount_minor
FROM finance_transaction_savings_links l
JOIN finance_transactions t ON t.id = l.transaction_id
WHERE l.savings_id = ANY($1::uuid[])
  AND t.type = 'savings'
  AND t.deleted_at IS NULL
ORDER BY l.transaction_id ASC, l.created_at ASC, l.id ASC`

		rows, err := tx.QueryContext(ctx, q, pqStringArray(normalizedIDs))
		if err != nil {
			return err
		}
		defer rows.Close()

		type allocationRow struct {
			transactionID string
			savingsID     string
			allocated     int64
			amountMinor   int64
		}
		byTransaction := make(map[string][]allocationRow)
		for rows.Next() {
			var row allocationRow
			if err := rows.Scan(&row.transactionID, &row.savingsID, &row.allocated, &row.amountMinor); err != nil {
				return err
			}
			byTransaction[row.transactionID] = append(byTransaction[row.transactionID], row)
		}
		if err := rows.Err(); err != nil {
			return err
		}

		totals := make(map[string]int64, len(normalizedIDs))
		for _, rowsByTrx := range byTransaction {
			if len(rowsByTrx) == 0 {
				continue
			}
			allZero := true
			goalIDs := make([]string, 0, len(rowsByTrx))
			for _, row := range rowsByTrx {
				goalIDs = append(goalIDs, row.savingsID)
				if row.allocated != 0 {
					allZero = false
				}
			}

			if allZero && rowsByTrx[0].amountMinor > 0 {
				recovered := allocateSavingsAmounts(rowsByTrx[0].amountMinor, goalIDs)
				for goalID, amount := range recovered {
					totals[goalID] += amount
				}
				continue
			}

			for _, row := range rowsByTrx {
				totals[row.savingsID] += row.allocated
			}
		}

		const updateQuery = `
UPDATE finance_savings
SET current_amount_minor = $1,
    updated_by = $2,
    updated_at = now()
WHERE id = $3
  AND deleted_at IS NULL`

		for _, savingsID := range normalizedIDs {
			amountMinor := totals[savingsID]
			if _, err := tx.ExecContext(ctx, updateQuery, amountMinor, actorUserID, savingsID); err != nil {
				return err
			}
		}
		return nil
	})
}

func allocateSavingsAmounts(amountMinor int64, savingsIDs []string) map[string]int64 {
	out := make(map[string]int64, len(savingsIDs))
	if len(savingsIDs) == 0 || amountMinor <= 0 {
		return out
	}

	portion := amountMinor / int64(len(savingsIDs))
	remainder := amountMinor % int64(len(savingsIDs))
	for idx, savingsID := range savingsIDs {
		allocated := portion
		if int64(idx) < remainder {
			allocated++
		}
		out[savingsID] = allocated
	}
	return out
}

func pqStringArray(items []string) string {
	quoted := make([]string, 0, len(items))
	for _, item := range items {
		clean := strings.TrimSpace(item)
		if clean == "" {
			continue
		}
		quoted = append(quoted, `"`+clean+`"`)
	}
	return "{" + strings.Join(quoted, ",") + "}"
}

func (r *RepositoryPG) ReplaceItems(ctx context.Context, tenantID, transactionID, actorUserID string, items []domain.TransactionItem) error {
	return db.WithTenantTx(ctx, r.conn, func(tx *sql.Tx) error {
		const deleteQuery = `DELETE FROM finance_transaction_items WHERE transaction_id = $1`
		if _, err := tx.ExecContext(ctx, deleteQuery, transactionID); err != nil {
			return err
		}

		if len(items) == 0 {
			return nil
		}

		const insertQuery = `
INSERT INTO finance_transaction_items (
  id, transaction_id, item_name, quantity, price_minor, discount_minor, total_minor, notes, created_by, updated_by, created_at, updated_at
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`

		for _, item := range items {
			id := item.ID
			if id == "" {
				id = uuid.NewString()
			}
			actor := item.CreatedBy
			if actor == "" {
				actor = actorUserID
			}
			createdAt := item.CreatedAt
			if createdAt.IsZero() {
				createdAt = time.Now().UTC()
			}
			updatedAt := item.UpdatedAt
			if updatedAt.IsZero() {
				updatedAt = time.Now().UTC()
			}
			if _, err := tx.ExecContext(ctx, insertQuery,
				id, transactionID, item.ItemName, item.Quantity, item.PriceMinor, item.DiscountMinor, item.TotalMinor, item.Notes, actor, actorUserID, createdAt, updatedAt,
			); err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *RepositoryPG) ListItems(ctx context.Context, tenantID, transactionID string) ([]domain.TransactionItem, error) {
	var out []domain.TransactionItem
	err := db.WithTenantTx(ctx, r.conn, func(tx *sql.Tx) error {
		const q = `
SELECT id, transaction_id, item_name, quantity, price_minor, discount_minor, total_minor, notes, COALESCE(created_by::text, '') AS created_by, COALESCE(updated_by::text, '') AS updated_by, created_at, updated_at
FROM finance_transaction_items
WHERE transaction_id = $1
ORDER BY created_at ASC`

		rows, err := tx.QueryContext(ctx, q, transactionID)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var item domain.TransactionItem
			if err := rows.Scan(
				&item.ID, &item.TransactionID, &item.ItemName, &item.Quantity, &item.PriceMinor, &item.DiscountMinor, &item.TotalMinor, &item.Notes, &item.CreatedBy, &item.UpdatedBy, &item.CreatedAt, &item.UpdatedAt,
			); err != nil {
				return err
			}
			item.TenantID = tenantID
			out = append(out, item)
		}
		return rows.Err()
	})
	return out, err
}

func (r *RepositoryPG) ListItemsByTransactionIDs(ctx context.Context, tenantID string, transactionIDs []string) (map[string][]domain.TransactionItem, error) {
	out := make(map[string][]domain.TransactionItem)
	if len(transactionIDs) == 0 {
		return out, nil
	}

	err := db.WithTenantTx(ctx, r.conn, func(tx *sql.Tx) error {
		const q = `
SELECT id, transaction_id, item_name, quantity, price_minor, discount_minor, total_minor, notes, COALESCE(created_by::text, '') AS created_by, COALESCE(updated_by::text, '') AS updated_by, created_at, updated_at
FROM finance_transaction_items
WHERE transaction_id = ANY($1::uuid[])
ORDER BY created_at ASC`

		rows, err := tx.QueryContext(ctx, q, pqStringArray(transactionIDs))
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var item domain.TransactionItem
			if err := rows.Scan(
				&item.ID, &item.TransactionID, &item.ItemName, &item.Quantity, &item.PriceMinor, &item.DiscountMinor, &item.TotalMinor, &item.Notes, &item.CreatedBy, &item.UpdatedBy, &item.CreatedAt, &item.UpdatedAt,
			); err != nil {
				return err
			}
			item.TenantID = tenantID
			out[item.TransactionID] = append(out[item.TransactionID], item)
		}
		return rows.Err()
	})
	return out, err
}
