package service

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"server-master/internal/config"
	"server-master/internal/model"
	"server-master/pkg/utils"
	"strconv"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"
	"gopkg.in/yaml.v3"
)

type SubscriptionService struct {
	cfg        *config.Config
	queue      *utils.Queue[string]
	httpClient *http.Client
	tokens     utils.Set[string]
	cache      *utils.SafeMap[string, any]
}

type baseCacheEntry struct {
	data    *model.ClashConfig
	modTime time.Time
}

type depCacheEntry struct {
	data    *Dependency
	expires time.Time
}

func NewSubscriptionService(cfg *config.Config, queue *utils.Queue[string]) *SubscriptionService {
	tokenSet := utils.NewSet[string]()
	tokenSet.AddAll(cfg.Tokens)

	return &SubscriptionService{
		cfg:    cfg,
		queue:  queue,
		tokens: tokenSet,
		cache:  utils.NewSafeMap[string, any](),
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

type Dependency struct {
	Proxies      []model.ClashProxy
	ProxyGroups  []model.ClashProxyGroup
	PrependRules []string
	UserInfo     string
}

func (s *SubscriptionService) GetDependencies(ctx context.Context) (*Dependency, error) {
	// Check Cache first
	if val, ok := s.cache.Get("deps"); ok {
		entry := val.(depCacheEntry)
		if time.Now().Before(entry.expires) {
			return entry.data, nil
		}
	}

	additions := s.cfg.Additions
	if len(additions) == 0 {
		return &Dependency{UserInfo: "upload=0; download=0; total=0; expire=0"}, nil
	}

	dependency := &Dependency{}
	var mu sync.Mutex
	var userInfo string

	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(5)

	for _, it := range additions {
		g.Go(func() error {
			req, err := http.NewRequestWithContext(ctx, "GET", it.URL, nil)
			if err != nil {
				slog.Error("Failed to create request for subscription", "url", it.URL, "error", err)
				return nil
			}
			req.Header.Set("User-Agent", "Clash")

			resp, err := s.httpClient.Do(req)
			if err != nil {
				slog.Error("Failed to fetch subscription", "url", it.URL, "error", err)
				return nil
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				slog.Error("Subscription returned non-OK status", "url", it.URL, "status", resp.Status)
				return nil
			}

			var data model.ClashConfig
			if err := yaml.NewDecoder(resp.Body).Decode(&data); err != nil {
				slog.Error("Failed to decode subscription YAML", "url", it.URL, "error", err)
				return nil
			}

			mu.Lock()
			defer mu.Unlock()

			dependency.Proxies = append(dependency.Proxies, data.Proxies...)
			group := model.ClashProxyGroup{
				Name:    it.GroupName,
				Type:    it.GroupType,
				Proxies: make([]string, 0, len(data.Proxies)),
			}
			for _, p := range data.Proxies {
				group.Proxies = append(group.Proxies, p.Name)
			}
			dependency.ProxyGroups = append(dependency.ProxyGroups, group)
			dependency.PrependRules = append(dependency.PrependRules, it.PrependRules...)

			if info := resp.Header.Get("Subscription-Userinfo"); info != "" {
				if userInfo == "" || (len(info) > len(userInfo)) {
					userInfo = info
				}
			}
			return nil
		})
	}

	_ = g.Wait()

	if userInfo == "" {
		userInfo = "upload=0; download=0; total=0; expire=0"
	}
	dependency.UserInfo = userInfo

	// Update Cache
	s.cache.Set("deps", depCacheEntry{
		data:    dependency,
		expires: time.Now().Add(5 * time.Minute),
	})

	return dependency, nil
}

func (s *SubscriptionService) getBaseConfig() (*model.ClashConfig, error) {
	info, err := os.Stat(s.cfg.ProxyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat base proxy file: %w", err)
	}

	// Check cache
	if val, ok := s.cache.Get("base"); ok {
		entry := val.(baseCacheEntry)
		if entry.modTime.Equal(info.ModTime()) {
			return entry.data.Clone(), nil
		}
	}

	// Load and parse
	data, err := os.ReadFile(s.cfg.ProxyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read base proxy file: %w", err)
	}

	var proxy model.ClashConfig
	if err := yaml.Unmarshal(data, &proxy); err != nil {
		return nil, fmt.Errorf("failed to unmarshal base proxy: %w", err)
	}

	// Update cache
	s.cache.Set("base", baseCacheEntry{
		data:    &proxy,
		modTime: info.ModTime(),
	})

	return proxy.Clone(), nil
}

func (s *SubscriptionService) GenerateConfig(ctx context.Context) (*model.ClashConfig, string, error) {
	// 1. Get base config (with ModTime caching)
	proxy, err := s.getBaseConfig()
	if err != nil {
		return nil, "", err
	}

	// 2. Randomize ports if queue is available
	if s.queue != nil && !s.queue.IsEmpty() {
		for i := range proxy.Proxies {
			portStr := s.queue.Rand()
			if portStr != "" {
				if port, err := strconv.Atoi(portStr); err == nil {
					proxy.Proxies[i].Port = port
				}
			}
		}
	}

	// 3. Get external dependencies (with TTL caching)
	dp, err := s.GetDependencies(ctx)
	if err != nil {
		return nil, "", err
	}

	if dp != nil {
		proxy.Proxies = append(proxy.Proxies, dp.Proxies...)
		proxy.ProxyGroups = append(proxy.ProxyGroups, dp.ProxyGroups...)
		proxy.Rules = append(dp.PrependRules, proxy.Rules...)
	}

	return proxy, dp.UserInfo, nil
}

func (s *SubscriptionService) ValidateToken(token string) bool {
	return s.tokens.Has(token)
}

func (s *SubscriptionService) GetConfig() config.SubscriptionConfig {
	return s.cfg.Subscription
}
