package mihomo

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"server-master/internal/client"
	"sync"
	"syscall"
	"time"
)

type Manager struct {
	cfg     *client.Config
	cmd     *exec.Cmd
	mu      sync.Mutex
	running bool
	cancel  context.CancelFunc
}

func NewManager(cfg *client.Config) *Manager {
	return &Manager{
		cfg: cfg,
	}
}

func (m *Manager) Start(ctx context.Context) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.running {
		return
	}
	m.running = true

	// Create a dedicated context for the supervisor
	supCtx, cancel := context.WithCancel(ctx)
	m.cancel = cancel

	go m.supervisor(supCtx)
}

func (m *Manager) supervisor(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			err := m.runOnce(ctx)
			if err != nil {
				slog.Error("Mihomo kernel exited with error", "error", err)
				if ctx.Err() == nil {
					slog.Info("Restarting mihomo in 5 seconds... (like Restart=always)")
					time.Sleep(5 * time.Second)
				}
			} else {
				slog.Info("Mihomo kernel exited gracefully")
				if ctx.Err() == nil {
					time.Sleep(1 * time.Second)
				}
			}
		}
	}
}

func (m *Manager) runOnce(ctx context.Context) error {
	binPath, err := filepath.Abs(m.cfg.Mihomo.BinPath)
	if err != nil {
		return err
	}

	workDir, err := filepath.Abs(m.cfg.Mihomo.WorkDir)
	if err != nil {
		return err
	}

	logPath, err := filepath.Abs(m.cfg.Mihomo.LogPath)
	if err != nil {
		return err
	}

	// Prepare log file
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}
	defer logFile.Close()

	cmd := exec.CommandContext(ctx, binPath, "-d", workDir)
	cmd.Dir = workDir

	mw := io.MultiWriter(os.Stdout, logFile)
	cmd.Stdout = mw
	cmd.Stderr = mw

	m.mu.Lock()
	m.cmd = cmd
	m.mu.Unlock()

	slog.Info("Mihomo kernel starting", "bin", binPath, "workDir", workDir)
	return cmd.Run()
}

func (m *Manager) Reload() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.cmd == nil || m.cmd.Process == nil {
		return fmt.Errorf("mihomo is not running")
	}

	slog.Info("Sending SIGHUP to Mihomo for configuration reload")
	return m.cmd.Process.Signal(syscall.SIGHUP)
}

func (m *Manager) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.cancel != nil {
		m.cancel()
	}

	if m.cmd != nil && m.cmd.Process != nil {
		slog.Info("Terminating Mihomo kernel...")
		_ = m.cmd.Process.Signal(os.Interrupt)
	}
	m.running = false
}
