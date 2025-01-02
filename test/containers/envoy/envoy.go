package envoy

import (
	"bytes"
	"context"
	"fmt"
	"net/url"

	_ "embed"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

//go:embed envoy.yml
var config []byte

type Container struct {
	testcontainers.Container
}

func Run(ctx context.Context, img string, opts ...testcontainers.ContainerCustomizer) (*Container, error) {
	req := testcontainers.ContainerRequest{
		Image: img,
	}

	genericContainerReq := testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	}

	for _, opt := range opts {
		if err := opt.Customize(&genericContainerReq); err != nil {
			return nil, fmt.Errorf("customize: %w", err)
		}
	}

	container, err := testcontainers.GenericContainer(ctx, genericContainerReq)
	ctr := &Container{
		Container: container,
	}
	if err != nil {
		return ctr, fmt.Errorf("could not create generic container: %w", err)
	}
	return ctr, nil
}

type TestContainer struct {
	testcontainers.Container
	overrides    testcontainers.GenericContainerRequest
	waitStrategy wait.Strategy
}

func NewTestContainer(opts ...TestContainerOption) *TestContainer {
	c := &TestContainer{}
	for _, opt := range opts {
		opt(c)
	}

	if len(c.overrides.Files) == 0 {
		opts = append(opts, WithFiles(testcontainers.ContainerFile{
			ContainerFilePath: "/etc/envoy/envoy.yml",
			Reader:            bytes.NewReader(config),
		}))
	}

	if len(c.overrides.Entrypoint) == 0 {
		entrypoint := []string{"/usr/local/bin/envoy", "--log-level warn", "-c", "/etc/envoy/envoy.yml"}
		opts = append(opts, WithEntrypoint(entrypoint...))
	}

	if len(c.overrides.ExposedPorts) == 0 {
		opts = append(opts, WithExposedPorts("10000"))
	}

	if len(c.overrides.HostAccessPorts) == 0 {
		opts = append(opts, WithHostAccessPorts(8080, 8081))
	}

	if len(c.overrides.ExtraHosts) == 0 {
		opts = append(opts, WithExtraHosts(fmt.Sprintf("%s:host-gateway", testcontainers.HostInternal)))
	}

	if c.waitStrategy == nil {
		opts = append(opts, WithWaitStrategy(wait.ForExposedPort()))
	}

	for _, opt := range opts {
		opt(c)
	}
	return c
}

type TestContainerOption func(*TestContainer)

func WithFiles(files ...testcontainers.ContainerFile) TestContainerOption {
	return func(c *TestContainer) {
		c.overrides.Files = append(c.overrides.Files, files...)
	}
}

func WithEntrypoint(entrypoint ...string) TestContainerOption {
	return func(c *TestContainer) {
		c.overrides.Entrypoint = entrypoint
	}
}

func WithExposedPorts(ports ...string) TestContainerOption {
	return func(c *TestContainer) {
		c.overrides.ExposedPorts = ports
	}
}

func WithHostAccessPorts(ports ...int) TestContainerOption {
	return func(c *TestContainer) {
		c.overrides.HostAccessPorts = ports
	}
}

func WithExtraHosts(hosts ...string) TestContainerOption {
	return func(c *TestContainer) {
		c.overrides.ExtraHosts = hosts
	}
}

func WithWaitStrategy(strategy wait.Strategy) TestContainerOption {
	return func(c *TestContainer) {
		c.waitStrategy = strategy
	}
}

func (c *TestContainer) Run(ctx context.Context, img string, opts ...testcontainers.ContainerCustomizer) (*url.URL, error) {
	for _, opt := range opts {
		if err := opt.Customize(&c.overrides); err != nil {
			return nil, fmt.Errorf("customize: %w", err)
		}
	}

	ctr, err := Run(ctx, img, testcontainers.CustomizeRequest(c.overrides))
	c.Container = ctr
	if err != nil {
		return nil, fmt.Errorf("could not run container: %w", err)
	}

	if err := c.waitStrategy.WaitUntilReady(ctx, ctr); err != nil {
		return nil, fmt.Errorf("container not ready: %w", err)
	}

	hostIP, err := ctr.Host(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not get host ip: %w", err)
	}

	mappedPort, err := ctr.MappedPort(ctx, "10000")
	if err != nil {
		return nil, fmt.Errorf("could not get mapped port: %w", err)
	}

	rawURL := fmt.Sprintf("http://%s:%s", hostIP, mappedPort.Port())
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("could not parse url: %w", err)
	}

	return u, nil
}
