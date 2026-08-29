package domain

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Currency string

const (
	USD Currency = "USD"
	EUR Currency = "EUR"
	MXN Currency = "MXN"
)

var supportedCurrencies = map[Currency]struct{}{
	USD: {},
	EUR: {},
	MXN: {},
}

type CurrencyPair struct {
	Base  Currency
	Quote Currency
}

func ParseCurrencyPair(value string) (CurrencyPair, error) {
	parts := strings.Split(strings.ToUpper(strings.TrimSpace(value)), "/")
	if len(parts) != 2 {
		return CurrencyPair{}, fmt.Errorf("%w: expected format BASE/QUOTE", ErrInvalidCurrencyPair)
	}

	base := Currency(parts[0])
	quote := Currency(parts[1])
	if _, ok := supportedCurrencies[base]; !ok {
		return CurrencyPair{}, fmt.Errorf("%w: currency %q is not supported", ErrInvalidCurrencyPair, base)
	}
	if _, ok := supportedCurrencies[quote]; !ok {
		return CurrencyPair{}, fmt.Errorf("%w: currency %q is not supported", ErrInvalidCurrencyPair, quote)
	}
	if base == quote {
		return CurrencyPair{}, fmt.Errorf("%w: base and quote currencies must differ", ErrInvalidCurrencyPair)
	}

	return CurrencyPair{Base: base, Quote: quote}, nil
}

func (p CurrencyPair) String() string {
	return string(p.Base) + "/" + string(p.Quote)
}

func NewUpdateID() string {
	return uuid.NewString()
}

func ValidateUpdateID(value string) error {
	if _, err := uuid.Parse(value); err != nil {
		return fmt.Errorf("%w: must be a UUID", ErrInvalidUpdateID)
	}
	return nil
}

func NormalizeIdempotencyKey(value string) (string, error) {
	key := strings.TrimSpace(value)
	if key == "" {
		return "", fmt.Errorf("%w: header is required", ErrInvalidIdempotencyKey)
	}
	if len(key) > 128 {
		return "", fmt.Errorf("%w: must not exceed 128 bytes", ErrInvalidIdempotencyKey)
	}
	for _, r := range key {
		if r < 0x21 || r > 0x7e {
			return "", fmt.Errorf("%w: must contain only printable ASCII characters without spaces", ErrInvalidIdempotencyKey)
		}
	}
	return key, nil
}

type UpdateStatus string

const (
	StatusQueued     UpdateStatus = "queued"
	StatusProcessing UpdateStatus = "processing"
	StatusSucceeded  UpdateStatus = "succeeded"
	StatusFailed     UpdateStatus = "failed"
)

func ParseUpdateStatus(value string) (UpdateStatus, error) {
	status := UpdateStatus(value)
	switch status {
	case StatusQueued, StatusProcessing, StatusSucceeded, StatusFailed:
		return status, nil
	default:
		return "", fmt.Errorf("%w: %q", ErrInvalidUpdateStatus, value)
	}
}

type QuoteUpdate struct {
	ID           string
	Pair         CurrencyPair
	Status       UpdateStatus
	Price        *float64
	Attempts     int
	CreatedAt    time.Time
	UpdatedAt    time.Time
	LeaseToken   string
	ErrorMessage string
}

type Quote struct {
	Price float64
}
