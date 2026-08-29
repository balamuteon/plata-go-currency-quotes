// Package quote contains HTTP handlers for quote operations.
package quote

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/balamuteon/plata-go-currency-quotes/internal/domain"
	utils "github.com/balamuteon/plata-go-currency-quotes/internal/handler"
	"github.com/balamuteon/plata-go-currency-quotes/pkg/observability/logger"
	"github.com/go-chi/chi/v5"
)

const maximumRequestBodyBytes int64 = 1 << 10

type Handler struct {
	service quoteService
	log     logger.Logger
}

func NewHandler(service quoteService, logger logger.Logger) *Handler {
	return &Handler{service: service, log: logger}
}

func (h *Handler) RequestUpdate(w http.ResponseWriter, r *http.Request) {
	var reqDTO requestUpdateDTO
	r.Body = http.MaxBytesReader(w, r.Body, maximumRequestBodyBytes)
	if err := json.NewDecoder(r.Body).Decode(&reqDTO); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			utils.NewErrorResponse(w, http.StatusRequestEntityTooLarge, ErrRequestBodyTooLarge)
			return
		}
		utils.NewErrorResponse(w, http.StatusBadRequest, ErrInvalidJSON)
		return
	}

	update, replayed, err := h.service.RequestUpdate(
		r.Context(),
		reqDTO.Pair,
		r.Header.Get("Idempotency-Key"),
	)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrInvalidCurrencyPair),
			errors.Is(err, domain.ErrInvalidIdempotencyKey):
			utils.NewErrorResponse(w, http.StatusBadRequest, err)
		case errors.Is(err, domain.ErrIdempotencyConflict):
			utils.NewErrorResponse(w, http.StatusConflict, err)
		default:
			h.log.Error("failed to request quote update", "error", err)
			utils.NewErrorResponse(w, http.StatusInternalServerError, ErrStatusInternalServerError)
		}
		return
	}

	w.Header().Set("Location", "/v1/quote-updates/"+update.ID)
	if replayed {
		w.Header().Set("Idempotency-Replayed", "true")
	}
	if err := utils.WriteJSON(w, http.StatusAccepted, updateFromDomain(update)); err != nil {
		h.log.Error("failed to write response (network error)", "error", err)
	}
}

func (h *Handler) GetUpdate(w http.ResponseWriter, r *http.Request) {
	update, err := h.service.GetUpdate(r.Context(), chi.URLParam(r, "update_id"))
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrInvalidUpdateID):
			utils.NewErrorResponse(w, http.StatusBadRequest, err)
		case errors.Is(err, domain.ErrNotFound):
			utils.NewErrorResponse(w, http.StatusNotFound, ErrQuoteUpdateNotFound)
		default:
			h.log.Error("failed to get quote update", "error", err)
			utils.NewErrorResponse(w, http.StatusInternalServerError, ErrStatusInternalServerError)
		}
		return
	}

	if err := utils.WriteJSON(w, http.StatusOK, updateFromDomain(update)); err != nil {
		h.log.Error("failed to write response (network error)", "error", err)
	}
}

func (h *Handler) GetLatest(w http.ResponseWriter, r *http.Request) {
	update, err := h.service.GetLatest(r.Context(), r.URL.Query().Get("pair"))
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrInvalidCurrencyPair):
			utils.NewErrorResponse(w, http.StatusBadRequest, err)
		case errors.Is(err, domain.ErrNotFound):
			utils.NewErrorResponse(w, http.StatusNotFound, ErrQuoteNotFound)
		default:
			h.log.Error("failed to get latest quote", "error", err)
			utils.NewErrorResponse(w, http.StatusInternalServerError, ErrStatusInternalServerError)
		}
		return
	}
	if update.Price == nil {
		h.log.Error("successful quote is missing result", "update_id", update.ID)
		utils.NewErrorResponse(w, http.StatusInternalServerError, ErrStatusInternalServerError)
		return
	}
	if err := utils.WriteJSON(w, http.StatusOK, latestFromDomain(update)); err != nil {
		h.log.Error("failed to write response (network error)", "error", err)
	}
}
