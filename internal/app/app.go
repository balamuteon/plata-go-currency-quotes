// Package app manages service lifecycle and graceful shutdown.
package app

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/balamuteon/plata-go-currency-quotes/pkg/observability/logger"
)

type workerRunner interface {
	Run(ctx context.Context)
}

type App struct {
	server          *http.Server
	shutdownTimeout time.Duration
	logger          logger.Logger
	workers         []workerRunner
}

func New(
	server *http.Server,
	shutdownTimeout time.Duration,
	logger logger.Logger,
	workers ...workerRunner,
) *App {
	return &App{
		server:          server,
		shutdownTimeout: shutdownTimeout,
		logger:          logger,
		workers:         workers,
	}
}

func (a *App) Run(ctx context.Context) error {
	if a.server == nil {
		return errors.New("HTTP server is not initialized")
	}

	workerContext, stopWorkers := context.WithCancel(ctx)
	var wg sync.WaitGroup
	wg.Add(len(a.workers))
	for _, backgroundWorker := range a.workers {
		go func() {
			defer wg.Done()
			backgroundWorker.Run(workerContext)
		}()
	}

	serverError := make(chan error, 1)
	go func() {
		a.logger.Info("HTTP server started", "address", a.server.Addr)
		serverError <- a.server.ListenAndServe()
	}()

	var runError error
	select {
	case <-ctx.Done():
	case err := <-serverError:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			runError = fmt.Errorf("serve HTTP: %w", err)
		}
	}

	stopWorkers()
	shutdownContext, cancelShutdown := context.WithTimeout(context.Background(), a.shutdownTimeout)
	defer cancelShutdown()

	if err := a.server.Shutdown(shutdownContext); err != nil && runError == nil {
		runError = fmt.Errorf("shut down HTTP server: %w", err)
	}

	workerDone := make(chan struct{})
	go func() {
		wg.Wait()
		close(workerDone)
	}()
	select {
	case <-workerDone:
	case <-shutdownContext.Done():
		if runError == nil {
			runError = errors.New("workers did not stop before shutdown timeout")
		}
	}

	a.logger.Info("service stopped")
	return runError
}
