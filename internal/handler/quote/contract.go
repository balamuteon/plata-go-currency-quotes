package quote

import (
	"context"

	"github.com/balamuteon/plata-go-currency-quotes/internal/domain"
)

//go:generate go tool mockgen -source=contract.go -destination=mocks_test.go -package=quote_test

type quoteService interface {
	RequestUpdate(ctx context.Context, pair, idempotencyKey string) (domain.QuoteUpdate, bool, error)
	GetUpdate(ctx context.Context, id string) (domain.QuoteUpdate, error)
	GetLatest(ctx context.Context, pair string) (domain.QuoteUpdate, error)
}
