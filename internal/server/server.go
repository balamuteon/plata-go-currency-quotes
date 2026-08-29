// Package server creates the configured HTTP server.
package server

import (
	"net/http"

	"github.com/balamuteon/plata-go-currency-quotes/internal/config"
)

const maximumHeaderBytes = 1 << 20

func New(cfg config.HTTP, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           handler,
		ReadHeaderTimeout: cfg.ReadTimeout,
		ReadTimeout:       cfg.ReadTimeout,
		WriteTimeout:      cfg.WriteTimeout,
		IdleTimeout:       cfg.IdleTimeout,
		MaxHeaderBytes:    maximumHeaderBytes,
	}
}
