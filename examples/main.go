package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"

	extproc "github.com/envoyproxy/go-control-plane/envoy/service/ext_proc/v3"
	"github.com/getyourguide/extproc-go/examples/filters"
	"github.com/getyourguide/extproc-go/filter"
	"github.com/getyourguide/extproc-go/server"
)

func headersToMap(headers http.Header) map[string]string {
	result := make(map[string]string)
	for k, v := range headers {
		result[k] = strings.Join(v, ",")
	}
	return result
}

func onStreamEnd(req *filter.RequestContext, msg *extproc.ProcessingRequest) {
	type Summary struct {
		RequestID       string
		RequestHeaders  map[string]string
		ResponseHeaders map[string]string
	}

	b, err := json.MarshalIndent(Summary{
		RequestID:       req.RequestID(),
		RequestHeaders:  headersToMap(req.RequestHeaders),
		ResponseHeaders: headersToMap(req.ResponseHeaders),
	}, "", "  ")

	if err != nil {
		fmt.Printf("could not marshal response: %s", err.Error())
	} else {
		fmt.Println(string(b))
	}
}

func main() {
	slog.Info("starting server")
	err := server.New(context.Background(),
		server.WithFilters(
			&filters.SameSiteLaxMode{},
			&filters.StepController{},
		),
		server.WithOnStreamEndFn(onStreamEnd),
		server.WithEcho(),
	).Serve()

	if err != nil {
		slog.Error("failed to start server", "error", err)
		os.Exit(1)
	}
}
