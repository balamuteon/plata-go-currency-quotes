package quote_test

import (
	"context"
	"testing"
	"time"

	"github.com/balamuteon/plata-go-currency-quotes/internal/domain"
	quoteservice "github.com/balamuteon/plata-go-currency-quotes/internal/service/quote"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestServiceRequestUpdate(t *testing.T) {
	t.Parallel()

	pair := domain.CurrencyPair{Base: domain.EUR, Quote: domain.MXN}
	createdAt := time.Date(2026, time.August, 28, 10, 0, 0, 0, time.UTC)
	update := domain.QuoteUpdate{
		ID:        "7e0f3931-9319-4c95-9ae8-029bf498264b",
		Pair:      pair,
		Status:    domain.StatusQueued,
		CreatedAt: createdAt,
		UpdatedAt: createdAt,
	}

	tests := []struct {
		name       string
		pairValue  string
		keyValue   string
		mock       func(*Mockrepository)
		wantReplay bool
		wantErr    error
		wantUpdate domain.QuoteUpdate
	}{
		{
			name:      "creates update",
			pairValue: " eur/mxn ",
			keyValue:  " request-1 ",
			mock: func(repository *Mockrepository) {
				repository.EXPECT().
					Enqueue(gomock.Any(), gomock.Any(), pair, "request-1").
					DoAndReturn(func(_ context.Context, id string, _ domain.CurrencyPair, _ string) (domain.QuoteUpdate, bool, error) {
						assert.NoError(t, domain.ValidateUpdateID(id))
						return update, false, nil
					})
			},
			wantUpdate: update,
		},
		{
			name:      "returns idempotent replay",
			pairValue: "EUR/MXN",
			keyValue:  "request-1",
			mock: func(repository *Mockrepository) {
				repository.EXPECT().
					Enqueue(gomock.Any(), gomock.Any(), pair, "request-1").
					Return(update, true, nil)
			},
			wantReplay: true,
			wantUpdate: update,
		},
		{
			name:      "rejects invalid pair before repository",
			pairValue: "EUR/EUR",
			keyValue:  "request-1",
			mock:      func(_ *Mockrepository) {},
			wantErr:   domain.ErrInvalidCurrencyPair,
		},
		{
			name:      "rejects invalid idempotency key before repository",
			pairValue: "EUR/MXN",
			keyValue:  "contains space",
			mock:      func(_ *Mockrepository) {},
			wantErr:   domain.ErrInvalidIdempotencyKey,
		},
		{
			name:      "returns repository error",
			pairValue: "EUR/MXN",
			keyValue:  "request-1",
			mock: func(repository *Mockrepository) {
				repository.EXPECT().
					Enqueue(gomock.Any(), gomock.Any(), pair, "request-1").
					Return(domain.QuoteUpdate{}, false, domain.ErrIdempotencyConflict)
			},
			wantErr: domain.ErrIdempotencyConflict,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			repository := NewMockrepository(ctrl)
			test.mock(repository)

			service := quoteservice.NewService(repository)

			got, replayed, err := service.RequestUpdate(context.Background(), test.pairValue, test.keyValue)

			if test.wantErr != nil {
				require.ErrorIs(t, err, test.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, test.wantReplay, replayed)
			assert.Equal(t, test.wantUpdate, got)
		})
	}
}

func TestServiceGetUpdate(t *testing.T) {
	t.Parallel()

	id := "7e0f3931-9319-4c95-9ae8-029bf498264b"
	update := domain.QuoteUpdate{
		ID:     id,
		Pair:   domain.CurrencyPair{Base: domain.EUR, Quote: domain.MXN},
		Status: domain.StatusSucceeded,
	}

	t.Run("returns update", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		repository := NewMockrepository(ctrl)
		repository.EXPECT().FindByID(gomock.Any(), id).Return(update, nil)

		service := quoteservice.NewService(repository)

		got, err := service.GetUpdate(context.Background(), id)
		require.NoError(t, err)
		assert.Equal(t, update, got)
	})

	t.Run("rejects invalid id before repository", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		repository := NewMockrepository(ctrl)
		service := quoteservice.NewService(repository)

		_, err := service.GetUpdate(context.Background(), "bad-id")
		require.ErrorIs(t, err, domain.ErrInvalidUpdateID)
	})
}

func TestServiceGetLatest(t *testing.T) {
	t.Parallel()

	pair := domain.CurrencyPair{Base: domain.EUR, Quote: domain.MXN}
	update := domain.QuoteUpdate{
		ID:     "7e0f3931-9319-4c95-9ae8-029bf498264b",
		Pair:   pair,
		Status: domain.StatusSucceeded,
	}

	t.Run("returns latest quote", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		repository := NewMockrepository(ctrl)
		repository.EXPECT().FindLatest(gomock.Any(), pair).Return(update, nil)

		service := quoteservice.NewService(repository)

		got, err := service.GetLatest(context.Background(), " eur/mxn ")
		require.NoError(t, err)
		assert.Equal(t, update, got)
	})

	t.Run("rejects invalid pair before repository", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		repository := NewMockrepository(ctrl)
		service := quoteservice.NewService(repository)

		_, err := service.GetLatest(context.Background(), "USD/USD")
		require.ErrorIs(t, err, domain.ErrInvalidCurrencyPair)
	})
}
