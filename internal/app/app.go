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
	cfg, err := config.Load(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	logFile, err := os.OpenFile(cfg.LogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}
	logger.Init(logFile, cfg.Log.Level, cfg.Log.Format)

	gin.SetMode(cfg.GinMode)

	slog.Info("Initializing ServerMaster...")

	dynamicPortRegistry := make(map[string]*service.DynamicPortRuntime)
	portServices := make([]*service.PortService, 0, len(cfg.Cron.DynamicPorts))
	for _, dynamicPort := range cfg.Cron.DynamicPorts {
		if !dynamicPort.Enable {
			continue
		}
		dpCfg := dynamicPort
		var queue *utils.Queue[string]
		if dpCfg.Mode != config.DynamicPortModeRange {
			queue = utils.NewQueue[string](dpCfg.ActiveNum)
		}
		portService := service.NewPortService(dpCfg, queue)
		if queue != nil {
			dynamicPortRegistry[dpCfg.Name] = &service.DynamicPortRuntime{
				Config: &dpCfg,
				Queue:  queue,
				Port:   portService,
			}
		}
		portServices = append(portServices, portService)
	}

	svcs := service.NewContainer(cfg, dynamicPortRegistry, portServices)

	cronService := service.NewCronService()
	for _, portService := range svcs.PortServices {
		if err := cronService.AddTask(portService); err != nil {
			slog.Error("Failed to register dynamic port task", "service", portService.Name(), "error", err)
		}
	}
	if cfg.Cron.RuleSet.Enable {
		if err := cronService.AddTask(svcs.Ruleset); err != nil {
			slog.Error("Failed to register ruleset task", "error", err)
		}
	}

	router := api.NewDefaultRouter(svcs)

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
	a.cronService.Start()
	slog.Info("Cron tasks started")

	errChan := make(chan error, 1)
	go func() {
		slog.Info("Server listening on " + a.cfg.Listen)
		if err := a.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errChan <- fmt.Errorf("server failed: %w", err)
		}
	}()

	select {
	case <-ctx.Done():
		slog.Info("Shutting down gracefully...")
	case err := <-errChan:
		return err
	}

	return a.Shutdown()
}

// Shutdown performs cleanup tasks before the application exits.
func (a *App) Shutdown() error {
	a.cronService.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := a.server.Shutdown(ctx); err != nil {
		return fmt.Errorf("server shutdown failed: %w", err)
	}

	slog.Info("Server exited")
	return nil
}
