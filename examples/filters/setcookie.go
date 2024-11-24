package filters

import (
	"context"
	"net/http"

	extproc "github.com/envoyproxy/go-control-plane/envoy/service/ext_proc/v3"
	"github.com/getyourguide/extproc-go/filter"
)

// SameSiteLaxMode is a filter that sets the SameSite attribute to Lax and HttpOnly to true for all cookies.
type SameSiteLaxMode struct {
	filter.NoOpFilter
}

var _ filter.Filter = &SameSiteLaxMode{}

func (f *SameSiteLaxMode) ResponseHeaders(ctx context.Context, crw *filter.CommonResponseWriter, req *filter.RequestContext) (*extproc.ProcessingResponse_ImmediateResponse, error) {
	for i, cookie := range req.SetCookies() {
		cookie.SameSite = http.SameSiteLaxMode
		cookie.HttpOnly = true
		// The first set-cookie we overwrite the header, the others we append
		if i == 0 {
			crw.SetHeader("set-cookie", cookie.String())
			continue
		}
		crw.AppendHeader("set-cookie", cookie.String())
	}
	return nil, nil
}
