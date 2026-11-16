package test_test

import (
	"context"
	"log"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/getyourguide/extproc-go/server"
	extproctest "github.com/getyourguide/extproc-go/test"
	"github.com/getyourguide/extproc-go/test/containers/envoy"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

func TestLoadRejectsUnknownFields(t *testing.T) {
	// Load fails via t.FailNow, so assert on it from a child process.
	if os.Getenv("EXTPROC_LOAD_CHILD") == "1" {
		extproctest.Load(t, "testdata/unknown_field.yml")
		return
	}
	if testing.Short() {
		t.Skip()
	}

	cmd := exec.Command(os.Args[0], "-test.run=^TestLoadRejectsUnknownFields$")
	cmd.Env = append(os.Environ(), "EXTPROC_LOAD_CHILD=1")
	out, err := cmd.CombinedOutput()
	require.Error(t, err, "Load should fail on unknown fields:\n%s", out)
	require.Contains(t, string(out), `unknown field "expect"`)
}

func TestIntegration(t *testing.T) {
	suite.Run(t, &IntegrationTestSuite{})
}

type IntegrationTestSuite struct {
	suite.Suite
	container *envoy.TestContainer
	url       string
	ctx       context.Context
}

func (suite *IntegrationTestSuite) SetupSuite() {
	suite.ctx = context.Background()
	suite.container = envoy.NewTestContainer()
	if err := suite.container.Run(suite.ctx, "istio/proxyv2:1.24.2"); err != nil {
		log.Fatal(err)
	}
	suite.url = suite.container.URL.String()
}

func (suite *IntegrationTestSuite) TearDownSuite() {
	if err := suite.container.Terminate(suite.ctx); err != nil {
		log.Fatalf("error terminating postgres container: %s", err)
	}
}

func (suite *IntegrationTestSuite) TestIntegrationTest() {
	t := suite.T()
	srv := server.New(context.Background(), server.WithEcho())
	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Serve()
	}()
	err := server.WaitReady(srv, 10*time.Second)
	require.NoError(t, err)

	templateData := struct {
		HeaderName  string
		HeaderValue string
	}{
		HeaderName:  "x-custom-header",
		HeaderValue: "value-1",
	}
	testcases := extproctest.LoadTemplate(t, "testdata/httptest.yml", templateData)
	require.NotEmpty(t, testcases)
	testcases.Run(t, extproctest.WithURL(suite.url))

	require.NoError(t, srv.Stop())
}
