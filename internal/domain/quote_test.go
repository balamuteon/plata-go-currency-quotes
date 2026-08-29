package domain_test

import (
	"testing"

	"github.com/balamuteon/plata-go-currency-quotes/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseCurrencyPair(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		value     string
		want      domain.CurrencyPair
		wantError bool
	}{
		{name: "normalizes input", value: " eur/mxn ", want: domain.CurrencyPair{Base: domain.EUR, Quote: domain.MXN}},
		{name: "rejects unsupported currency", value: "EUR/GBP", wantError: true},
		{name: "rejects equal currencies", value: "USD/USD", wantError: true},
		{name: "rejects malformed pair", value: "EUR-MXN", wantError: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got, err := domain.ParseCurrencyPair(test.value)
			if test.wantError {
				require.ErrorIs(t, err, domain.ErrInvalidCurrencyPair)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, test.want, got)
		})
	}
}

func TestNormalizeIdempotencyKey(t *testing.T) {
	t.Parallel()

	key, err := domain.NormalizeIdempotencyKey(" request-42 ")
	require.NoError(t, err)
	assert.Equal(t, "request-42", key)

	_, err = domain.NormalizeIdempotencyKey("contains space")
	require.ErrorIs(t, err, domain.ErrInvalidIdempotencyKey)

	_, err = domain.NormalizeIdempotencyKey("ключ")
	require.ErrorIs(t, err, domain.ErrInvalidIdempotencyKey)

	_, err = domain.NormalizeIdempotencyKey("   ")
	require.ErrorIs(t, err, domain.ErrInvalidIdempotencyKey)
}

func TestParseUpdateStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		value   string
		want    domain.UpdateStatus
		wantErr bool
	}{
		{name: "queued", value: "queued", want: domain.StatusQueued},
		{name: "processing", value: "processing", want: domain.StatusProcessing},
		{name: "succeeded", value: "succeeded", want: domain.StatusSucceeded},
		{name: "failed", value: "failed", want: domain.StatusFailed},
		{name: "unknown", value: "unknown", wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got, err := domain.ParseUpdateStatus(test.value)
			if test.wantErr {
				require.ErrorIs(t, err, domain.ErrInvalidUpdateStatus)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, test.want, got)
		})
	}
}

func TestValidateUpdateID(t *testing.T) {
	t.Parallel()

	require.NoError(t, domain.ValidateUpdateID("7e0f3931-9319-4c95-9ae8-029bf498264b"))

	require.ErrorIs(t, domain.ValidateUpdateID("not-a-uuid"), domain.ErrInvalidUpdateID)
}

func TestCurrencyPairString(t *testing.T) {
	t.Parallel()

	pair := domain.CurrencyPair{Base: domain.EUR, Quote: domain.MXN}
	assert.Equal(t, "EUR/MXN", pair.String())
}

func TestNewUpdateID(t *testing.T) {
	t.Parallel()

	id := domain.NewUpdateID()
	require.NoError(t, domain.ValidateUpdateID(id))
}
