package infra

import (
	"context"
	"database/sql"

	"github.com/google/uuid"

	"pekan/backend/internal/modules/finance/master/domain"
	"pekan/backend/internal/platform/db"
)

type RepositoryPG struct {
	conn *sql.DB
}

func NewRepositoryPG(conn *sql.DB) *RepositoryPG {
	return &RepositoryPG{conn: conn}
}

func (r *RepositoryPG) CreateAccount(ctx context.Context, account domain.Account) (domain.Account, error) {
	err := db.WithTenantTx(ctx, r.conn, func(tx *sql.Tx) error {
		account.ID = uuid.NewString()
		const q = `
INSERT INTO finance_accounts (
  id, name, account_type, currency, opening_balance_minor, is_active, created_by, created_at, updated_at
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`

		_, err := tx.ExecContext(ctx, q,
			account.ID, account.Name, account.AccountType, account.Currency,
			account.OpeningBalanceMinor, account.IsActive, account.CreatedBy, account.CreatedAt, account.UpdatedAt,
		)
		return err
	})
	if err != nil {
		return domain.Account{}, err
	}
	return account, nil
}

func (r *RepositoryPG) ListAccounts(ctx context.Context, tenantID string) ([]domain.Account, error) {
	if err := r.EnsureDefaultData(ctx, tenantID); err != nil {
		// Log error but continue to try list
	}



	var out []domain.Account
	err := db.WithTenantTx(ctx, r.conn, func(tx *sql.Tx) error {
		const q = `
SELECT id, name, account_type, currency, opening_balance_minor, is_active, created_by, created_at, updated_at
FROM finance_accounts
ORDER BY name ASC`

		rows, err := tx.QueryContext(ctx, q)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var row domain.Account
			if err := rows.Scan(
				&row.ID, &row.Name, &row.AccountType, &row.Currency, &row.OpeningBalanceMinor,
				&row.IsActive, &row.CreatedBy, &row.CreatedAt, &row.UpdatedAt,
			); err != nil {
				return err
			}
			row.TenantID = tenantID
			out = append(out, row)
		}
		return rows.Err()
	})
	return out, err
}


func (r *RepositoryPG) CreateCategory(ctx context.Context, category domain.Category) (domain.Category, error) {
	err := db.WithTenantTx(ctx, r.conn, func(tx *sql.Tx) error {
		category.ID = uuid.NewString()
		const q = `
INSERT INTO finance_categories (
  id, name, category_type, parent_id, is_active, created_by, created_at, updated_at
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`

		_, err := tx.ExecContext(ctx, q,
			category.ID, category.Name, category.CategoryType, category.ParentID,
			category.IsActive, category.CreatedBy, category.CreatedAt, category.UpdatedAt,
		)
		return err
	})
	if err != nil {
		return domain.Category{}, err
	}
	return category, nil
}

func (r *RepositoryPG) ListCategories(ctx context.Context, tenantID string) ([]domain.Category, error) {
	if err := r.EnsureDefaultData(ctx, tenantID); err != nil {
		// Log error but continue
	}



	var out []domain.Category
	err := db.WithTenantTx(ctx, r.conn, func(tx *sql.Tx) error {
		const q = `
SELECT id, name, category_type, parent_id, is_active, created_by, created_at, updated_at
FROM finance_categories
ORDER BY category_type ASC, name ASC`

		rows, err := tx.QueryContext(ctx, q)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var row domain.Category
			if err := rows.Scan(
				&row.ID, &row.Name, &row.CategoryType, &row.ParentID, &row.IsActive, &row.CreatedBy, &row.CreatedAt, &row.UpdatedAt,
			); err != nil {
				return err
			}
			row.TenantID = tenantID
			out = append(out, row)
		}
		return rows.Err()
	})
	return out, err
}


