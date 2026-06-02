package infra

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"pekan/backend/internal/modules/finance/reminders/domain"
	"pekan/backend/internal/platform/db"
)

type RepositoryPG struct {
	conn *sql.DB
}

func NewRepositoryPG(conn *sql.DB) *RepositoryPG {
	return &RepositoryPG{conn: conn}
}

func (r *RepositoryPG) Create(ctx context.Context, in domain.Reminder) (domain.Reminder, error) {
	var out domain.Reminder
	err := db.WithTenantTx(ctx, r.conn, func(tx *sql.Tx) error {
		in.ID = uuid.NewString()
		const q = `
INSERT INTO finance_reminders (
  id, title, description, amount_minor, currency, due_date, repeat_interval, status,
  total_tenor, current_tenor,
  last_triggered_at, created_by, updated_by, created_at, updated_at
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)`

		_, err := tx.ExecContext(ctx, q,
			in.ID, in.Title, in.Description, in.AmountMinor, in.Currency, in.DueDate, in.RepeatInterval,
			in.Status, in.TotalTenor, in.CurrentTenor, in.LastTriggeredAt, in.CreatedBy, in.UpdatedBy, in.CreatedAt, in.UpdatedAt,
		)
		if err != nil {
			return err
		}
		out = in
		return nil
	})
	return out, err
}

func (r *RepositoryPG) GetByID(ctx context.Context, tenantID, reminderID string) (domain.Reminder, error) {
	var out domain.Reminder
	err := db.WithTenantTx(ctx, r.conn, func(tx *sql.Tx) error {
		const q = `
SELECT id, title, description, amount_minor, currency, due_date, repeat_interval, status,
       total_tenor, current_tenor,
       last_triggered_at, created_by, updated_by, created_at, updated_at, deleted_at
FROM finance_reminders
WHERE id = $1 AND deleted_at IS NULL`

		err := tx.QueryRowContext(ctx, q, reminderID).Scan(
			&out.ID, &out.Title, &out.Description, &out.AmountMinor, &out.Currency, &out.DueDate,
			&out.RepeatInterval, &out.Status, &out.TotalTenor, &out.CurrentTenor, &out.LastTriggeredAt, &out.CreatedBy, &out.UpdatedBy, &out.CreatedAt, &out.UpdatedAt, &out.DeletedAt,
		)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return domain.ErrReminderNotFound
			}
			return err
		}
		out.TenantID = tenantID
		return nil
	})
	return out, err
}

