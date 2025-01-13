package filters

import (
	"context"
	"log/slog"
	"strings"

	extproc "github.com/envoyproxy/go-control-plane/envoy/service/ext_proc/v3"
	"github.com/getyourguide/extproc-go/filter"
)

type Observer struct {
	filter.NoOpFilter
}

var _ filter.Filter = &Observer{}

func (f *Observer) ResponseHeaders(ctx context.Context, crw *filter.CommonResponseWriter, req *filter.RequestContext) (*extproc.ProcessingResponse_ImmediateResponse, error) {
	slog.Info("====== ResponseHeaders ======")
	slog.Info("saw request response", "request_id", req.RequestID())
	slog.Info("request headers")
	for k, v := range req.RequestHeaders {
		slog.Info(k, "values", strings.Join(v, ","))
	}

	slog.Info("response headers")
	for k, v := range req.ResponseHeaders {
		slog.Info(k, "values", strings.Join(v, ","))
	}
	return nil, nil
}
