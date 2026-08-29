package quote_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/balamuteon/plata-go-currency-quotes/internal/domain"
	quotehandler "github.com/balamuteon/plata-go-currency-quotes/internal/handler/quote"
	"github.com/balamuteon/plata-go-currency-quotes/pkg/observability/logger"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestHandlerRequestUpdate(t *testing.T) {
	t.Parallel()

	createdAt := time.Date(2026, time.August, 28, 10, 0, 0, 0, time.UTC)
	update := domain.QuoteUpdate{
		ID:        "7e0f3931-9319-4c95-9ae8-029bf498264b",
		Pair:      domain.CurrencyPair{Base: domain.EUR, Quote: domain.MXN},
		Status:    domain.StatusQueued,
		CreatedAt: createdAt,
		UpdatedAt: createdAt,
	}

	tests := []struct {
		name                 string
		body                 string
		idempotencyKey       string
		mock                 func(*MockquoteService)
		expectedStatusCode   int
		expectedError        string
		expectedReplayHeader string
	}{
		{
			name:           "accepts update request",
			body:           `{"pair":"EUR/MXN"}`,
			idempotencyKey: "request-1",
			mock: func(service *MockquoteService) {
				service.EXPECT().
					RequestUpdate(gomock.Any(), "EUR/MXN", "request-1").
					Return(update, false, nil)
			},
			expectedStatusCode: http.StatusAccepted,
		},
		{
			name:           "marks idempotent replay",
			body:           `{"pair":"EUR/MXN"}`,
			idempotencyKey: "request-1",
			mock: func(service *MockquoteService) {
				service.EXPECT().
					RequestUpdate(gomock.Any(), "EUR/MXN", "request-1").
					Return(update, true, nil)
			},
			expectedStatusCode:   http.StatusAccepted,
			expectedReplayHeader: "true",
		},
		{
			name:               "rejects invalid JSON",
			body:               `{"pair":`,
			mock:               func(_ *MockquoteService) {},
			expectedStatusCode: http.StatusBadRequest,
			expectedError:      "invalid json",
		},
		{
			name:               "rejects too large body",
			body:               `{"pair":"` + strings.Repeat("A", 2000) + `"}`,
			mock:               func(_ *MockquoteService) {},
			expectedStatusCode: http.StatusRequestEntityTooLarge,
			expectedError:      "request body is too large",
		},
		{
			name:           "maps validation error",
			body:           `{"pair":"EUR/EUR"}`,
			idempotencyKey: "request-1",
			mock: func(service *MockquoteService) {
				service.EXPECT().
					RequestUpdate(gomock.Any(), "EUR/EUR", "request-1").
					Return(domain.QuoteUpdate{}, false, domain.ErrInvalidCurrencyPair)
			},
			expectedStatusCode: http.StatusBadRequest,
			expectedError:      "invalid currency pair",
		},
		{
			name:           "maps idempotency conflict",
			body:           `{"pair":"EUR/MXN"}`,
			idempotencyKey: "request-1",
			mock: func(service *MockquoteService) {
				service.EXPECT().
					RequestUpdate(gomock.Any(), "EUR/MXN", "request-1").
					Return(domain.QuoteUpdate{}, false, domain.ErrIdempotencyConflict)
			},
			expectedStatusCode: http.StatusConflict,
			expectedError:      "idempotency key is already used for another request",
		},
		{
			name:           "maps unexpected error",
			body:           `{"pair":"EUR/MXN"}`,
			idempotencyKey: "request-1",
			mock: func(service *MockquoteService) {
				service.EXPECT().
					RequestUpdate(gomock.Any(), "EUR/MXN", "request-1").
					Return(domain.QuoteUpdate{}, false, errors.New("database down"))
			},
			expectedStatusCode: http.StatusInternalServerError,
			expectedError:      "internal server error",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			service := NewMockquoteService(ctrl)
			test.mock(service)
			handler := quotehandler.NewHandler(service, logger.NewNoop())

			request := httptest.NewRequest(
				http.MethodPost,
				"/v1/quote-updates",
				bytes.NewBufferString(test.body),
			)
			if test.idempotencyKey != "" {
				request.Header.Set("Idempotency-Key", test.idempotencyKey)
			}
			response := httptest.NewRecorder()

			handler.RequestUpdate(response, request)

			require.Equal(t, test.expectedStatusCode, response.Code, response.Body.String())
			assert.Equal(t, test.expectedReplayHeader, response.Header().Get("Idempotency-Replayed"))
			if test.expectedError != "" {
				assertErrorResponse(t, response.Body.Bytes(), test.expectedError)
				return
			}
			assert.Equal(t, "/v1/quote-updates/"+update.ID, response.Header().Get("Location"))
			assertJSONFields(t, response.Body.Bytes(), map[string]any{
				"update_id":  update.ID,
				"pair":       "EUR/MXN",
				"status":     "queued",
				"created_at": createdAt.Format(time.RFC3339),
				"updated_at": createdAt.Format(time.RFC3339),
			})
		})
	}
}

