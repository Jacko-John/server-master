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

	s := NewSubscriptionService(cfg, utils.NewQueue[string](10))
	
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
