package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"server-master/internal/model"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestSyncer_Sync_Merge(t *testing.T) {
	// 1. Setup mock ServerMaster
	serverMaster := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/sub" {
			cfg := model.ClashConfig{
				Proxies: []model.ClashProxy{{Name: "ServerProxy"}},
				Rules:   []string{"MATCH,DIRECT"},
			}
			yaml.NewEncoder(w).Encode(cfg)
		}
	}))
	defer serverMaster.Close()

	// 2. Setup mock Additional Subscription
	additionalSub := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg := model.ClashConfig{
			Proxies: []model.ClashProxy{{Name: "AdditionalProxy"}},
		}
		yaml.NewEncoder(w).Encode(cfg)
	}))
	defer additionalSub.Close()

	// 3. Setup client config
	tmpDir, _ := os.MkdirTemp("", "sm-client-merge-test")
	defer os.RemoveAll(tmpDir)
	configPath := filepath.Join(tmpDir, "final.yaml")

	cfg := &Config{
		ServerURL:  serverMaster.URL + "/sub",
		ConfigPath: configPath,
		Additions: []Addition{
			{
				URL:       additionalSub.URL,
				GroupName: "External",
				GroupType: "select",
			},
		},
		PrependRules: []string{"DOMAIN-SUFFIX,google.com,Proxy"},
	}

	syncer := NewSyncer(cfg)

	// 4. Execute
	err := syncer.Sync(context.Background())
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}

	// 5. Verify Merge Results
	data, _ := os.ReadFile(configPath)
	var final model.ClashConfig
	yaml.Unmarshal(data, &final)

	// Check proxies (1 from server + 1 from addition)
	if len(final.Proxies) != 2 {
		t.Errorf("Expected 2 proxies, got %d", len(final.Proxies))
	}

	// Check rules (1 custom + 1 from server)
	if len(final.Rules) != 2 {
		t.Errorf("Expected 2 rules, got %d", len(final.Rules))
	}
	if final.Rules[0] != "DOMAIN-SUFFIX,google.com,Proxy" {
		t.Errorf("Custom rule should be at the top")
	}

	// Check proxy groups (1 added for the addition)
	if len(final.ProxyGroups) != 1 {
		t.Errorf("Expected 1 proxy group, got %d", len(final.ProxyGroups))
	}
	if final.ProxyGroups[0].Name != "External" {
		t.Errorf("Expected group name 'External', got %s", final.ProxyGroups[0].Name)
	}
}

func TestSyncer_Sync_Overrides(t *testing.T) {
	// 1. Setup mock ServerMaster with default config
	defaultPort := 7890
	serverMaster := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/sub" {
			cfg := model.ClashConfig{
				MixedPort:  defaultPort,
				AllowLan:   false,
				Mode:       "direct",
				LogLevel:   "silent",
				Proxies:    []model.ClashProxy{{Name: "ServerProxy"}},
				Rules:      []string{"MATCH,DIRECT"},
				DNS: model.DNSConfig{
					Enable:       false,
					EnhancedMode: "redir-host",
				},
			}
			yaml.NewEncoder(w).Encode(cfg)
		}
	}))
	defer serverMaster.Close()

	// 2. Setup client config with overrides
	tmpDir, _ := os.MkdirTemp("", "sm-client-override-test")
	defer os.RemoveAll(tmpDir)
	configPath := filepath.Join(tmpDir, "final.yaml")

	overridePort := 8080
	allowLan := true
	mode := "rule"
	logLevel := "info"

	cfg := &Config{
		ServerURL:  serverMaster.URL + "/sub",
		ConfigPath: configPath,
		Overrides: &ConfigOverrides{
			MixedPort: &overridePort,
			AllowLan:  &allowLan,
			Mode:      &mode,
			LogLevel:  &logLevel,
			DNS: &model.DNSConfig{
				Enable:       true,
				EnhancedMode: "fake-ip",
				Nameserver:   []string{"223.5.5.5"},
			},
		},
	}

	syncer := NewSyncer(cfg)

	// 3. Execute sync
	err := syncer.Sync(context.Background())
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}

	// 4. Verify overrides were applied
	data, _ := os.ReadFile(configPath)
	var final model.ClashConfig
	yaml.Unmarshal(data, &final)

	// Check basic overrides
	if final.MixedPort != overridePort {
		t.Errorf("Expected MixedPort %d, got %d", overridePort, final.MixedPort)
	}
	if final.AllowLan != allowLan {
		t.Errorf("Expected AllowLan %v, got %v", allowLan, final.AllowLan)
	}
	if final.Mode != mode {
		t.Errorf("Expected Mode %s, got %s", mode, final.Mode)
	}
	if final.LogLevel != logLevel {
		t.Errorf("Expected LogLevel %s, got %s", logLevel, final.LogLevel)
	}

	// Check DNS override
	if !final.DNS.Enable {
		t.Errorf("Expected DNS.Enable true, got false")
	}
	if final.DNS.EnhancedMode != "fake-ip" {
		t.Errorf("Expected DNS.EnhancedMode 'fake-ip', got %s", final.DNS.EnhancedMode)
	}
	if len(final.DNS.Nameserver) != 1 || final.DNS.Nameserver[0] != "223.5.5.5" {
		t.Errorf("Expected DNS.Nameserver ['223.5.5.5'], got %v", final.DNS.Nameserver)
	}
}
