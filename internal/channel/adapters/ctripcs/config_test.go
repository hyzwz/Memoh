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
		{
			name: "defaults entry url when omitted",
			raw: map[string]any{
				"browserContextId": "ctx-456",
				"accountLabel":     "  Agent B  ",
			},
			want: map[string]any{
				"browserContextId": "ctx-456",
				"entryUrl":         defaultEntryURL,
				"accountLabel":     "Agent B",
				"pollIntervalMs":   1500,
				"inboxPageUrl":     defaultEntryURL,
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

	tests := []struct {
		name string
		raw  map[string]any
		want string
	}{
		{
			name: "entryUrl host validation",
			raw: map[string]any{
				"browserContextId": "ctx-123",
				"entryUrl":         "https://example.com/inbox",
			},
			want: "ctrip_cs entryUrl must be under a ctrip host",
		},
		{
			name: "inboxPageUrl host validation",
			raw: map[string]any{
				"browserContextId": "ctx-123",
				"entryUrl":         "https://m.ctrip.com/customer-service/inbox",
				"inboxPageUrl":     "https://example.com/inbox",
			},
			want: "ctrip_cs inboxPageUrl must be under a ctrip host",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := normalizeConfig(tt.raw)
			if err == nil {
				t.Fatal("expected entry URL validation error")
			}
			if err.Error() != tt.want {
				t.Fatalf("unexpected error: %v", err)
			}
		})
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

func TestDiscoverSelfRequiresAccountLabel(t *testing.T) {
	t.Parallel()

	adapter := NewAdapter(nil, config.BrowserGatewayConfig{})

	_, _, err := adapter.DiscoverSelf(context.Background(), map[string]any{
		"browserContextId": "ctx-123",
		"entryUrl":         "https://m.ctrip.com/customer-service/inbox",
	})
	if err == nil {
		t.Fatal("expected accountLabel validation error")
	}
}

func TestAdapterDescriptor(t *testing.T) {
	t.Parallel()

	desc := NewAdapter(nil, config.BrowserGatewayConfig{}).Descriptor()
	if desc.Type != Type {
		t.Fatalf("unexpected type: %q", desc.Type)
	}
	if desc.DisplayName != "Ctrip Customer Service" {
		t.Fatalf("unexpected display name: %q", desc.DisplayName)
	}
	if !desc.Capabilities.Text || !desc.Capabilities.Reply {
		t.Fatalf("unexpected capabilities: %#v", desc.Capabilities)
	}
	if desc.ConfigSchema.Version != 1 {
		t.Fatalf("unexpected config schema version: %d", desc.ConfigSchema.Version)
	}
	for _, field := range []string{"browserContextId", "entryUrl", "accountLabel", "pollIntervalMs", "inboxPageUrl"} {
		if _, ok := desc.ConfigSchema.Fields[field]; !ok {
			t.Fatalf("missing config field %q", field)
		}
	}
	if desc.TargetSpec.Format != "session:<opaque-session-id>" {
		t.Fatalf("unexpected target format: %q", desc.TargetSpec.Format)
	}
}

func TestNormalizeConfigPollIntervalValidationRejectsStringAndFractionalAndNonPositive(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		raw  map[string]any
	}{
		{
			name: "string",
			raw: map[string]any{
				"browserContextId": "ctx-123",
				"entryUrl":         "https://m.ctrip.com/customer-service/inbox",
				"pollIntervalMs":   "1500ms",
			},
		},
		{
			name: "fractional",
			raw: map[string]any{
				"browserContextId": "ctx-123",
				"entryUrl":         "https://m.ctrip.com/customer-service/inbox",
				"pollIntervalMs":   1500.5,
			},
		},
		{
			name: "non-positive",
			raw: map[string]any{
				"browserContextId": "ctx-123",
				"entryUrl":         "https://m.ctrip.com/customer-service/inbox",
				"pollIntervalMs":   0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := normalizeConfig(tt.raw)
			if err == nil {
				t.Fatal("expected pollIntervalMs validation error")
			}
		})
	}
}
