package infra

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"pekan/backend/internal/modules/finance/whatsapp/domain"
	"pekan/backend/internal/platform/db"
	"pekan/backend/internal/platform/security"
	"pekan/backend/internal/platform/tenancy"
)

type RepositoryPG struct {
	conn   *sql.DB
	cipher *security.Cipher
}

func NewRepositoryPG(conn *sql.DB, cipher *security.Cipher) *RepositoryPG {
	return &RepositoryPG{conn: conn, cipher: cipher}
}

func (r *RepositoryPG) CreateOTPToken(ctx context.Context, in domain.OTPToken) error {
	return db.WithTenantTx(ctx, r.conn, func(tx *sql.Tx) error {
		const q = `
INSERT INTO public.whatsapp_otp_tokens (token, tenant_id, user_id, expires_at, created_at)
VALUES ($1, $2, $3, $4, $5)`
		_, err := tx.ExecContext(ctx, q, in.Token, in.TenantID, in.UserID, in.ExpiresAt, in.CreatedAt)
		return err
	})
}

func (r *RepositoryPG) GetOTPToken(ctx context.Context, token string) (domain.OTPToken, error) {
	var out domain.OTPToken
	err := db.WithTenantTx(ctx, r.conn, func(tx *sql.Tx) error {
		const q = `
SELECT token, tenant_id, user_id, expires_at, created_at
FROM public.whatsapp_otp_tokens
WHERE token = $1`
		err := tx.QueryRowContext(ctx, q, token).Scan(&out.Token, &out.TenantID, &out.UserID, &out.ExpiresAt, &out.CreatedAt)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return domain.ErrTokenNotFound
			}
			return err
		}
		return nil
	})
	return out, err
}

func (r *RepositoryPG) DeleteOTPToken(ctx context.Context, token string) error {
	return db.WithTenantTx(ctx, r.conn, func(tx *sql.Tx) error {
		const q = `DELETE FROM public.whatsapp_otp_tokens WHERE token = $1`
		_, err := tx.ExecContext(ctx, q, token)
		return err
	})
}

func (r *RepositoryPG) DeleteExpiredTokens(ctx context.Context) error {
	return db.WithTenantTx(ctx, r.conn, func(tx *sql.Tx) error {
		const q = `DELETE FROM public.whatsapp_otp_tokens WHERE expires_at <= $1`
		_, err := tx.ExecContext(ctx, q, time.Now().UTC())
		return err
	})
}

func (r *RepositoryPG) CreateSession(ctx context.Context, in domain.Session) error {
	return db.WithTenantTx(ctx, r.conn, func(tx *sql.Tx) error {
		const q = `
INSERT INTO public.whatsapp_sessions (phone_number, tenant_id, user_id, last_active, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6)`
		_, err := tx.ExecContext(ctx, q, in.PhoneNumber, in.TenantID, in.UserID, in.LastActive, in.CreatedAt, in.UpdatedAt)
		return err
	})
}

func (r *RepositoryPG) GetSessionByPhone(ctx context.Context, phoneNumber string) (domain.Session, error) {
	var out domain.Session
	err := db.WithTenantTx(ctx, r.conn, func(tx *sql.Tx) error {
		const q = `
SELECT phone_number, tenant_id, user_id, last_active, created_at, updated_at
FROM public.whatsapp_sessions
WHERE phone_number = $1`
		err := tx.QueryRowContext(ctx, q, phoneNumber).Scan(&out.PhoneNumber, &out.TenantID, &out.UserID, &out.LastActive, &out.CreatedAt, &out.UpdatedAt)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return domain.ErrSessionNotFound
			}
			return err
		}
		return nil
	})
	return out, err
}

