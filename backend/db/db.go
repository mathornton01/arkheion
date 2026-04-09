// Package db provides the PostgreSQL connection pool for the Arkheion backend.
package db

import (
	"context"
	"fmt"
	"time"

	"github.com/mathornton01/arkheion/config"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Connect creates and returns a validated pgxpool connection pool.
// The caller is responsible for calling pool.Close() when done.
func Connect(cfg *config.Config) (*pgxpool.Pool, error) {
	poolCfg, err := pgxpool.ParseConfig(cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse database URL: %w", err)
	}

	poolCfg.MaxConns = int32(cfg.DBMaxOpenConns)
	poolCfg.MinConns = int32(cfg.DBMaxIdleConns)
	poolCfg.MaxConnLifetime = time.Duration(cfg.DBConnMaxLifetimeMinutes) * time.Minute
	poolCfg.MaxConnIdleTime = 10 * time.Minute
	poolCfg.HealthCheckPeriod = 1 * time.Minute

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("create connection pool: %w", err)
	}

	// Verify connectivity.
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	return pool, nil
}
