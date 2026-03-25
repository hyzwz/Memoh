package ctripcs

import (
	"context"
	"testing"

	"github.com/memohai/memoh/internal/config"
)

func TestNormalizeConfigDefaults(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		raw  map[string]any
		want map[string]any
	}{
		{
			name: "normalizes aliases and applies defaults",
			raw: map[string]any{
				"browser_context_id": "  ctx-123  ",
				"entry_url":          "  https://m.ctrip.com/customer-service/inbox  ",
			},
			want: map[string]any{
				"browserContextId": "ctx-123",
				"entryUrl":         "https://m.ctrip.com/customer-service/inbox",
				"accountLabel":     "",
				"pollIntervalMs":   1500,
				"inboxPageUrl":     "https://m.ctrip.com/customer-service/inbox",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := normalizeConfig(tt.raw)
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			for key, want := range tt.want {
				if got[key] != want {
					t.Fatalf("unexpected %s: got %#v want %#v", key, got[key], want)
				}
			}
		})
	}
}

func TestNormalizeConfigRequiresBrowserContextID(t *testing.T) {
	t.Parallel()

	_, err := normalizeConfig(map[string]any{
		"entryUrl":       "https://m.ctrip.com/customer-service/inbox",
		"accountLabel":   "Agent",
		"pollIntervalMs": 1500,
	})
	if err == nil {
		t.Fatal("expected browserContextId validation error")
	}
}

func TestNormalizeConfigRejectsNonCtripEntryURL(t *testing.T) {
	t.Parallel()

	_, err := normalizeConfig(map[string]any{
		"browserContextId": "ctx-123",
		"entryUrl":         "https://example.com/inbox",
	})
	if err == nil {
		t.Fatal("expected entryUrl host validation error")
	}
}

func TestDiscoverSelfUsesAccountLabel(t *testing.T) {
	t.Parallel()

	adapter := NewAdapter(nil, config.BrowserGatewayConfig{})

	identity, externalID, err := adapter.DiscoverSelf(context.Background(), map[string]any{
		"browserContextId": "ctx-123",
		"entryUrl":         "https://m.ctrip.com/customer-service/inbox",
		"accountLabel":     "Ctrip Support A",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if externalID != "Ctrip Support A" {
		t.Fatalf("unexpected externalID: %q", externalID)
	}
	if identity["accountLabel"] != "Ctrip Support A" {
		t.Fatalf("unexpected accountLabel: %#v", identity["accountLabel"])
	}
	metadata, ok := identity["metadata"].(map[string]any)
	if !ok {
		t.Fatalf("expected metadata map, got %#v", identity["metadata"])
	}
	if metadata["account_label"] != "Ctrip Support A" {
		t.Fatalf("unexpected metadata.account_label: %#v", metadata["account_label"])
	}
}
