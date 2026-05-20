package tests

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"

	transactiondomain "pekan/backend/internal/modules/finance/transactions/domain"
	transactioninfra "pekan/backend/internal/modules/finance/transactions/infra"
)

func TestRepositoryGetByIDTenantScoped(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock new: %v", err)
	}
	defer db.Close()

	repo := transactioninfra.NewRepositoryPG(db)
	now := time.Now().UTC()

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("SET LOCAL search_path TO public, public")).
		WillReturnResult(sqlmock.NewResult(0, 0))

	mock.ExpectQuery(regexp.QuoteMeta(`
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
WHERE t.id = $1 AND t.deleted_at IS NULL`)).
		WithArgs("trx-1").
		WillReturnRows(sqlmock.NewRows([]string{
			"id",
			"account_id",
			"account_name",
			"category_id",
			"category_name",
			"type",
			"amount_minor",
			"currency",
			"input_date",
			"transaction_date",
			"description",
			"merchant_name",
			"receipt_number",
			"payment_method",
			"subtotal_minor",
			"tax_minor",
			"service_charge_minor",
			"receipt_discount_minor",
			"notes",
			"created_by",
			"created_by_name",
			"updated_by",
			"created_at",
			"updated_at",
		}).AddRow("trx-1", "acc-1", "Cash Wallet", nil, nil, "expense", int64(1000), "IDR", now, now, nil, nil, nil, nil, int64(0), int64(0), int64(0), int64(0), nil, "user-1", "User 1", "user-1", now, now))
	mock.ExpectCommit()

	out, err := repo.GetByID(context.Background(), "tenant-a", "trx-1")
	if err != nil {
		t.Fatalf("GetByID err: %v", err)
	}
	if out.TenantID != "tenant-a" {
		t.Fatalf("unexpected tenant: %s", out.TenantID)
	}
}

func TestRepositoryGetByIDCrossTenantReturnsNotFound(t *testing.T) {
	t.Parallel()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock new: %v", err)
	}
	defer db.Close()

	repo := transactioninfra.NewRepositoryPG(db)

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("SET LOCAL search_path TO public, public")).
		WillReturnResult(sqlmock.NewResult(0, 0))

	mock.ExpectQuery(regexp.QuoteMeta(`
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
WHERE t.id = $1 AND t.deleted_at IS NULL`)).
		WithArgs("trx-2").
		WillReturnRows(sqlmock.NewRows([]string{
			"id",
			"account_id",
			"account_name",
			"category_id",
			"category_name",
			"type",
			"amount_minor",
			"currency",
			"input_date",
			"transaction_date",
			"description",
			"merchant_name",
			"receipt_number",
			"payment_method",
			"subtotal_minor",
			"tax_minor",
			"service_charge_minor",
			"receipt_discount_minor",
			"notes",
			"created_by",
			"created_by_name",
			"updated_by",
			"created_at",
			"updated_at",
		}))
	mock.ExpectRollback()

	_, err = repo.GetByID(context.Background(), "tenant-a", "trx-2")
	if err == nil {
		t.Fatalf("expected not found error")
	}
	if err != transactiondomain.ErrTransactionNotFound {
		t.Fatalf("expected ErrTransactionNotFound got=%v", err)
	}
}