func TestHandlerGetUpdate(t *testing.T) {
	t.Parallel()

	createdAt := time.Date(2026, time.August, 28, 10, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2026, time.August, 28, 10, 0, 1, 0, time.UTC)
	price := 21.75
	update := domain.QuoteUpdate{
		ID:        "7e0f3931-9319-4c95-9ae8-029bf498264b",
		Pair:      domain.CurrencyPair{Base: domain.EUR, Quote: domain.MXN},
		Status:    domain.StatusSucceeded,
		Price:     &price,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}

	tests := []struct {
		name               string
		updateID           string
		mock               func(*MockquoteService)
		expectedStatusCode int
		expectedError      string
	}{
		{
			name:     "returns update",
			updateID: update.ID,
			mock: func(service *MockquoteService) {
				service.EXPECT().GetUpdate(gomock.Any(), update.ID).Return(update, nil)
			},
			expectedStatusCode: http.StatusOK,
		},
		{
			name:     "maps invalid id",
			updateID: "bad-id",
			mock: func(service *MockquoteService) {
				service.EXPECT().GetUpdate(gomock.Any(), "bad-id").Return(domain.QuoteUpdate{}, domain.ErrInvalidUpdateID)
			},
			expectedStatusCode: http.StatusBadRequest,
			expectedError:      "invalid update ID",
		},
		{
			name:     "maps not found",
			updateID: update.ID,
			mock: func(service *MockquoteService) {
				service.EXPECT().GetUpdate(gomock.Any(), update.ID).Return(domain.QuoteUpdate{}, domain.ErrNotFound)
			},
			expectedStatusCode: http.StatusNotFound,
			expectedError:      "quote update was not found",
		},
		{
			name:     "maps unexpected error",
			updateID: update.ID,
			mock: func(service *MockquoteService) {
				service.EXPECT().GetUpdate(gomock.Any(), update.ID).Return(domain.QuoteUpdate{}, errors.New("database down"))
			},
			expectedStatusCode: http.StatusInternalServerError,
			expectedError:      "internal server error",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			service := NewMockquoteService(ctrl)
			test.mock(service)
			handler := quotehandler.NewHandler(service, logger.NewNoop())

			request := requestWithURLParam(
				httptest.NewRequest(http.MethodGet, "/v1/quote-updates/"+test.updateID, nil),
				"update_id",
				test.updateID,
			)
			response := httptest.NewRecorder()

			handler.GetUpdate(response, request)

			require.Equal(t, test.expectedStatusCode, response.Code, response.Body.String())
			if test.expectedError != "" {
				assertErrorResponse(t, response.Body.Bytes(), test.expectedError)
				return
			}
			assertJSONFields(t, response.Body.Bytes(), map[string]any{
				"update_id":  update.ID,
				"pair":       "EUR/MXN",
				"status":     "succeeded",
				"price":      price,
				"created_at": createdAt.Format(time.RFC3339),
				"updated_at": updatedAt.Format(time.RFC3339),
			})
		})
	}
}

