package echo_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/getyourguide/extproc-go/httptest/echo"
	"github.com/stretchr/testify/require"
)

func TestRequestHeaders(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
	req.Header.Set("X-Test-Header", "test-value")
	rr := httptest.NewRecorder()

	echo.RequestHeaders(rr, req)

	require.Equal(t, http.StatusOK, rr.Code, "expected HTTP status OK")

	var resp echo.RequestHeaderResponse
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp), "failed to decode response body")

	expectedHeaders := map[string]string{
		"X-Test-Header": "test-value",
		"Host":          "example.com",
		"Method":        http.MethodGet,
	}
	for key, expectedValue := range expectedHeaders {
		require.Equal(t, expectedValue, resp.Headers[key], "mismatch for header %s", key)
	}
}

func TestResponseHeaders(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "http://example.com?status=201&X-Test-Response=test-value", nil)
	rr := httptest.NewRecorder()

	echo.ResponseHeaders(rr, req)

	require.Equal(t, http.StatusCreated, rr.Code, "expected HTTP status Created")
	require.Equal(t, "test-value", rr.Header().Get("X-Test-Response"), "mismatch for header X-Test-Response")

	var resp echo.ResponseHeaderResponse
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp), "failed to decode response body")
	require.Equal(t, "test-value", resp["X-Test-Response"], "mismatch for X-Test-Response in body")
}

func TestResponseHeadersInvalidStatus(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "http://example.com?status=invalid", nil)
	rr := httptest.NewRecorder()

	echo.ResponseHeaders(rr, req)

	require.Equal(t, http.StatusBadRequest, rr.Code, "expected HTTP status Bad Request")

	var resp echo.ErrorResponse
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&resp), "failed to decode response body")
	require.NotEmpty(t, resp.Error, "error message should not be empty")
}
