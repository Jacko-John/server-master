package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"server-master/internal/app"
	"syscall"
)

func main() {
	configPath := flag.String("c", "./config.yaml", "Path to config file")
	flag.Parse()

	// 1. Setup Signal Handling
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// 2. Initialize Application
	application, err := app.New(*configPath)
	if err != nil {
		slog.Error("Failed to initialize application", "error", err)
		os.Exit(1)
	}

	// 3. Run Application
	if err := application.Run(ctx); err != nil {
		slog.Error("Application exited with error", "error", err)
		os.Exit(1)
	}
}
