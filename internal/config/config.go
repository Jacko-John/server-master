package config

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config represents the root configuration structure
type Config struct {
	Listen       string             `yaml:"listen" json:"listen"`
	GinMode      string             `yaml:"gin-mode" json:"gin_mode"`
	Log          LogConfig          `yaml:"log" json:"log"`
	ProxyPath    string             `yaml:"proxy-path" json:"proxy_path"`
	Tokens       []string           `yaml:"tokens" json:"tokens"`
	LogPath      string             `yaml:"log-path" json:"log_path"`
	RulePath     string             `yaml:"rule-path" json:"rule_path"`
	Additions    []Addition         `yaml:"additions" json:"additions"`
	Cron         CronConfig         `yaml:"cron" json:"cron"`
	Subscription SubscriptionConfig `yaml:"subscription" json:"subscription"`
}

// LogConfig holds logger settings
type LogConfig struct {
	Level  string `yaml:"level" json:"level"`
	Format string `yaml:"format" json:"format"`
}

// SubscriptionConfig holds settings for subscription response headers
type SubscriptionConfig struct {
	Filename       string `yaml:"filename" json:"filename"`
	UpdateInterval int    `yaml:"update-interval" json:"update_interval"`
	ProfileURL     string `yaml:"profile-url" json:"profile_url"`
}

// Addition represents an external subscription to be merged
type Addition struct {
	URL          string   `yaml:"url" json:"url"`
	GroupName    string   `yaml:"group-name" json:"group_name"`
	GroupType    string   `yaml:"group-type" json:"group_type"`
	UserAgent    string   `yaml:"user-agent" json:"user_agent"`
	PrependRules []string `yaml:"prepend-rules" json:"prepend_rules"`
}

// CronConfig holds configurations for background tasks
type CronConfig struct {
	DynamicPorts []DynamicPortServiceConfig `yaml:"dynamic-ports,omitempty" json:"dynamic_ports,omitempty"`
	RuleSet      RuleSetConfig              `yaml:"rule-set" json:"rule_set"`
}

// DynamicPortServiceConfig holds settings for one dynamic port service.
type DynamicPortServiceConfig struct {
	Name       string   `yaml:"name" json:"name"`
	Enable     bool     `yaml:"enable" json:"enable"`
	Protocol   string   `yaml:"protocol" json:"protocol"`
	Max        int      `yaml:"max" json:"max"`
	Min        int      `yaml:"min" json:"min"`
	ActiveNum  int      `yaml:"active-num" json:"active_num"`
	TargetPort int      `yaml:"target-port" json:"target_port"`
	Cycle      string   `yaml:"cycle" json:"cycle"`
	Proxies    []string `yaml:"proxies,omitempty" json:"proxies,omitempty"`
}

// RuleSetConfig holds settings for automated rule updates
type RuleSetConfig struct {
	Enable bool     `yaml:"enable" json:"enable"`
	Direct []string `yaml:"direct" json:"direct"`
	Proxy  []string `yaml:"proxy" json:"proxy"`
	Reject []string `yaml:"reject" json:"reject"`
	Cycle  string   `yaml:"cycle" json:"cycle"`
}

// Load loads the configuration from the given path
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &cfg, nil
}

// Validate checks the configuration for required fields and logical consistency
func (c *Config) Validate() error {
	if c.Listen == "" {
		return fmt.Errorf("listen address is required")
	}
	if c.GinMode == "" {
		c.GinMode = "release"
	}
	if c.Log.Level == "" {
		c.Log.Level = "info"
	}
	if c.Log.Format == "" {
		c.Log.Format = "json"
	}
	if c.ProxyPath == "" {
		return fmt.Errorf("proxy-path is required")
	}
	if len(c.Tokens) == 0 {
		return fmt.Errorf("at least one token is required")
	}
	if c.LogPath == "" {
		c.LogPath = "server.log"
	}
	if c.RulePath == "" {
		return fmt.Errorf("rule-path is required")
	}
	for i, add := range c.Additions {
		if add.URL == "" {
			return fmt.Errorf("addition[%d]: URL is required", i)
		}
		if add.GroupName == "" {
			return fmt.Errorf("addition[%d]: group-name is required", i)
		}
	}

	if err := c.validateDynamicPorts(); err != nil {
		return err
	}

	if c.Cron.RuleSet.Enable {
		if c.Cron.RuleSet.Cycle == "" {
			c.Cron.RuleSet.Cycle = "@every 1h"
		}
	}

	// Set default values for subscription
	if c.Subscription.Filename == "" {
		c.Subscription.Filename = "Jacko.yaml"
	}
	if c.Subscription.UpdateInterval <= 0 {
		c.Subscription.UpdateInterval = 18
	}
	if c.Subscription.ProfileURL == "" {
		c.Subscription.ProfileURL = "https://jacko-john.top"
	}

	return nil
}

func (c *Config) validateDynamicPorts() error {
	enabledCount := 0
	wildcardCount := 0
	names := make(map[string]struct{})
	proxyBindings := make(map[string]string)

	for i := range c.Cron.DynamicPorts {
		dp := &c.Cron.DynamicPorts[i]
		if !dp.Enable {
			continue
		}

		enabledCount++
		if dp.Name == "" {
			return fmt.Errorf("cron.dynamic-ports[%d]: name is required", i)
		}
		if _, exists := names[dp.Name]; exists {
			return fmt.Errorf("cron.dynamic-ports[%d]: duplicated name %q", i, dp.Name)
		}
		names[dp.Name] = struct{}{}

		dp.Protocol = strings.ToLower(dp.Protocol)
		if dp.Protocol == "" {
			dp.Protocol = "tcp"
		}
		if dp.Protocol != "tcp" && dp.Protocol != "udp" {
			return fmt.Errorf("cron.dynamic-ports[%d]: protocol must be tcp or udp", i)
		}
		if dp.Min < 1 || dp.Max > 65535 {
			return fmt.Errorf("cron.dynamic-ports[%d]: ports must be in range 1-65535", i)
		}
		if dp.Max <= dp.Min {
			return fmt.Errorf("cron.dynamic-ports[%d]: max (%d) must be greater than min (%d)", i, dp.Max, dp.Min)
		}
		if dp.ActiveNum <= 0 {
			return fmt.Errorf("cron.dynamic-ports[%d]: active-num must be greater than 0", i)
		}
		if dp.Max-dp.Min+1 < dp.ActiveNum {
			return fmt.Errorf("cron.dynamic-ports[%d]: port range [%d, %d] is smaller than active-num %d", i, dp.Min, dp.Max, dp.ActiveNum)
		}
		if dp.TargetPort <= 0 || dp.TargetPort > 65535 {
			return fmt.Errorf("cron.dynamic-ports[%d]: target-port must be in range 1-65535", i)
		}
		if dp.Cycle == "" {
			dp.Cycle = "@every 1m"
		}
		if len(dp.Proxies) == 0 {
			wildcardCount++
		}
		for _, proxyName := range dp.Proxies {
			if proxyName == "" {
				return fmt.Errorf("cron.dynamic-ports[%d]: proxies contains empty proxy name", i)
			}
			if boundTo, exists := proxyBindings[proxyName]; exists {
				return fmt.Errorf("cron.dynamic-ports[%d]: proxy %q is already bound to %q", i, proxyName, boundTo)
			}
			proxyBindings[proxyName] = dp.Name
		}
	}

	if wildcardCount > 0 && enabledCount > 1 {
		return fmt.Errorf("only one enabled dynamic port service may omit proxies")
	}

	return nil
}
