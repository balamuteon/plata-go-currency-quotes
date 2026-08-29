package quote_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/balamuteon/plata-go-currency-quotes/internal/config"
	quoteworker "github.com/balamuteon/plata-go-currency-quotes/internal/worker/quote"
	"github.com/balamuteon/plata-go-currency-quotes/pkg/observability/logger"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestWorkerRun(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		mock func(context.CancelFunc, *Mockprocessor)
		cfg  config.Worker
	}{
		{
			name: "stops when processor found nothing and context is cancelled",
			mock: func(cancel context.CancelFunc, processor *Mockprocessor) {
				processor.EXPECT().
					ProcessNext(gomock.Any()).
					DoAndReturn(func(context.Context) (bool, error) {
						cancel()
						return false, nil
					})
			},
			cfg: testWorkerConfig(time.Hour),
		},
		{
			name: "continues immediately after processed update",
			mock: func(cancel context.CancelFunc, processor *Mockprocessor) {
				gomock.InOrder(
					processor.EXPECT().
						ProcessNext(gomock.Any()).
						Return(true, nil),
					processor.EXPECT().
						ProcessNext(gomock.Any()).
						DoAndReturn(func(context.Context) (bool, error) {
							cancel()
							return false, nil
						}),
				)
			},
			cfg: testWorkerConfig(time.Hour),
		},
		{
			name: "continues after processor error",
			mock: func(cancel context.CancelFunc, processor *Mockprocessor) {
				gomock.InOrder(
					processor.EXPECT().
						ProcessNext(gomock.Any()).
						Return(false, errors.New("process quote update")),
					processor.EXPECT().
						ProcessNext(gomock.Any()).
						DoAndReturn(func(context.Context) (bool, error) {
							cancel()
							return false, nil
						}),
				)
			},
			cfg: testWorkerConfig(time.Millisecond),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			processor := NewMockprocessor(ctrl)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			test.mock(cancel, processor)

			worker := quoteworker.New(processor, test.cfg, logger.NewNoop())
			done := make(chan struct{})

			go func() {
				defer close(done)
				worker.Run(ctx)
			}()

			require.Eventually(t, func() bool {
				select {
				case <-done:
					return true
				default:
					return false
				}
			}, time.Second, 10*time.Millisecond)
		})
	}
}

func TestWorkerRunStopsWhenContextIsCancelled(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	processor := NewMockprocessor(ctrl)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	worker := quoteworker.New(processor, testWorkerConfig(time.Hour), logger.NewNoop())

	done := make(chan struct{})
	go func() {
		defer close(done)
		worker.Run(ctx)
	}()

	require.Eventually(t, func() bool {
		select {
		case <-done:
			return true
		default:
			return false
		}
	}, time.Second, 10*time.Millisecond)
}

func testWorkerConfig(pollInterval time.Duration) config.Worker {
	return config.Worker{
		Count:        1,
		PollInterval: pollInterval,
	}
}
