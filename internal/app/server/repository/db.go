package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type DBConnection struct {
	Pool *pgxpool.Pool
}

func NewDBConnection(dsn string) (*DBConnection, error) {
	if dsn == "" {
		return nil, nil
	}

	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database DSN: %w", err)
	}

	config.ConnConfig.RuntimeParams["extra_float_digits"] = "3"

	config.MaxConns = 5
	config.MinConns = 1
	config.HealthCheckPeriod = 30 * time.Second

	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		return nil, fmt.Errorf("failed to create pgx pool: %w", err)
	}

	if err := pool.Ping(context.Background()); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &DBConnection{Pool: pool}, nil
}

// CreateMetricsTable создаёт таблицу для хранения метрик, если она не существует.
// В качестве примера в одной таблице хранятся и counter, и gauge:
//   - mtype = 'counter' или 'gauge'
//   - ivalue — для counter
//   - fvalue — для gauge

func (db *DBConnection) Ping(ctx context.Context) error {
	if db == nil || db.Pool == nil {
		return fmt.Errorf("database not configured")
	}
	return db.Pool.Ping(ctx)
}

func (db *DBConnection) Close() {
	if db != nil && db.Pool != nil {
		db.Pool.Close()
	}
}
