package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"server-master/internal/client"
	"server-master/internal/client/mihomo"
	"server-master/pkg/logger"
	"syscall"
	"time"
)

func main() {
	configPath := flag.String("c", "client.yaml", "Path to client configuration file")
	daemonMode := flag.Bool("d", false, "Run in daemon mode")
	flag.Parse()

	cfg, err := client.LoadConfig(*configPath)
	if err != nil {
		slog.Error("Failed to load configuration", "error", err)
		os.Exit(1)
	}

	// Initialize Logger
	if cfg.Log.Path != "" {
		logFile, err := os.OpenFile(cfg.Log.Path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			slog.Error("Failed to open log file", "error", err)
			os.Exit(1)
		}
		logger.Init(logFile, cfg.Log.Level, cfg.Log.Format)
	}

	syncer := client.NewSyncer(cfg)

	if !*daemonMode {
		if err := syncer.Sync(context.Background()); err != nil {
			slog.Error("Sync failed", "error", err)
			os.Exit(1)
		}
		return
	}

	runDaemon(syncer, cfg)
}

func runDaemon(syncer *client.Syncer, cfg *client.Config) {
	slog.Info("Starting client in daemon mode", "interval", cfg.UpdateInterval)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Manage Mihomo process
	var mm *mihomo.Manager
	if cfg.Mihomo.Enable {
		mm = mihomo.NewManager(cfg)
	}

	// Initial sync
	if err := syncer.Sync(ctx); err != nil {
		slog.Error("Initial sync failed", "error", err)
	}

	// Start Mihomo after initial sync
	if mm != nil {
		mm.Start(ctx)
		// Set ReloadFunc to use SIGHUP
		syncer.ReloadFunc = func(ctx context.Context) error {
			return mm.Reload()
		}
	}

	ticker := time.NewTicker(time.Duration(cfg.UpdateInterval) * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-sigChan:
			slog.Info("Shutting down daemon...")
			if mm != nil {
				mm.Stop()
			}
			return
		case <-ticker.C:
			if err := syncer.Sync(ctx); err != nil {
				slog.Error("Scheduled sync failed", "error", err)
			}
		}
	}
}
