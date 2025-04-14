package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/getyourguide/extproc-go/examples/filters"
	"github.com/getyourguide/extproc-go/server"
)

func main() {
	slog.Info("starting server")
	err := server.New(context.Background(),
		server.WithFilters(&filters.SameSiteLaxMode{}),
		server.WithEcho(),
	).Serve()
	if err != nil {
		slog.Error("failed to start server", "error", err)
		os.Exit(1)
	}
}
