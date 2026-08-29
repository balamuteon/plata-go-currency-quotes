// Package quote implements quote-update use cases.
package quote

import (
	"context"

	"github.com/balamuteon/plata-go-currency-quotes/internal/domain"
)

type Service struct {
	repository repository
}

func NewService(repository repository) *Service {
	return &Service{repository: repository}
}

func (s *Service) RequestUpdate(
	ctx context.Context,
	pairValue string,
	idempotencyKeyValue string,
) (domain.QuoteUpdate, bool, error) {
	pair, err := domain.ParseCurrencyPair(pairValue)
	if err != nil {
		return domain.QuoteUpdate{}, false, err
	}
	idempotencyKey, err := domain.NormalizeIdempotencyKey(idempotencyKeyValue)
	if err != nil {
		return domain.QuoteUpdate{}, false, err
	}

	return s.repository.Enqueue(ctx, domain.NewUpdateID(), pair, idempotencyKey)
}

func (s *Service) GetUpdate(ctx context.Context, id string) (domain.QuoteUpdate, error) {
	if err := domain.ValidateUpdateID(id); err != nil {
		return domain.QuoteUpdate{}, err
	}
	return s.repository.FindByID(ctx, id)
}

func (s *Service) GetLatest(ctx context.Context, pairValue string) (domain.QuoteUpdate, error) {
	pair, err := domain.ParseCurrencyPair(pairValue)
	if err != nil {
		return domain.QuoteUpdate{}, err
	}
	return s.repository.FindLatest(ctx, pair)
}
