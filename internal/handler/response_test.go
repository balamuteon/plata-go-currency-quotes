package handler_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/balamuteon/plata-go-currency-quotes/internal/handler"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteJSON(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		response := httptest.NewRecorder()

		err := handler.WriteJSON(response, http.StatusOK, map[string]string{"status": "ok"})

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, response.Code)
		assert.Equal(t, "application/json; charset=utf-8", response.Header().Get("Content-Type"))
		assertJSONEqual(t, response.Body.Bytes(), map[string]string{"status": "ok"})
	})

	t.Run("marshal error", func(t *testing.T) {
		t.Parallel()

		response := httptest.NewRecorder()

		err := handler.WriteJSON(response, http.StatusOK, make(chan int))

		require.Error(t, err)
		assert.Equal(t, http.StatusInternalServerError, response.Code)
		assertJSONEqual(t, response.Body.Bytes(), map[string]string{"error": "internal server error"})
	})
}

func TestNewErrorResponse(t *testing.T) {
	t.Parallel()

	response := httptest.NewRecorder()
	handler.NewErrorResponse(response, http.StatusBadRequest, errors.New("something went wrong"))

	assert.Equal(t, http.StatusBadRequest, response.Code)
	assertJSONEqual(t, response.Body.Bytes(), map[string]string{"error": "something went wrong"})
}

func assertJSONEqual(t *testing.T, data []byte, expected map[string]string) {
	t.Helper()

	var actual map[string]string
	require.NoError(t, json.Unmarshal(data, &actual))
	assert.Equal(t, expected, actual)
}
