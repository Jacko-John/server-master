package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"server-master/internal/config"
	"server-master/pkg/utils"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestSubscriptionService_ValidateToken(t *testing.T) {
	cfg := &config.Config{
		Tokens: []string{"token1", "token2"},
	}
	s := NewSubscriptionService(cfg, nil)

	if !s.ValidateToken("token1") {
		t.Error("expected token1 to be valid")
	}
	if !s.ValidateToken("token2") {
		t.Error("expected token2 to be valid")
	}
	if s.ValidateToken("wrong") {
		t.Error("expected wrong token to be invalid")
	}
}

func TestSubscriptionService_GenerateConfig(t *testing.T) {
	tempDir := t.TempDir()
	proxyPath := filepath.Join(tempDir, "proxy.yaml")

	// Create dummy base proxy
	baseProxy := `proxies: [{name: "base", type: "ss"}]
rules: []`
	if err := os.WriteFile(proxyPath, []byte(baseProxy), 0644); err != nil {
		t.Fatal(err)
	}

	// Create a dummy remote server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Subscription-Userinfo", "upload=100; download=200; total=1000")
		w.Write([]byte(`proxies: [{name: "remote", type: "vmess"}]`))
	}))
	defer server.Close()

	cfg := &config.Config{
		ProxyPath: proxyPath,
		Additions: []config.Addition{
			{
				URL:       server.URL,
				GroupName: "RemoteGroup",
				GroupType: "select",
			},
		},
		Tokens: []string{"test"},
	}

	s := NewSubscriptionService(cfg, nil)

	ctx := context.Background()
	config, userInfo, err := s.GenerateConfig(ctx)
	if err != nil {
		t.Fatalf("GenerateConfig failed: %v", err)
	}

	if userInfo != "upload=100; download=200; total=1000" {
		t.Errorf("unexpected userInfo: %s", userInfo)
	}

	foundBase := false
	foundRemote := false
	for _, p := range config.Proxies {
		if p.Name == "base" {
			foundBase = true
		}
		if p.Name == "remote" {
			foundRemote = true
		}
	}

	if !foundBase || !foundRemote {
		t.Errorf("missing proxies: base=%v, remote=%v", foundBase, foundRemote)
	}

	if len(config.ProxyGroups) != 1 || config.ProxyGroups[0].Name != "RemoteGroup" {
		t.Errorf("unexpected proxy groups: %+v", config.ProxyGroups)
	}
}

func TestSubscriptionService_GenerateConfig_PreservesProxyExtras(t *testing.T) {
	tempDir := t.TempDir()
	proxyPath := filepath.Join(tempDir, "proxy.yaml")

	baseProxy := `proxies:
  - name: base-anytls
    type: anytls
    server: 127.0.0.1
    port: 18443
    password: base-secret
    client-fingerprint: chrome
    udp: true
    idle-session-check-interval: 30
    idle-session-timeout: 30
    min-idle-session: 0
    sni: test.com
    alpn:
      - h2
      - http/1.1
    skip-cert-verify: true
rules: []`
	if err := os.WriteFile(proxyPath, []byte(baseProxy), 0644); err != nil {
		t.Fatal(err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`proxies:
  - name: remote-vless
    type: vless
    server: example.com
    port: 443
    uuid: test-uuid
    tls: true
    network: ws
    client-fingerprint: chrome
    alpn:
      - h2`))
	}))
	defer server.Close()

	cfg := &config.Config{
		ProxyPath: proxyPath,
		Additions: []config.Addition{{
			URL:       server.URL,
			GroupName: "RemoteGroup",
			GroupType: "select",
		}},
		Tokens: []string{"test"},
	}

	s := NewSubscriptionService(cfg, nil)
	result, _, err := s.GenerateConfig(context.Background())
	if err != nil {
		t.Fatalf("GenerateConfig failed: %v", err)
	}

	proxyMaps := mustMarshalProxyMaps(t, result)
	base := findProxyMap(t, proxyMaps, "base-anytls")
	remote := findProxyMap(t, proxyMaps, "remote-vless")

	if got := base["client-fingerprint"]; got != "chrome" {
		t.Fatalf("expected base client-fingerprint to be preserved, got %#v", got)
	}
	if got := base["idle-session-check-interval"]; got != 30 {
		t.Fatalf("expected base idle-session-check-interval to be 30, got %#v", got)
	}
	if got := base["idle-session-timeout"]; got != 30 {
		t.Fatalf("expected base idle-session-timeout to be 30, got %#v", got)
	}
	if got := base["min-idle-session"]; got != 0 {
		t.Fatalf("expected base min-idle-session to be 0, got %#v", got)
	}
	assertStringSliceField(t, base, "alpn", []string{"h2", "http/1.1"})

	if got := remote["uuid"]; got != "test-uuid" {
		t.Fatalf("expected remote uuid to be preserved, got %#v", got)
	}
	if got := remote["tls"]; got != true {
		t.Fatalf("expected remote tls to be preserved, got %#v", got)
	}
	if got := remote["network"]; got != "ws" {
		t.Fatalf("expected remote network to be preserved, got %#v", got)
	}
	if got := remote["client-fingerprint"]; got != "chrome" {
		t.Fatalf("expected remote client-fingerprint to be preserved, got %#v", got)
	}
	assertStringSliceField(t, remote, "alpn", []string{"h2"})

	if len(result.ProxyGroups) != 1 || result.ProxyGroups[0].Name != "RemoteGroup" {
		t.Fatalf("unexpected proxy groups: %+v", result.ProxyGroups)
	}
}

