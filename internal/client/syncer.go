package client

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"server-master/internal/model"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"
	"gopkg.in/yaml.v3"
)

type Syncer struct {
	cfg        *Config
	httpClient *http.Client
	ReloadFunc func(context.Context) error
}

func NewSyncer(cfg *Config) *Syncer {
	return &Syncer{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (s *Syncer) Sync(ctx context.Context) error {
	slog.Info("Starting synchronization and merge...")

	var finalCfg *model.ClashConfig

	// 1. Fetch upstream from ServerMaster if configured
	if s.cfg.ServerURL != "" {
		upstream, err := s.fetchUpstream(ctx)
		if err != nil {
			slog.Error("Failed to fetch upstream config", "error", err)
		} else {
			finalCfg = upstream
		}
	}

	if finalCfg == nil {
		return fmt.Errorf("no valid upstream configuration found")
	}

	// 2. Fetch additional subscriptions
	if len(s.cfg.Additions) > 0 {
		if err := s.mergeAdditions(ctx, finalCfg); err != nil {
			slog.Warn("Some additions failed to merge", "error", err)
		}
	}

	// 3. Apply prepend rules (highest priority)
	if len(s.cfg.PrependRules) > 0 {
		finalCfg.Rules = append(s.cfg.PrependRules, finalCfg.Rules...)
	}

	// 4. Save final configuration
	if err := s.saveConfig(finalCfg); err != nil {
		return err
	}

	// 5. Trigger reload
	if s.ReloadFunc != nil {
		if err := s.ReloadFunc(ctx); err != nil {
			slog.Warn("Custom reload failed", "error", err)
		} else {
			slog.Info("Custom reload triggered successfully")
		}
	}
	slog.Info("Synchronization and merge completed")
	return nil
}

func (s *Syncer) fetchUpstream(ctx context.Context) (*model.ClashConfig, error) {
	subURL, err := url.Parse(s.cfg.ServerURL)
	if err != nil {
		return nil, err
	}
	subURL.Path = "/sub"
	q := subURL.Query()
	q.Set("token", s.cfg.Token)
	subURL.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, "GET", subURL.String(), nil)
	if err != nil {
		return nil, err
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("upstream returned %s", resp.Status)
	}

	var cfg model.ClashConfig
	if err := yaml.NewDecoder(resp.Body).Decode(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (s *Syncer) mergeAdditions(ctx context.Context, base *model.ClashConfig) error {
	var mu sync.Mutex
	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(3)

	for _, it := range s.cfg.Additions {
		g.Go(func() error {
			req, err := http.NewRequestWithContext(ctx, "GET", it.URL, nil)
			if err != nil {
				return err
			}
			req.Header.Set("User-Agent", "Clash")

			resp, err := s.httpClient.Do(req)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				return fmt.Errorf("addition %s returned %s", it.URL, resp.Status)
			}

			var data model.ClashConfig
			if err := yaml.NewDecoder(resp.Body).Decode(&data); err != nil {
				return err
			}

			mu.Lock()
			defer mu.Unlock()

			base.Proxies = append(base.Proxies, data.Proxies...)
			group := model.ClashProxyGroup{
				Name:    it.GroupName,
				Type:    it.GroupType,
				Proxies: make([]string, 0, len(data.Proxies)),
			}
			for _, p := range data.Proxies {
				group.Proxies = append(group.Proxies, p.Name)
			}
			base.ProxyGroups = append(base.ProxyGroups, group)
			base.Rules = append(it.PrependRules, base.Rules...)

			return nil
		})
	}

	return g.Wait()
}

func (s *Syncer) saveConfig(cfg *model.ClashConfig) error {
	if err := os.MkdirAll(filepath.Dir(s.cfg.ConfigPath), 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal final config: %w", err)
	}

	if err := os.WriteFile(s.cfg.ConfigPath, data, 0644); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	slog.Info("Configuration saved", "path", s.cfg.ConfigPath)
	return nil
}
