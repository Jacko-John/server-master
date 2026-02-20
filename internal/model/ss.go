package model

import (
	"encoding/base64"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

type SSLink struct {
	Name       string
	Type       string
	Server     string
	Port       string
	Cipher     string
	Password   string
	UDP        bool
	Plugin     string
	PluginOpts map[string]string
}

func (s *SSLink) ParseSSLink(ssURL string) error {
	const prefix = "ss://"
	if !strings.HasPrefix(ssURL, prefix) {
		return fmt.Errorf("invalid scheme prefix")
	}
	ssURL = strings.TrimPrefix(ssURL, prefix)

	// Extract remark info
	if idx := strings.Index(ssURL, "#"); idx != -1 {
		name, err := url.QueryUnescape(ssURL[idx+1:])
		if err != nil {
			return fmt.Errorf("failed to URL decode remark: %v", err)
		}
		s.Name, ssURL = name, ssURL[:idx]
	}

	// Parse plugin params
	if idx := strings.Index(ssURL, "/?plugin="); idx != -1 {
		if err := s.parsePluginParams(ssURL[idx+len("/?plugin="):]); err != nil {
			return fmt.Errorf("plugin params error: %w", err)
		}
		ssURL = ssURL[:idx]
	}

	// Parse core params
	if err := s.parseCoreParams(ssURL); err != nil {
		return fmt.Errorf("core params error: %w", err)
	}

	// Set defaults
	s.Type = "ss"
	s.UDP = true

	return nil
}

func (s *SSLink) parsePluginParams(encodedParams string) error {
	pluginStr, err := url.QueryUnescape(encodedParams)
	if err != nil {
		return fmt.Errorf("URL decode failed: %w", err)
	}

	parts := strings.Split(pluginStr, ";")
	if len(parts) < 1 {
		return fmt.Errorf("empty plugin parameters")
	}

	s.Plugin = parts[0]
	s.PluginOpts = make(map[string]string)

	for _, param := range parts[1:] {
		if param == "" {
			continue
		}
		kv := strings.SplitN(param, "=", 2)
		if len(kv) != 2 {
			return fmt.Errorf("invalid parameter format: %q", param)
		}
		s.PluginOpts[kv[0]] = kv[1]
	}
	return nil
}

func (s *SSLink) parseCoreParams(encoded string) error {
	parts := strings.SplitN(encoded, "@", 2)
	if len(parts) == 1 {
		// Handle full base64 encoding format
		decoded, err := base64.RawURLEncoding.DecodeString(encoded)
		if err != nil {
			return fmt.Errorf("base64 decode error: %w", err)
		}
		return s.parseHostPort(string(decoded))
	}

	// Handle method:password@host:port format
	methodPass, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return fmt.Errorf("method-pass decode error: %w", err)
	}

	mp := strings.SplitN(string(methodPass), ":", 2)
	if len(mp) != 2 {
		return fmt.Errorf("invalid method:password format: %q", methodPass)
	}

	s.Cipher, s.Password = mp[0], mp[1]
	return s.parseHostPort(parts[1])
}

func (s *SSLink) parseHostPort(hostPort string) error {
	hp := strings.SplitN(hostPort, ":", 2)
	if len(hp) != 2 {
		return fmt.Errorf("invalid host:port format: %q", hostPort)
	}

	if _, err := strconv.ParseUint(hp[1], 10, 16); err != nil {
		return fmt.Errorf("invalid port number: %w", err)
	}

	s.Server = hp[0]
	s.Port = hp[1]
	return nil
}