func TestSubscriptionService_GenerateConfig_ClonePreservesProxyExtrasIsolation(t *testing.T) {
	tempDir := t.TempDir()
	proxyPath := filepath.Join(tempDir, "proxy.yaml")

	baseProxy := `proxies:
  - name: cached-anytls
    type: anytls
    server: 127.0.0.1
    port: 18443
    password: cache-secret
    client-fingerprint: chrome
    alpn:
      - h2
      - http/1.1
    metadata:
      labels:
        - alpha
        - beta
      flag: true
rules: []`
	if err := os.WriteFile(proxyPath, []byte(baseProxy), 0644); err != nil {
		t.Fatal(err)
	}

	s := NewSubscriptionService(&config.Config{ProxyPath: proxyPath}, nil)

	first, _, err := s.GenerateConfig(context.Background())
	if err != nil {
		t.Fatalf("first GenerateConfig failed: %v", err)
	}

	first.Proxies[0].Extra["client-fingerprint"] = "firefox"
	metadata, ok := first.Proxies[0].Extra["metadata"].(map[string]any)
	if !ok {
		t.Fatalf("expected metadata to be a map, got %#v", first.Proxies[0].Extra["metadata"])
	}
	labels, ok := metadata["labels"].([]any)
	if !ok {
		t.Fatalf("expected metadata.labels to be a slice, got %#v", metadata["labels"])
	}
	labels[0] = "changed"
	metadata["flag"] = false

	second, _, err := s.GenerateConfig(context.Background())
	if err != nil {
		t.Fatalf("second GenerateConfig failed: %v", err)
	}

	if got := second.Proxies[0].Extra["client-fingerprint"]; got != "chrome" {
		t.Fatalf("expected cached client-fingerprint to stay chrome, got %#v", got)
	}
	metadata, ok = second.Proxies[0].Extra["metadata"].(map[string]any)
	if !ok {
		t.Fatalf("expected second metadata to be a map, got %#v", second.Proxies[0].Extra["metadata"])
	}
	labels, ok = metadata["labels"].([]any)
	if !ok {
		t.Fatalf("expected second metadata.labels to be a slice, got %#v", metadata["labels"])
	}
	if labels[0] != "alpha" {
		t.Fatalf("expected cached labels[0] to stay alpha, got %#v", labels[0])
	}
	if metadata["flag"] != true {
		t.Fatalf("expected cached metadata.flag to stay true, got %#v", metadata["flag"])
	}
}

