// Package main provides the release-oriented CLI entrypoint for gormdb2struct.
package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/dan-sherwin/gormdb2struct/cmd/app"
)

func main() {
	if err := app.Run(context.Background(), os.Args[1:]); err != nil {
		slog.Error("Application failed", slog.String("error", err.Error()))
		os.Exit(1)
	}
}