func (r *RepositoryPG) GetSessionByUser(ctx context.Context, tenantID, userID string) (domain.Session, error) {
	var out domain.Session
	err := db.WithTenantTx(ctx, r.conn, func(tx *sql.Tx) error {
		const q = `
SELECT phone_number, tenant_id, user_id, last_active, created_at, updated_at
FROM public.whatsapp_sessions
WHERE tenant_id = $1 AND user_id = $2`
		err := tx.QueryRowContext(ctx, q, tenantID, userID).Scan(&out.PhoneNumber, &out.TenantID, &out.UserID, &out.LastActive, &out.CreatedAt, &out.UpdatedAt)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return domain.ErrSessionNotFound
			}
			return err
		}
		return nil
	})
	return out, err
}

func (r *RepositoryPG) UpdateLastActive(ctx context.Context, phoneNumber string) error {
	return db.WithTenantTx(ctx, r.conn, func(tx *sql.Tx) error {
		const q = `UPDATE public.whatsapp_sessions SET last_active = $1, updated_at = $1 WHERE phone_number = $2`
		_, err := tx.ExecContext(ctx, q, time.Now().UTC(), phoneNumber)
		return err
	})
}

func (r *RepositoryPG) DeleteSessionByUser(ctx context.Context, tenantID, userID string) error {
	return db.WithTenantTx(ctx, r.conn, func(tx *sql.Tx) error {
		const q = `DELETE FROM public.whatsapp_sessions WHERE tenant_id = $1 AND user_id = $2`
		_, err := tx.ExecContext(ctx, q, tenantID, userID)
		return err
	})
}

func (r *RepositoryPG) DeleteSessionByPhone(ctx context.Context, phoneNumber string) error {
	return db.WithTenantTx(ctx, r.conn, func(tx *sql.Tx) error {
		const q = `DELETE FROM public.whatsapp_sessions WHERE phone_number = $1`
		_, err := tx.ExecContext(ctx, q, phoneNumber)
		return err
	})
}

func (r *RepositoryPG) GetTenantCode(ctx context.Context, tenantID string) (string, error) {
	var code string
	const q = `SELECT code FROM public.tenants WHERE id = $1`
	err := r.conn.QueryRowContext(ctx, q, tenantID).Scan(&code)
	if err != nil {
		return "", err
	}
	return code, nil
}

func (r *RepositoryPG) GetUserPhone(ctx context.Context, userID string) (string, error) {
	var phone sql.NullString
	const q = `SELECT phone FROM public.user_profiles WHERE user_id = $1`
	err := r.conn.QueryRowContext(ctx, q, userID).Scan(&phone)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", nil
		}
		return "", err
	}
	if phone.Valid {
		if r.cipher != nil {
			if dec, err := r.cipher.Decrypt(phone.String); err == nil && dec != "" {
				return dec, nil
			}
		}
		return phone.String, nil
	}
	return "", nil
}

