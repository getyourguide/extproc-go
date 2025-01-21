package filtertest

import (
	"context"
	"fmt"
	"os"

	extproc "github.com/envoyproxy/go-control-plane/envoy/service/ext_proc/v3"
	"github.com/getyourguide/extproc-go/filter"
	"sigs.k8s.io/yaml"
)

type Configuration struct {
	RequestHeaders struct {
		HeaderMutation    headerMutation     `json:"headerMutation"`
		ImmediateResponse *immediateResponse `json:"immediateResponse"`
	} `json:"requestHeaders"`
	ResponseHeaders struct {
		HeaderMutation    headerMutation     `json:"headerMutation"`
		ImmediateResponse *immediateResponse `json:"immediateResponse"`
	} `json:"responseHeaders"`
}

type immediateResponse struct {
	Status int    `json:"status"`
	Body   string `json:"body"`
}
type headerMutation struct {
	RemoveHeader []string          `json:"remove"`
	SetHeader    map[string]string `json:"set"`
	AppendHeader map[string]string `json:"append"`
}

type Filter struct {
	Configuration Configuration
}

var _ filter.Filter = &Filter{}

func (f *Filter) RequestHeaders(ctx context.Context, crw *filter.CommonResponseWriter, req *filter.RequestContext) (*extproc.ProcessingResponse_ImmediateResponse, error) {
	if f.Configuration.RequestHeaders.ImmediateResponse != nil {
		return filter.NewImmediateResponseBuilder().
			HTTPStatus(f.Configuration.RequestHeaders.ImmediateResponse.Status).
			Body([]byte(f.Configuration.RequestHeaders.ImmediateResponse.Body)).
			ImmediateResponse(), nil
	}
	crw.RemoveHeaders(f.Configuration.RequestHeaders.HeaderMutation.RemoveHeader...)
	for k, v := range f.Configuration.RequestHeaders.HeaderMutation.SetHeader {
		crw.SetHeader(k, v)
	}
	for k, v := range f.Configuration.RequestHeaders.HeaderMutation.AppendHeader {
		crw.AppendHeader(k, v)
	}
	return nil, nil
}

func (f *Filter) ResponseHeaders(ctx context.Context, crw *filter.CommonResponseWriter, req *filter.RequestContext) (*extproc.ProcessingResponse_ImmediateResponse, error) {
	if f.Configuration.ResponseHeaders.ImmediateResponse != nil {
		return filter.NewImmediateResponseBuilder().
			HTTPStatus(f.Configuration.ResponseHeaders.ImmediateResponse.Status).
			Body([]byte(f.Configuration.ResponseHeaders.ImmediateResponse.Body)).
			ImmediateResponse(), nil
	}
	crw.RemoveHeaders(f.Configuration.ResponseHeaders.HeaderMutation.RemoveHeader...)
	for k, v := range f.Configuration.ResponseHeaders.HeaderMutation.SetHeader {
		crw.SetHeader(k, v)
	}
	for k, v := range f.Configuration.ResponseHeaders.HeaderMutation.AppendHeader {
		crw.AppendHeader(k, v)
	}
	return nil, nil
}

func New(config []byte) (*Filter, error) {
	var cfg Configuration
	err := yaml.Unmarshal(config, &cfg)
	if err != nil {
		return nil, fmt.Errorf("could not unmarshal file: %w", err)
	}
	return &Filter{
		Configuration: cfg,
	}, nil
}

func NewFromFile(file string) (*Filter, error) {
	f, err := os.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("could not read file: %w", err)
	}
	return New(f)
}
