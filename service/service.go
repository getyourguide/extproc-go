package service

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"

	extproc "github.com/envoyproxy/go-control-plane/envoy/service/ext_proc/v3"
	"github.com/getyourguide/extproc-go/filter"
	"github.com/go-logr/logr"
	"go.opentelemetry.io/otel/codes"
	grpcodes "google.golang.org/grpc/codes"

	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
	"google.golang.org/grpc/status"
)

const (
	TraceMessageOperationName = "grpc.message"
)

var (
	ProcessResourceName          = "Process"
	RequestHeadersResourceName   = "RequestHeaders"
	RequestBodyResourceName      = "RequestBody"
	RequestTrailersResourceName  = "RequestTrailers"
	ResponseHeadersResourceName  = "ResponseHeaders"
	ResponseBodyResourceName     = "ResponseBody"
	ResponseTrailersResourceName = "ResponseTrailers"
)

type ExtProcessor struct {
	filters         []filter.Filter
	streamCallbacks []filter.Stream
	log             logr.Logger
	tracer          trace.Tracer
}

var _ extproc.ExternalProcessorServer = &ExtProcessor{}

func New(options ...Option) *ExtProcessor {
	f := &ExtProcessor{}
	for _, opt := range options {
		opt.apply(f)
	}
	if f.tracer == nil {
		f.tracer = noop.NewTracerProvider().Tracer(TraceMessageOperationName)
	}

	return f
}

// Process is the main entry point for the ExternalProcessor service.
// The protocol itself is based on a bidirectional gRPC stream. Envoy will send the server ProcessingRequest messages, and the server must reply with ProcessingResponse.
// https://www.envoyproxy.io/docs/envoy/latest/api-v3/extensions/filters/http/ext_proc/v3/ext_proc.proto#envoy-v3-api-msg-extensions-filters-http-ext-proc-v3-externalfilter
func (svc *ExtProcessor) Process(procsrv extproc.ExternalProcessor_ProcessServer) error {
	req := filter.NewRequestContext()
	ctx := procsrv.Context()
	if len(svc.streamCallbacks) > 0 {
		defer func() {
			for _, s := range svc.streamCallbacks {
				if err := s.OnStreamComplete(req); err != nil {
					slog.Error(fmt.Sprintf("%T.OnStreamComplete returned an error", s), "err", err.Error())
				}
			}
		}()
	}

	for {
		procreq, err := procsrv.Recv()
		if err != nil {
			return IgnoreCanceled(err)
		}

		ctx := logr.NewContext(ctx, svc.log)
		switch msg := procreq.Request.(type) {
		case *extproc.ProcessingRequest_RequestHeaders:
			ctx, span := svc.tracer.Start(ctx, RequestHeadersResourceName)
			if err := svc.requestHeadersMessage(ctx, req, msg, procsrv); err != nil {
				span.End()
				return IgnoreCanceled(err)
			}
			span.End()
		case *extproc.ProcessingRequest_RequestBody:
			ctx, span := svc.tracer.Start(ctx, RequestBodyResourceName)
			if err := svc.requestBodyMessage(ctx, req, msg, procsrv); err != nil {
				span.End()
				return IgnoreCanceled(err)
			}
			span.End()
		case *extproc.ProcessingRequest_RequestTrailers:
			ctx, span := svc.tracer.Start(ctx, RequestTrailersResourceName)
			if err := svc.requestTrailersMessage(ctx, req, msg, procsrv); err != nil {
				span.End()
				return IgnoreCanceled(err)
			}
			span.End()
		case *extproc.ProcessingRequest_ResponseHeaders:
			ctx, span := svc.tracer.Start(ctx, ResponseHeadersResourceName)
			if err := svc.responseHeadersMessage(ctx, req, msg, procsrv); err != nil {
				span.End()
				return IgnoreCanceled(err)
			}
			span.End()
		case *extproc.ProcessingRequest_ResponseBody:
			ctx, span := svc.tracer.Start(ctx, ResponseBodyResourceName)
			if err := svc.responseBodyMessage(ctx, req, msg, procsrv); err != nil {
				span.End()
				return IgnoreCanceled(err)
			}
			span.End()
		case *extproc.ProcessingRequest_ResponseTrailers:
			ctx, span := svc.tracer.Start(ctx, ResponseTrailersResourceName)
			if err := svc.responseTrailersMessage(ctx, req, msg, procsrv); err != nil {
				span.End()
				return IgnoreCanceled(err)
			}
			span.End()
		default:
			return fmt.Errorf("unknown request type: %T", procreq.Request)
		}
	}
}

