// Package quote processes queued quote updates in the background.
package quote

import (
	"context"
	"sync"
	"time"

	"github.com/balamuteon/plata-go-currency-quotes/internal/config"
	"github.com/balamuteon/plata-go-currency-quotes/pkg/observability/logger"
)

type Worker struct {
	processor processor
	config    config.Worker
	logger    logger.Logger
}

func New(processor processor, cfg config.Worker, logger logger.Logger) *Worker {
	return &Worker{
		processor: processor,
		config:    cfg,
		logger:    logger,
	}
}

func (w *Worker) Run(ctx context.Context) {
	var wg sync.WaitGroup
	wg.Add(w.config.Count)

	for workerID := 1; workerID <= w.config.Count; workerID++ {
		go func() {
			defer wg.Done()
			w.runProcessor(ctx, workerID)
		}()
	}

	wg.Wait()
}

func (w *Worker) runProcessor(ctx context.Context, workerID int) {
	logger := w.logger.With("worker_id", workerID)
	logger.Info("quote worker started")
	defer logger.Info("quote worker stopped")

	for {
		if ctx.Err() != nil {
			return
		}

		found, err := w.processor.ProcessNext(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			logger.Error("failed to process quote update", "error", err)
			if !wait(ctx, w.config.PollInterval) {
				return
			}
			continue
		}
		if found {
			continue
		}

		if !wait(ctx, w.config.PollInterval) {
			return
		}
	}
}

func wait(ctx context.Context, duration time.Duration) bool {
	timer := time.NewTimer(duration)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}