func (r *RepositoryPG) CreateChatTransaction(ctx context.Context, tenantID, userID, tenantCode string, amount int64, typeStr, description, categoryName string, transactionDate string, items []domain.ChatItem) (string, error) {
	tc := tenancy.Context{
		TenantID:   tenantID,
		UserID:     userID,
		SchemaName: tenancy.GetSchemaName(tenantCode),
	}
	tenantCtx := tenancy.WithContext(ctx, tc)

	transactionID := uuid.NewString()

	err := db.WithTenantTx(tenantCtx, r.conn, func(tx *sql.Tx) error {
		var accountID string
		const findAccountQ = `SELECT id FROM finance_accounts WHERE is_active = TRUE LIMIT 1`
		err := tx.QueryRowContext(tenantCtx, findAccountQ).Scan(&accountID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				// No active account exists. Let's create a default one!
				accountID = uuid.NewString()
				const insertAccountQ = `
INSERT INTO finance_accounts (id, name, account_type, currency, opening_balance_minor, is_active, created_by, created_at, updated_at)
VALUES ($1, 'Cash / Tunai', 'cash', 'IDR', 0, TRUE, $2, now(), now())`
				_, err = tx.ExecContext(tenantCtx, insertAccountQ, accountID, userID)
				if err != nil {
					return fmt.Errorf("failed to create default account: %w", err)
				}
			} else {
				return fmt.Errorf("failed to find active account: %w", err)
			}
		}

		var categoryID *string
		if categoryName != "" && typeStr != "transfer" && typeStr != "savings" {
			var catID string
			const findCategoryQ = `
SELECT id FROM finance_categories
WHERE category_type = $1 AND LOWER(name) = LOWER($2) AND is_active = TRUE
LIMIT 1`
			err = tx.QueryRowContext(tenantCtx, findCategoryQ, typeStr, categoryName).Scan(&catID)
			if err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					// Create new category
					catID = uuid.NewString()
					const insertCategoryQ = `
INSERT INTO finance_categories (id, name, category_type, parent_id, is_active, created_by, created_at, updated_at)
VALUES ($1, $2, $3, NULL, TRUE, $4, now(), now())`
					_, err = tx.ExecContext(tenantCtx, insertCategoryQ, catID, categoryName, typeStr, userID)
					if err != nil {
						return fmt.Errorf("failed to create category: %w", err)
					}
				} else {
					return fmt.Errorf("failed to find category: %w", err)
				}
			}
			categoryID = &catID
		}

		now := time.Now().UTC()
		const insertTxQ = `
INSERT INTO finance_transactions (
  id, account_id, category_id, type, amount_minor, currency, input_date, transaction_date,
  description, merchant_name, receipt_number, payment_method, subtotal_minor, tax_minor,
  service_charge_minor, receipt_discount_minor, created_by, updated_by, created_at, updated_at
) VALUES ($1, $2, $3, $4, $5, 'IDR', $6, $7, $8, NULL, NULL, NULL, $9, 0, 0, 0, $10, $10, $6, $6)`

		// transaction_date needs to be a Date. We can use the current date in UTC or transactionDate.
		txDate := transactionDate
		if txDate == "" {
			txDate = now.Format("2006-01-02")
		}

		_, err = tx.ExecContext(tenantCtx, insertTxQ,
			transactionID, accountID, categoryID, typeStr, amount, now, txDate,
			description, amount, userID,
		)
		if err != nil {
			return fmt.Errorf("failed to insert transaction: %w", err)
		}

		if len(items) > 0 {
			const insertItemQ = `
INSERT INTO finance_transaction_items (
  id, transaction_id, item_name, quantity, price_minor, discount_minor, total_minor, notes, created_by, updated_by, created_at, updated_at
) VALUES ($1, $2, $3, $4, $5, 0, $6, NULL, $7, $7, now(), now())`
			for _, item := range items {
				itemID := uuid.NewString()
				totalMinor := int64(item.Qty * float64(item.Price))
				_, err = tx.ExecContext(tenantCtx, insertItemQ,
					itemID, transactionID, item.Name, item.Qty, item.Price, totalMinor, userID,
				)
				if err != nil {
					return fmt.Errorf("failed to insert transaction item: %w", err)
				}
			}
		}

		return nil
	})
	if err != nil {
		return "", err
	}
	return transactionID, nil
}

func (r *RepositoryPG) FindChatTransaction(ctx context.Context, tenantID, userID, tenantCode, txID string) (*domain.RecentTxItem, error) {
	tc := tenancy.Context{
		TenantID:   tenantID,
		UserID:     userID,
		SchemaName: tenancy.GetSchemaName(tenantCode),
	}
	tenantCtx := tenancy.WithContext(ctx, tc)

	var item domain.RecentTxItem
	var txDate time.Time

	err := db.WithTenantTx(tenantCtx, r.conn, func(tx *sql.Tx) error {
		cleanID := strings.TrimPrefix(strings.ToLower(txID), "tx-")
		cleanID = strings.TrimPrefix(cleanID, "trx-")

		const q = `
			SELECT t.id, t.amount_minor, t.type, t.description, COALESCE(c.name, 'Tanpa Kategori') as category_name, t.transaction_date
			FROM finance_transactions t
			LEFT JOIN finance_categories c ON t.category_id = c.id
			WHERE t.deleted_at IS NULL AND (t.id::text = $1 OR LOWER(t.id::text) LIKE $2)
			LIMIT 1`
		
		return tx.QueryRowContext(tenantCtx, q, txID, cleanID+"%").Scan(&item.ID, &item.Amount, &item.Type, &item.Description, &item.CategoryName, &txDate)
	})
	if err != nil {
		return nil, err
	}
	item.TxDate = txDate.Format("2006-01-02")
	return &item, nil
}

