// Package observability provides HTTP observability middleware.
package observability

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/balamuteon/plata-go-currency-quotes/pkg/observability/logger"
	"github.com/go-chi/chi/v5/middleware"
)

// GetLogMiddleware returns HTTP request logging middleware.
func GetLogMiddleware(l logger.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			next.ServeHTTP(ww, r)

			l.Info("HTTP Request",
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Int("status", ww.Status()),
				slog.String("duration", time.Since(start).String()),
			)
		})
	}
}
