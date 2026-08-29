package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/balamuteon/plata-go-currency-quotes/internal/app"
	"github.com/balamuteon/plata-go-currency-quotes/internal/config"
	frankfurtergateway "github.com/balamuteon/plata-go-currency-quotes/internal/gateway/frankfurter"
	healthhandler "github.com/balamuteon/plata-go-currency-quotes/internal/handler/health"
	quotehandler "github.com/balamuteon/plata-go-currency-quotes/internal/handler/quote"
	quoterepository "github.com/balamuteon/plata-go-currency-quotes/internal/repository/quote"
	"github.com/balamuteon/plata-go-currency-quotes/internal/router"
	"github.com/balamuteon/plata-go-currency-quotes/internal/server"
	quoteservice "github.com/balamuteon/plata-go-currency-quotes/internal/service/quote"
	quoteworker "github.com/balamuteon/plata-go-currency-quotes/internal/worker/quote"
	"github.com/balamuteon/plata-go-currency-quotes/pkg/db"
	"github.com/balamuteon/plata-go-currency-quotes/pkg/observability/logger"
)

func main() {
	logger := setupLogger()

	if err := run(logger); err != nil {
		logger.Error("service terminated", "error", err)
		os.Exit(1)
	}
}

func setupLogger() logger.Logger {
	slogHandler := slog.NewJSONHandler(os.Stdout, nil)
	certainLogger := slog.New(slogHandler)
	slog.SetDefault(certainLogger)
	return logger.NewSlog(certainLogger)
}

func run(logger logger.Logger) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	databasePool, err := db.NewPostgresPool(ctx, cfg.Database.URL, logger)
	if err != nil {
		return err
	}
	defer databasePool.Close()

	quoteRepository := quoterepository.NewRepository(databasePool)
	quoteService := quoteservice.NewService(quoteRepository)
	quoteHandler := quotehandler.NewHandler(quoteService, logger)
	healthHandler := healthhandler.NewHandler(logger)

	httpRouter := router.NewRouter(quoteHandler, healthHandler, logger)
	httpServer := server.New(cfg.HTTP, httpRouter)

	quoteProvider := frankfurtergateway.NewClient(cfg.Provider.BaseURL, cfg.Provider.Timeout)
	quoteProcessor := quoteservice.NewProcessor(quoteRepository, quoteProvider, cfg.Worker, logger)
	quoteWorker := quoteworker.New(quoteProcessor, cfg.Worker, logger)

	application := app.New(httpServer, cfg.HTTP.ShutdownTimeout, logger, quoteWorker)
	return application.Run(ctx)
}
