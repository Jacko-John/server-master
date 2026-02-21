package model

import "maps"

type ClashConfig struct {
	MixedPort          int                     `yaml:"mixed-port" json:"mixed_port"`
	AllowLan           bool                    `yaml:"allow-lan" json:"allow_lan"`
	BindAddress        string                  `yaml:"bind-address" json:"bind_address"`
	Mode               string                  `yaml:"mode" json:"mode"`
	LogLevel           string                  `yaml:"log-level" json:"log_level"`
	ExternalController string                  `yaml:"external-controller,omitempty" json:"external_controller"`
	DNS                DNSConfig               `yaml:"dns" json:"dns"`
	Proxies            []ClashProxy            `yaml:"proxies" json:"proxies"`
	ProxyGroups        []ClashProxyGroup       `yaml:"proxy-groups" json:"proxy_groups"`
	Rules              []string                `yaml:"rules" json:"rules"`
	RuleProviders      map[string]RuleProvider `yaml:"rule-providers" json:"rule_providers"`
}

type DNSConfig struct {
	Enable            bool     `yaml:"enable" json:"enable"`
	IPv6              bool     `yaml:"ipv6" json:"ipv6"`
	DefaultNameserver []string `yaml:"default-nameserver" json:"default_nameserver"`
	Nameserver        []string `yaml:"nameserver" json:"nameserver"`
	Fallback          []string `yaml:"fallback" json:"fallback"`
	FallbackFilter    struct {
		Geoip     bool     `yaml:"geoip" json:"geoip"`
		GeoipCode string   `yaml:"geoip-code" json:"geoip_code"`
		IPCidr    []string `yaml:"ipcidr" json:"ipcidr"`
	} `yaml:"fallback-filter" json:"fallback_filter"`
	EnhancedMode string   `yaml:"enhanced-mode" json:"enhanced_mode"`
	FakeIPRange  string   `yaml:"fake-ip-range" json:"fake_ip_range"`
	FakeIPFilter []string `yaml:"fake-ip-filter" json:"fake_ip_filter"`
}

type RuleProvider struct {
	Type      string              `yaml:"type" json:"type"`
	Behavior  string              `yaml:"behavior" json:"behavior"`
	Format    string              `yaml:"format" json:"format"`
	Path      string              `yaml:"path" json:"path"`
	URL       string              `yaml:"url" json:"url"`
	Interval  int                 `yaml:"interval,omitempty" json:"interval,omitempty"`
	Proxy     string              `yaml:"proxy,omitempty" json:"proxy,omitempty"`
	SizeLimit int64               `yaml:"size-limit,omitempty" json:"size_limit,omitempty"`
	Header    map[string][]string `yaml:"header,omitempty" json:"header,omitempty"`
}

func (r *RuleProvider) Clone() RuleProvider {
	newR := *r
	if r.Header != nil {
		newR.Header = make(map[string][]string)
		for k, v := range r.Header {
			newV := make([]string, len(v))
			copy(newV, v)
			newR.Header[k] = newV
		}
	}
	return newR
}

type ClashProxyGroup struct {
	Name    string   `yaml:"name" json:"name"`
	Type    string   `yaml:"type" json:"type"`
	Proxies []string `yaml:"proxies" json:"proxies"`
}

func (g *ClashProxyGroup) Clone() ClashProxyGroup {
	newG := *g
	if g.Proxies != nil {
		newG.Proxies = make([]string, len(g.Proxies))
		copy(newG.Proxies, g.Proxies)
	}
	return newG
}

type ClashProxy struct {
	Name           string            `yaml:"name,omitempty" json:"name,omitempty"`
	Type           string            `yaml:"type,omitempty" json:"type,omitempty"`
	Server         string            `yaml:"server,omitempty" json:"server,omitempty"`
	Port           int               `yaml:"port,omitempty" json:"port,omitempty"`
	Password       string            `yaml:"password,omitempty" json:"password,omitempty"`
	UDP            bool              `yaml:"udp,omitempty" json:"udp,omitempty"`
	SNI            string            `yaml:"sni,omitempty" json:"sni,omitempty"`
	SkipCertVerify bool              `yaml:"skip-cert-verify,omitempty" json:"skip_cert_verify,omitempty"`
	Cipher         string            `yaml:"cipher,omitempty" json:"cipher,omitempty"`
	Plugin         string            `yaml:"plugin,omitempty" json:"plugin,omitempty"`
	PluginOpts     map[string]string `yaml:"plugin-opts,flow,omitempty" json:"plugin_opts,omitempty"`
}

func (p *ClashProxy) Clone() ClashProxy {
	newP := *p
	if p.PluginOpts != nil {
		newP.PluginOpts = make(map[string]string)
		maps.Copy(newP.PluginOpts, p.PluginOpts)
	}
	return newP
}

func (c *ClashConfig) Clone() *ClashConfig {
	newCfg := &ClashConfig{}
	*newCfg = *c

	if c.Proxies != nil {
		newCfg.Proxies = make([]ClashProxy, len(c.Proxies))
		for i := range c.Proxies {
			newCfg.Proxies[i] = c.Proxies[i].Clone()
		}
	}

	if c.ProxyGroups != nil {
		newCfg.ProxyGroups = make([]ClashProxyGroup, len(c.ProxyGroups))
		for i := range c.ProxyGroups {
			newCfg.ProxyGroups[i] = c.ProxyGroups[i].Clone()
		}
	}

	if c.Rules != nil {
		newCfg.Rules = make([]string, len(c.Rules))
		copy(newCfg.Rules, c.Rules)
	}

	if c.RuleProviders != nil {
		newCfg.RuleProviders = make(map[string]RuleProvider)
		for k, v := range c.RuleProviders {
			newCfg.RuleProviders[k] = v.Clone()
		}
	}

	return newCfg
}