func (r *RepositoryPG) List(ctx context.Context, filter domain.ListFilter) ([]domain.Reminder, int64, error) {
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.PageSize <= 0 || filter.PageSize > 100 {
		filter.PageSize = 20
	}

	var items []domain.Reminder
	var total int64

	err := db.WithTenantTx(ctx, r.conn, func(tx *sql.Tx) error {
		var (
			clauses = []string{"deleted_at IS NULL"}
			args    []any
			idx     = 1
		)

		if filter.Status != nil && strings.TrimSpace(*filter.Status) != "" {
			clauses = append(clauses, fmt.Sprintf("status = $%d", idx))
			args = append(args, strings.ToLower(strings.TrimSpace(*filter.Status)))
			idx++
		}
		if filter.DateFrom != nil && *filter.DateFrom != "" {
			clauses = append(clauses, fmt.Sprintf("due_date >= $%d", idx))
			args = append(args, *filter.DateFrom)
			idx++
		}
		if filter.DateTo != nil && *filter.DateTo != "" {
			clauses = append(clauses, fmt.Sprintf("due_date <= $%d", idx))
			args = append(args, *filter.DateTo)
			idx++
		}

		where := strings.Join(clauses, " AND ")
		countQuery := "SELECT COUNT(1) FROM finance_reminders WHERE " + where

		if err := tx.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
			return err
		}

		offset := (filter.Page - 1) * filter.PageSize
		dataArgs := append(args, filter.PageSize, offset)
		limitIdx := idx
		offsetIdx := idx + 1

		dataQuery := fmt.Sprintf(`
SELECT id, title, description, amount_minor, currency, due_date, repeat_interval, status,
       total_tenor, current_tenor,
       last_triggered_at, created_by, updated_by, created_at, updated_at, deleted_at
FROM finance_reminders
WHERE %s
ORDER BY due_date ASC, created_at DESC
LIMIT $%d OFFSET $%d`, where, limitIdx, offsetIdx)

		rows, err := tx.QueryContext(ctx, dataQuery, dataArgs...)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var item domain.Reminder
			if err := rows.Scan(
				&item.ID, &item.Title, &item.Description, &item.AmountMinor, &item.Currency, &item.DueDate,
				&item.RepeatInterval, &item.Status, &item.TotalTenor, &item.CurrentTenor, &item.LastTriggeredAt, &item.CreatedBy, &item.UpdatedBy, &item.CreatedAt, &item.UpdatedAt, &item.DeletedAt,
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

func (r *RepositoryPG) ListDue(ctx context.Context, tenantID string) ([]domain.Reminder, error) {
	var items []domain.Reminder
	err := db.WithTenantTx(ctx, r.conn, func(tx *sql.Tx) error {
		const q = `
SELECT id, title, description, amount_minor, currency, due_date, repeat_interval, status,
       total_tenor, current_tenor,
       last_triggered_at, created_by, updated_by, created_at, updated_at, deleted_at
FROM finance_reminders
WHERE status = 'pending' AND deleted_at IS NULL AND (
    CASE 
      WHEN total_tenor IS NOT NULL AND total_tenor > 1 THEN
        CASE 
          WHEN repeat_interval = 'daily' THEN due_date + (COALESCE(current_tenor, 0) * INTERVAL '1 day')
          WHEN repeat_interval = 'weekly' THEN due_date + (COALESCE(current_tenor, 0) * INTERVAL '1 week')
          WHEN repeat_interval = 'monthly' THEN due_date + (COALESCE(current_tenor, 0) * INTERVAL '1 month')
          ELSE due_date
        END
      ELSE due_date
    END
  ) <= CURRENT_DATE
ORDER BY due_date ASC`

		rows, err := tx.QueryContext(ctx, q)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var item domain.Reminder
			if err := rows.Scan(
				&item.ID, &item.Title, &item.Description, &item.AmountMinor, &item.Currency, &item.DueDate,
				&item.RepeatInterval, &item.Status, &item.TotalTenor, &item.CurrentTenor, &item.LastTriggeredAt, &item.CreatedBy, &item.UpdatedBy, &item.CreatedAt, &item.UpdatedAt, &item.DeletedAt,
			); err != nil {
				return err
			}
			item.TenantID = tenantID
			items = append(items, item)
		}
		return rows.Err()
	})
	return items, err
}

func (r *RepositoryPG) ListDueForProcessing(ctx context.Context, limit int) ([]domain.Reminder, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	var items []domain.Reminder
	err := db.WithTenantTx(ctx, r.conn, func(tx *sql.Tx) error {
		const q = `
SELECT id, tenant_id, title, description, amount_minor, currency, due_date, repeat_interval, status,
       total_tenor, current_tenor,
       last_triggered_at, created_by, updated_by, created_at, updated_at, deleted_at
FROM finance_reminders
WHERE status = 'pending'
  AND deleted_at IS NULL
  AND (
    CASE 
      WHEN total_tenor IS NOT NULL AND total_tenor > 1 THEN
        CASE 
          WHEN repeat_interval = 'daily' THEN due_date + (COALESCE(current_tenor, 0) * INTERVAL '1 day')
          WHEN repeat_interval = 'weekly' THEN due_date + (COALESCE(current_tenor, 0) * INTERVAL '1 week')
          WHEN repeat_interval = 'monthly' THEN due_date + (COALESCE(current_tenor, 0) * INTERVAL '1 month')
          ELSE due_date
        END
      ELSE due_date
    END
  ) <= CURRENT_DATE
  AND (
    last_triggered_at IS NULL 
    OR last_triggered_at::date < (
      CASE 
        WHEN total_tenor IS NOT NULL AND total_tenor > 1 THEN
          CASE 
            WHEN repeat_interval = 'daily' THEN due_date + (COALESCE(current_tenor, 0) * INTERVAL '1 day')
            WHEN repeat_interval = 'weekly' THEN due_date + (COALESCE(current_tenor, 0) * INTERVAL '1 week')
            WHEN repeat_interval = 'monthly' THEN due_date + (COALESCE(current_tenor, 0) * INTERVAL '1 month')
            ELSE due_date
          END
        ELSE due_date
      END
    )::date
  )
ORDER BY due_date ASC
LIMIT $1`

		rows, err := tx.QueryContext(ctx, q, limit)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var item domain.Reminder
			if err := rows.Scan(
				&item.ID, &item.TenantID, &item.Title, &item.Description, &item.AmountMinor, &item.Currency, &item.DueDate,
				&item.RepeatInterval, &item.Status, &item.TotalTenor, &item.CurrentTenor, &item.LastTriggeredAt, &item.CreatedBy, &item.UpdatedBy, &item.CreatedAt, &item.UpdatedAt, &item.DeletedAt,
			); err != nil {
				return err
			}
			items = append(items, item)
		}
		return rows.Err()
	})
	return items, err
}

func (r *RepositoryPG) MarkTriggered(ctx context.Context, tenantID, reminderID string, nextDueDate *time.Time, actorUserID string) error {
	return db.WithTenantTx(ctx, r.conn, func(tx *sql.Tx) error {
		const q = `
UPDATE finance_reminders
SET last_triggered_at = $1,
    due_date = COALESCE($2, due_date),
    updated_by = $3,
    updated_at = $1
WHERE id = $4 AND deleted_at IS NULL`

		now := time.Now().UTC()
		res, err := tx.ExecContext(ctx, q, now, nextDueDate, actorUserID, reminderID)
		if err != nil {
			return err
		}
		affected, err := res.RowsAffected()
		if err != nil {
			return err
		}
		if affected == 0 {
			return domain.ErrReminderNotFound
		}
		return nil
	})
}

func (r *RepositoryPG) Update(ctx context.Context, in domain.Reminder) (domain.Reminder, error) {
	err := db.WithTenantTx(ctx, r.conn, func(tx *sql.Tx) error {
		const q = `
UPDATE finance_reminders
SET title = $1, description = $2, amount_minor = $3, currency = $4, due_date = $5, repeat_interval = $6,
    status = $7, total_tenor = $8, current_tenor = $9, updated_by = $10, updated_at = $11
WHERE id = $12 AND deleted_at IS NULL`

		res, err := tx.ExecContext(ctx, q,
			in.Title, in.Description, in.AmountMinor, in.Currency, in.DueDate, in.RepeatInterval, in.Status,
			in.TotalTenor, in.CurrentTenor, in.UpdatedBy, in.UpdatedAt, in.ID,
		)
		if err != nil {
			return err
		}
		affected, err := res.RowsAffected()
		if err != nil {
			return err
		}
		if affected == 0 {
			return domain.ErrReminderNotFound
		}
		return nil
	})
	if err != nil {
		return domain.Reminder{}, err
	}
	return in, nil
}

func (r *RepositoryPG) UpdateStatus(ctx context.Context, tenantID, reminderID, status, actorUserID string) (domain.Reminder, error) {
	var out domain.Reminder
	err := db.WithTenantTx(ctx, r.conn, func(tx *sql.Tx) error {
		const q = `
UPDATE finance_reminders
SET status = $1, updated_by = $2, updated_at = $3
WHERE id = $4 AND deleted_at IS NULL
RETURNING id, title, description, amount_minor, currency, due_date, repeat_interval, status,
          total_tenor, current_tenor,
          last_triggered_at, created_by, updated_by, created_at, updated_at, deleted_at`

		now := time.Now().UTC()
		err := tx.QueryRowContext(ctx, q, status, actorUserID, now, reminderID).Scan(
			&out.ID, &out.Title, &out.Description, &out.AmountMinor, &out.Currency, &out.DueDate,
			&out.RepeatInterval, &out.Status, &out.TotalTenor, &out.CurrentTenor, &out.LastTriggeredAt, &out.CreatedBy, &out.UpdatedBy, &out.CreatedAt, &out.UpdatedAt, &out.DeletedAt,
		)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return domain.ErrReminderNotFound
			}
			return err
		}
		out.TenantID = tenantID
		return nil
	})
	return out, err
}

