package quote

import (
	"context"
	"errors"
	"time"

	"github.com/balamuteon/plata-go-currency-quotes/internal/config"
	"github.com/balamuteon/plata-go-currency-quotes/internal/domain"
	"github.com/balamuteon/plata-go-currency-quotes/pkg/observability/logger"
)

type Processor struct {
	queue    queue
	provider provider
	config   config.Worker
	logger   logger.Logger
}

func NewProcessor(queue queue, provider provider, cfg config.Worker, logger logger.Logger) *Processor {
	return &Processor{
		queue:    queue,
		provider: provider,
		config:   cfg,
		logger:   logger,
	}
}

func (p *Processor) ProcessNext(ctx context.Context) (bool, error) {
	update, found, err := p.queue.Claim(ctx, p.config.LeaseDuration)
	if err != nil {
		return false, err
	}
	if !found {
		return false, nil
	}

	p.process(ctx, update)
	return true, nil
}

func (p *Processor) process(ctx context.Context, update domain.QuoteUpdate) {
	log := p.logger.With(
		"update_id", update.ID,
		"pair", update.Pair.String(),
		"attempt", update.Attempts,
	)

	quote, err := p.provider.Fetch(ctx, update.Pair)
	if err == nil {
		if err = p.queue.Complete(ctx, update.ID, update.LeaseToken, quote); err != nil {
			logStateTransitionError(log, "complete", err)
			return
		}
		log.Info("quote update completed", "price", quote.Price)
		return
	}

	if ctx.Err() != nil {
		// The lease will expire and the job will be reclaimed after a restart.
		return
	}

	if update.Attempts >= p.config.MaxAttempts {
		if stateErr := p.queue.Fail(ctx, update.ID, update.LeaseToken, err.Error()); stateErr != nil {
			logStateTransitionError(log, "fail", stateErr)
			return
		}
		log.Error("quote update exhausted retries", "error", err)
		return
	}

	delay := retryDelay(p.config.RetryBaseDelay, p.config.RetryMaxDelay, update.Attempts)
	nextAttemptAt := time.Now().UTC().Add(delay)
	if stateErr := p.queue.Retry(ctx, update.ID, update.LeaseToken, nextAttemptAt, err.Error()); stateErr != nil {
		logStateTransitionError(log, "retry", stateErr)
		return
	}
	log.Warn("quote provider request failed; update scheduled for retry",
		"error", err,
		"retry_in", delay,
	)
}

func logStateTransitionError(logger logger.Logger, operation string, err error) {
	if errors.Is(err, domain.ErrLeaseLost) {
		logger.Warn("quote update lease was lost", "operation", operation)
		return
	}
	logger.Error("failed to persist quote update state", "operation", operation, "error", err)
}

func retryDelay(base, maximum time.Duration, attempt int) time.Duration {
	delay := base
	for current := 1; current < attempt; current++ {
		if delay >= maximum/2 {
			return maximum
		}
		delay *= 2
	}
	if delay > maximum {
		return maximum
	}
	return delay
}
