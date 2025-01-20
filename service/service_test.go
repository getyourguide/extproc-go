package service_test

import (
	"context"
	"log"
	"os"
	"testing"
	"time"

	"github.com/getyourguide/extproc-go/filter"
	"github.com/getyourguide/extproc-go/server"
	extproctest "github.com/getyourguide/extproc-go/test"
	"github.com/getyourguide/extproc-go/test/containers/envoy"
	filtertest "github.com/getyourguide/extproc-go/test/filter"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"sigs.k8s.io/yaml"
)

func TestService(t *testing.T) {
	suite.Run(t, &ServiceTestSuite{})
}

type ServiceTestSuite struct {
	suite.Suite
	container *envoy.TestContainer
	url       string
	ctx       context.Context
}

func (suite *ServiceTestSuite) SetupSuite() {
	suite.ctx = context.Background()
	suite.container = envoy.NewTestContainer()
	if err := suite.container.Run(suite.ctx, "istio/proxyv2:1.24.2"); err != nil {
		log.Fatal(err)
	}
	suite.url = suite.container.URL.String()
}

func (suite *ServiceTestSuite) TearDownSuite() {
	if err := suite.container.Terminate(suite.ctx); err != nil {
		log.Fatalf("error terminating postgres container: %s", err)
	}
}

func (suite *ServiceTestSuite) TestRequestHeaders() {
	suite.Run("testdata/request_header_order.yml")
}

func (suite *ServiceTestSuite) TestResponseHeaders() {
	suite.Run("testdata/response_header_order.yml")
}

func (suite *ServiceTestSuite) Run(fileName string) {
	t := suite.T()
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

	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Serve()
	}()

	err = server.WaitReady(srv, 15*time.Second)
	require.NoError(t, err)

	tt.TestCases.Run(t,
		extproctest.WithURL(suite.url),
	)
	require.NoError(t, srv.Stop())
	err = <-errCh
	require.NoError(t, err)
}

type serviceTest struct {
	Filters   []filtertest.Configuration `json:"filters"`
	TestCases extproctest.TestCases      `json:"tests"`
}
