package health_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	healthhandler "github.com/balamuteon/plata-go-currency-quotes/internal/handler/health"
	"github.com/balamuteon/plata-go-currency-quotes/pkg/observability/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandlerPing(t *testing.T) {
	t.Parallel()

	handler := healthhandler.NewHandler(logger.NewNoop())
	request := httptest.NewRequest(http.MethodGet, "/ping", nil)
	response := httptest.NewRecorder()

	handler.Ping(response, request)

	require.Equal(t, http.StatusOK, response.Code)

	var body map[string]string
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &body))
	assert.Equal(t, "pong", body["message"])
}

func TestHandlerHealthCheck(t *testing.T) {
	t.Parallel()

	handler := healthhandler.NewHandler(logger.NewNoop())
	request := httptest.NewRequest(http.MethodHead, "/healthcheck", nil)
	response := httptest.NewRecorder()

	handler.HealthCheck(response, request)

	require.Equal(t, http.StatusNoContent, response.Code)
	assert.Empty(t, response.Body.String())
}
