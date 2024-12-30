package filters_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/getyourguide/extproc-go/examples/filters"
	"github.com/getyourguide/extproc-go/server"
	extproctest "github.com/getyourguide/extproc-go/test"
	"github.com/getyourguide/extproc-go/test/containers/envoy"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
)

var baseURL string

func TestMain(m *testing.M) {
	container := envoy.NewTestContainer()
	url, err := container.Run(context.Background(), "istio/proxyv2:1.24.2")
	defer testcontainers.TerminateContainer(container)
	if err != nil {
		fmt.Printf("could not start container: %v", err)
		os.Exit(1)
	}
	baseURL = url.String()

	code := m.Run()
	os.Exit(code)
}

func TestSameSiteLax(t *testing.T) {
	srv := server.New(context.Background(),
		server.WithEcho(),
		server.WithFilters(&filters.SameSiteLaxMode{}))
	go func() {
		require.NoError(t, srv.Serve())
	}()
	defer srv.Stop()
	server.WaitReady(srv, 5*time.Second)

	tc := extproctest.Load(t, "testdata/setcookie.yml")
	tc.Run(t, extproctest.WithURL(baseURL))
}
