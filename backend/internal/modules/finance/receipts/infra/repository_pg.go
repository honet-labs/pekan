package infra

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	"github.com/google/uuid"

	"pekan/backend/internal/modules/finance/receipts/domain"
	"pekan/backend/internal/platform/db"
)

type RepositoryPG struct{ conn *sql.DB }

func NewRepositoryPG(conn *sql.DB) *RepositoryPG { return &RepositoryPG{conn: conn} }

func (r *RepositoryPG) ListProviderConfigs(ctx context.Context, tenantID string) ([]domain.ProviderConfig, error) {
	var items []domain.ProviderConfig
	err := db.WithTenantTx(ctx, r.conn, func(tx *sql.Tx) error {
		const q = `
SELECT id, provider_code, display_name, base_url, model_name, is_enabled,
       api_key_ciphertext, created_at, updated_at
FROM finance_receipt_provider_configs
ORDER BY provider_code ASC`
		rows, err := tx.QueryContext(ctx, q)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var item domain.ProviderConfig
			if err := rows.Scan(&item.ID, &item.ProviderCode, &item.DisplayName, &item.BaseURL, &item.ModelName, &item.IsEnabled, &item.APIKeyCiphertext, &item.CreatedAt, &item.UpdatedAt); err != nil {
				return err
			}
			item.TenantID = tenantID
			item.HasAPIKey = item.APIKeyCiphertext != nil && *item.APIKeyCiphertext != ""
			items = append(items, item)
		}
		return rows.Err()
	})
	return items, err
}

func (r *RepositoryPG) GetProviderConfig(ctx context.Context, tenantID, providerCode string) (domain.ProviderConfig, error) {
	var item domain.ProviderConfig
	err := db.WithTenantTx(ctx, r.conn, func(tx *sql.Tx) error {
		const q = `
SELECT id, provider_code, display_name, base_url, model_name, is_enabled,
       api_key_ciphertext, created_at, updated_at
FROM finance_receipt_provider_configs
WHERE provider_code = $1`
		err := tx.QueryRowContext(ctx, q, providerCode).Scan(
			&item.ID, &item.ProviderCode, &item.DisplayName, &item.BaseURL,
			&item.ModelName, &item.IsEnabled, &item.APIKeyCiphertext, &item.CreatedAt, &item.UpdatedAt,
		)
		if err != nil {
			return err
		}
		item.TenantID = tenantID
		item.HasAPIKey = item.APIKeyCiphertext != nil && *item.APIKeyCiphertext != ""
		return nil
	})
	return item, err
}

func (r *RepositoryPG) UpsertProviderConfig(ctx context.Context, item domain.ProviderConfig) (domain.ProviderConfig, error) {
	var out domain.ProviderConfig
	err := db.WithTenantTx(ctx, r.conn, func(tx *sql.Tx) error {
		if item.ID == "" {
			item.ID = uuid.NewString()
		}
		const q = `
INSERT INTO finance_receipt_provider_configs (
  id, provider_code, display_name, base_url, model_name, is_enabled,
  api_key_ciphertext, created_by, updated_by, created_at, updated_at
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
ON CONFLICT (provider_code)
DO UPDATE SET
  display_name = EXCLUDED.display_name,
  base_url = EXCLUDED.base_url,
  model_name = EXCLUDED.model_name,
  is_enabled = EXCLUDED.is_enabled,
  api_key_ciphertext = COALESCE(EXCLUDED.api_key_ciphertext, finance_receipt_provider_configs.api_key_ciphertext),
  updated_by = EXCLUDED.updated_by,
  updated_at = EXCLUDED.updated_at
RETURNING id, provider_code, display_name, base_url, model_name, is_enabled, api_key_ciphertext, created_at, updated_at`

		err := tx.QueryRowContext(ctx, q,
			item.ID, item.ProviderCode, item.DisplayName, item.BaseURL, item.ModelName, item.IsEnabled,
			item.APIKeyCiphertext, item.CreatedBy, item.UpdatedBy, item.CreatedAt, item.UpdatedAt,
		).Scan(&out.ID, &out.ProviderCode, &out.DisplayName, &out.BaseURL, &out.ModelName, &out.IsEnabled, &out.APIKeyCiphertext, &out.CreatedAt, &out.UpdatedAt)
		if err != nil {
			return err
		}
		out.TenantID = item.TenantID
		out.HasAPIKey = out.APIKeyCiphertext != nil && *out.APIKeyCiphertext != ""
		return nil
	})
	return out, err
}

func scanReceiptScanRow(scanner interface{ Scan(dest ...any) error }, item *domain.ReceiptScan) error {
	var extracted []byte
	if err := scanner.Scan(
		&item.ID,
		&item.ProviderCode,
		&item.ModelName,
		&item.Status,
		&item.OriginalFilename,
		&item.MimeType,
		&extracted,
		&item.ErrorMessage,
		&item.CreatedAt,
		&item.UpdatedAt,
	); err != nil {
		return err
	}
	if len(extracted) > 0 {
		item.ExtractedJSON = json.RawMessage(append([]byte(nil), extracted...))
	} else {
		item.ExtractedJSON = nil
	}
	return nil
}

