// Package quote implements quote persistence and the PostgreSQL-backed work queue.
package quote

import (
	"context"
	"errors"
	"fmt"

	"github.com/balamuteon/plata-go-currency-quotes/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type storage interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

type Repository struct {
	storage storage
}

func NewRepository(storage storage) *Repository {
	return &Repository{storage: storage}
}

func (r *Repository) FindByID(ctx context.Context, id string) (domain.QuoteUpdate, error) {
	update, err := scanQuoteUpdate(r.storage.QueryRow(ctx, `
		SELECT `+returningFields+`
		FROM quote_updates
		WHERE id = $1::uuid`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.QuoteUpdate{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.QuoteUpdate{}, fmt.Errorf("find quote update: %w", err)
	}
	return update, nil
}

func (r *Repository) FindLatest(ctx context.Context, pair domain.CurrencyPair) (domain.QuoteUpdate, error) {
	update, err := scanQuoteUpdate(r.storage.QueryRow(ctx, `
		SELECT `+returningFields+`
		FROM quote_updates
		WHERE pair = $1
		  AND status = 'succeeded'
		ORDER BY updated_at DESC, created_at DESC
		LIMIT 1`, pair.String()))
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.QuoteUpdate{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.QuoteUpdate{}, fmt.Errorf("find latest quote: %w", err)
	}
	return update, nil
}

func (r *Repository) Enqueue(
	ctx context.Context,
	id string,
	pair domain.CurrencyPair,
	idempotencyKey string,
) (domain.QuoteUpdate, bool, error) {
	update, err := scanQuoteUpdate(r.storage.QueryRow(ctx, `
		INSERT INTO quote_updates (id, pair, idempotency_key)
		VALUES ($1::uuid, $2, $3)
		ON CONFLICT (idempotency_key) DO NOTHING
		RETURNING `+returningFields, id, pair.String(), idempotencyKey))
	if err == nil {
		return update, false, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return domain.QuoteUpdate{}, false, fmt.Errorf("insert quote update: %w", err)
	}

	existing, err := scanQuoteUpdate(r.storage.QueryRow(ctx, `
		SELECT `+returningFields+`
		FROM quote_updates
		WHERE idempotency_key = $1`, idempotencyKey))
	if err != nil {
		return domain.QuoteUpdate{}, false, fmt.Errorf("find idempotent quote update: %w", err)
	}
	if existing.Pair != pair {
		return domain.QuoteUpdate{}, false, domain.ErrIdempotencyConflict
	}

	return existing, true, nil
}
