package frankfurter_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/balamuteon/plata-go-currency-quotes/internal/domain"
	"github.com/balamuteon/plata-go-currency-quotes/internal/gateway/frankfurter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClientFetch(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "/v2/rate/EUR/MXN", request.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"date":"2026-08-25","base":"EUR","quote":"MXN","rate":21.75}`))
	}))
	t.Cleanup(server.Close)

	client := frankfurter.NewClient(server.URL+"/v2", time.Second)

	quote, err := client.Fetch(context.Background(), domain.CurrencyPair{Base: domain.EUR, Quote: domain.MXN})
	require.NoError(t, err)
	assert.Equal(t, 21.75, quote.Price)
}

func TestClientFetchErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		statusCode int
		body       string
	}{
		{
			name:       "provider returns non OK status",
			statusCode: http.StatusTooManyRequests,
			body:       `{"error":"rate limit"}`,
		},
		{
			name:       "provider returns malformed JSON",
			statusCode: http.StatusOK,
			body:       `{bad json`,
		},
		{
			name:       "provider returns unexpected pair",
			statusCode: http.StatusOK,
			body:       `{"base":"USD","quote":"MXN","rate":21.75}`,
		},
		{
			name:       "provider returns invalid rate",
			statusCode: http.StatusOK,
			body:       `{"base":"EUR","quote":"MXN","rate":0}`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(test.statusCode)
				_, _ = w.Write([]byte(test.body))
			}))
			t.Cleanup(server.Close)

			client := frankfurter.NewClient(server.URL, time.Second)

			_, err := client.Fetch(context.Background(), domain.CurrencyPair{Base: domain.EUR, Quote: domain.MXN})
			require.Error(t, err)
		})
	}
}