func (r *RepositoryPG) SoftDelete(ctx context.Context, tenantID, reminderID, actorUserID string) error {
	return db.WithTenantTx(ctx, r.conn, func(tx *sql.Tx) error {
		const q = `
UPDATE finance_reminders
SET deleted_at = $1, updated_by = $2, updated_at = $1
WHERE id = $3 AND deleted_at IS NULL`

		now := time.Now().UTC()
		res, err := tx.ExecContext(ctx, q, now, actorUserID, reminderID)
		if err != nil {
			return err
		}
		affected, err := res.RowsAffected()
		if err != nil {
			return err
		}
		if affected == 0 {
			return domain.ErrReminderNotFound
		}
		return nil
	})
}

func (r *RepositoryPG) CreatePayment(ctx context.Context, in domain.ReminderPayment) (domain.ReminderPayment, error) {
	var out domain.ReminderPayment
	err := db.WithTenantTx(ctx, r.conn, func(tx *sql.Tx) error {
		in.ID = uuid.NewString()

		const q = `
INSERT INTO finance_reminder_payments (
    id, reminder_id, paid_at, amount_minor, status, notes, proof_image_url,
    created_by, updated_by, created_at, updated_at
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`

		_, err := tx.ExecContext(ctx, q,
			in.ID, in.ReminderID, in.PaidAt, in.AmountMinor, in.Status, in.Notes, in.ProofImageURL,
			in.CreatedBy, in.UpdatedBy, in.CreatedAt, in.UpdatedAt,
		)
		if err != nil {
			return err
		}

		if in.ProofImageURL != nil && *in.ProofImageURL != "" && in.TransientProofName != "" {
			fileID := uuid.NewString()
			attachmentID := uuid.NewString()
			
			const insertFileQuery = `
INSERT INTO public.files (
  id, tenant_id, module_code, owner_type, owner_id, provider, object_key,
  original_filename, stored_filename, mime_type, scan_status, size_bytes, uploaded_by, created_at, deleted_at
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,NULL)`
			
			_, err = tx.ExecContext(ctx, insertFileQuery,
				fileID, in.TenantID, "finance.reminders", "reminders", in.ReminderID, "local", *in.ProofImageURL,
				in.TransientProofName, in.TransientProofName, in.TransientProofMime, "clean", 0, in.CreatedBy, in.CreatedAt,
			)
			if err != nil {
				return err
			}

			const insertAttachmentQuery = `
INSERT INTO finance_entity_attachments (
  id, owner_type, owner_id, file_id, created_by, created_at
) VALUES ($1,$2,$3,$4,$5,$6)`

			_, err = tx.ExecContext(ctx, insertAttachmentQuery,
				attachmentID, "reminders", in.ReminderID, fileID, in.CreatedBy, in.CreatedAt,
			)
			if err != nil {
				return err
			}
		}

		// Fetch reminder details
		var (
			reminderTitle string
			totalTenor    *int
			currentTenor  *int
			currency      string
		)
		const reminderQuery = `
			SELECT title, total_tenor, current_tenor, currency 
			FROM finance_reminders 
			WHERE id = $1 AND deleted_at IS NULL`
		err = tx.QueryRowContext(ctx, reminderQuery, in.ReminderID).Scan(&reminderTitle, &totalTenor, &currentTenor, &currency)
		if err != nil {
			return fmt.Errorf("failed to find reminder: %w", err)
		}

		// Find or create active account
		var accountID string
		const findAccountQ = `SELECT id FROM finance_accounts WHERE is_active = TRUE LIMIT 1`
		err = tx.QueryRowContext(ctx, findAccountQ).Scan(&accountID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				// No active account exists. Let's create a default one!
				accountID = uuid.NewString()
				const insertAccountQ = `
INSERT INTO finance_accounts (id, name, account_type, currency, opening_balance_minor, is_active, created_by, created_at, updated_at)
VALUES ($1, 'Cash / Tunai', 'cash', 'IDR', 0, TRUE, $2, now(), now())`
				_, err = tx.ExecContext(ctx, insertAccountQ, accountID, in.CreatedBy)
				if err != nil {
					return fmt.Errorf("failed to create default account: %w", err)
				}
			} else {
				return fmt.Errorf("failed to find active account: %w", err)
			}
		}

		// Find or create 'Cicilan' category
		var categoryID string
		const findCategoryQ = `
			SELECT id FROM finance_categories
			WHERE category_type = 'expense' AND LOWER(name) = 'cicilan' AND is_active = TRUE
			LIMIT 1`
		err = tx.QueryRowContext(ctx, findCategoryQ).Scan(&categoryID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				// Create new category
				categoryID = uuid.NewString()
				const insertCategoryQ = `
					INSERT INTO finance_categories (id, name, category_type, parent_id, is_active, created_by, created_at, updated_at)
					VALUES ($1, 'Cicilan', 'expense', NULL, TRUE, $2, now(), now())`
				_, err = tx.ExecContext(ctx, insertCategoryQ, categoryID, in.CreatedBy)
				if err != nil {
					return fmt.Errorf("failed to create category: %w", err)
				}
			} else {
				return fmt.Errorf("failed to find category: %w", err)
			}
		}

		// Determine transaction description
		// e.g. "pembayaran tenor 4 - cicilan kulkas" or "pembayaran - cicilan kulkas"
		description := "pembayaran - " + reminderTitle
		if totalTenor != nil && *totalTenor > 1 {
			var paidCount int
			const countPaymentsQ = `
				SELECT COUNT(1) 
				FROM finance_reminder_payments 
				WHERE reminder_id = $1`
			err = tx.QueryRowContext(ctx, countPaymentsQ, in.ReminderID).Scan(&paidCount)
			if err != nil {
				return fmt.Errorf("failed to count reminder payments: %w", err)
			}
			description = fmt.Sprintf("pembayaran tenor %d - %s", paidCount, reminderTitle)
		} else if currentTenor != nil && totalTenor != nil {
			// Fallback for tenor info if exists
			description = fmt.Sprintf("pembayaran tenor %d - %s", *currentTenor, reminderTitle)
		}

		// Clean and validate currency
		cleanCurrency := strings.ToUpper(strings.TrimSpace(currency))
		if len(cleanCurrency) != 3 {
			cleanCurrency = "IDR"
		}

		// Insert auto-generated transaction
		transactionID := uuid.NewString()
		const insertTxQ = `
			INSERT INTO finance_transactions (
				id, account_id, category_id, type, amount_minor, currency, input_date, transaction_date,
				description, merchant_name, receipt_number, payment_method, subtotal_minor, tax_minor,
				service_charge_minor, receipt_discount_minor, notes, created_by, updated_by, created_at, updated_at
			) VALUES ($1, $2, $3, 'expense', $4, $5, $6, $7, $8, NULL, NULL, NULL, $9, 0, 0, 0, $10, $11, $11, $6, $6)`

		now := time.Now().UTC()
		_, err = tx.ExecContext(ctx, insertTxQ,
			transactionID, accountID, categoryID, in.AmountMinor, cleanCurrency, now, in.PaidAt,
			description, in.AmountMinor, in.Notes, in.CreatedBy,
		)
		if err != nil {
			return fmt.Errorf("failed to auto-log transaction: %w", err)
		}

		out = in
		return nil
	})
	return out, err
}