func TestHandlerGetLatest(t *testing.T) {
	t.Parallel()

	updatedAt := time.Date(2026, time.August, 28, 10, 0, 1, 0, time.UTC)
	price := 21.75
	update := domain.QuoteUpdate{
		ID:        "7e0f3931-9319-4c95-9ae8-029bf498264b",
		Pair:      domain.CurrencyPair{Base: domain.EUR, Quote: domain.MXN},
		Status:    domain.StatusSucceeded,
		Price:     &price,
		UpdatedAt: updatedAt,
	}

	tests := []struct {
		name               string
		pair               string
		mock               func(*MockquoteService)
		expectedStatusCode int
		expectedError      string
	}{
		{
			name: "returns latest quote",
			pair: "EUR/MXN",
			mock: func(service *MockquoteService) {
				service.EXPECT().GetLatest(gomock.Any(), "EUR/MXN").Return(update, nil)
			},
			expectedStatusCode: http.StatusOK,
		},
		{
			name: "maps invalid pair",
			pair: "EUR/EUR",
			mock: func(service *MockquoteService) {
				service.EXPECT().GetLatest(gomock.Any(), "EUR/EUR").Return(domain.QuoteUpdate{}, domain.ErrInvalidCurrencyPair)
			},
			expectedStatusCode: http.StatusBadRequest,
			expectedError:      "invalid currency pair",
		},
		{
			name: "maps not found",
			pair: "EUR/MXN",
			mock: func(service *MockquoteService) {
				service.EXPECT().GetLatest(gomock.Any(), "EUR/MXN").Return(domain.QuoteUpdate{}, domain.ErrNotFound)
			},
			expectedStatusCode: http.StatusNotFound,
			expectedError:      "quote was not found",
		},
		{
			name: "maps successful quote without price as internal error",
			pair: "EUR/MXN",
			mock: func(service *MockquoteService) {
				withoutPrice := update
				withoutPrice.Price = nil
				service.EXPECT().GetLatest(gomock.Any(), "EUR/MXN").Return(withoutPrice, nil)
			},
			expectedStatusCode: http.StatusInternalServerError,
			expectedError:      "internal server error",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			service := NewMockquoteService(ctrl)
			test.mock(service)
			handler := quotehandler.NewHandler(service, logger.NewNoop())

			request := httptest.NewRequest(http.MethodGet, "/v1/quotes/latest?pair="+test.pair, nil)
			response := httptest.NewRecorder()

			handler.GetLatest(response, request)

			require.Equal(t, test.expectedStatusCode, response.Code, response.Body.String())
			if test.expectedError != "" {
				assertErrorResponse(t, response.Body.Bytes(), test.expectedError)
				return
			}
			assertJSONFields(t, response.Body.Bytes(), map[string]any{
				"update_id":  update.ID,
				"pair":       "EUR/MXN",
				"price":      price,
				"updated_at": updatedAt.Format(time.RFC3339),
			})
		})
	}
}

func requestWithURLParam(request *http.Request, key, value string) *http.Request {
	routeContext := chi.NewRouteContext()
	routeContext.URLParams.Add(key, value)

	return request.WithContext(context.WithValue(request.Context(), chi.RouteCtxKey, routeContext))
}

func assertErrorResponse(t *testing.T, data []byte, expectedError string) {
	t.Helper()

	assertJSONFields(t, data, map[string]any{"error": expectedError})
}

func assertJSONFields(t *testing.T, data []byte, expected map[string]any) {
	t.Helper()

	var actual map[string]any
	require.NoError(t, json.Unmarshal(data, &actual))

	for key, expectedValue := range expected {
		assert.Equal(t, expectedValue, actual[key], "field %q; full response = %#v", key, actual)
	}
}
