// Package wecom implements the WeCom (企业微信) channel adapter.
package wecom

import (
	"log/slog"

	"github.com/memohai/memoh/internal/channel"
)

// Type is the registered ChannelType identifier for WeCom.
const Type channel.ChannelType = "wecom"

// WeComAdapter implements the channel.Adapter interface for WeCom.
type WeComAdapter struct {
	logger *slog.Logger
}

// NewWeComAdapter creates a WeComAdapter with the given logger.
func NewWeComAdapter(log *slog.Logger) *WeComAdapter {
	if log == nil {
		log = slog.Default()
	}
	return &WeComAdapter{
		logger: log.With(slog.String("adapter", "wecom")),
	}
}

// Type returns the WeCom channel type.
func (a *WeComAdapter) Type() channel.ChannelType {
	return Type
}

// Descriptor returns the WeCom channel metadata.
func (a *WeComAdapter) Descriptor() channel.Descriptor {
	return channel.Descriptor{
		Type:        Type,
		DisplayName: "WeCom",
		Capabilities: channel.ChannelCapabilities{
			Text:        true,
			RichText:    false,
			Attachments: true,
			Media:       true,
			Reactions:   false,
			Reply:       false,
			Streaming:   false,
		},
		ConfigSchema: channel.ConfigSchema{
			Version: 1,
			Fields: map[string]channel.FieldSchema{
				"corpId":         {Type: channel.FieldString, Required: true, Title: "Corp ID"},
				"agentId":        {Type: channel.FieldString, Required: true, Title: "Agent ID"},
				"secret":         {Type: channel.FieldSecret, Required: true, Title: "Secret"},
				"token":          {Type: channel.FieldSecret, Required: true, Title: "Token"},
				"encodingAESKey": {Type: channel.FieldSecret, Required: true, Title: "Encoding AES Key"},
			},
		},
		UserConfigSchema: channel.ConfigSchema{
			Version: 1,
			Fields: map[string]channel.FieldSchema{
				"userid": {Type: channel.FieldString},
			},
		},
		TargetSpec: channel.TargetSpec{
			Format: "userid:xxx",
			Hints: []channel.TargetHint{
				{Label: "User ID", Example: "userid:zhangsan"},
			},
		},
	}
}

// NormalizeConfig validates and normalizes a WeCom channel configuration map.
func (a *WeComAdapter) NormalizeConfig(raw map[string]any) (map[string]any, error) {
	return normalizeConfig(raw)
}
