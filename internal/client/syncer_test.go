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
