package test_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/getyourguide/extproc-go/server"
	extproctest "github.com/getyourguide/extproc-go/test"
	"github.com/getyourguide/extproc-go/test/containers/envoy"
	"github.com/testcontainers/testcontainers-go"

	"github.com/stretchr/testify/require"
)

var baseURL string

func TestMain(t *testing.M) {
	container := envoy.NewTestContainer()
	url, err := container.Run(context.Background(), "istio/proxyv2:1.24.2")
	defer testcontainers.TerminateContainer(container)
	if err != nil {
		fmt.Printf("could not start container: %v", err)
		os.Exit(1)
	}
	baseURL = url.String()

	srv := server.New(
		context.Background(),
		server.WithEcho(),
	)
	code := t.Run()
	srv.Stop()

	os.Exit(code)
}

func TestIntegrationTest(t *testing.T) {
	templateData := struct {
		HeaderName  string
		HeaderValue string
	}{
		HeaderName:  "x-custom-header",
		HeaderValue: "value-1",
	}
	testcases := extproctest.LoadTemplate(t, "testdata/httptest.yml", templateData)
	require.NotEmpty(t, testcases)
	testcases.Run(t, extproctest.WithURL(baseURL))
}
