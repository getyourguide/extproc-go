package service

import (
	"github.com/getyourguide/extproc-go/filter"
	"github.com/go-logr/logr"
	"go.opentelemetry.io/otel/trace"
)

type Option interface {
	apply(c *ExtProcessor)
}

type optionFunc func(*ExtProcessor)

func (o optionFunc) apply(f *ExtProcessor) {
	o(f)
}

// WithLogger configures the service with a logger
func WithLogger(log logr.Logger) Option {
	return optionFunc(func(svc *ExtProcessor) {
		svc.log = log
	})
}

func WithFilters(filters ...filter.Filter) Option {
	return optionFunc(func(svc *ExtProcessor) {
		svc.filters = filters
	})
}

func WithTracer(tracer trace.Tracer) Option {
	return optionFunc(func(svc *ExtProcessor) {
		svc.tracer = tracer
	})
}
