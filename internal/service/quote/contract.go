package quote

import (
	"context"
	"time"

	"github.com/balamuteon/plata-go-currency-quotes/internal/domain"
)

//go:generate go tool mockgen -source=contract.go -destination=mocks_test.go -package=quote_test

type repository interface {
	Enqueue(ctx context.Context, id string, pair domain.CurrencyPair, idempotencyKey string) (update domain.QuoteUpdate, replayed bool, err error)
	FindByID(ctx context.Context, id string) (domain.QuoteUpdate, error)
	FindLatest(ctx context.Context, pair domain.CurrencyPair) (domain.QuoteUpdate, error)
}

type queue interface {
	Claim(ctx context.Context, leaseDuration time.Duration) (domain.QuoteUpdate, bool, error)
	Complete(ctx context.Context, id, leaseToken string, quote domain.Quote) error
	Retry(ctx context.Context, id, leaseToken string, nextAttemptAt time.Time, errorMessage string) error
	Fail(ctx context.Context, id, leaseToken, errorMessage string) error
}

type provider interface {
	Fetch(ctx context.Context, pair domain.CurrencyPair) (domain.Quote, error)
}
