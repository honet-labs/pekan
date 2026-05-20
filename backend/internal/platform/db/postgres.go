package db

import (
	"context"
	"database/sql"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type PoolConfig struct {
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
}

// DB is a common interface for sql.DB and sql.Tx
type DB interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

func NewPostgres(databaseURL string, pool PoolConfig) (*sql.DB, error) {
	conn, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return nil, err
	}

	maxOpen := pool.MaxOpenConns
	if maxOpen <= 0 {
		maxOpen = 25
	}
	maxIdle := pool.MaxIdleConns
	if maxIdle <= 0 {
		maxIdle = 25
	}
	maxLifetime := pool.ConnMaxLifetime
	if maxLifetime <= 0 {
		maxLifetime = 30 * time.Minute
	}
	maxIdleTime := pool.ConnMaxIdleTime
	if maxIdleTime <= 0 {
		maxIdleTime = 5 * time.Minute
	}

	conn.SetMaxOpenConns(maxOpen)
	conn.SetMaxIdleConns(maxIdle)
	conn.SetConnMaxLifetime(maxLifetime)
	conn.SetConnMaxIdleTime(maxIdleTime)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := conn.PingContext(ctx); err != nil {
		return nil, err
	}

	// Ensure pgcrypto is enabled for gen_random_uuid()
	if _, err := conn.ExecContext(ctx, "CREATE EXTENSION IF NOT EXISTS pgcrypto"); err != nil {
		// Log error but don't fail, as it might already be enabled or user doesn't have permission
		// which would be handled later if a query fails.
	}

	return conn, nil
}


func WithTx(ctx context.Context, db *sql.DB, fn func(*sql.Tx) error) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := fn(tx); err != nil {
		return err
	}

	return tx.Commit()
}

// CurrentDB returns the connection to use. 
// In the future, this could return a Tx from context if we implement transaction-per-request.
func CurrentDB(ctx context.Context, defaultDB *sql.DB) DB {
	// For now, we return the default DB. 
	// The separate schema logic will be handled by ExecContextWithTenant for write operations,
	// and we might need a better way for Read operations if we don't want to wrap everything in Tx.
	return defaultDB
}
