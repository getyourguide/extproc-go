package main

import (
	"context"
	"flag"
	"log"
	"log/slog"
	"os"
	"strings"

	"github.com/getyourguide/extproc-go/examples/filters"
	"github.com/getyourguide/extproc-go/filter"
	"github.com/getyourguide/extproc-go/server"
)

func main() {
	var enabledFiltersParam string
	flag.StringVar(&enabledFiltersParam, "filters", "", "the filters to enable")
	flag.Parse()
	enabledFilters := strings.Split(enabledFiltersParam, ",")

	if len(enabledFilters) == 0 {
		log.Fatal("no filters enabled: pass --filters x,y,z to set filters to enable")
	}

	filterMap := map[string]filter.Filter{
		"reject":  &filters.Rejector{},
		"observe": &filters.Observer{},
	}

	var serverFilters []filter.Filter
	for _, filterName := range enabledFilters {
		f, ok := filterMap[filterName]
		if !ok {
			log.Fatalf("no filter called %s exists", filterName)
		}
		serverFilters = append(serverFilters, f)
	}

	slog.Info("starting server", "filters", enabledFilters)
	err := server.New(context.Background(),
		server.WithFilters(serverFilters...),
		server.WithEcho(),
	).Serve()

	if err != nil {
		slog.Error("failed to start server", "error", err)
		os.Exit(1)
	}
}
