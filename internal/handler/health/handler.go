// Package health contains the service health check handler.
package health

import (
	"net/http"

	utils "github.com/balamuteon/plata-go-currency-quotes/internal/handler"
	"github.com/balamuteon/plata-go-currency-quotes/pkg/observability/logger"
)

type Handler struct {
	log logger.Logger
}

func NewHandler(log logger.Logger) *Handler {
	return &Handler{log: log}
}

// Ping отвечает на запросы пинга.
func (h *Handler) Ping(w http.ResponseWriter, _ *http.Request) {
	if err := utils.WriteJSON(w, http.StatusOK, map[string]string{
		"message": "pong",
	}); err != nil {
		h.log.Error("ERROR: failed to write response", "error", err)
	}
}

// HealthCheck отвечает на запросы проверки состояния сервиса.
func (h *Handler) HealthCheck(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusNoContent)
}
