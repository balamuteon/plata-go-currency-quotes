package domain

import "errors"

var (
	ErrInvalidCurrencyPair   = errors.New("invalid currency pair")
	ErrInvalidUpdateID       = errors.New("invalid update ID")
	ErrInvalidIdempotencyKey = errors.New("invalid idempotency key")
	ErrInvalidUpdateStatus   = errors.New("invalid update status")
	ErrNotFound              = errors.New("quote update not found")
	ErrIdempotencyConflict   = errors.New("idempotency key is already used for another request")
	ErrLeaseLost             = errors.New("quote update lease is no longer owned")
)
