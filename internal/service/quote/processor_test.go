package quote_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/balamuteon/plata-go-currency-quotes/internal/config"
	"github.com/balamuteon/plata-go-currency-quotes/internal/domain"
	quoteservice "github.com/balamuteon/plata-go-currency-quotes/internal/service/quote"
	"github.com/balamuteon/plata-go-currency-quotes/pkg/observability/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestProcessorProcessNext(t *testing.T) {
	t.Parallel()

	pair := domain.CurrencyPair{Base: domain.EUR, Quote: domain.MXN}
	update := domain.QuoteUpdate{
		ID:         "7e0f3931-9319-4c95-9ae8-029bf498264b",
		Pair:       pair,
		Status:     domain.StatusProcessing,
		Attempts:   1,
		LeaseToken: testLeaseValue(),
	}
	quote := domain.Quote{Price: 21.75}

	tests := []struct {
		name      string
		mock      func(*testing.T, *Mockqueue, *Mockprovider, config.Worker)
		wantFound bool
		wantErr   error
	}{
		{
			name: "returns not found when queue is empty",
			mock: func(_ *testing.T, queue *Mockqueue, _ *Mockprovider, cfg config.Worker) {
				queue.EXPECT().
					Claim(gomock.Any(), cfg.LeaseDuration).
					Return(domain.QuoteUpdate{}, false, nil)
			},
		},
		{
			name: "returns claim error",
			mock: func(_ *testing.T, queue *Mockqueue, _ *Mockprovider, cfg config.Worker) {
				queue.EXPECT().
					Claim(gomock.Any(), cfg.LeaseDuration).
					Return(domain.QuoteUpdate{}, false, domain.ErrNotFound)
			},
			wantErr: domain.ErrNotFound,
		},
		{
			name: "completes claimed update",
			mock: func(_ *testing.T, queue *Mockqueue, provider *Mockprovider, cfg config.Worker) {
				gomock.InOrder(
					queue.EXPECT().
						Claim(gomock.Any(), cfg.LeaseDuration).
						Return(update, true, nil),
					provider.EXPECT().
						Fetch(gomock.Any(), pair).
						Return(quote, nil),
					queue.EXPECT().
						Complete(gomock.Any(), update.ID, update.LeaseToken, quote).
						Return(nil),
				)
			},
			wantFound: true,
		},
		{
			name: "swallows complete lease error after claimed update",
			mock: func(_ *testing.T, queue *Mockqueue, provider *Mockprovider, cfg config.Worker) {
				gomock.InOrder(
					queue.EXPECT().
						Claim(gomock.Any(), cfg.LeaseDuration).
						Return(update, true, nil),
					provider.EXPECT().
						Fetch(gomock.Any(), pair).
						Return(quote, nil),
					queue.EXPECT().
						Complete(gomock.Any(), update.ID, update.LeaseToken, quote).
						Return(domain.ErrLeaseLost),
				)
			},
			wantFound: true,
		},
		{
			name: "retries provider error while attempts remain",
			mock: func(t *testing.T, queue *Mockqueue, provider *Mockprovider, cfg config.Worker) {
				t.Helper()

				providerErr := errors.New("provider down")

				gomock.InOrder(
					queue.EXPECT().
						Claim(gomock.Any(), cfg.LeaseDuration).
						Return(update, true, nil),
					provider.EXPECT().
						Fetch(gomock.Any(), pair).
						Return(domain.Quote{}, providerErr),
					queue.EXPECT().
						Retry(gomock.Any(), update.ID, update.LeaseToken, gomock.Any(), providerErr.Error()).
						DoAndReturn(func(_ context.Context, _ string, _ string, nextAttemptAt time.Time, _ string) error {
							assert.True(t, nextAttemptAt.After(time.Now().UTC().Add(59*time.Minute)))
							return nil
						}),
				)
			},
			wantFound: true,
		},
		{
			name: "fails provider error when attempts are exhausted",
			mock: func(_ *testing.T, queue *Mockqueue, provider *Mockprovider, cfg config.Worker) {
				exhaustedUpdate := update
				exhaustedUpdate.Attempts = cfg.MaxAttempts
				providerErr := errors.New("provider down")

				gomock.InOrder(
					queue.EXPECT().
						Claim(gomock.Any(), cfg.LeaseDuration).
						Return(exhaustedUpdate, true, nil),
					provider.EXPECT().
						Fetch(gomock.Any(), pair).
						Return(domain.Quote{}, providerErr),
					queue.EXPECT().
						Fail(gomock.Any(), update.ID, update.LeaseToken, providerErr.Error()).
						Return(nil),
				)
			},
			wantFound: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			queue := NewMockqueue(ctrl)
			provider := NewMockprovider(ctrl)
			cfg := testProcessorConfig()

			test.mock(t, queue, provider, cfg)

			processor := quoteservice.NewProcessor(queue, provider, cfg, logger.NewNoop())

			found, err := processor.ProcessNext(context.Background())

			if test.wantErr != nil {
				require.ErrorIs(t, err, test.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, test.wantFound, found)
		})
	}
}

func TestProcessorProcessNextSkipsRetryWhenContextIsCancelled(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	queue := NewMockqueue(ctrl)
	provider := NewMockprovider(ctrl)
	cfg := testProcessorConfig()
	pair := domain.CurrencyPair{Base: domain.EUR, Quote: domain.MXN}
	update := domain.QuoteUpdate{
		ID:         "7e0f3931-9319-4c95-9ae8-029bf498264b",
		Pair:       pair,
		Status:     domain.StatusProcessing,
		Attempts:   1,
		LeaseToken: testLeaseValue(),
	}
	providerErr := errors.New("provider down")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	gomock.InOrder(
		queue.EXPECT().
			Claim(gomock.Any(), cfg.LeaseDuration).
			Return(update, true, nil),
		provider.EXPECT().
			Fetch(gomock.Any(), pair).
			DoAndReturn(func(context.Context, domain.CurrencyPair) (domain.Quote, error) {
				cancel()
				return domain.Quote{}, providerErr
			}),
	)

	processor := quoteservice.NewProcessor(queue, provider, cfg, logger.NewNoop())

	found, err := processor.ProcessNext(ctx)

	require.NoError(t, err)
	assert.True(t, found)
}

func testProcessorConfig() config.Worker {
	return config.Worker{
		LeaseDuration:  time.Minute,
		MaxAttempts:    3,
		RetryBaseDelay: time.Hour,
		RetryMaxDelay:  2 * time.Hour,
	}
}

func testLeaseValue() string {
	return "lease-1"
}
