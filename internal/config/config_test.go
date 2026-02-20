package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfigLoad(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	
	content := `
listen: ":8080"
proxy-path: "proxies.yaml"
tokens: ["test-token"]
log-path: "test.log"
rule-path: "rules/"
`
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create temp config: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if cfg.Listen != ":8080" {
		t.Errorf("expected listen :8080, got %s", cfg.Listen)
	}

	if len(cfg.Tokens) != 1 || cfg.Tokens[0] != "test-token" {
		t.Error("tokens not loaded correctly")
	}

	// Verify default subscription configuration
	if cfg.Subscription.Filename != "Jacko.yaml" {
		t.Errorf("expected default filename Jacko.yaml, got %s", cfg.Subscription.Filename)
	}
	if cfg.Subscription.UpdateInterval != 18 {
		t.Errorf("expected default interval 18, got %d", cfg.Subscription.UpdateInterval)
	}
	if cfg.Subscription.ProfileURL != "https://jacko-john.top" {
		t.Errorf("expected default profile url, got %s", cfg.Subscription.ProfileURL)
	}
}

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{
			name: "valid config",
			cfg: Config{
				Listen:    ":8080",
				ProxyPath: "p.yaml",
				Tokens:    []string{"t"},
				RulePath:  "r/",
			},
			wantErr: false,
		},
		{
			name: "missing listen",
			cfg: Config{
				ProxyPath: "p.yaml",
				Tokens:    []string{"t"},
				RulePath:  "r/",
			},
			wantErr: true,
		},
		{
			name: "invalid cron ports",
			cfg: Config{
				Listen:    ":8080",
				ProxyPath: "p.yaml",
				Tokens:    []string{"t"},
				RulePath:  "r/",
				Cron: CronConfig{
					DynamicPort: DynamicPortConfig{
						Enable: true,
						Max:    100,
						Min:    200,
					},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
