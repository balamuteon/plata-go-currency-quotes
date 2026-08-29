// Package testdb предоставляет функции для настройки тестовой базы данных.
package testdb

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"runtime"
	"time"

	// Драйвер pgx нужен для работы goose с базой через database/sql.
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// SetupTestDatabase поднимает контейнер с Postgres, применяет миграции и возвращает DSN и функцию очистки.
func SetupTestDatabase(ctx context.Context) (string, func(), error) {
	pgContainer, err := postgres.Run(ctx,
		"postgres:16",
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("test_user"),
		postgres.WithPassword("test_password"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second),
		),
	)
	if err != nil {
		return "", nil, fmt.Errorf("failed to start postgres container: %w", err)
	}

	teardown := func() {
		if err = pgContainer.Terminate(ctx); err != nil {
			fmt.Printf("failed to terminate container: %v\n", err)
		}
	}

	dsn, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		teardown()
		return "", nil, fmt.Errorf("failed to get connection string: %w", err)
	}

	if err := runMigrations(dsn); err != nil {
		teardown()
		return "", nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return dsn, teardown, nil
}

func runMigrations(dsn string) error {
	// Определяем путь к папке migrations относительно текущего файла.
	_, b, _, _ := runtime.Caller(0)
	basepath := filepath.Dir(b)
	migrationsDir := filepath.Join(basepath, "../../migrations")

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return err
	}
	defer func() {
		_ = db.Close()
	}()

	if err := goose.SetDialect("postgres"); err != nil {
		return err
	}

	if err := goose.Up(db, migrationsDir); err != nil {
		return err
	}

	return nil
}
