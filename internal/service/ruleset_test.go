package service

import (
	"testing"
)

func TestCategorizeRule(t *testing.T) {
	s := &RulesetService{}

	tests := []struct {
		name         string
		rule         string
		wantCategory string
		wantProcessed string
	}{
		// Formal Clash Rules
		{"Domain Standard", "DOMAIN,google.com", "domain", "google.com"},
		{"Domain Suffix", "DOMAIN-SUFFIX,google.com", "domain", "+.google.com"},
		{"IP CIDR", "IP-CIDR,127.0.0.1/32", "ip", "127.0.0.1/32"},
		{"IP CIDR6", "IP-CIDR6,2001:db8::/32", "ip", "2001:db8::/32"},
		{"Clash Rule with Options", "DOMAIN,google.com,no-resolve", "domain", "google.com"},
		{"Classic Rule (Keyword)", "DOMAIN-KEYWORD,google", "classic", "DOMAIN-KEYWORD,google"},
		{"Classic Rule (GeoIP)", "GEOIP,CN", "classic", "GEOIP,CN"},

		// Plain Rules / Heuristics
		{"Plain Domain", "google.com", "domain", "google.com"},
		{"Plus Prefix Domain", "+.google.com", "domain", "+.google.com"},
		{"Plain IP", "1.1.1.1", "ip", "1.1.1.1"},
		{"Plain CIDR", "192.168.1.0/24", "ip", "192.168.1.0/24"},
		{"IPv6 Address", "::1", "ip", "::1"},
		{"Domain starting with digit", "1password.com", "domain", "1password.com"},
		
		// Edge cases
		{"Empty", "", "classic", ""},
		{"Space trimmed", "  DOMAIN,google.com  ", "domain", "google.com"},
		{"Classic (Process)", "PROCESS-NAME,curl", "classic", "PROCESS-NAME,curl"},
		{"No Dot String", "localhost", "classic", "localhost"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotCategory, gotProcessed := s.categorizeRule(tt.rule)
			if gotCategory != tt.wantCategory {
				t.Errorf("categorizeRule() gotCategory = %v, want %v", gotCategory, tt.wantCategory)
			}
			if gotProcessed != tt.wantProcessed {
				t.Errorf("categorizeRule() gotProcessed = %v, want %v", gotProcessed, tt.wantProcessed)
			}
		})
	}
}
