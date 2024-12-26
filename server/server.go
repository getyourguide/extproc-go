package server

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"time"

	extproc "github.com/envoyproxy/go-control-plane/envoy/service/ext_proc/v3"
	"github.com/getyourguide/extproc-go/filter"
	"github.com/getyourguide/extproc-go/httptest/echo"
	"github.com/getyourguide/extproc-go/service"
	"github.com/go-logr/logr"
	"google.golang.org/grpc"
)

const (
	defaultGrpcNetwork  = "unix"
	defaultGrpcAddress  = "/tmp/extproc.sock"
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
	log         logr.Logger
}

type echoConfig struct {
	enabled     bool
	bindAddress string
	mux         *http.ServeMux
	httpsrv     *http.Server
}

func New(ctx context.Context) *Server {
	return &Server{
		ctx: ctx,
	}
}

func (s *Server) WithFilters(f ...filter.Filter) *Server {
	s.serviceOpts = append(s.serviceOpts, service.WithFilters(f...))
	return s
}

func (s *Server) WithGrpcServer(server *grpc.Server, network string, address string) *Server {
	s.grpcServer = server
	s.grpcNetwork = network
	s.grpcAddress = address
	return s
}

func (s *Server) WithEcho() *Server {
	s.echoConfig.enabled = true
	return s
}

func (s *Server) WithEchoServerMux(mux *http.ServeMux, address string) *Server {
	srv := http.Server{}
	srv.Handler = mux
	s.echoConfig.enabled = true
	s.echoConfig.mux = mux
	s.echoConfig.bindAddress = address
	return s
}

func (s *Server) Serve() error {
	if s.ctx == nil {
		s.ctx = context.TODO()
	}

	errCh := make(chan error, 1)
	if s.echoConfig.enabled {
		if s.echoConfig.mux == nil {
			s.echoConfig.mux = http.NewServeMux()
		}
		if s.echoConfig.bindAddress == "" {
			s.echoConfig.bindAddress = defaultHTTPBindAddr
		}

		s.echoConfig.mux.HandleFunc("/headers", echo.RequestHeaders)
		s.echoConfig.mux.HandleFunc("/response-headers", echo.ResponseHeaders)
		go func() {
			s.echoConfig.httpsrv = &http.Server{
				Addr: s.echoConfig.bindAddress,
			}
			s.echoConfig.httpsrv.Handler = s.echoConfig.mux
			errCh <- s.echoConfig.httpsrv.ListenAndServe()
		}()
	}

	go func() {
		if s.grpcAddress == "" {
			s.grpcAddress = defaultGrpcAddress
		}
		if s.grpcNetwork == "" {
			s.grpcNetwork = defaultGrpcNetwork
		}
		if s.grpcNetwork == "unix" {
			os.RemoveAll(s.grpcAddress) // nolint:errcheck
		}
		listener, err := net.Listen(s.grpcNetwork, s.grpcAddress)
		if err != nil {
			errCh <- fmt.Errorf("cannot listen: %w", err)
			return
		}
		if s.grpcServer == nil {
			s.grpcServer = grpc.NewServer()
		}
		extprocService := service.New(s.serviceOpts...)
		extproc.RegisterExternalProcessorServer(s.grpcServer, extprocService)

		errCh <- s.grpcServer.Serve(listener)
	}()

	select {
	case <-s.ctx.Done():
		s.grpcServer.GracefulStop()
		if s.grpcNetwork == "unix" {
			os.RemoveAll(s.grpcAddress) // nolint:errcheck
		}
		if s.echoConfig.httpsrv == nil {
			return nil
		}
		ctx, cancel := context.WithTimeout(context.Background(), defaultShutdownWait)
		s.echoConfig.httpsrv.Shutdown(ctx)
		cancel()
	case err := <-errCh:
		return err
	}
	return nil
}