func (r *RepositoryPG) CreateReceiptScan(ctx context.Context, item domain.ReceiptScan) (domain.ReceiptScan, error) {
	var out domain.ReceiptScan
	err := db.WithTenantTx(ctx, r.conn, func(tx *sql.Tx) error {
		if item.ID == "" {
			item.ID = uuid.NewString()
		}
		const q = `
INSERT INTO finance_receipt_scans (
  id, provider_code, model_name, status, original_filename, mime_type,
  extracted_json, error_message, created_by, created_at, updated_at
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
RETURNING id, provider_code, model_name, status, original_filename, mime_type, extracted_json, error_message, created_at, updated_at`

		err := scanReceiptScanRow(
			tx.QueryRowContext(ctx, q,
				item.ID, item.ProviderCode, item.ModelName, item.Status, item.OriginalFilename, item.MimeType,
				item.ExtractedJSON, item.ErrorMessage, item.CreatedBy, item.CreatedAt, item.UpdatedAt,
			),
			&out,
		)
		if err != nil {
			return err
		}
		out.TenantID = item.TenantID
		return nil
	})
	return out, err
}

func (r *RepositoryPG) UpdateReceiptScanResult(ctx context.Context, item domain.ReceiptScan) (domain.ReceiptScan, error) {
	var out domain.ReceiptScan
	err := db.WithTenantTx(ctx, r.conn, func(tx *sql.Tx) error {
		const q = `
UPDATE finance_receipt_scans
SET status = $2,
    extracted_json = $3,
    error_message = $4,
    updated_at = $5,
    model_name = $6,
    provider_code = $7
WHERE id = $1
RETURNING id, provider_code, model_name, status, original_filename, mime_type, extracted_json, error_message, created_at, updated_at`

		err := scanReceiptScanRow(
			tx.QueryRowContext(ctx, q, item.ID, item.Status, item.ExtractedJSON, item.ErrorMessage, item.UpdatedAt, item.ModelName, item.ProviderCode),
			&out,
		)
		if err != nil {
			return err
		}
		out.TenantID = item.TenantID
		return nil
	})
	return out, err
}

func (r *RepositoryPG) GetReceiptScanByID(ctx context.Context, tenantID, scanID string) (domain.ReceiptScan, error) {
	var item domain.ReceiptScan
	err := db.WithTenantTx(ctx, r.conn, func(tx *sql.Tx) error {
		const q = `
SELECT id, provider_code, model_name, status, original_filename, mime_type, extracted_json, error_message, created_at, updated_at
FROM finance_receipt_scans
WHERE id = $1`
		err := scanReceiptScanRow(tx.QueryRowContext(ctx, q, scanID), &item)
		if err != nil {
			return err
		}
		item.TenantID = tenantID
		return nil
	})
	return item, err
}

func (r *RepositoryPG) ListReceiptScans(ctx context.Context, tenantID string, limit int) ([]domain.ReceiptScan, error) {
	if limit <= 0 {
		limit = 20
	}
	var items []domain.ReceiptScan
	err := db.WithTenantTx(ctx, r.conn, func(tx *sql.Tx) error {
		const q = `
SELECT id, provider_code, model_name, status, original_filename, mime_type, extracted_json, error_message, created_at, updated_at
FROM finance_receipt_scans
ORDER BY created_at DESC
LIMIT $1`
		rows, err := tx.QueryContext(ctx, q, limit)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var item domain.ReceiptScan
			if err := scanReceiptScanRow(rows, &item); err != nil {
				return err
			}
			item.TenantID = tenantID
			items = append(items, item)
		}
		return rows.Err()
	})
	return items, err
}

func (r *RepositoryPG) DeleteReceiptScan(ctx context.Context, tenantID, scanID string) error {
	return db.WithTenantTx(ctx, r.conn, func(tx *sql.Tx) error {
		const q = `DELETE FROM finance_receipt_scans WHERE id = $1`
		_, err := tx.ExecContext(ctx, q, scanID)
		return err
	})
}

func (r *RepositoryPG) ClearReceiptScans(ctx context.Context, tenantID string) error {
	return db.WithTenantTx(ctx, r.conn, func(tx *sql.Tx) error {
		const q = `DELETE FROM finance_receipt_scans`
		_, err := tx.ExecContext(ctx, q)
		return err
	})
}

func (r *RepositoryPG) GetGlobalSetting(ctx context.Context, key string) (string, bool, error) {
	var val string
	var enc bool
	// Use WithTenantTx just to get a connection and handle search_path if needed,
	// though global_settings is likely in 'public' schema.
	err := db.WithTenantTx(ctx, r.conn, func(tx *sql.Tx) error {
		const q = `SELECT value, is_encrypted FROM global_settings WHERE key = $1`
		err := tx.QueryRowContext(ctx, q, key).Scan(&val, &enc)
		if err == sql.ErrNoRows {
			return nil // handled outside
		}
		return err
	})
	if err == sql.ErrNoRows || (err == nil && val == "") {
		// If WithTenantTx returns nil but we didn't find anything, we might need to check how it handles ErrNoRows
		// Actually WithTenantTx returns the error from the function.
	}
	return val, enc, err
}

var _ domain.Repository = (*RepositoryPG)(nil)

func IsNotFound(err error) bool { return errors.Is(err, sql.ErrNoRows) }