func TestSubscriptionService_GenerateConfig_AppliesDynamicPortsOnlyToLocalProxies(t *testing.T) {
	tempDir := t.TempDir()
	proxyPath := filepath.Join(tempDir, "proxy.yaml")

	baseProxy := `proxies:
  - name: local-tcp
    type: trojan
    server: example.com
    port: 443
  - name: local-udp
    type: hysteria2
    server: example.com
    port: 8443
  - name: local-static
    type: ss
    server: static.example.com
    port: 8388
rules: []`
	if err := os.WriteFile(proxyPath, []byte(baseProxy), 0644); err != nil {
		t.Fatal(err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`proxies:
  - name: local-tcp-addition
    type: trojan
    server: remote.example.com
    port: 9443`))
	}))
	defer server.Close()

	cfg := &config.Config{
		ProxyPath: proxyPath,
		Additions: []config.Addition{{
			URL:       server.URL,
			GroupName: "RemoteGroup",
			GroupType: "select",
		}},
		Cron: config.CronConfig{
			DynamicPorts: []config.DynamicPortServiceConfig{
				{
					Name:       "tcp-service",
					Enable:     true,
					Protocol:   "tcp",
					Min:        10000,
					Max:        10010,
					ActiveNum:  1,
					TargetPort: 443,
					Cycle:      "@every 1m",
					Proxies:    []string{"local-tcp"},
				},
				{
					Name:       "udp-service",
					Enable:     true,
					Protocol:   "udp",
					Min:        20000,
					Max:        20010,
					ActiveNum:  1,
					TargetPort: 8443,
					Cycle:      "@every 1m",
					Proxies:    []string{"local-udp"},
				},
			},
		},
	}

	registry := map[string]*DynamicPortRuntime{
		"tcp-service": newDynamicPortRuntime("tcp-service", "tcp", "15001"),
		"udp-service": newDynamicPortRuntime("udp-service", "udp", "25001"),
	}

	s := NewSubscriptionService(cfg, registry)
	result, _, err := s.GenerateConfig(context.Background())
	if err != nil {
		t.Fatalf("GenerateConfig failed: %v", err)
	}

	proxyMaps := mustMarshalProxyMaps(t, result)
	localTCP := findProxyMap(t, proxyMaps, "local-tcp")
	localUDP := findProxyMap(t, proxyMaps, "local-udp")
	localStatic := findProxyMap(t, proxyMaps, "local-static")
	addition := findProxyMap(t, proxyMaps, "local-tcp-addition")

	if got := localTCP["port"]; got != 15001 {
		t.Fatalf("expected local-tcp port to be 15001, got %#v", got)
	}
	if got := localUDP["port"]; got != 25001 {
		t.Fatalf("expected local-udp port to be 25001, got %#v", got)
	}
	if got := localStatic["port"]; got != 8388 {
		t.Fatalf("expected local-static port to stay 8388, got %#v", got)
	}
	if got := addition["port"]; got != 9443 {
		t.Fatalf("expected addition port to stay 9443, got %#v", got)
	}
	if _, ok := localTCP["dynamic-port-service"]; ok {
		t.Fatalf("expected internal dynamic-port-service field to stay hidden from output")
	}
}

func TestSubscriptionService_GenerateConfig_IgnoresRangeModeServices(t *testing.T) {
	tempDir := t.TempDir()
	proxyPath := filepath.Join(tempDir, "proxy.yaml")

	baseProxy := `proxies:
  - name: local-range
    type: trojan
    server: example.com
    port: 443
rules: []`
	if err := os.WriteFile(proxyPath, []byte(baseProxy), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{
		ProxyPath: proxyPath,
		Cron: config.CronConfig{
			DynamicPorts: []config.DynamicPortServiceConfig{{
				Name:       "range-service",
				Enable:     true,
				Mode:       config.DynamicPortModeRange,
				Protocol:   "tcp",
				Min:        10000,
				Max:        10010,
				TargetPort: 443,
				Cycle:      "@every 1m",
			}},
		},
	}

	registry := map[string]*DynamicPortRuntime{
		"range-service": newDynamicPortRuntime("range-service", "tcp", "15001"),
	}

	s := NewSubscriptionService(cfg, registry)
	result, _, err := s.GenerateConfig(context.Background())
	if err != nil {
		t.Fatalf("GenerateConfig failed: %v", err)
	}

	proxyMaps := mustMarshalProxyMaps(t, result)
	localRange := findProxyMap(t, proxyMaps, "local-range")
	if got := localRange["port"]; got != 443 {
		t.Fatalf("expected local-range port to stay 443, got %#v", got)
	}
}

func newDynamicPortRuntime(name, protocol string, ports ...string) *DynamicPortRuntime {
	queue := utils.NewQueue[string](len(ports))
	for _, port := range ports {
		queue.Enqueue(port)
	}
	cfg := &config.DynamicPortServiceConfig{
		Name:     name,
		Enable:   true,
		Protocol: protocol,
	}
	return &DynamicPortRuntime{
		Config: cfg,
		Queue:  queue,
	}
}

func mustMarshalProxyMaps(t *testing.T, cfg any) []map[string]any {
	t.Helper()

	data, err := yaml.Marshal(cfg)
	if err != nil {
		t.Fatalf("marshal config failed: %v", err)
	}

	var raw struct {
		Proxies []map[string]any `yaml:"proxies"`
	}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unmarshal config failed: %v", err)
	}
	return raw.Proxies
}

func findProxyMap(t *testing.T, proxies []map[string]any, name string) map[string]any {
	t.Helper()

	for _, proxy := range proxies {
		if proxy["name"] == name {
			return proxy
		}
	}
	t.Fatalf("proxy %q not found", name)
	return nil
}

func assertStringSliceField(t *testing.T, proxy map[string]any, key string, expected []string) {
	t.Helper()

	raw, ok := proxy[key].([]any)
	if !ok {
		t.Fatalf("expected %s to be a slice, got %#v", key, proxy[key])
	}
	if len(raw) != len(expected) {
		t.Fatalf("expected %s length %d, got %d", key, len(expected), len(raw))
	}
	for i, want := range expected {
		if raw[i] != want {
			t.Fatalf("expected %s[%d] to be %q, got %#v", key, i, want, raw[i])
		}
	}
}
