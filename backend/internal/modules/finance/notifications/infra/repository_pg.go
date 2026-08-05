package infra

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"pekan/backend/internal/modules/finance/notifications/domain"
	"pekan/backend/internal/platform/db"
)

type RepositoryPG struct {
	conn *sql.DB
}

func NewRepositoryPG(conn *sql.DB) *RepositoryPG {
	return &RepositoryPG{conn: conn}
}

func (r *RepositoryPG) Create(ctx context.Context, in domain.Notification) (domain.Notification, error) {
	var out domain.Notification
	err := db.WithTenantTx(ctx, r.conn, func(tx *sql.Tx) error {
		in.ID = uuid.NewString()
		const q = `
INSERT INTO finance_notifications (
  id, notification_type, title, message, status, metadata, created_by, created_at, read_at
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`

		_, err := tx.ExecContext(ctx, q,
			in.ID, in.NotificationType, in.Title, in.Message, in.Status, in.Metadata, in.CreatedBy, in.CreatedAt, in.ReadAt,
		)
		if err != nil {
			return err
		}
		out = in
		return nil
	})
	return out, err
}

func (r *RepositoryPG) List(ctx context.Context, filter domain.ListFilter) ([]domain.Notification, int64, error) {
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.PageSize <= 0 || filter.PageSize > 100 {
		filter.PageSize = 20
	}

	var items []domain.Notification
	var total int64

	err := db.WithTenantTx(ctx, r.conn, func(tx *sql.Tx) error {
		var (
			clauses = []string{"1=1"}
			args    = []any{}
			idx     = 1
		)

		if filter.Status != nil && strings.TrimSpace(*filter.Status) != "" {
			clauses = append(clauses, fmt.Sprintf("status = $%d", idx))
			args = append(args, strings.ToLower(strings.TrimSpace(*filter.Status)))
			idx++
		}

		where := strings.Join(clauses, " AND ")
		countQuery := "SELECT COUNT(1) FROM finance_notifications WHERE " + where

		if err := tx.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
			return err
		}

		offset := (filter.Page - 1) * filter.PageSize
		args = append(args, filter.PageSize, offset)
		dataQuery := fmt.Sprintf(`
SELECT id, notification_type, title, message, status, metadata, created_by, created_at, read_at
FROM finance_notifications
WHERE %s
ORDER BY created_at DESC
LIMIT $%d OFFSET $%d`, where, idx, idx+1)

		rows, err := tx.QueryContext(ctx, dataQuery, args...)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var item domain.Notification
			if err := rows.Scan(
				&item.ID, &item.NotificationType, &item.Title, &item.Message, &item.Status, &item.Metadata,
				&item.CreatedBy, &item.CreatedAt, &item.ReadAt,
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

func (r *RepositoryPG) MarkRead(ctx context.Context, tenantID, notificationID string) (domain.Notification, error) {
	var out domain.Notification
	err := db.WithTenantTx(ctx, r.conn, func(tx *sql.Tx) error {
		const q = `
UPDATE finance_notifications
SET status = 'read', read_at = $1
WHERE id = $2 AND status != 'read'
RETURNING id, notification_type, title, message, status, metadata, created_by, created_at, read_at`

		now := time.Now().UTC()
		err := tx.QueryRowContext(ctx, q, now, notificationID).Scan(
			&out.ID, &out.NotificationType, &out.Title, &out.Message, &out.Status, &out.Metadata,
			&out.CreatedBy, &out.CreatedAt, &out.ReadAt,
		)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return domain.ErrNotificationNotFound
			}
			return err
		}
		out.TenantID = tenantID
		return nil
	})
	return out, err
}

