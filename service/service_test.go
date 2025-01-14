package service_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/getyourguide/extproc-go/filter"
	"github.com/getyourguide/extproc-go/server"
	"github.com/getyourguide/extproc-go/test"
	"github.com/getyourguide/extproc-go/test/containers/envoy"
	filtertest "github.com/getyourguide/extproc-go/test/filter"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"sigs.k8s.io/yaml"
)

var envoyProxy *envoy.TestContainer

func TestMain(m *testing.M) {
	ctx := context.Background()
	envoyProxy = envoy.NewTestContainer()
	if err := envoyProxy.Run(ctx, "istio/proxyv2:1.24.2"); err != nil {
		panic(err)
	}
	m.Run()
	if err := testcontainers.TerminateContainer(envoyProxy); err != nil {
		panic(err)
	}
}

func TestRequestHeaders(t *testing.T) {
	runServiceTest(t, "testdata/request_header_order.yml")
}

func TestResponseHeaders(t *testing.T) {
	runServiceTest(t, "testdata/response_header_order.yml")
}

type serviceTest struct {
	Filters   []filtertest.Configuration `json:"filters"`
	TestCases test.TestCases             `json:"tests"`
}

func runServiceTest(t *testing.T, fileName string) {
	f, err := os.ReadFile(fileName)
	require.NoError(t, err)

	var tt serviceTest
	err = yaml.Unmarshal(f, &tt)
	require.NoError(t, err)

	var filters []filter.Filter
	for _, cfg := range tt.Filters {
		filters = append(filters, &filtertest.Filter{
			Configuration: cfg,
		})
	}

	srv := server.New(context.Background(),
		server.WithEcho(),
		server.WithFilters(filters...),
	)
	go func() {
		require.NoError(t, srv.Serve())
	}()
	waitTimeout := 5 * time.Second

	err = server.WaitReady(srv, waitTimeout)
	require.NoError(t, err)

	tt.TestCases.Run(t,
		test.WithURL(envoyProxy.URL.String()),
	)
	require.NoError(t, srv.Stop())
}
