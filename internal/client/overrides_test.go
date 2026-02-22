package client

import (
	"server-master/internal/model"
	"testing"
)

func TestApplyOverrides(t *testing.T) {
	tests := []struct {
		name     string
		base     *model.ClashConfig
		opts     *ConfigOverrides
		expected *model.ClashConfig
	}{
		{
			name: "apply mixed port override",
			base: &model.ClashConfig{
				MixedPort: 7890,
				Mode:      "rule",
				LogLevel:  "info",
			},
			opts: &ConfigOverrides{
				MixedPort: intPtr(8080),
			},
			expected: &model.ClashConfig{
				MixedPort: 8080,
				Mode:      "rule",
				LogLevel:  "info",
			},
		},
		{
			name: "apply multiple overrides",
			base: &model.ClashConfig{
				MixedPort: 7890,
				AllowLan:  false,
				Mode:      "direct",
				LogLevel:  "silent",
			},
			opts: &ConfigOverrides{
				MixedPort: intPtr(7890),
				AllowLan:  boolPtr(true),
				Mode:      strPtr("rule"),
				LogLevel:  strPtr("info"),
			},
			expected: &model.ClashConfig{
				MixedPort: 7890,
				AllowLan:  true,
				Mode:      "rule",
				LogLevel:  "info",
			},
		},
		{
			name: "apply DNS override",
			base: &model.ClashConfig{
				MixedPort: 7890,
				DNS: model.DNSConfig{
					Enable: false,
				},
			},
			opts: &ConfigOverrides{
				DNS: &model.DNSConfig{
					Enable:       true,
					EnhancedMode: "fake-ip",
					Nameserver:   []string{"223.5.5.5"},
				},
			},
			expected: &model.ClashConfig{
				MixedPort: 7890,
				DNS: model.DNSConfig{
					Enable:       true,
					EnhancedMode: "fake-ip",
					Nameserver:   []string{"223.5.5.5"},
				},
			},
		},
		{
			name: "nil opts does nothing",
			base: &model.ClashConfig{
				MixedPort: 7890,
				Mode:      "rule",
			},
			opts: nil,
			expected: &model.ClashConfig{
				MixedPort: 7890,
				Mode:      "rule",
			},
		},
		{
			name: "override with zero values",
			base: &model.ClashConfig{
				MixedPort:   7890,
				AllowLan:    true,
				BindAddress: "*",
			},
			opts: &ConfigOverrides{
				MixedPort:   intPtr(0),
				AllowLan:    boolPtr(false),
				BindAddress: strPtr(""),
			},
			expected: &model.ClashConfig{
				MixedPort:   0,
				AllowLan:    false,
				BindAddress: "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			applyOverrides(tt.base, tt.opts)

			if tt.base.MixedPort != tt.expected.MixedPort {
				t.Errorf("MixedPort = %v, want %v", tt.base.MixedPort, tt.expected.MixedPort)
			}
			if tt.base.AllowLan != tt.expected.AllowLan {
				t.Errorf("AllowLan = %v, want %v", tt.base.AllowLan, tt.expected.AllowLan)
			}
			if tt.base.BindAddress != tt.expected.BindAddress {
				t.Errorf("BindAddress = %v, want %v", tt.base.BindAddress, tt.expected.BindAddress)
			}
			if tt.base.Mode != tt.expected.Mode {
				t.Errorf("Mode = %v, want %v", tt.base.Mode, tt.expected.Mode)
			}
			if tt.base.LogLevel != tt.expected.LogLevel {
				t.Errorf("LogLevel = %v, want %v", tt.base.LogLevel, tt.expected.LogLevel)
			}
			if tt.base.DNS.Enable != tt.expected.DNS.Enable {
				t.Errorf("DNS.Enable = %v, want %v", tt.base.DNS.Enable, tt.expected.DNS.Enable)
			}
			if tt.base.DNS.EnhancedMode != tt.expected.DNS.EnhancedMode {
				t.Errorf("DNS.EnhancedMode = %v, want %v", tt.base.DNS.EnhancedMode, tt.expected.DNS.EnhancedMode)
			}
		})
	}
}

// Helper functions to create pointers
func intPtr(i int) *int {
	return &i
}

func boolPtr(b bool) *bool {
	return &b
}

func strPtr(s string) *string {
	return &s
}