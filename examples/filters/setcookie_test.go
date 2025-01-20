package filters_test

import (
	"context"
	"log"
	"testing"
	"time"

	"github.com/getyourguide/extproc-go/examples/filters"
	"github.com/getyourguide/extproc-go/server"
	extproctest "github.com/getyourguide/extproc-go/test"
	"github.com/getyourguide/extproc-go/test/containers/envoy"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

func TestFilters(t *testing.T) {
	suite.Run(t, &FiltersTestSuite{})
}

type FiltersTestSuite struct {
	suite.Suite
	container *envoy.TestContainer
	url       string
	ctx       context.Context
}

func (suite *FiltersTestSuite) SetupSuite() {
	suite.ctx = context.Background()
	suite.container = envoy.NewTestContainer()
	if err := suite.container.Run(suite.ctx, "istio/proxyv2:1.24.2"); err != nil {
		log.Fatal(err)
	}
	suite.url = suite.container.URL.String()
}

func (suite *FiltersTestSuite) TearDownSuite() {
	if err := suite.container.Terminate(suite.ctx); err != nil {
		log.Fatalf("error terminating postgres container: %s", err)
	}
}

func (suite *FiltersTestSuite) TestSameSiteLaxMode() {
	t := suite.T()
	srv := server.New(context.Background(),
		server.WithEcho(),
		server.WithFilters(&filters.SameSiteLaxMode{}),
	)

	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Serve()
	}()
	err := server.WaitReady(srv, 10*time.Second)
	require.NoError(t, err)

	tc := extproctest.Load(t, "testdata/setcookie.yml")
	tc.Run(t, extproctest.WithURL(suite.url))

	require.NoError(t, srv.Stop())
	err = <-errCh
	require.NoError(t, err)
}
