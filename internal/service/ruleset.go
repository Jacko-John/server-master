package service

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/netip"
	"os"
	"path/filepath"
	"server-master/internal/config"
	"server-master/pkg/utils"
	"sort"
	"strings"
	"time"

	"golang.org/x/sync/errgroup"
	"gopkg.in/yaml.v3"
)

type RulesetService struct {
	cfg        *config.Config
	httpClient *http.Client
}

func NewRulesetService(cfg *config.Config) *RulesetService {
	return &RulesetService{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: 30 * time.Second, // Increased timeout for slow rule sources
		},
	}
}

type rules struct {
	Payload []string `yaml:"payload"`
}

// UpdateAll downloads and updates all configured rule sets
func (s *RulesetService) UpdateAll() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	c := s.cfg.Cron.RuleSet
	slog.Info("Starting rule-set update task")

	// 1. Load rules in parallel
	var dr, pr, rj rules
	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		var err error
		dr, err = s.loadRulesParallel(ctx, c.Direct, "direct")
		return err
	})
	g.Go(func() error {
		var err error
		pr, err = s.loadRulesParallel(ctx, c.Proxy, "proxy")
		return err
	})
	g.Go(func() error {
		var err error
		rj, err = s.loadRulesParallel(ctx, c.Reject, "reject")
		return err
	})

	if err := g.Wait(); err != nil {
		slog.Error("Rule-set update task failed", "error", err)
		return
	}

	// 2. Process and write to files
	s.parseAndWriteDirect(dr, "direct")
	s.atomicWriteToFile(pr, "proxy")
	s.atomicWriteToFile(rj, "reject")

	slog.Info("Rule-set update task completed")
}

func (s *RulesetService) loadRulesParallel(ctx context.Context, links []string, name string) (rules, error) {
	if len(links) == 0 {
		return rules{Payload: []string{}}, nil
	}

	slog.Info("Loading rule set category", "name", name, "count", len(links))

	results := make(chan []string, len(links))
	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(4) // Limit concurrency to prevent overloading

	for _, link := range links {
		url := link
		g.Go(func() error {
			payloads, err := s.fetchOne(ctx, url, name)
			if err != nil {
				return err
			}
			select {
			case results <- payloads:
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		})
	}

	// Wait for all fetches to complete
	if err := g.Wait(); err != nil {
		return rules{}, err
	}
	close(results)

	res := rules{Payload: utils.CollectUnique(results)}
	sort.Strings(res.Payload)
	return res, nil
}

func (s *RulesetService) fetchOne(ctx context.Context, link string, category string) ([]string, error) {
	var reader io.ReadCloser
	if strings.HasPrefix(link, "http") {
		req, err := http.NewRequestWithContext(ctx, "GET", link, nil)
		if err != nil {
			return nil, fmt.Errorf("create request failed: %w", err)
		}
		resp, err := s.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("download rule set failed (%s): %w", category, err)
		}
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("download rule set failed (%s): status %d", category, resp.StatusCode)
		}
		reader = resp.Body
	} else {
		file, err := os.Open(link)
		if err != nil {
			return nil, fmt.Errorf("open rule set file failed (%s): %w", category, err)
		}
		reader = file
	}
	defer reader.Close()
	var rs rules
	if err := yaml.NewDecoder(reader).Decode(&rs); err != nil {
		return nil, fmt.Errorf("decode rule set failed (%s): %w", category, err)
	}
	return rs.Payload, nil
}

func (s *RulesetService) atomicWriteToFile(rs rules, name string) {
	path := filepath.Join(s.cfg.RulePath, name+".yaml")
	tmpPath := path + ".tmp"

	file, err := os.Create(tmpPath)
	if err != nil {
		slog.Error("Create temporary rule file failed", "path", tmpPath, "error", err)
		return
	}

	if err := yaml.NewEncoder(file).Encode(rs); err != nil {
		file.Close()
		slog.Error("Write to temporary rule file failed", "path", tmpPath, "error", err)
		return
	}
	file.Close()

	if err := os.Rename(tmpPath, path); err != nil {
		slog.Error("Failed to rename temporary rule file", "from", tmpPath, "to", path, "error", err)
		return
	}
	slog.Debug("Updated rule file", "path", path)
}

func (s *RulesetService) parseAndWriteDirect(rs rules, name string) {
	sets := map[string]utils.Set[string]{
		"ip":      utils.NewSet[string](),
		"domain":  utils.NewSet[string](),
		"classic": utils.NewSet[string](),
	}

	for _, rule := range rs.Payload {
		if rule == "" {
			continue
		}

		category, processed := s.categorizeRule(rule)
		sets[category].Add(processed)
	}

	for suffix, set := range sets {
		s.writeSetToRuleFile(set, fmt.Sprintf("%s-%s", name, suffix))
	}
}

func (s *RulesetService) categorizeRule(rule string) (string, string) {
	rule = strings.TrimSpace(rule)
	if rule == "" {
		return "classic", ""
	}

	// 1. Formal Clash Rule Detection (TYPE,VALUE[,OPTIONS])
	if before, after, ok := strings.Cut(rule, ","); ok {
		prefix := strings.ToUpper(before)
		val := strings.TrimSpace(after)

		// Extract value before next comma (ignore options like no-resolve)
		cleanVal := val
		if before0, _, ok0 := strings.Cut(val, ","); ok0 {
			cleanVal = strings.TrimSpace(before0)
		}

		switch prefix {
		case "DOMAIN":
			return "domain", cleanVal
		case "DOMAIN-SUFFIX":
			return "domain", "+." + cleanVal
		case "IP-CIDR", "IP-CIDR6":
			return "ip", cleanVal
		default:
			// GEOIP, DOMAIN-KEYWORD, PROCESS-NAME, etc.
			return "classic", rule
		}
	}

	// 2. Heuristics for plain rules (no comma)
	if strings.HasPrefix(rule, "+") {
		return "domain", rule
	}

	// Accurate IP/CIDR detection using netip
	if _, err := netip.ParsePrefix(rule); err == nil {
		return "ip", rule
	}

	// Domain heuristic: contains dot and no space
	if strings.Contains(rule, ".") && !strings.ContainsAny(rule, " \t\n\r") {
		return "domain", rule
	}

	return "classic", rule
}

func (s *RulesetService) writeSetToRuleFile(set utils.Set[string], name string) {
	if set.Size() == 0 {
		return
	}
	r := rules{Payload: set.ToSlice()}
	sort.Strings(r.Payload)
	s.atomicWriteToFile(r, name)
}

// Task interface implementation

func (s *RulesetService) Name() string {
	return "RuleSetUpdate"
}

func (s *RulesetService) Spec() string {
	return s.cfg.Cron.RuleSet.Cycle
}

func (s *RulesetService) Run() {
	s.UpdateAll()
}
