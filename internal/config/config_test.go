package config_test

import (
	"testing"
	"time"

	"github.com/balamuteon/plata-go-currency-quotes/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad(t *testing.T) {
	setValidEnv(t)

	cfg, err := config.Load()
	require.NoError(t, err)

	assert.Equal(t, "8080", cfg.HTTP.Port)
	assert.Equal(t, 2*time.Second, cfg.HTTP.ReadTimeout)
	assert.Equal(t, "postgres://quotes:quotes@localhost:5432/testdb?sslmode=disable", cfg.Database.URL)
	assert.Equal(t, "https://api.frankfurter.dev/v2", cfg.Provider.BaseURL)
	assert.Equal(t, time.Second, cfg.Provider.Timeout)
	assert.Equal(t, 2, cfg.Worker.Count)
	assert.Equal(t, 3, cfg.Worker.MaxAttempts)
}

func TestLoadErrors(t *testing.T) {
	tests := []struct {
		name   string
		change func(t *testing.T)
	}{
		{
			name: "missing HTTP port",
			change: func(t *testing.T) {
				t.Setenv("HTTP_PORT", "")
			},
		},
		{
			name: "invalid HTTP port",
			change: func(t *testing.T) {
				t.Setenv("HTTP_PORT", "70000")
			},
		},
		{
			name: "missing database host",
			change: func(t *testing.T) {
				t.Setenv("POSTGRES_HOST", "")
			},
		},
		{
			name: "invalid database port",
			change: func(t *testing.T) {
				t.Setenv("POSTGRES_PORT", "70000")
			},
		},
		{
			name: "invalid provider URL",
			change: func(t *testing.T) {
				t.Setenv("QUOTE_PROVIDER_BASE_URL", "localhost:8080")
			},
		},
		{
			name: "non positive worker count",
			change: func(t *testing.T) {
				t.Setenv("WORKER_COUNT", "0")
			},
		},
		{
			name: "retry max delay less than base delay",
			change: func(t *testing.T) {
				t.Setenv("WORKER_RETRY_BASE_DELAY", "2s")
				t.Setenv("WORKER_RETRY_MAX_DELAY", "1s")
			},
		},
		{
			name: "lease duration not greater than provider timeout",
			change: func(t *testing.T) {
				t.Setenv("QUOTE_PROVIDER_TIMEOUT", "5s")
				t.Setenv("WORKER_LEASE_DURATION", "5s")
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			setValidEnv(t)
			test.change(t)

			_, err := config.Load()
			require.Error(t, err)
		})
	}
}

func setValidEnv(t *testing.T) {
	t.Helper()

	t.Setenv("HTTP_PORT", "8080")
	t.Setenv("HTTP_READ_TIMEOUT", "2s")
	t.Setenv("HTTP_WRITE_TIMEOUT", "3s")
	t.Setenv("HTTP_IDLE_TIMEOUT", "30s")
	t.Setenv("SHUTDOWN_TIMEOUT", "5s")
	t.Setenv("POSTGRES_HOST", "localhost")
	t.Setenv("POSTGRES_PORT", "5432")
	t.Setenv("POSTGRES_USER", "quotes")
	t.Setenv("POSTGRES_PASSWORD", "quotes")
	t.Setenv("POSTGRES_DB", "testdb")
	t.Setenv("POSTGRES_SSLMODE", "disable")
	t.Setenv("QUOTE_PROVIDER_BASE_URL", "https://api.frankfurter.dev/v2")
	t.Setenv("QUOTE_PROVIDER_TIMEOUT", "1s")
	t.Setenv("WORKER_COUNT", "2")
	t.Setenv("WORKER_POLL_INTERVAL", "500ms")
	t.Setenv("WORKER_LEASE_DURATION", "10s")
	t.Setenv("WORKER_MAX_ATTEMPTS", "3")
	t.Setenv("WORKER_RETRY_BASE_DELAY", "1s")
	t.Setenv("WORKER_RETRY_MAX_DELAY", "10s")
}
