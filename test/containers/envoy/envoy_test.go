package envoy_test

import (
	"context"
	"testing"

	"github.com/getyourguide/extproc-go/test/containers/envoy"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
)

func TestRunContainer(t *testing.T) {
	container := envoy.NewTestContainer()
	u, err := container.Run(context.Background(), "istio/proxyv2:1.24.2")
	defer testcontainers.CleanupContainer(t, container)

	require.NoError(t, err)
	require.Contains(t, u.String(), "http://localhost:")
}
