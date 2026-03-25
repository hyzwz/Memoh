package ctripcs

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/memohai/memoh/internal/channel"
	"github.com/memohai/memoh/internal/config"
)

const Type channel.ChannelType = "ctrip_cs"

type Adapter struct {
	logger         *slog.Logger
	browserGateway browserActionRunner
}

func NewAdapter(log *slog.Logger, gatewayCfg config.BrowserGatewayConfig) *Adapter {
	if log == nil {
		log = slog.Default()
	}
	baseURL := gatewayCfg.BaseURL()
	return &Adapter{
		logger: log.With(slog.String("adapter", string(Type))),
		browserGateway: &browserGatewayClient{
			baseURL: baseURL,
			client:  &http.Client{Timeout: 10 * time.Second},
		},
	}
}

func (*Adapter) Type() channel.ChannelType {
	return Type
}

func (*Adapter) Descriptor() channel.Descriptor {
	return channel.Descriptor{
		Type:        Type,
		DisplayName: "Ctrip Customer Service",
		Capabilities: channel.ChannelCapabilities{
			Text:      true,
			Reply:     true,
			Streaming: true,
		},
		ConfigSchema: channel.ConfigSchema{
			Version: 1,
			Fields: map[string]channel.FieldSchema{
				"browserContextId": {
					Type:     channel.FieldString,
					Required: true,
					Title:    "Browser Context ID",
				},
				"entryUrl": {
					Type:        channel.FieldString,
					Title:       "Entry URL",
					Description: "Ctrip page the adapter opens before polling customer-service sessions.",
					Example:     "https://m.ctrip.com/customer-service/inbox",
				},
				"accountLabel": {
					Type:        channel.FieldString,
					Required:    true,
					Title:       "Account Label",
					Description: "Human-readable label used as the adapter self identity.",
				},
				"pollIntervalMs": {
					Type:        channel.FieldNumber,
					Title:       "Poll Interval (ms)",
					Description: "Polling interval used by the browser worker.",
					Example:     1500,
				},
				"inboxPageUrl": {
					Type:        channel.FieldString,
					Title:       "Inbox Page URL",
					Description: "Optional override for the page that the browser worker polls.",
					Example:     "https://m.ctrip.com/customer-service/inbox",
				},
			},
		},
		TargetSpec: channel.TargetSpec{
			Format: "session:<opaque-session-id>",
			Hints: []channel.TargetHint{
				{Label: "Session ID", Example: "session:ctrip-session-123"},
			},
		},
	}
}

func (*Adapter) NormalizeConfig(raw map[string]any) (map[string]any, error) {
	return normalizeConfig(raw)
}

func (a *Adapter) DiscoverSelf(_ context.Context, credentials map[string]any) (map[string]any, string, error) {
	cfg, err := parseConfig(credentials)
	if err != nil {
		return nil, "", err
	}
	label := strings.TrimSpace(cfg.AccountLabel)
	if label == "" {
		return nil, "", errors.New("ctrip_cs accountLabel is required for self discovery")
	}
	identity := map[string]any{
		"accountLabel": label,
		"metadata": map[string]any{
			"account_label": label,
		},
	}
	return identity, label, nil
}
