package quote_test

import (
	"context"
	"sync"
	"time"

	"github.com/balamuteon/plata-go-currency-quotes/internal/domain"
	"github.com/google/uuid"
)

func (s *RepositorySuite) TestEnqueue() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pair := domain.CurrencyPair{Base: domain.EUR, Quote: domain.MXN}
	updateID := uuid.NewString()
	idempotencyKey := uuid.NewString()

	created, replayed, err := s.repository.Enqueue(ctx, updateID, pair, idempotencyKey)
	s.Require().NoError(err)

	s.False(replayed)
	s.Equal(updateID, created.ID)
	s.Equal(pair, created.Pair)
	s.Equal(domain.StatusQueued, created.Status)
	s.Nil(created.Price)
	s.Zero(created.Attempts)
	s.NotZero(created.CreatedAt)
	s.NotZero(created.UpdatedAt)

	replayedUpdate, replayed, err := s.repository.Enqueue(ctx, uuid.NewString(), pair, idempotencyKey)
	s.Require().NoError(err)

	s.True(replayed)
	s.Equal(created.ID, replayedUpdate.ID)
	s.Equal(created.Pair, replayedUpdate.Pair)

	conflictingPair := domain.CurrencyPair{Base: domain.USD, Quote: domain.MXN}
	_, _, err = s.repository.Enqueue(ctx, uuid.NewString(), conflictingPair, idempotencyKey)
	s.ErrorIs(err, domain.ErrIdempotencyConflict)
}

func (s *RepositorySuite) TestFindByID() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pair := domain.CurrencyPair{Base: domain.EUR, Quote: domain.MXN}
	created := s.enqueueQuoteUpdate(ctx, pair)

	found, err := s.repository.FindByID(ctx, created.ID)
	s.Require().NoError(err)

	s.Equal(created.ID, found.ID)
	s.Equal(pair, found.Pair)
	s.Equal(domain.StatusQueued, found.Status)

	_, err = s.repository.FindByID(ctx, uuid.NewString())
	s.ErrorIs(err, domain.ErrNotFound)
}

func (s *RepositorySuite) TestClaimCompleteAndFindLatest() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pair := domain.CurrencyPair{Base: domain.EUR, Quote: domain.MXN}
	first := s.enqueueQuoteUpdate(ctx, pair)
	second := s.enqueueQuoteUpdate(ctx, pair)

	firstClaimed := s.claimQuoteUpdate(ctx)
	s.Equal(first.ID, firstClaimed.ID)

	err := s.repository.Complete(ctx, firstClaimed.ID, uuid.NewString(), domain.Quote{Price: 19.77})
	s.ErrorIs(err, domain.ErrLeaseLost)

	err = s.repository.Complete(ctx, firstClaimed.ID, firstClaimed.LeaseToken, domain.Quote{Price: 19.77})
	s.Require().NoError(err)

	time.Sleep(10 * time.Millisecond)

	secondClaimed := s.claimQuoteUpdate(ctx)
	s.Equal(second.ID, secondClaimed.ID)

	err = s.repository.Complete(ctx, secondClaimed.ID, secondClaimed.LeaseToken, domain.Quote{Price: 21.75})
	s.Require().NoError(err)

	completed, err := s.repository.FindByID(ctx, first.ID)
	s.Require().NoError(err)

	s.Require().NotNil(completed.Price)
	s.Equal(domain.StatusSucceeded, completed.Status)
	s.InDelta(19.77, *completed.Price, 0.000001)
	s.Empty(completed.LeaseToken)
	s.Empty(completed.ErrorMessage)
	s.False(completed.UpdatedAt.Before(firstClaimed.UpdatedAt))

	latest, err := s.repository.FindLatest(ctx, pair)
	s.Require().NoError(err)

	s.Require().NotNil(latest.Price)
	s.Equal(second.ID, latest.ID)
	s.InDelta(21.75, *latest.Price, 0.000001)
}

func (s *RepositorySuite) TestRetry() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pair := domain.CurrencyPair{Base: domain.EUR, Quote: domain.MXN}
	update := s.enqueueQuoteUpdate(ctx, pair)
	claimed := s.claimQuoteUpdate(ctx)
	s.Equal(update.ID, claimed.ID)

	err := s.repository.Retry(
		ctx,
		claimed.ID,
		uuid.NewString(),
		time.Now().UTC().Add(time.Second),
		"provider is not available",
	)
	s.ErrorIs(err, domain.ErrLeaseLost)

	nextAttemptAt := time.Now().UTC().Add(time.Hour)
	err = s.repository.Retry(ctx, claimed.ID, claimed.LeaseToken, nextAttemptAt, "provider is not available")
	s.Require().NoError(err)

	stored, err := s.repository.FindByID(ctx, update.ID)
	s.Require().NoError(err)

	s.Equal(domain.StatusQueued, stored.Status)
	s.Equal("provider is not available", stored.ErrorMessage)
	s.Empty(stored.LeaseToken)

	_, found, err := s.repository.Claim(ctx, time.Minute)
	s.Require().NoError(err)
	s.False(found)
}

