package wecombot

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/memohai/memoh/internal/channel"
)

const Type channel.ChannelType = "wecom_ai_bot"

type WeComBotAdapter struct {
	logger *slog.Logger

	mu      sync.RWMutex
	clients map[string]*wsClient
}

func NewWeComBotAdapter(log *slog.Logger) *WeComBotAdapter {
	if log == nil {
		log = slog.Default()
	}
	return &WeComBotAdapter{
		logger:  log.With(slog.String("adapter", "wecom_ai_bot")),
		clients: make(map[string]*wsClient),
	}
}

func (*WeComBotAdapter) Type() channel.ChannelType {
	return Type
}

func (*WeComBotAdapter) Descriptor() channel.Descriptor {
	return channel.Descriptor{
		Type:        Type,
		DisplayName: "WeCom AI Bot",
		Capabilities: channel.ChannelCapabilities{
			Text:           true,
			Markdown:       true,
			Reply:          true,
			Streaming:      true,
			BlockStreaming: true,
			Attachments:    true,
			Media:          false,
			ChatTypes:      []string{"direct", "group"},
		},
		OutboundPolicy: channel.OutboundPolicy{
			TextChunkLimit: 4000,
			ChunkerMode:    channel.ChunkerModeMarkdown,
		},
		ConfigSchema: channel.ConfigSchema{
			Version: 1,
			Fields: map[string]channel.FieldSchema{
				"botId": {
					Type:        channel.FieldString,
					Required:    true,
					Title:       "Bot ID",
					Description: "WeCom AI Bot identifier from the enterprise WeChat console.",
				},
				"secret": {
					Type:        channel.FieldSecret,
					Required:    true,
					Title:       "Secret",
					Description: "WeCom AI Bot secret used for websocket authentication.",
				},
				"websocketUrl": {
					Type:        channel.FieldString,
					Title:       "WebSocket URL",
					Description: "Override only if Tencent provides a different endpoint.",
					Example:     defaultWebSocketURL,
				},
				"sendThinkingPrompt": {
					Type:        channel.FieldBool,
					Title:       "Send Thinking Prompt",
					Description: "Emit the initial <think></think> placeholder when streaming starts.",
				},
			},
		},
		UserConfigSchema: channel.ConfigSchema{
			Version: 1,
			Fields: map[string]channel.FieldSchema{
				"userid": {
					Type:  channel.FieldString,
					Title: "User ID",
				},
				"chatid": {
					Type:        channel.FieldString,
					Title:       "Chat ID",
					Description: "Preferred for proactive delivery. Direct chats can persist this automatically from inbound messages.",
				},
			},
		},
		TargetSpec: channel.TargetSpec{
			Format: "userid:xxx | chatid:xxx",
			Hints: []channel.TargetHint{
				{Label: "User ID", Example: "userid:zhangsan"},
				{Label: "Chat ID", Example: "chatid:ww-group-chat-id"},
			},
		},
	}
}

func (*WeComBotAdapter) NormalizeConfig(raw map[string]any) (map[string]any, error) {
	return normalizeConfig(raw)
}

func (*WeComBotAdapter) NormalizeUserConfig(raw map[string]any) (map[string]any, error) {
	return normalizeUserConfig(raw)
}

func (*WeComBotAdapter) NormalizeTarget(raw string) string {
	return normalizeTarget(raw)
}

func (*WeComBotAdapter) ResolveTarget(userConfig map[string]any) (string, error) {
	return resolveTarget(userConfig)
}

func (*WeComBotAdapter) MatchBinding(config map[string]any, criteria channel.BindingCriteria) bool {
	return matchBinding(config, criteria)
}

func (*WeComBotAdapter) BuildUserConfig(identity channel.Identity) map[string]any {
	return buildUserConfig(identity)
}

func (*WeComBotAdapter) DiscoverSelf(_ context.Context, credentials map[string]any) (map[string]any, string, error) {
	cfg, err := parseConfig(credentials)
	if err != nil {
		return nil, "", err
	}
	return map[string]any{"bot_id": cfg.BotID}, cfg.BotID, nil
}

