package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfigLoad(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	content := `
listen: ":8080"
proxy-path: "proxies.yaml"
tokens: ["test-token"]
log-path: "test.log"
rule-path: "rules/"
cron:
  dynamic-ports:
    - name: "default"
      enable: true
      protocol: "tcp"
      min: 10000
      max: 10010
      active-num: 2
      target-port: 443
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
	if len(cfg.Cron.DynamicPorts) != 1 {
		t.Fatalf("expected normalized dynamic ports length 1, got %d", len(cfg.Cron.DynamicPorts))
	}
	if cfg.Cron.DynamicPorts[0].Name != "default" {
		t.Errorf("expected default dynamic port name, got %s", cfg.Cron.DynamicPorts[0].Name)
	}
	if cfg.Cron.DynamicPorts[0].Mode != DynamicPortModeRotate {
		t.Errorf("expected default mode %s, got %s", DynamicPortModeRotate, cfg.Cron.DynamicPorts[0].Mode)
	}
	if cfg.Cron.DynamicPorts[0].Protocol != "tcp" {
		t.Errorf("expected default protocol tcp, got %s", cfg.Cron.DynamicPorts[0].Protocol)
	}
	if cfg.Cron.DynamicPorts[0].TargetPort != 443 {
		t.Errorf("expected target port 443, got %d", cfg.Cron.DynamicPorts[0].TargetPort)
	}
	if cfg.Cron.DynamicPorts[0].Cycle != "@every 1m" {
		t.Errorf("expected default cycle @every 1m, got %s", cfg.Cron.DynamicPorts[0].Cycle)
	}

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
				Cron: CronConfig{
					DynamicPorts: []DynamicPortServiceConfig{{
						Name:       "tcp-service",
						Enable:     true,
						Protocol:   "tcp",
						Min:        10000,
						Max:        10010,
						ActiveNum:  2,
						TargetPort: 443,
						Proxies:    []string{"local-a", "local-b"},
					}},
				},
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
			name: "invalid protocol",
			cfg: Config{
				Listen:    ":8080",
				ProxyPath: "p.yaml",
				Tokens:    []string{"t"},
				RulePath:  "r/",
				Cron: CronConfig{
					DynamicPorts: []DynamicPortServiceConfig{{
						Name:       "bad-service",
						Enable:     true,
						Protocol:   "icmp",
						Min:        10000,
						Max:        10010,
						ActiveNum:  1,
						TargetPort: 443,
					}},
				},
			},
			wantErr: true,
		},
		{
			name: "range mode valid config",
			cfg: Config{
				Listen:    ":8080",
				ProxyPath: "p.yaml",
				Tokens:    []string{"t"},
				RulePath:  "r/",
				Cron: CronConfig{
					DynamicPorts: []DynamicPortServiceConfig{{
						Name:       "range-service",
						Enable:     true,
						Mode:       DynamicPortModeRange,
						Protocol:   "tcp",
						Min:        10000,
						Max:        10010,
						TargetPort: 443,
					}},
				},
			},
			wantErr: false,
		},
		{
			name: "range mode rejects proxies",
			cfg: Config{
				Listen:    ":8080",
				ProxyPath: "p.yaml",
				Tokens:    []string{"t"},
				RulePath:  "r/",
				Cron: CronConfig{
					DynamicPorts: []DynamicPortServiceConfig{{
						Name:       "range-service",
						Enable:     true,
						Mode:       DynamicPortModeRange,
						Protocol:   "tcp",
						Min:        10000,
						Max:        10010,
						TargetPort: 443,
						Proxies:    []string{"local-a"},
					}},
				},
			},
			wantErr: true,
		},
		{
			name: "range mode rejects active num",
			cfg: Config{
				Listen:    ":8080",
				ProxyPath: "p.yaml",
				Tokens:    []string{"t"},
				RulePath:  "r/",
				Cron: CronConfig{
					DynamicPorts: []DynamicPortServiceConfig{{
						Name:       "range-service",
						Enable:     true,
						Mode:       DynamicPortModeRange,
						Protocol:   "tcp",
						Min:        10000,
						Max:        10010,
						ActiveNum:  1,
						TargetPort: 443,
					}},
				},
			},
			wantErr: true,
		},
		{
			name: "rotate wildcard can coexist with range service",
			cfg: Config{
				Listen:    ":8080",
				ProxyPath: "p.yaml",
				Tokens:    []string{"t"},
				RulePath:  "r/",
				Cron: CronConfig{
					DynamicPorts: []DynamicPortServiceConfig{
						{
							Name:       "default",
							Enable:     true,
							Mode:       DynamicPortModeRotate,
							Protocol:   "tcp",
							Min:        10000,
							Max:        10010,
							ActiveNum:  1,
							TargetPort: 443,
						},
						{
							Name:       "range-service",
							Enable:     true,
							Mode:       DynamicPortModeRange,
							Protocol:   "udp",
							Min:        20000,
							Max:        20010,
							TargetPort: 8443,
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "duplicate proxy binding",
			cfg: Config{
				Listen:    ":8080",
				ProxyPath: "p.yaml",
				Tokens:    []string{"t"},
				RulePath:  "r/",
				Cron: CronConfig{
					DynamicPorts: []DynamicPortServiceConfig{
						{
							Name:       "tcp-a",
							Enable:     true,
							Protocol:   "tcp",
							Min:        10000,
							Max:        10010,
							ActiveNum:  1,
							TargetPort: 443,
							Proxies:    []string{"local-a"},
						},
						{
							Name:       "udp-b",
							Enable:     true,
							Protocol:   "udp",
							Min:        20000,
							Max:        20010,
							ActiveNum:  1,
							TargetPort: 8443,
							Proxies:    []string{"local-a"},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "wildcard with multiple enabled services",
			cfg: Config{
				Listen:    ":8080",
				ProxyPath: "p.yaml",
				Tokens:    []string{"t"},
				RulePath:  "r/",
				Cron: CronConfig{
					DynamicPorts: []DynamicPortServiceConfig{
						{
							Name:       "default",
							Enable:     true,
							Protocol:   "tcp",
							Min:        10000,
							Max:        10010,
							ActiveNum:  1,
							TargetPort: 443,
						},
						{
							Name:       "other",
							Enable:     true,
							Protocol:   "udp",
							Min:        20000,
							Max:        20010,
							ActiveNum:  1,
							TargetPort: 8443,
							Proxies:    []string{"local-udp"},
						},
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
