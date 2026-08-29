package quote

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/balamuteon/plata-go-currency-quotes/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const (
	// claimQuery atomically takes one due job from the queue.
	// New jobs are taken by next_attempt_at, expired processing jobs are reclaimed
	// by lease_until, and SKIP LOCKED lets concurrent workers avoid each other.
	claimQuery = `
		WITH candidate AS (
			SELECT id AS candidate_id
			FROM quote_updates
			WHERE (status = 'queued' AND next_attempt_at <= NOW())
			   OR (status = 'processing' AND lease_until <= NOW())
			ORDER BY
				CASE
					WHEN status = 'queued' THEN next_attempt_at
					ELSE lease_until
				END,
				created_at
			FOR UPDATE SKIP LOCKED
			LIMIT 1
		)
		UPDATE quote_updates AS u
		SET status = 'processing',
			attempts = attempts + 1,
			lease_until = NOW() + ($1::double precision * interval '1 second'),
			lease_token = $2::uuid,
			updated_at = NOW()
		FROM candidate
		WHERE u.id = candidate.candidate_id
		RETURNING ` + returningFields

	completeQuery = `
		UPDATE quote_updates
		SET status = 'succeeded',
			price = $1,
			lease_until = NULL,
			lease_token = NULL,
			last_error = NULL,
			updated_at = NOW()
		WHERE id = $2::uuid
		  AND status = 'processing'
		  AND lease_token = $3::uuid`

	retryQuery = `
		UPDATE quote_updates
		SET status = 'queued',
			next_attempt_at = $1,
			lease_until = NULL,
			lease_token = NULL,
			last_error = $2,
			updated_at = NOW()
		WHERE id = $3::uuid
		  AND status = 'processing'
		  AND lease_token = $4::uuid`

	failQuery = `
		UPDATE quote_updates
		SET status = 'failed',
			lease_until = NULL,
			lease_token = NULL,
			last_error = $1,
			updated_at = NOW()
		WHERE id = $2::uuid
		  AND status = 'processing'
		  AND lease_token = $3::uuid`
)

func (r *Repository) Claim(ctx context.Context, leaseDuration time.Duration) (domain.QuoteUpdate, bool, error) {
	leaseToken := uuid.NewString()

	update, err := scanQuoteUpdate(r.storage.QueryRow(ctx, claimQuery, leaseDuration.Seconds(), leaseToken))
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.QuoteUpdate{}, false, nil
	}
	if err != nil {
		return domain.QuoteUpdate{}, false, fmt.Errorf("claim quote update: %w", err)
	}
	return update, true, nil
}

func (r *Repository) Complete(
	ctx context.Context,
	id string,
	leaseToken string,
	quote domain.Quote,
) error {
	tag, err := r.storage.Exec(ctx, completeQuery, quote.Price, id, leaseToken)
	if err != nil {
		return fmt.Errorf("complete quote update: %w", err)
	}
	return ensureLeaseUpdated(tag)
}

func (r *Repository) Retry(
	ctx context.Context,
	id string,
	leaseToken string,
	nextAttemptAt time.Time,
	errorMessage string,
) error {
	tag, err := r.storage.Exec(ctx, retryQuery, nextAttemptAt, errorMessage, id, leaseToken)
	if err != nil {
		return fmt.Errorf("retry quote update: %w", err)
	}
	return ensureLeaseUpdated(tag)
}

func (r *Repository) Fail(ctx context.Context, id, leaseToken, errorMessage string) error {
	tag, err := r.storage.Exec(ctx, failQuery, errorMessage, id, leaseToken)
	if err != nil {
		return fmt.Errorf("fail quote update: %w", err)
	}
	return ensureLeaseUpdated(tag)
}

func ensureLeaseUpdated(tag pgconn.CommandTag) error {
	if tag.RowsAffected() == 0 {
		return domain.ErrLeaseLost
	}
	return nil
}