func (r *RepositoryPG) UpdatePayment(ctx context.Context, in domain.ReminderPayment) (domain.ReminderPayment, error) {
	err := db.WithTenantTx(ctx, r.conn, func(tx *sql.Tx) error {
		const q = `
UPDATE finance_reminder_payments
SET paid_at = $1, amount_minor = $2, status = $3, notes = $4, proof_image_url = COALESCE($5, proof_image_url),
    updated_by = $6, updated_at = $7
WHERE reminder_id = $8 AND id = $9`

		now := time.Now().UTC()
		in.UpdatedAt = now
		res, err := tx.ExecContext(ctx, q,
			in.PaidAt, in.AmountMinor, in.Status, in.Notes, in.ProofImageURL,
			in.UpdatedBy, in.UpdatedAt, in.ReminderID, in.ID,
		)
		if err != nil {
			return err
		}
		affected, err := res.RowsAffected()
		if err != nil {
			return err
		}
		if affected == 0 {
			return errors.New("payment not found")
		}

		if in.ProofImageURL != nil && *in.ProofImageURL != "" && in.TransientProofName != "" {
			fileID := uuid.NewString()
			attachmentID := uuid.NewString()
			
			const insertFileQuery = `
INSERT INTO public.files (
  id, tenant_id, module_code, owner_type, owner_id, provider, object_key,
  original_filename, stored_filename, mime_type, scan_status, size_bytes, uploaded_by, created_at, deleted_at
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,NULL)`
			
			_, err = tx.ExecContext(ctx, insertFileQuery,
				fileID, in.TenantID, "finance.reminders", "reminders", in.ReminderID, "local", *in.ProofImageURL,
				in.TransientProofName, in.TransientProofName, in.TransientProofMime, "clean", 0, in.UpdatedBy, in.UpdatedAt,
			)
			if err != nil {
				return err
			}

			const insertAttachmentQuery = `
INSERT INTO finance_entity_attachments (
  id, owner_type, owner_id, file_id, created_by, created_at
) VALUES ($1,$2,$3,$4,$5,$6)`

			_, err = tx.ExecContext(ctx, insertAttachmentQuery,
				attachmentID, "reminders", in.ReminderID, fileID, in.UpdatedBy, in.UpdatedAt,
			)
			if err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return domain.ReminderPayment{}, err
	}
	return in, nil
}

func (r *RepositoryPG) DeletePayment(ctx context.Context, tenantID, reminderID, paymentID, actorUserID string) error {
	return db.WithTenantTx(ctx, r.conn, func(tx *sql.Tx) error {
		// Fetch payment details before deleting
		var (
			amountMinor int64
			paidAt      time.Time
		)
		const paymentQuery = `
			SELECT amount_minor, paid_at 
			FROM finance_reminder_payments 
			WHERE reminder_id = $1 AND id = $2`
		err := tx.QueryRowContext(ctx, paymentQuery, reminderID, paymentID).Scan(&amountMinor, &paidAt)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return errors.New("payment not found")
			}
			return err
		}

		// Fetch reminder details
		var (
			reminderTitle string
			totalTenor    *int
		)
		const reminderQuery = `
			SELECT title, total_tenor 
			FROM finance_reminders 
			WHERE id = $1 AND deleted_at IS NULL`
		err = tx.QueryRowContext(ctx, reminderQuery, reminderID).Scan(&reminderTitle, &totalTenor)
		if err != nil {
			return err
		}

		// Determine the description of the transaction to delete
		description := "pembayaran - " + reminderTitle
		if totalTenor != nil && *totalTenor > 1 {
			// Find the sequence number of this payment being deleted
			// It is the count of payments that were created on or before this payment's creation date/time
			var sequence int
			const countPrevQuery = `
				SELECT COUNT(1) 
				FROM finance_reminder_payments 
				WHERE reminder_id = $1 
				  AND created_at <= (SELECT created_at FROM finance_reminder_payments WHERE id = $2)`
			err = tx.QueryRowContext(ctx, countPrevQuery, reminderID, paymentID).Scan(&sequence)
			if err == nil && sequence > 0 {
				description = fmt.Sprintf("pembayaran tenor %d - %s", sequence, reminderTitle)
			}
		}

		// Perform soft delete on the transaction matching this description, date, and amount
		const deleteTxQuery = `
			UPDATE finance_transactions 
			SET deleted_at = now(), updated_by = $1, updated_at = now() 
			WHERE type = 'expense' 
			  AND amount_minor = $2 
			  AND transaction_date = $3 
			  AND description = $4 
			  AND deleted_at IS NULL`
		_, _ = tx.ExecContext(ctx, deleteTxQuery, actorUserID, amountMinor, paidAt, description)

		const q = `DELETE FROM finance_reminder_payments WHERE reminder_id = $1 AND id = $2`
		res, err := tx.ExecContext(ctx, q, reminderID, paymentID)
		if err != nil {
			return err
		}
		affected, err := res.RowsAffected()
		if err != nil {
			return err
		}
		if affected == 0 {
			return errors.New("payment not found")
		}
		return nil
	})
}

func (r *RepositoryPG) ListPayments(ctx context.Context, tenantID, reminderID string) ([]domain.ReminderPayment, error) {
	var out []domain.ReminderPayment
	err := db.WithTenantTx(ctx, r.conn, func(tx *sql.Tx) error {
		const q = `
SELECT id, reminder_id, paid_at, amount_minor, status, notes, proof_image_url,
       created_by, updated_by, created_at, updated_at
FROM finance_reminder_payments
WHERE reminder_id = $1
ORDER BY paid_at DESC, created_at DESC`

		rows, err := tx.QueryContext(ctx, q, reminderID)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var item domain.ReminderPayment
			err := rows.Scan(
				&item.ID, &item.ReminderID, &item.PaidAt, &item.AmountMinor, &item.Status, &item.Notes, &item.ProofImageURL,
				&item.CreatedBy, &item.UpdatedBy, &item.CreatedAt, &item.UpdatedAt,
			)
			if err != nil {
				return err
			}
			item.TenantID = tenantID
			out = append(out, item)
		}
		return rows.Err()
	})
	return out, err
}