func (s *RepositorySuite) TestPastDueRetryCanBeClaimedAgain() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pair := domain.CurrencyPair{Base: domain.EUR, Quote: domain.MXN}
	update := s.enqueueQuoteUpdate(ctx, pair)
	claimed := s.claimQuoteUpdate(ctx)
	s.Equal(update.ID, claimed.ID)

	err := s.repository.Retry(
		ctx,
		claimed.ID,
		claimed.LeaseToken,
		time.Now().UTC().Add(-time.Second),
		"temporary provider error",
	)
	s.Require().NoError(err)

	reclaimed := s.claimQuoteUpdate(ctx)

	s.Equal(update.ID, reclaimed.ID)
	s.Equal(2, reclaimed.Attempts)
	s.NotEqual(claimed.LeaseToken, reclaimed.LeaseToken)
}

func (s *RepositorySuite) TestFail() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pair := domain.CurrencyPair{Base: domain.EUR, Quote: domain.MXN}
	update := s.enqueueQuoteUpdate(ctx, pair)
	claimed := s.claimQuoteUpdate(ctx)
	s.Equal(update.ID, claimed.ID)

	err := s.repository.Fail(ctx, claimed.ID, uuid.NewString(), "provider is not available")
	s.ErrorIs(err, domain.ErrLeaseLost)

	err = s.repository.Fail(ctx, claimed.ID, claimed.LeaseToken, "provider is not available")
	s.Require().NoError(err)

	stored, err := s.repository.FindByID(ctx, update.ID)
	s.Require().NoError(err)

	s.Equal(domain.StatusFailed, stored.Status)
	s.Equal("provider is not available", stored.ErrorMessage)
	s.Empty(stored.LeaseToken)

	_, found, err := s.repository.Claim(ctx, time.Minute)
	s.Require().NoError(err)
	s.False(found)

	_, err = s.repository.FindLatest(ctx, pair)
	s.ErrorIs(err, domain.ErrNotFound)
}

func (s *RepositorySuite) TestExpiredLeaseCanBeReclaimed() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pair := domain.CurrencyPair{Base: domain.EUR, Quote: domain.MXN}
	update := s.enqueueQuoteUpdate(ctx, pair)

	claimed, found, err := s.repository.Claim(ctx, 30*time.Millisecond)
	s.Require().NoError(err)
	s.Require().True(found)
	s.Equal(update.ID, claimed.ID)

	_, found, err = s.repository.Claim(ctx, time.Minute)
	s.Require().NoError(err)
	s.False(found)

	var reclaimed domain.QuoteUpdate
	s.Require().Eventually(func() bool {
		var claimErr error
		reclaimed, found, claimErr = s.repository.Claim(ctx, time.Minute)
		return claimErr == nil && found && reclaimed.ID == update.ID
	}, 2*time.Second, 20*time.Millisecond)

	s.Equal(2, reclaimed.Attempts)
	s.NotEqual(claimed.LeaseToken, reclaimed.LeaseToken)
}

func (s *RepositorySuite) TestConcurrentClaimsDoNotReturnSameUpdate() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pair := domain.CurrencyPair{Base: domain.EUR, Quote: domain.MXN}
	first := s.enqueueQuoteUpdate(ctx, pair)
	second := s.enqueueQuoteUpdate(ctx, pair)

	start := make(chan struct{})
	results := make(chan claimResult, 2)

	var wg sync.WaitGroup
	wg.Add(2)
	for range 2 {
		go func() {
			defer wg.Done()
			<-start

			claimed, found, err := s.repository.Claim(ctx, time.Minute)
			results <- claimResult{update: claimed, found: found, err: err}
		}()
	}

	close(start)
	wg.Wait()
	close(results)

	claimedIDs := make(map[string]struct{}, 2)
	for result := range results {
		s.Require().NoError(result.err)
		s.Require().True(result.found)
		claimedIDs[result.update.ID] = struct{}{}
	}

	s.Contains(claimedIDs, first.ID)
	s.Contains(claimedIDs, second.ID)
	s.Len(claimedIDs, 2)
}

type claimResult struct {
	update domain.QuoteUpdate
	found  bool
	err    error
}

func (s *RepositorySuite) enqueueQuoteUpdate(ctx context.Context, pair domain.CurrencyPair) domain.QuoteUpdate {
	update, replayed, err := s.repository.Enqueue(ctx, uuid.NewString(), pair, uuid.NewString())
	s.Require().NoError(err)
	s.Require().False(replayed)
	s.Equal(domain.StatusQueued, update.Status)

	return update
}

func (s *RepositorySuite) claimQuoteUpdate(ctx context.Context) domain.QuoteUpdate {
	update, found, err := s.repository.Claim(ctx, time.Minute)
	s.Require().NoError(err)
	s.Require().True(found)
	s.Equal(domain.StatusProcessing, update.Status)
	s.NotEmpty(update.LeaseToken)

	return update
}
