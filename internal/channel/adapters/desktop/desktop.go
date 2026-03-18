package desktop

import (
	"context"
	"log/slog"

	"github.com/memohai/memoh/internal/channel"
)

// DesktopAdapter implements channel.Adapter, channel.ConfigNormalizer,
// channel.BindingMatcher, channel.TargetResolver, channel.Sender, channel.StreamSender,
// and channel.Receiver for desktop clients connected via Tailscale.
type DesktopAdapter struct {
	logger *slog.Logger
}

// NewDesktopAdapter creates a new desktop channel adapter.
func NewDesktopAdapter(log *slog.Logger) *DesktopAdapter {
	return &DesktopAdapter{
		logger: log.With(slog.String("adapter", "desktop")),
	}
}

// Type returns the channel type identifier.
func (*DesktopAdapter) Type() channel.ChannelType {
	return Type
}

// Descriptor returns the channel metadata.
func (*DesktopAdapter) Descriptor() channel.Descriptor {
	return channel.Descriptor{
		Type:        Type,
		DisplayName: "Desktop",
		Configless:  false,
		Capabilities: channel.ChannelCapabilities{
			Text:        true,
			Markdown:    true,
			Attachments: true,
			Streaming:   true,
			Reply:       true,
		},
		OutboundPolicy: channel.OutboundPolicy{
			TextChunkLimit: 0, // no limit for desktop
		},
		ConfigSchema: channel.ConfigSchema{
			Version: 1,
			Fields: map[string]channel.FieldSchema{
				"hostname": {
					Type:        channel.FieldString,
					Required:    true,
					Title:       "Hostname",
					Description: "Desktop client hostname",
				},
				"platform": {
					Type:        channel.FieldEnum,
					Required:    true,
					Title:       "Platform",
					Description: "Operating system",
					Enum:        []string{"darwin", "windows", "linux"},
				},
			},
		},
		UserConfigSchema: channel.ConfigSchema{
			Version: 1,
			Fields: map[string]channel.FieldSchema{
				"device_id": {
					Type:        channel.FieldString,
					Required:    true,
					Title:       "Device ID",
					Description: "Remote device UUID",
				},
			},
		},
	}
}

// NormalizeConfig validates and normalizes desktop channel configuration.
func (*DesktopAdapter) NormalizeConfig(raw map[string]any) (map[string]any, error) {
	return normalizeConfig(raw)
}

// NormalizeUserConfig validates and normalizes desktop user binding configuration.
func (*DesktopAdapter) NormalizeUserConfig(raw map[string]any) (map[string]any, error) {
	return normalizeUserConfig(raw)
}

// MatchBinding checks if a user binding matches the given criteria.
func (*DesktopAdapter) MatchBinding(config map[string]any, criteria channel.BindingCriteria) bool {
	return matchBinding(config, criteria)
}

// BuildUserConfig constructs a user binding config from an identity.
func (*DesktopAdapter) BuildUserConfig(identity channel.Identity) map[string]any {
	return buildUserConfig(identity)
}

// NormalizeTarget normalizes a desktop delivery target string.
func (*DesktopAdapter) NormalizeTarget(raw string) string {
	return normalizeTarget(raw)
}

// ResolveTarget derives a delivery target from a desktop user-binding configuration.
func (*DesktopAdapter) ResolveTarget(userConfig map[string]any) (string, error) {
	return resolveTarget(userConfig)
}

// Send delivers an outbound message to the desktop client.
func (a *DesktopAdapter) Send(ctx context.Context, cfg channel.ChannelConfig, msg channel.OutboundMessage) error {
	// Desktop messages are delivered via the WebSocket connection managed by the local handler.
	// This adapter acts as a routing declaration; actual delivery is handled by the local channel handler.
	a.logger.Debug("desktop send",
		slog.String("config_id", cfg.ID),
		slog.String("target", msg.Target),
	)
	return nil
}

// OpenStream opens a streaming session for the desktop client.
func (a *DesktopAdapter) OpenStream(ctx context.Context, cfg channel.ChannelConfig, target string, opts channel.StreamOptions) (channel.OutboundStream, error) {
	a.logger.Debug("desktop open stream",
		slog.String("config_id", cfg.ID),
		slog.String("target", target),
	)
	return &desktopStream{logger: a.logger}, nil
}

// Connect implements channel.Receiver. Desktop clients connect via HTTP API
// (not a persistent server-initiated connection like Telegram/Feishu), so this
// returns a passive connection that stays "running" until explicitly stopped.
func (a *DesktopAdapter) Connect(_ context.Context, cfg channel.ChannelConfig, _ channel.InboundHandler) (channel.Connection, error) {
	a.logger.Info("desktop channel connected",
		slog.String("config_id", cfg.ID),
		slog.String("bot_id", cfg.BotID),
	)
	return channel.NewConnection(cfg, func(_ context.Context) error {
		a.logger.Info("desktop channel disconnected", slog.String("config_id", cfg.ID))
		return nil
	}), nil
}
