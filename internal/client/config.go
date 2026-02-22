package client

import (
	"fmt"
	"os"
	"path/filepath"

	"server-master/internal/model"

	"gopkg.in/yaml.v3"
)

type Config struct {
	ServerURL      string           `yaml:"server-url" json:"server_url"`
	ConfigPath     string           `yaml:"config-path" json:"config_path"`
	UpdateInterval int              `yaml:"update-interval" json:"update_interval"`
	Additions      []Addition       `yaml:"additions" json:"additions"`
	PrependRules   []string         `yaml:"prepend-rules" json:"prepend_rules"`
	Overrides      *ConfigOverrides `yaml:"overrides,omitempty" json:"overrides,omitempty"`
	Mihomo         MihomoConfig     `yaml:"mihomo" json:"mihomo"`
	Log            LogConfig        `yaml:"log" json:"log"`
}

type LogConfig struct {
	Level  string `yaml:"level" json:"level"`
	Path   string `yaml:"path" json:"path"`
	Format string `yaml:"format" json:"format"` // json or text
}

type MihomoConfig struct {
	Enable  bool   `yaml:"enable" json:"enable"`
	BinPath string `yaml:"bin-path" json:"bin_path"`
	WorkDir string `yaml:"work-dir" json:"work_dir"`
	LogPath string `yaml:"log-path" json:"log_path"`
	Args    string `yaml:"args" json:"args"`
}

type Addition struct {
	URL          string   `yaml:"url" json:"url"`
	GroupName    string   `yaml:"group-name" json:"group_name"`
	GroupType    string   `yaml:"group-type" json:"group_type"`
	PrependRules []string `yaml:"prepend-rules" json:"prepend_rules"`
}

// ConfigOverrides 定义需要覆盖的 Clash 配置项
// 使用指针类型可以区分"未设置"和"设置为零值"的情况
type ConfigOverrides struct {
	// 基础配置覆盖
	MixedPort          *int             `yaml:"mixed-port,omitempty" json:"mixed_port,omitempty"`
	AllowLan           *bool            `yaml:"allow-lan,omitempty" json:"allow_lan,omitempty"`
	BindAddress        *string          `yaml:"bind-address,omitempty" json:"bind_address,omitempty"`
	Mode               *string          `yaml:"mode,omitempty" json:"mode,omitempty"`
	LogLevel           *string          `yaml:"log-level,omitempty" json:"log_level,omitempty"`
	ExternalController *string          `yaml:"external-controller,omitempty" json:"external_controller,omitempty"`
	// DNS 配置覆盖（完全替换）
	DNS                *model.DNSConfig `yaml:"dns,omitempty" json:"dns,omitempty"`
}

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read client config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal client config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid client configuration: %w", err)
	}

	return &cfg, nil
}

func (c *Config) Validate() error {
	if c.ServerURL == "" && len(c.Additions) == 0 {
		return fmt.Errorf("either server-url or additions must be provided")
	}
	if c.ConfigPath == "" {
		c.ConfigPath = "config.yaml"
	}
	if c.UpdateInterval <= 0 {
		c.UpdateInterval = 15
	}
	for i, add := range c.Additions {
		if add.URL == "" {
			return fmt.Errorf("addition[%d]: URL is required", i)
		}
		if add.GroupName == "" {
			return fmt.Errorf("addition[%d]: group-name is required", i)
		}
	}
	if c.Mihomo.Enable {
		if c.Mihomo.BinPath == "" {
			c.Mihomo.BinPath = "./mihomo"
		}
		if c.Mihomo.WorkDir == "" {
			c.Mihomo.WorkDir = "./"
		}
		if c.Mihomo.LogPath == "" {
			c.Mihomo.LogPath = "mihomo.log"
		}
		// Mihomo defaults to reading config.yaml in the work directory
		c.ConfigPath = filepath.Join(c.Mihomo.WorkDir, "config.yaml")
	}

	if c.Log.Level == "" {
		c.Log.Level = "info"
	}
	if c.Log.Format == "" {
		c.Log.Format = "text"
	}

	return nil
}
