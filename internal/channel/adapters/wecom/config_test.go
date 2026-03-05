package wecom

import (
	"testing"
)

func TestParseConfigValid(t *testing.T) {
	raw := map[string]any{
		"corpId":        "ww1234567890",
		"agentId":       "1000002",
		"secret":        "test-secret-value",
		"token":         "test-token",
		"encodingAESKey": "abcdefghijklmnopqrstuvwxyz0123456789ABCDEFG",
	}
	cfg, err := parseConfig(raw)
	if err != nil {
		t.Fatalf("parseConfig: %v", err)
	}
	if cfg.CorpID != "ww1234567890" {
		t.Fatalf("CorpID = %q", cfg.CorpID)
	}
	if cfg.AgentID != "1000002" {
		t.Fatalf("AgentID = %q", cfg.AgentID)
	}
	if cfg.Secret != "test-secret-value" {
		t.Fatalf("Secret = %q", cfg.Secret)
	}
	if cfg.Token != "test-token" {
		t.Fatalf("Token = %q", cfg.Token)
	}
	if cfg.EncodingAESKey != "abcdefghijklmnopqrstuvwxyz0123456789ABCDEFG" {
		t.Fatalf("EncodingAESKey = %q", cfg.EncodingAESKey)
	}
}

func TestParseConfigSnakeCase(t *testing.T) {
	raw := map[string]any{
		"corp_id":          "ww1234567890",
		"agent_id":         "1000002",
		"secret":           "s",
		"token":            "t",
		"encoding_aes_key": "abcdefghijklmnopqrstuvwxyz0123456789ABCDEFG",
	}
	cfg, err := parseConfig(raw)
	if err != nil {
		t.Fatalf("parseConfig: %v", err)
	}
	if cfg.CorpID != "ww1234567890" {
		t.Fatalf("CorpID = %q", cfg.CorpID)
	}
}

func TestParseConfigMissingRequired(t *testing.T) {
	tests := []struct {
		name string
		raw  map[string]any
	}{
		{"missing corpId", map[string]any{"agentId": "1", "secret": "s", "token": "t", "encodingAESKey": "abcdefghijklmnopqrstuvwxyz0123456789ABCDEFG"}},
		{"missing secret", map[string]any{"corpId": "ww1", "agentId": "1", "token": "t", "encodingAESKey": "abcdefghijklmnopqrstuvwxyz0123456789ABCDEFG"}},
		{"missing token", map[string]any{"corpId": "ww1", "agentId": "1", "secret": "s", "encodingAESKey": "abcdefghijklmnopqrstuvwxyz0123456789ABCDEFG"}},
		{"missing encodingAESKey", map[string]any{"corpId": "ww1", "agentId": "1", "secret": "s", "token": "t"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseConfig(tt.raw)
			if err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestNormalizeConfigRoundTrip(t *testing.T) {
	raw := map[string]any{
		"corp_id":          "ww123",
		"agent_id":         "100",
		"secret":           "sec",
		"token":            "tok",
		"encoding_aes_key": "abcdefghijklmnopqrstuvwxyz0123456789ABCDEFG",
	}
	normalized, err := normalizeConfig(raw)
	if err != nil {
		t.Fatalf("normalizeConfig: %v", err)
	}
	// Should be able to parse the normalized output.
	_, err = parseConfig(normalized)
	if err != nil {
		t.Fatalf("parseConfig from normalized: %v", err)
	}
}