func (r *RepositoryPG) UpdateCategory(ctx context.Context, tenantID, categoryID, name, categoryType string) (domain.Category, error) {
	var row domain.Category
	err := db.WithTenantTx(ctx, r.conn, func(tx *sql.Tx) error {
		const q = `
UPDATE finance_categories
SET name = $1, category_type = $2, updated_at = NOW()
WHERE id = $3
RETURNING id, name, category_type, parent_id, is_active, created_by, created_at, updated_at`
		err := tx.QueryRowContext(ctx, q, name, categoryType, categoryID).Scan(
			&row.ID, &row.Name, &row.CategoryType, &row.ParentID, &row.IsActive, &row.CreatedBy, &row.CreatedAt, &row.UpdatedAt,
		)
		row.TenantID = tenantID
		return err
	})
	if err != nil {
		return domain.Category{}, err
	}
	return row, nil
}

func (r *RepositoryPG) DeleteCategory(ctx context.Context, tenantID, categoryID string) error {
	return db.WithTenantTx(ctx, r.conn, func(tx *sql.Tx) error {
		const q = `DELETE FROM finance_categories WHERE id = $1`
		_, err := tx.ExecContext(ctx, q, categoryID)
		return err
	})
}

func (r *RepositoryPG) GetCategory(ctx context.Context, tenantID, categoryID string) (domain.Category, error) {
	var row domain.Category
	err := db.WithTenantTx(ctx, r.conn, func(tx *sql.Tx) error {
		const q = `
SELECT id, name, category_type, parent_id, is_active, created_by, created_at, updated_at
FROM finance_categories
WHERE id = $1`
		err := tx.QueryRowContext(ctx, q, categoryID).Scan(
			&row.ID, &row.Name, &row.CategoryType, &row.ParentID, &row.IsActive, &row.CreatedBy, &row.CreatedAt, &row.UpdatedAt,
		)
		row.TenantID = tenantID
		return err
	})
	if err != nil {
		return domain.Category{}, err
	}
	return row, nil
}

func (r *RepositoryPG) EnsureDefaultData(ctx context.Context, tenantID string) error {
	return db.WithTenantTx(ctx, r.conn, func(tx *sql.Tx) error {
		// 1. Check if accounts exist
		var accCount int
		if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM finance_accounts`).Scan(&accCount); err != nil {
			return err
		}

		// 2. Check if categories exist
		var catCount int
		if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM finance_categories`).Scan(&catCount); err != nil {
			return err
		}

		if accCount > 0 && catCount > 0 {
			return nil // Data already exists
		}

		// 3. Get first owner/admin of the workspace to attribute creation
		var userID string
		qUser := `SELECT user_id FROM public.tenant_memberships WHERE tenant_id = $1 ORDER BY joined_at ASC LIMIT 1`
		if err := tx.QueryRowContext(ctx, qUser, tenantID).Scan(&userID); err != nil {
			userID = uuid.NewString()
		}


		// 4. Seed Account if empty
		if accCount == 0 {
			const qAcc = `
INSERT INTO finance_accounts (id, name, account_type, currency, opening_balance_minor, is_active, created_by, created_at, updated_at)
VALUES (gen_random_uuid(), 'Kas Utama', 'cash', 'IDR', 0, TRUE, $1, NOW(), NOW())`
			if _, err := tx.ExecContext(ctx, qAcc, userID); err != nil {
				return err
			}
		}

		// 5. Seed Categories if empty
		if catCount == 0 {
			const qCat = `
INSERT INTO finance_categories (id, name, category_type, is_active, created_by, created_at, updated_at)
VALUES 
    (gen_random_uuid(), 'Makanan & Minuman', 'expense', TRUE, $1, NOW(), NOW()),
    (gen_random_uuid(), 'Transportasi', 'expense', TRUE, $1, NOW(), NOW()),
    (gen_random_uuid(), 'Peralatan Rumah', 'expense', TRUE, $1, NOW(), NOW()),
    (gen_random_uuid(), 'Belanja', 'expense', TRUE, $1, NOW(), NOW()),
    (gen_random_uuid(), 'Gaji', 'income', TRUE, $1, NOW(), NOW()),
    (gen_random_uuid(), 'Bonus', 'income', TRUE, $1, NOW(), NOW())`
			if _, err := tx.ExecContext(ctx, qCat, userID); err != nil {
				return err
			}
		}

		return nil
	})
}


