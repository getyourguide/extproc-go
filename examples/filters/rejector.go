package filters

import (
	"context"
	"log/slog"
	"net/http"

	extproc "github.com/envoyproxy/go-control-plane/envoy/service/ext_proc/v3"
	"github.com/getyourguide/extproc-go/filter"
)

// Rejector is a filter that immediately responds with a 403
type Rejector struct {
	filter.NoOpFilter
}

var _ filter.Filter = &Rejector{}

func (f *Rejector) RequestHeaders(ctx context.Context, crw *filter.CommonResponseWriter, req *filter.RequestContext) (*extproc.ProcessingResponse_ImmediateResponse, error) {
	slog.Info("rejecting a request", "request", req.RequestID())
	return filter.NewImmediateResponseBuilder().
		HTTPStatus(http.StatusForbidden).
		ImmediateResponse(), nil
}
