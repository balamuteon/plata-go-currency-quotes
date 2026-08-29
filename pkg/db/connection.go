// Package db contains reusable PostgreSQL connection setup.
package db

import (
	"context"
	"fmt"
	"time"

	"github.com/balamuteon/plata-go-currency-quotes/pkg/observability/logger"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	defaultConnectTimeout = 5 * time.Second
	defaultMaxAttempts    = 5
	defaultRetryDelay     = 2 * time.Second
)

// NewPostgresPool creates and verifies a PostgreSQL pool. Failed startup
// connections are retried so an application can safely start alongside its
// database in a container environment.
func NewPostgresPool(ctx context.Context, databaseURL string, logger logger.Logger) (*pgxpool.Pool, error) {
	poolConfig, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse PostgreSQL configuration: %w", err)
	}
	poolConfig.ConnConfig.ConnectTimeout = defaultConnectTimeout

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("create PostgreSQL pool: %w", err)
	}

	var lastError error
	for attempt := 1; attempt <= defaultMaxAttempts; attempt++ {
		if err := pool.Ping(ctx); err == nil {
			return pool, nil
		} else {
			lastError = err
		}

		if attempt == defaultMaxAttempts {
			break
		}
		logger.Warn("PostgreSQL is not ready; connection will be retried",
			"attempt", attempt,
			"max_attempts", defaultMaxAttempts,
			"retry_in", defaultRetryDelay,
			"error", lastError,
		)

		select {
		case <-ctx.Done():
			pool.Close()
			return nil, fmt.Errorf("connect to PostgreSQL: %w", ctx.Err())
		case <-time.After(defaultRetryDelay):
		}
	}

	pool.Close()
	return nil, fmt.Errorf("connect to PostgreSQL after %d attempts: %w", defaultMaxAttempts, lastError)
}
