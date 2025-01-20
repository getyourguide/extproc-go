package server

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"time"

	extproc "github.com/envoyproxy/go-control-plane/envoy/service/ext_proc/v3"
	"github.com/getyourguide/extproc-go/filter"
	"github.com/getyourguide/extproc-go/service"
	"github.com/getyourguide/extproc-go/test/echo"
	"google.golang.org/grpc"
)

const (
	defaultGrpcNetwork  = "tcp"
	defaultGrpcAddress  = ":8081"
	defaultHTTPBindAddr = ":8080"
	defaultShutdownWait = 5 * time.Second
)

type Server struct {
	serviceOpts []service.Option
	grpcServer  *grpc.Server
	grpcNetwork string
	grpcAddress string
	echoConfig  echoConfig
	ctx         context.Context
}

type echoConfig struct {
	enabled     bool
	bindAddress string
	mux         *http.ServeMux
	httpsrv     *http.Server
}

type Option func(*Server)

func New(ctx context.Context, opts ...Option) *Server {
	srv := &Server{
		ctx: ctx,
	}

	for _, opt := range opts {
		opt(srv)
	}

	if srv.grpcNetwork == "" {
		srv.grpcNetwork = defaultGrpcNetwork
	}
	if srv.grpcAddress == "" {
		srv.grpcAddress = defaultGrpcAddress
	}
	if srv.grpcServer == nil {
		srv.grpcServer = grpc.NewServer()
	}

	if srv.echoConfig.enabled {
		if srv.echoConfig.mux == nil {
			srv.echoConfig.mux = http.NewServeMux()
		}
		if srv.echoConfig.bindAddress == "" {
			srv.echoConfig.bindAddress = defaultHTTPBindAddr
		}

		srv.echoConfig.mux.HandleFunc("/headers", echo.RequestHeaders)
		srv.echoConfig.mux.HandleFunc("/response-headers", echo.ResponseHeaders)
		srv.echoConfig.httpsrv = &http.Server{
			Addr: srv.echoConfig.bindAddress,
		}
		srv.echoConfig.httpsrv.Handler = srv.echoConfig.mux

	}
	return srv
}

func WithFilters(f ...filter.Filter) Option {
	return func(s *Server) {
		s.serviceOpts = append(s.serviceOpts, service.WithFilters(f...))
	}
}

func WithGrpcServer(server *grpc.Server, network string, address string) Option {
	return func(s *Server) {
		s.grpcServer = server
		s.grpcNetwork = network
		s.grpcAddress = address
	}
}

func WithEcho() Option {
	return func(s *Server) {
		s.echoConfig.enabled = true
	}
}

func WithEchoServerMux(mux *http.ServeMux, address string) Option {
	return func(s *Server) {
		srv := http.Server{}
		srv.Handler = mux
		s.echoConfig.enabled = true
		s.echoConfig.mux = mux
		s.echoConfig.bindAddress = address
	}
}

func (s *Server) Serve() error {
	if s.ctx == nil {
		s.ctx = context.TODO()
	}

	errCh := make(chan error, 1)
	if s.echoConfig.enabled {
		go func() {
			slog.Info("starting http server", "address", s.echoConfig.bindAddress)
			errCh <- s.echoConfig.httpsrv.ListenAndServe()
		}()
	}

	go func() {
		if s.grpcNetwork == "unix" {
			os.RemoveAll(s.grpcAddress) // nolint:errcheck
		}
		listener, err := net.Listen(s.grpcNetwork, s.grpcAddress)
		if err != nil {
			errCh <- fmt.Errorf("cannot listen: %w", err)
			return
		}
		extprocService := service.New(s.serviceOpts...)
		extproc.RegisterExternalProcessorServer(s.grpcServer, extprocService)
		slog.Info("starting grpc server", "address", s.grpcAddress)
		errCh <- s.grpcServer.Serve(listener)
	}()

	select {
	case <-s.ctx.Done():
		return s.Stop()
	case err := <-errCh:
		return err
	}
}

func (s *Server) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if s.grpcServer != nil {
		slog.Info("stopping grpc server")
		s.grpcServer.GracefulStop()
	}
	if s.grpcNetwork == "unix" {
		os.RemoveAll(s.grpcAddress) // nolint:errcheck
	}
	if s.echoConfig.httpsrv != nil {
		slog.Info("stopping http server")
		if err := s.echoConfig.httpsrv.Shutdown(ctx); err != nil {
			return fmt.Errorf("http server shutdown error: %w", err)
		}
	}
	return nil
}

func IsReady(s *Server) bool {
	if s.echoConfig.enabled {
		req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("http://%s/headers", s.echoConfig.bindAddress), nil)
		if err != nil {
			return false
		}
		httpClient := http.Client{
			Timeout: 5 * time.Second,
		}
		res, err := httpClient.Do(req)
		if err != nil {
			return false
		}
		if res.StatusCode != http.StatusOK {
			return false
		}
	}
	return true
}

func WaitReady(s *Server, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	tck := time.NewTicker(500 * time.Millisecond)
	defer tck.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-tck.C:
			if IsReady(s) {
				return nil
			}
		}
	}
}
