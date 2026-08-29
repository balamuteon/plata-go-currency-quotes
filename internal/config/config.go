// Package config loads and validates application configuration.
package config

import (
	"fmt"
	"net"
	"net/url"
	"time"
)

type Config struct {
	HTTP     HTTP
	Database Database
	Provider Provider
	Worker   Worker
}

type HTTP struct {
	Port            string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	ShutdownTimeout time.Duration
}

type Database struct {
	URL string
}

type Provider struct {
	BaseURL string
	Timeout time.Duration
}

type Worker struct {
	Count          int
	PollInterval   time.Duration
	LeaseDuration  time.Duration
	MaxAttempts    int
	RetryBaseDelay time.Duration
	RetryMaxDelay  time.Duration
}

func Load() (*Config, error) {
	httpConfig, err := loadHTTP()
	if err != nil {
		return nil, fmt.Errorf("HTTP config: %w", err)
	}
	databaseConfig, err := loadDatabase()
	if err != nil {
		return nil, fmt.Errorf("database config: %w", err)
	}
	providerConfig, err := loadProvider()
	if err != nil {
		return nil, fmt.Errorf("provider config: %w", err)
	}
	workerConfig, err := loadWorker()
	if err != nil {
		return nil, fmt.Errorf("worker config: %w", err)
	}
	if workerConfig.LeaseDuration <= providerConfig.Timeout {
		return nil, fmt.Errorf(
			"worker config: WORKER_LEASE_DURATION must exceed QUOTE_PROVIDER_TIMEOUT",
		)
	}

	return &Config{
		HTTP:     httpConfig,
		Database: databaseConfig,
		Provider: providerConfig,
		Worker:   workerConfig,
	}, nil
}

func loadHTTP() (HTTP, error) {
	var (
		cfg HTTP
		err error
	)

	if cfg.Port, err = portEnv("HTTP_PORT"); err != nil {
		return HTTP{}, err
	}

	if cfg.ReadTimeout, err = durationEnv("HTTP_READ_TIMEOUT"); err != nil {
		return HTTP{}, err
	}
	if cfg.WriteTimeout, err = durationEnv("HTTP_WRITE_TIMEOUT"); err != nil {
		return HTTP{}, err
	}
	if cfg.IdleTimeout, err = durationEnv("HTTP_IDLE_TIMEOUT"); err != nil {
		return HTTP{}, err
	}
	if cfg.ShutdownTimeout, err = durationEnv("SHUTDOWN_TIMEOUT"); err != nil {
		return HTTP{}, err
	}

	return cfg, nil
}

func loadDatabase() (Database, error) {
	host, err := stringEnv("POSTGRES_HOST")
	if err != nil {
		return Database{}, err
	}
	port, err := portEnv("POSTGRES_PORT")
	if err != nil {
		return Database{}, err
	}
	user, err := stringEnv("POSTGRES_USER")
	if err != nil {
		return Database{}, err
	}
	password, err := stringEnv("POSTGRES_PASSWORD")
	if err != nil {
		return Database{}, err
	}
	databaseName, err := stringEnv("POSTGRES_DB")
	if err != nil {
		return Database{}, err
	}
	sslMode, err := stringEnv("POSTGRES_SSLMODE")
	if err != nil {
		return Database{}, err
	}

	databaseURL := url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(user, password),
		Host:   net.JoinHostPort(host, port),
		Path:   databaseName,
	}
	query := databaseURL.Query()
	query.Set("sslmode", sslMode)
	databaseURL.RawQuery = query.Encode()

	return Database{URL: databaseURL.String()}, nil
}

func loadProvider() (Provider, error) {
	var (
		cfg Provider
		err error
	)

	if cfg.BaseURL, err = stringEnv("QUOTE_PROVIDER_BASE_URL"); err != nil {
		return Provider{}, err
	}

	providerURL, err := url.Parse(cfg.BaseURL)
	if err != nil || providerURL.Host == "" || (providerURL.Scheme != "http" && providerURL.Scheme != "https") {
		return Provider{}, fmt.Errorf("QUOTE_PROVIDER_BASE_URL must be an absolute HTTP(S) URL")
	}
	if cfg.Timeout, err = durationEnv("QUOTE_PROVIDER_TIMEOUT"); err != nil {
		return Provider{}, err
	}

	return cfg, nil
}

func loadWorker() (Worker, error) {
	var (
		cfg Worker
		err error
	)

	if cfg.Count, err = intEnv("WORKER_COUNT"); err != nil {
		return Worker{}, err
	}
	if cfg.PollInterval, err = durationEnv("WORKER_POLL_INTERVAL"); err != nil {
		return Worker{}, err
	}
	if cfg.LeaseDuration, err = durationEnv("WORKER_LEASE_DURATION"); err != nil {
		return Worker{}, err
	}
	if cfg.MaxAttempts, err = intEnv("WORKER_MAX_ATTEMPTS"); err != nil {
		return Worker{}, err
	}
	if cfg.RetryBaseDelay, err = durationEnv("WORKER_RETRY_BASE_DELAY"); err != nil {
		return Worker{}, err
	}
	if cfg.RetryMaxDelay, err = durationEnv("WORKER_RETRY_MAX_DELAY"); err != nil {
		return Worker{}, err
	}

	if cfg.Count <= 0 {
		return Worker{}, fmt.Errorf("WORKER_COUNT must be positive")
	}
	if cfg.MaxAttempts <= 0 {
		return Worker{}, fmt.Errorf("WORKER_MAX_ATTEMPTS must be positive")
	}
	if cfg.RetryMaxDelay < cfg.RetryBaseDelay {
		return Worker{}, fmt.Errorf("WORKER_RETRY_MAX_DELAY must not be less than WORKER_RETRY_BASE_DELAY")
	}

	return cfg, nil
}