// Step 1. Request headers: Contains the headers from the original HTTP request.
func (svc *ExtProcessor) requestHeadersMessage(ctx context.Context, req *filter.RequestContext, msg *extproc.ProcessingRequest_RequestHeaders, procsrv extproc.ExternalProcessor_ProcessServer) error {
	for _, header := range msg.RequestHeaders.GetHeaders().GetHeaders() {
		headerValue := cmp.Or(string(header.GetRawValue()), header.GetValue())
		req.RequestHeaders.Add(header.Key, headerValue)
	}
	crw := filter.NewCommonResponseWriter(req.RequestHeaders)

	for _, f := range svc.filters {
		select {
		case <-ctx.Done():
			return nil
		default:
		}
		resourceName := fmt.Sprintf("%T/RequestHeaders", f)
		ctx, span := svc.tracer.Start(ctx, resourceName)
		// span.AddAttributes(trace.StringAttribute("filter", fmt.Sprintf("%T", f)))

		immediateResponse, err := f.RequestHeaders(ctx, crw, req)
		if err != nil {
			span.SetStatus(codes.Error, err.Error())
			span.End()
			return fmt.Errorf("RequestHeaders: failed running filter %T: %w", f, err)
		}
		if immediateResponse != nil {
			span.End()
			return procsrv.Send(&extproc.ProcessingResponse{
				Response: immediateResponse,
			})
		}
		if err := crw.CommonResponse().Validate(); err != nil {
			span.End()
			return fmt.Errorf("RequestHeaders: failed validating response in filter %T: %w", f, err)
		}
		span.End()
	}
	r := &extproc.ProcessingResponse{
		Response: &extproc.ProcessingResponse_RequestHeaders{
			RequestHeaders: &extproc.HeadersResponse{
				Response: crw.CommonResponse(),
			},
		},
	}
	if err := r.ValidateAll(); err != nil {
		return fmt.Errorf("RequestHeaders: failed validating response in filter: %w", err)
	}

	if err := procsrv.Send(r); err != nil {
		return fmt.Errorf("RequestHeaders: failed sending response: %w", err)
	}
	return nil
}

// Step 2. (Not implemented) Request body: Delivered if they are present and sent in a single message if the BUFFERED or BUFFERED_PARTIAL mode is chosen, in multiple messages if the STREAMED mode is chosen, and not at all otherwise.
func (svc *ExtProcessor) requestBodyMessage(_ context.Context, _ *filter.RequestContext, _ *extproc.ProcessingRequest_RequestBody, procsrv extproc.ExternalProcessor_ProcessServer) error {
	r := &extproc.ProcessingResponse{
		Response: &extproc.ProcessingResponse_RequestBody{},
	}
	if err := procsrv.Send(r); err != nil {
		return fmt.Errorf("RequestBody: failed sending response: %w", err)
	}
	return nil
}

// Step 3. (Not implemented) Request trailers: Delivered if they are present and if the trailer mode is set to SEND.
func (svc *ExtProcessor) requestTrailersMessage(_ context.Context, _ *filter.RequestContext, _ *extproc.ProcessingRequest_RequestTrailers, procsrv extproc.ExternalProcessor_ProcessServer) error {
	r := &extproc.ProcessingResponse{
		Response: &extproc.ProcessingResponse_RequestTrailers{},
	}
	if err := procsrv.Send(r); err != nil {
		return fmt.Errorf("RequestTrailers: failed sending response: %w", err)
	}
	return nil
}

