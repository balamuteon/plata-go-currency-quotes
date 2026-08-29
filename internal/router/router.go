// Package router configures HTTP routes and middleware.
package router

import (
	"net/http"

	healthhandler "github.com/balamuteon/plata-go-currency-quotes/internal/handler/health"
	quotehandler "github.com/balamuteon/plata-go-currency-quotes/internal/handler/quote"
	pkg "github.com/balamuteon/plata-go-currency-quotes/pkg/observability"
	"github.com/balamuteon/plata-go-currency-quotes/pkg/observability/logger"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func NewRouter(quoteHandler *quotehandler.Handler, healthHandler *healthhandler.Handler, l logger.Logger) http.Handler {
	router := chi.NewRouter()
	logMiddleware := pkg.GetLogMiddleware(l)

	router.Use(
		middleware.Recoverer,
		logMiddleware,
	)

	router.Route("/v1", func(router chi.Router) {
		router.Post("/quote-updates", quoteHandler.RequestUpdate)
		router.Get("/quote-updates/{update_id}", quoteHandler.GetUpdate)
		router.Get("/quotes/latest", quoteHandler.GetLatest)
	})

	// healthcheckers
	router.Get("/ping", healthHandler.Ping)
	router.Head("/healthcheck", healthHandler.HealthCheck)

	return router
}
