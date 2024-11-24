package main

import (
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"

	extproc "github.com/envoyproxy/go-control-plane/envoy/service/ext_proc/v3"
	"github.com/getyourguide/extproc-go/examples/filters"
	"github.com/getyourguide/extproc-go/httptest/echo"
	"github.com/getyourguide/extproc-go/service"
	"google.golang.org/grpc"
)

func main() {
	if err := run(); err != nil {
		slog.Error("oops", "error", err)
		os.Exit(1)
	}
}

func run() error {
	server := grpc.NewServer()
	extprocService := service.New(service.WithFilters(
		&filters.BasicAuth{},
		&filters.SameSiteLaxMode{},
	))
	extproc.RegisterExternalProcessorServer(server, extprocService)

	// For testing purposes, we also start an HTTP server that echoes headers back to the client.
	httpsrv := http.NewServeMux()
	httpsrv.HandleFunc("/headers", echo.RequestHeaders)
	httpsrv.HandleFunc("/response-headers", echo.ResponseHeaders)

	errCh := make(chan error, 1)
	go func() {
		slog.Info("listening", "addr", ":8080", "type", "http")
		errCh <- http.ListenAndServe(":8080", httpsrv)
	}()

	go func() {
		udsAddr := "/var/run/extproc-go/extproc-go.sock"
		os.RemoveAll(udsAddr) // nolint:errcheck

		listener, err := net.Listen("unix", udsAddr)
		if err != nil {
			errCh <- fmt.Errorf("cannot listen: %w", err)
		}
		slog.Info("listening", "addr", listener.Addr().String(), "type", "grpc")
		errCh <- server.Serve(listener)
	}()
	return <-errCh
}
