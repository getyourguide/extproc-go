package filter

import (
	"context"

	extproc "github.com/envoyproxy/go-control-plane/envoy/service/ext_proc/v3"
)

type Filter interface {
	RequestHeaders(ctx context.Context, crw *CommonResponseWriter, req *RequestContext) (*extproc.ProcessingResponse_ImmediateResponse, error)
	ResponseHeaders(ctx context.Context, crw *CommonResponseWriter, req *RequestContext) (*extproc.ProcessingResponse_ImmediateResponse, error)
}

type NoOpFilter struct{}

var _ Filter = &NoOpFilter{}

func (f *NoOpFilter) RequestHeaders(ctx context.Context, crw *CommonResponseWriter, req *RequestContext) (*extproc.ProcessingResponse_ImmediateResponse, error) {
	return nil, nil
}

func (f *NoOpFilter) ResponseHeaders(ctx context.Context, crw *CommonResponseWriter, req *RequestContext) (*extproc.ProcessingResponse_ImmediateResponse, error) {
	return nil, nil
}