// Step 4. Response headers: Contains the headers from the HTTP response. Keep in mind that if the upstream system sends them before processing the request body that this message may arrive before the complete body.
func (svc *ExtProcessor) responseHeadersMessage(ctx context.Context, req *filter.RequestContext, msg *extproc.ProcessingRequest_ResponseHeaders, procsrv extproc.ExternalProcessor_ProcessServer) error {
	for _, header := range msg.ResponseHeaders.GetHeaders().GetHeaders() {
		headerValue := cmp.Or(string(header.GetRawValue()), header.GetValue())
		req.ResponseHeaders.Add(header.Key, headerValue)
	}
	crw := filter.NewCommonResponseWriter(req.ResponseHeaders)

	for i := len(svc.filters) - 1; i >= 0; i-- {
		f := svc.filters[i]
		select {
		case <-ctx.Done():
			return nil
		default:
		}
		resourceName := fmt.Sprintf("%T/ResponseHeaders", f)
		ctx, span := svc.tracer.Start(ctx, resourceName)
		// span.AddAttributes(trace.StringAttribute("filter", fmt.Sprintf("%T", f)))

		immediateResponse, err := f.ResponseHeaders(ctx, crw, req)
		if err != nil {
			span.SetStatus(codes.Error, err.Error())
			span.End()
			return fmt.Errorf("ResponseHeaders: failed running filter %T: %w", f, err)
		}
		if immediateResponse != nil {
			span.End()
			return procsrv.Send(&extproc.ProcessingResponse{
				Response: immediateResponse,
			})
		}
		if err := crw.CommonResponse().Validate(); err != nil {
			span.SetStatus(codes.Error, err.Error())
			span.End()
			return fmt.Errorf("ResponseHeaders: failed validating response in filter %T: %w", f, err)
		}
		span.End()
	}
	r := &extproc.ProcessingResponse{
		Response: &extproc.ProcessingResponse_ResponseHeaders{
			ResponseHeaders: &extproc.HeadersResponse{
				Response: crw.CommonResponse(),
			},
		},
	}
	if err := r.ValidateAll(); err != nil {
		return fmt.Errorf("ResponseHeaders: failed validating response: %w", err)
	}
	if err := procsrv.Send(r); err != nil {
		return fmt.Errorf("ResponseHeaders: failed sending response: %w", err)
	}
	return nil
}

// Step 5. (Not implemented) Response body: Sent according to the processing mode like the request body.
func (svc *ExtProcessor) responseBodyMessage(_ context.Context, _ *filter.RequestContext, _ *extproc.ProcessingRequest_ResponseBody, procsrv extproc.ExternalProcessor_ProcessServer) error {
	r := &extproc.ProcessingResponse{
		Response: &extproc.ProcessingResponse_ResponseBody{},
	}
	if err := procsrv.Send(r); err != nil {
		return fmt.Errorf("ResponseBody: failed sending response: %w", err)
	}
	return nil
}

// Step 6. (Not implemented) Response trailers: Delivered according to the processing mode like the request trailers.
func (svc *ExtProcessor) responseTrailersMessage(_ context.Context, _ *filter.RequestContext, _ *extproc.ProcessingRequest_ResponseTrailers, procsrv extproc.ExternalProcessor_ProcessServer) error {
	r := &extproc.ProcessingResponse{
		Response: &extproc.ProcessingResponse_ResponseTrailers{},
	}
	if err := procsrv.Send(r); err != nil {
		return fmt.Errorf("ResponseTrailers: failed sending response: %w", err)
	}
	return nil
}

// IgnoreCanceled returns nil if the error is a context.Canceled error or an io.EOF error.
func IgnoreCanceled(err error) error {
	switch {
	case errors.Is(err, io.EOF), errors.Is(err, status.Error(grpcodes.Canceled, context.Canceled.Error())):
		return nil
	}
	return err
}
