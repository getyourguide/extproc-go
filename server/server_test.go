package server_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/getyourguide/extproc-go/filter"
	"github.com/getyourguide/extproc-go/server"
	"github.com/getyourguide/extproc-go/test/echo"
	"github.com/stretchr/testify/require"
)

func TestServer(t *testing.T) {
	t.Run("Serve with basic configuration", func(t *testing.T) {
		ctx, shutdown := context.WithCancel(context.Background())
		srv := server.New(ctx)
		errCh := make(chan error, 1)
		go func() {
			errCh <- srv.Serve()
		}()
		err := server.WaitReady(srv, 10*time.Second)
		require.NoError(t, err)

		shutdown()
		err = <-errCh
		require.NoError(t, err)
	})

	t.Run("Serve with echo", func(t *testing.T) {
		srv := server.New(context.Background(),
			server.WithEcho(),
			server.WithFilters(&filter.NoOpFilter{}),
		)
		errCh := make(chan error, 1)
		go func() {
			errCh <- srv.Serve()
		}()
		err := server.WaitReady(srv, 10*time.Second)
		require.NoError(t, err)

		httpClient := http.Client{
			Timeout: time.Second,
		}
		echoURL := "http://:8080/headers"
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
		require.NoError(t, srv.Stop())
		err = <-errCh
		require.NoError(t, err)
	})
}