func (r *RepositoryPG) DeleteChatTransaction(ctx context.Context, tenantID, userID, tenantCode, txID string) error {
	tc := tenancy.Context{
		TenantID:   tenantID,
		UserID:     userID,
		SchemaName: tenancy.GetSchemaName(tenantCode),
	}
	tenantCtx := tenancy.WithContext(ctx, tc)

	return db.WithTenantTx(tenantCtx, r.conn, func(tx *sql.Tx) error {
		cleanID := strings.TrimPrefix(strings.ToLower(txID), "tx-")
		cleanID = strings.TrimPrefix(cleanID, "trx-")

		var exactID string
		const findQ = `SELECT id FROM finance_transactions WHERE deleted_at IS NULL AND (id::text = $1 OR LOWER(id::text) LIKE $2) LIMIT 1`
		err := tx.QueryRowContext(tenantCtx, findQ, txID, cleanID+"%").Scan(&exactID)
		if err != nil {
			return err
		}

		const deleteItemsQ = `DELETE FROM finance_transaction_items WHERE transaction_id = $1`
		_, _ = tx.ExecContext(tenantCtx, deleteItemsQ, exactID)

		const deleteTxQ = `
			UPDATE finance_transactions 
			SET deleted_at = now(), updated_by = $1, updated_at = now() 
			WHERE id = $2 AND deleted_at IS NULL`
		_, err = tx.ExecContext(tenantCtx, deleteTxQ, userID, exactID)
		return err
	})
}

func (r *RepositoryPG) UpdateChatTransaction(ctx context.Context, tenantID, userID, tenantCode, txID string, amount int64, typeStr, description, categoryName string, transactionDate string) error {
	tc := tenancy.Context{
		TenantID:   tenantID,
		UserID:     userID,
		SchemaName: tenancy.GetSchemaName(tenantCode),
	}
	tenantCtx := tenancy.WithContext(ctx, tc)

	return db.WithTenantTx(tenantCtx, r.conn, func(tx *sql.Tx) error {
		cleanID := strings.TrimPrefix(strings.ToLower(txID), "tx-")
		cleanID = strings.TrimPrefix(cleanID, "trx-")

		var exactID string
		const findQ = `SELECT id FROM finance_transactions WHERE deleted_at IS NULL AND (id::text = $1 OR LOWER(id::text) LIKE $2) LIMIT 1`
		err := tx.QueryRowContext(tenantCtx, findQ, txID, cleanID+"%").Scan(&exactID)
		if err != nil {
			return err
		}

		var categoryID *string
		if categoryName != "" && typeStr != "transfer" && typeStr != "savings" {
			var catID string
			const findCategoryQ = `
				SELECT id FROM finance_categories
				WHERE category_type = $1 AND LOWER(name) = LOWER($2) AND is_active = TRUE
				LIMIT 1`
			err = tx.QueryRowContext(tenantCtx, findCategoryQ, typeStr, categoryName).Scan(&catID)
			if err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					catID = uuid.NewString()
					const insertCategoryQ = `
						INSERT INTO finance_categories (id, name, category_type, parent_id, is_active, created_by, created_at, updated_at)
						VALUES ($1, $2, $3, NULL, TRUE, $4, now(), now())`
					_, err = tx.ExecContext(tenantCtx, insertCategoryQ, catID, categoryName, typeStr, userID)
					if err != nil {
						return fmt.Errorf("failed to create category: %w", err)
					}
				} else {
					return fmt.Errorf("failed to find category: %w", err)
				}
			}
			categoryID = &catID
		}

		const updateQ = `
			UPDATE finance_transactions 
			SET amount_minor = $1, subtotal_minor = $1, type = $2, description = $3, category_id = $4, transaction_date = $5, updated_by = $6, updated_at = now() 
			WHERE id = $7 AND deleted_at IS NULL`
		_, err = tx.ExecContext(tenantCtx, updateQ, amount, typeStr, description, categoryID, transactionDate, userID, exactID)
		return err
	})
}

