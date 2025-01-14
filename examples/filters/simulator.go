package filters

import (
	"context"
	"net/http"
	"strconv"

	extproc "github.com/envoyproxy/go-control-plane/envoy/service/ext_proc/v3"
	"github.com/getyourguide/extproc-go/filter"
)

// StepController is a filter that allows triggering an immediate response on any step based on headers
type StepController struct {
	filter.NoOpFilter
}

var _ filter.Filter = &StepController{}

type stepInput string

const (
	stepInputCmdHeader  = "step-input-cmd"
	stepInputCodeHeader = "step-input-code"
	stepResultHeader    = "step-result"

	stepInputHaltRequest  = "halt-request"
	stepInputHaltResponse = "halt-response"
)

func getStepCode(req *filter.RequestContext) int {
	code, err := strconv.Atoi(req.RequestHeaders.Get(stepInputCodeHeader))
	if err != nil {
		return http.StatusUnprocessableEntity
	}
	return code

}

func (f *StepController) RequestHeaders(ctx context.Context, crw *filter.CommonResponseWriter, req *filter.RequestContext) (*extproc.ProcessingResponse_ImmediateResponse, error) {
	cmd := req.RequestHeader(stepInputCmdHeader)
	crw.SetHeader("test", "stepControllerRequestHeaders")
	if cmd == stepInputHaltRequest {
		return filter.NewImmediateResponseBuilder().
			HTTPStatus(getStepCode(req)).
			ImmediateResponse(), nil
	}

	return nil, nil
}

func (f *StepController) ResponseHeaders(ctx context.Context, crw *filter.CommonResponseWriter, req *filter.RequestContext) (*extproc.ProcessingResponse_ImmediateResponse, error) {
	cmd := req.RequestHeader(stepInputCmdHeader)
	if cmd == stepInputHaltResponse {
		return filter.NewImmediateResponseBuilder().
			HTTPStatus(getStepCode(req)).
			ImmediateResponse(), nil
	}
	return nil, nil
}
