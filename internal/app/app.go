package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"server-master/internal/api"
	"server-master/internal/config"
	"server-master/internal/service"
	"server-master/pkg/logger"
	"server-master/pkg/utils"
	"time"

	"github.com/gin-gonic/gin"
)

// App manages the application's lifecycle and dependencies.
type App struct {
	cfg         *config.Config
	cronService *service.CronService
	server      *http.Server
}

// New creates and assembles a new App instance.
func New(configPath string) (*App, error) {
	// 1. Load Config
	cfg, err := config.Load(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// 2. Init Logger
	logFile, err := os.OpenFile(cfg.LogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}
	logger.Init(logFile, cfg.Log.Level, cfg.Log.Format)

	// 3. Set Gin Mode
	gin.SetMode(cfg.GinMode)

	slog.Info("Initializing ServerMaster...")

	// 3. Build Business Services (Container)
	queue := utils.NewQueue[string](cfg.Cron.DynamicPort.ActiveNum)
	svcs := service.NewContainer(cfg, queue)

	// 4. Register Cron Tasks
	cronService := service.NewCronService()
	if cfg.Cron.DynamicPort.Enable {
		if err := cronService.AddTask(svcs.Port); err != nil {
			slog.Error("Failed to register dynamic port task", "error", err)
		}
	}
	if cfg.Cron.RuleSet.Enable {
		if err := cronService.AddTask(svcs.Ruleset); err != nil {
			slog.Error("Failed to register ruleset task", "error", err)
		}
	}

	// 5. Build Router using default services
	router := api.NewDefaultRouter(svcs)

	// 6. Build HTTP Server
	server := &http.Server{
		Addr:    cfg.Listen,
		Handler: router,
	}

	return &App{
		cfg:         cfg,
		cronService: cronService,
		server:      server,
	}, nil
}

// Run starts the application and blocks until the context is canceled.
func (a *App) Run(ctx context.Context) error {
	// 1. Start Cron Tasks
	a.cronService.Start()
	slog.Info("Cron tasks started")

	// 2. Start HTTP Server
	errChan := make(chan error, 1)
	go func() {
		slog.Info("Server listening on " + a.cfg.Listen)
		if err := a.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errChan <- fmt.Errorf("server failed: %w", err)
		}
	}()

	// 3. Wait for Termination Signal or Error
	select {
	case <-ctx.Done():
		slog.Info("Shutting down gracefully...")
	case err := <-errChan:
		return err
	}

	// 4. Graceful Shutdown
	return a.Shutdown()
}

// Shutdown performs cleanup tasks before the application exits.
func (a *App) Shutdown() error {
	// Stop Cron tasks
	a.cronService.Stop()

	// Shutdown HTTP Server
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := a.server.Shutdown(ctx); err != nil {
		return fmt.Errorf("server shutdown failed: %w", err)
	}

	slog.Info("Server exited")
	return nil
}