func (r *RepositoryPG) GetFinancialContext(ctx context.Context, tenantID, userID, tenantCode string) (*domain.FinancialContext, error) {
	tc := tenancy.Context{
		TenantID:   tenantID,
		UserID:     userID,
		SchemaName: tenancy.GetSchemaName(tenantCode),
	}
	tenantCtx := tenancy.WithContext(ctx, tc)

	var finContext domain.FinancialContext

	err := db.WithTenantTx(tenantCtx, r.conn, func(tx *sql.Tx) error {
		// 1. Get total income and expenses for current month using transaction_date and filtering deleted_at
		const summaryQ = `
			SELECT 
				COALESCE(SUM(CASE WHEN type = 'income' THEN amount_minor ELSE 0 END), 0) as total_income,
				COALESCE(SUM(CASE WHEN type = 'expense' THEN amount_minor ELSE 0 END), 0) as total_expense
			FROM finance_transactions
			WHERE EXTRACT(MONTH FROM transaction_date) = EXTRACT(MONTH FROM CURRENT_DATE)
			  AND EXTRACT(YEAR FROM transaction_date) = EXTRACT(YEAR FROM CURRENT_DATE)
			  AND deleted_at IS NULL`
		
		err := tx.QueryRowContext(tenantCtx, summaryQ).Scan(&finContext.TotalIncome, &finContext.TotalExpense)
		if err != nil {
			return fmt.Errorf("failed to get financial summary: %w", err)
		}

		// 2. Get recent transactions (limit 100) excluding deleted ones
		const recentQ = `
			SELECT t.id, t.amount_minor, t.type, t.description, COALESCE(c.name, 'Tanpa Kategori') as category_name, t.transaction_date
			FROM finance_transactions t
			LEFT JOIN finance_categories c ON t.category_id = c.id
			WHERE t.deleted_at IS NULL
			ORDER BY t.transaction_date DESC, t.created_at DESC
			LIMIT 100`
		
		rows, err := tx.QueryContext(tenantCtx, recentQ)
		if err != nil {
			return fmt.Errorf("failed to get recent transactions: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			var item domain.RecentTxItem
			var txDate time.Time
			err := rows.Scan(&item.ID, &item.Amount, &item.Type, &item.Description, &item.CategoryName, &txDate)
			if err != nil {
				return fmt.Errorf("failed to scan recent transaction: %w", err)
			}
			item.TxDate = txDate.Format("2006-01-02")
			finContext.RecentTx = append(finContext.RecentTx, item)
		}

		// 3. Get active budgets with spent_amount_minor calculated on the fly
		const budgetQ = `
			SELECT b.name, COALESCE(c.name, 'Semua Kategori') as category_name, b.amount_limit_minor, COALESCE(spent.spent_amount_minor, 0) as spent_amount_minor
			FROM finance_budgets b
			LEFT JOIN finance_categories c ON b.category_id = c.id
			LEFT JOIN LATERAL (
				SELECT COALESCE(SUM(t.amount_minor), 0) AS spent_amount_minor
				FROM finance_transactions t
				WHERE t.deleted_at IS NULL
				  AND t.type = 'expense'
				  AND t.transaction_date >= b.start_date
				  AND (b.end_date IS NULL OR t.transaction_date <= b.end_date)
				  AND (
					  b.category_id IS NULL 
					  OR t.category_id = b.category_id
					  OR t.category_id IN (
						  WITH RECURSIVE cat_tree AS (
							  SELECT id FROM finance_categories WHERE parent_id = b.category_id
							  UNION ALL
							  SELECT cc.id FROM finance_categories cc JOIN cat_tree ct ON cc.parent_id = ct.id
						  )
						  SELECT id FROM cat_tree
					  )
				  )
			) spent ON TRUE
			WHERE b.deleted_at IS NULL AND (b.status = 'active' OR b.status = 'approved')`
		
		brows, err := tx.QueryContext(tenantCtx, budgetQ)
		if err != nil {
			return fmt.Errorf("failed to query budgets: %w", err)
		}
		defer brows.Close()
		for brows.Next() {
			var bitem domain.BudgetSummaryItem
			err := brows.Scan(&bitem.Name, &bitem.CategoryName, &bitem.AmountLimitMinor, &bitem.SpentAmountMinor)
			if err != nil {
				return fmt.Errorf("failed to scan budget item: %w", err)
			}
			finContext.ActiveBudgets = append(finContext.ActiveBudgets, bitem)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return &finContext, nil
}

func (r *RepositoryPG) EnqueueMessage(ctx context.Context, phoneNumber, message string, tenantID, userID *string) (string, error) {
	var queueID string
	err := db.WithTenantTx(ctx, r.conn, func(tx *sql.Tx) error {
		const q = `
INSERT INTO public.whatsapp_bot_queue (phone_number, message, status, tenant_id, user_id, received_at)
VALUES ($1, $2, 'pending', $3, $4, NOW())
RETURNING id`
		var tID, uID sql.NullString
		if tenantID != nil && *tenantID != "" {
			tID = sql.NullString{String: *tenantID, Valid: true}
		}
		if userID != nil && *userID != "" {
			uID = sql.NullString{String: *userID, Valid: true}
		}
		return tx.QueryRowContext(ctx, q, phoneNumber, message, tID, uID).Scan(&queueID)
	})
	return queueID, err
}

func (r *RepositoryPG) GetPendingQueueItems(ctx context.Context, limit int) ([]domain.QueueItem, error) {
	var items []domain.QueueItem
	err := db.WithTenantTx(ctx, r.conn, func(tx *sql.Tx) error {
		const q = `
SELECT id, phone_number, message, status, tenant_id, user_id, received_at
FROM public.whatsapp_bot_queue
WHERE status = 'pending'
ORDER BY received_at ASC
LIMIT $1
FOR UPDATE SKIP LOCKED`
		rows, err := tx.QueryContext(ctx, q, limit)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var item domain.QueueItem
			var tID, uID sql.NullString
			err := rows.Scan(&item.ID, &item.PhoneNumber, &item.Message, &item.Status, &tID, &uID, &item.ReceivedAt)
			if err != nil {
				return err
			}
			if tID.Valid {
				s := tID.String
				item.TenantID = &s
			}
			if uID.Valid {
				s := uID.String
				item.UserID = &s
			}
			items = append(items, item)
		}
		return nil
	})
	return items, err
}

func (r *RepositoryPG) UpdateQueueItemStatus(ctx context.Context, id string, status string, replyMessage *string, errorMessage *string, latencyMs *int) error {
	return db.WithTenantTx(ctx, r.conn, func(tx *sql.Tx) error {
		var q string
		var err error

		var replyVal, errVal sql.NullString
		var latVal sql.NullInt32

		if replyMessage != nil {
			replyVal = sql.NullString{String: *replyMessage, Valid: true}
		}
		if errorMessage != nil {
			errVal = sql.NullString{String: *errorMessage, Valid: true}
		}
		if latencyMs != nil {
			latVal = sql.NullInt32{Int32: int32(*latencyMs), Valid: true}
		}

		if status == "processing" {
			q = `
UPDATE public.whatsapp_bot_queue
SET status = $1
WHERE id = $2`
			_, err = tx.ExecContext(ctx, q, status, id)
		} else {
			q = `
UPDATE public.whatsapp_bot_queue
SET status = $1, reply_message = $2, error_message = $3, processing_time_ms = $4, processed_at = NOW()
WHERE id = $5`
			_, err = tx.ExecContext(ctx, q, status, replyVal, errVal, latVal, id)
		}
		return err
	})
}


