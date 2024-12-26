package server_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/getyourguide/extproc-go/filter"
	"github.com/getyourguide/extproc-go/httptest/echo"
	"github.com/getyourguide/extproc-go/server"
	"github.com/stretchr/testify/require"
)

func TestServer(t *testing.T) {
	t.Run("Serve with basic configuration", func(t *testing.T) {
		ctx, shutdown := context.WithCancel(context.Background())

		errCh := make(chan error, 1)
		go func() {
			errCh <- server.New(ctx).Serve()
		}()

		waitFor := 5 * time.Second
		require.Eventually(t, func() bool {
			_, err := os.Stat("/tmp/extproc.sock")
			return err == nil
		}, waitFor, time.Millisecond)

		shutdown()
		err := <-errCh
		require.NoError(t, err)
	})

	t.Run("Serve with echo", func(t *testing.T) {
		ctx, shutdown := context.WithCancel(context.Background())

		errCh := make(chan error, 1)
		go func() {
			errCh <- server.New(ctx).
				WithFilters(&filter.NoOpFilter{}).
				WithEcho().
				Serve()
		}()

		waitFor := 5 * time.Second
		require.Eventually(t, func() bool {
			_, err := os.Stat("/tmp/extproc.sock")
			return err == nil
		}, waitFor, time.Millisecond)

		httpClient := http.Client{
			Timeout: time.Second,
		}
		echoURL := fmt.Sprintf("http://:8080/headers")
		req, err := http.NewRequest(http.MethodGet, echoURL, nil)
		require.NoError(t, err)
		req.Header.Set("X-Test-Header", "test-value")

		res, err := httpClient.Do(req)
		require.NoError(t, err)

		require.Equal(t, http.StatusOK, res.StatusCode)

		var resp echo.RequestHeaderResponse
		require.NoError(t, json.NewDecoder(res.Body).Decode(&resp))

		expectedHeaders := map[string]string{
			"X-Test-Header": "test-value",
			"Method":        http.MethodGet,
		}
		for key, expectedValue := range expectedHeaders {
			require.Equal(t, expectedValue, resp.Headers[key], "mismatch for header %s", key)
		}

		shutdown()

		err = <-errCh
		require.NoError(t, err)
	})
}