func (a *WeComBotAdapter) Connect(ctx context.Context, cfg channel.ChannelConfig, handler channel.InboundHandler) (channel.Connection, error) {
	parsed, err := parseConfig(cfg.Credentials)
	if err != nil {
		return nil, err
	}
	client := newWSClient(a.logger, parsed, func(frameCtx context.Context, frame wsFrame) error {
		msg, ok, err := buildInboundMessage(cfg, frame)
		if err != nil || !ok {
			return err
		}
		return handler(frameCtx, cfg, msg)
	})
	a.mu.Lock()
	if existing := a.clients[cfg.ID]; existing != nil {
		_ = existing.Close()
	}
	a.clients[cfg.ID] = client
	a.mu.Unlock()

	connCtx, cancel := context.WithCancel(ctx)
	go func() {
		client.Run(connCtx)
		a.mu.Lock()
		if current := a.clients[cfg.ID]; current == client {
			delete(a.clients, cfg.ID)
		}
		a.mu.Unlock()
	}()

	readyCtx, readyCancel := context.WithTimeout(ctx, connectTimeout+5*time.Second)
	defer readyCancel()
	if err := client.WaitReady(readyCtx); err != nil {
		cancel()
		a.mu.Lock()
		if current := a.clients[cfg.ID]; current == client {
			delete(a.clients, cfg.ID)
		}
		a.mu.Unlock()
		_ = client.Close()
		return nil, err
	}

	return channel.NewConnection(cfg, func(context.Context) error {
		cancel()
		a.mu.Lock()
		if current := a.clients[cfg.ID]; current == client {
			delete(a.clients, cfg.ID)
		}
		a.mu.Unlock()
		return client.Close()
	}), nil
}

func (a *WeComBotAdapter) Send(ctx context.Context, cfg channel.ChannelConfig, msg channel.OutboundMessage) error {
	client, _, targetKind, targetID, err := a.resolveSendContext(cfg, msg.Target)
	if err != nil {
		return err
	}
	if len(msg.Message.Attachments) > 0 {
		return errors.New("wecom ai bot outbound attachments are not supported yet")
	}
	content := strings.TrimSpace(msg.Message.PlainText())
	if content == "" {
		return errors.New("wecom ai bot message text is required")
	}
	if targetKind != targetChatID {
		return errors.New("wecom ai bot proactive send requires chatid target")
	}
	return client.SendMarkdown(ctx, targetID, content)
}

func (a *WeComBotAdapter) OpenStream(_ context.Context, cfg channel.ChannelConfig, target string, opts channel.StreamOptions) (channel.OutboundStream, error) {
	client, parsed, targetKind, targetID, err := a.resolveSendContext(cfg, target)
	if err != nil {
		return nil, err
	}
	sourceReqID := metadataString(opts.Metadata, "source_req_id")
	return newOutboundStream(client, targetKind, targetID, sourceReqID, parsed.SendThinkingPrompt), nil
}

func (a *WeComBotAdapter) resolveSendContext(cfg channel.ChannelConfig, rawTarget string) (*wsClient, Config, targetType, string, error) {
	parsed, err := parseConfig(cfg.Credentials)
	if err != nil {
		return nil, Config{}, "", "", err
	}
	targetKind, targetID, err := parseTarget(rawTarget)
	if err != nil {
		return nil, Config{}, "", "", err
	}
	a.mu.RLock()
	client := a.clients[cfg.ID]
	a.mu.RUnlock()
	if client == nil || !client.Connected() {
		return nil, Config{}, "", "", fmt.Errorf("wecom ai bot connection is not ready for config %s", cfg.ID)
	}
	switch targetKind {
	case targetUserID, targetChatID:
		return client, parsed, targetKind, targetID, nil
	default:
		return nil, Config{}, "", "", errors.New("unsupported wecom ai bot target")
	}
}

func metadataString(metadata map[string]any, key string) string {
	if metadata == nil {
		return ""
	}
	value, ok := metadata[key]
	if !ok {
		return ""
	}
	text, ok := value.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(text)
}
