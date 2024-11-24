package filters

import (
	"context"
	"fmt"

	extproc "github.com/envoyproxy/go-control-plane/envoy/service/ext_proc/v3"
	"github.com/getyourguide/extproc-go/filter"
)

type BasicAuth struct {
	filter.NoOpFilter
}

var _ filter.Filter = &BasicAuth{}

func (f *BasicAuth) RequestHeaders(ctx context.Context, crw *filter.CommonResponseWriter, req *filter.RequestContext) (*extproc.ProcessingResponse_ImmediateResponse, error) {
	fmt.Println("BasicAuth filter: RequestHeaders")
	authHeader := req.RequestHeader("authorization")
	if req.Authority() == "www.example.com" {
		return nil, nil
	}
	if authHeader == "" {
		return filter.NewImmediateResponseBuilder().
			HTTPStatus(401).
			SetHeader("WWW-Authenticate", "Basic realm=\"Please Authenticate\"").
			Body([]byte(`Unauthorized`)).
			ImmediateResponse(), nil
	}
	return nil, nil
}
