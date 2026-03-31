package dingtalk

import (
	"context"
	"errors"
	"log/slog"
	"strings"

	"github.com/memohai/memoh/internal/channel"
)

const Type channel.ChannelType = "dingtalk"

var errNotImplemented = errors.New("dingtalk adapter not implemented")

type Adapter struct {
	logger                *slog.Logger
	newClient             func(Config) *client
	newStreamClient       func(Config, streamEventHandler) (streamClient, error)
	sessionCommandHandler SessionCommandHandler
	assets                assetOpener
}

func NewDingTalkAdapter(log *slog.Logger) *Adapter {
	if log == nil {
		log = slog.Default()
	}
	return &Adapter{
		logger:          log.With(slog.String("adapter", "dingtalk")),
		newClient:       newClient,
		newStreamClient: newSDKStreamClient,
	}
}

type SessionCommandHandler interface {
	HandleReset(ctx context.Context, cfg channel.ChannelConfig, msg channel.InboundMessage, cmd SessionCommand) (string, error)
}

func (a *Adapter) SetSessionCommandHandler(handler SessionCommandHandler) {
	if a == nil {
		return
	}
	a.sessionCommandHandler = handler
}

func (a *Adapter) SetAssetOpener(opener assetOpener) {
	if a == nil {
		return
	}
	a.assets = opener
}

func (*Adapter) Type() channel.ChannelType {
	return Type
}

func (*Adapter) Descriptor() channel.Descriptor {
	return channel.Descriptor{
		Type:        Type,
		DisplayName: "DingTalk",
		Capabilities: channel.ChannelCapabilities{
			Text:           true,
			Attachments:    true,
			Media:          true,
			Reply:          true,
			Streaming:      true,
			BlockStreaming: true,
		},
		ConfigSchema: channel.ConfigSchema{
			Version: 1,
			Fields: map[string]channel.FieldSchema{
				"appKey": {
					Type:     channel.FieldString,
					Required: true,
					Title:    "App Key",
				},
				"appSecret": {
					Type:     channel.FieldSecret,
					Required: true,
					Title:    "App Secret",
				},
				"robotCode": {
					Type:  channel.FieldString,
					Title: "Robot Code",
				},
				"streamEndpoint": {
					Type:        channel.FieldString,
					Title:       "Stream Endpoint",
					Description: "Optional DingTalk stream endpoint override used by the inbound transport.",
				},
				"apiBaseURL": {
					Type:        channel.FieldString,
					Title:       "API Base URL",
					Description: "Optional DingTalk API base URL override.",
				},
				"groupSessionScope": {
					Type:        channel.FieldEnum,
					Title:       "Group Session Scope",
					Description: "Choose whether group sessions are shared or per-sender.",
					Enum:        []string{"group", "group_sender"},
					Example:     "group",
				},
				"requireMentionInGroup": {
					Type:        channel.FieldBool,
					Title:       "Require Mention In Group",
					Description: "Require @bot mentions before a group message starts processing.",
				},
				"enableSessionCommands": {
					Type:        channel.FieldBool,
					Title:       "Enable Session Commands",
					Description: "Enable session reset command handling such as /new.",
				},
				"sessionResetCommands": {
					Type:        channel.FieldString,
					Title:       "Session Reset Commands",
					Description: "Comma-separated session reset commands, for example /new,新会话.",
					Example:     "/new,新会话",
				},
				"enableAICard": {
					Type:        channel.FieldBool,
					Title:       "Enable AI Card",
					Description: "Enable AI Card streaming when DingTalk supports card updates.",
				},
				"cardUpdateThrottleMs": {
					Type:        channel.FieldNumber,
					Title:       "Card Update Throttle (ms)",
					Description: "Minimum interval between AI Card updates.",
					Example:     250,
				},
			},
		},
		UserConfigSchema: channel.ConfigSchema{
			Version: 1,
			Fields: map[string]channel.FieldSchema{
				"conversationId": {
					Type:  channel.FieldString,
					Title: "Conversation ID",
				},
				"staffId": {
					Type:  channel.FieldString,
					Title: "Staff ID",
				},
				"unionId": {
					Type:  channel.FieldString,
					Title: "Union ID",
				},
			},
		},
		TargetSpec: channel.TargetSpec{
			Format: "webhook:<sessionWebhook> | conversation:<conversationId> | user:staff:<staffId> | user:union:<unionId>",
			Hints: []channel.TargetHint{
				{Label: "Session Webhook", Example: "webhook:https://oapi.dingtalk.com/robot/send?access_token=..."},
				{Label: "Conversation", Example: "conversation:cid123456"},
				{Label: "Staff User", Example: "user:staff:staff123"},
				{Label: "Union User", Example: "user:union:union123"},
			},
		},
	}
}

func (*Adapter) NormalizeConfig(raw map[string]any) (map[string]any, error) {
	return normalizeConfig(raw)
}

func (*Adapter) NormalizeUserConfig(raw map[string]any) (map[string]any, error) {
	return normalizeUserConfig(raw)
}

func (*Adapter) NormalizeTarget(raw string) string {
	return normalizeTarget(raw)
}

func (*Adapter) ResolveTarget(userConfig map[string]any) (string, error) {
	return resolveTarget(userConfig)
}

func (a *Adapter) DiscoverSelf(ctx context.Context, credentials map[string]any) (map[string]any, string, error) {
	cfg, err := parseConfig(credentials)
	if err != nil {
		return nil, "", err
	}

	clientFactory := newClient
	if a != nil && a.newClient != nil {
		clientFactory = a.newClient
	}
	c := clientFactory(cfg)

	accessToken, err := c.accessToken(ctx, cfg)
	if err != nil {
		return nil, "", err
	}

	self, err := c.selfIdentity(ctx, accessToken)
	if err != nil {
		return nil, "", err
	}

	identity := map[string]any{}
	if value := strings.TrimSpace(self.AppID); value != "" {
		identity["appId"] = value
	}
	if value := strings.TrimSpace(self.UserID); value != "" {
		identity["chatbotUserId"] = value
	}
	if value := strings.TrimSpace(self.StaffID); value != "" {
		identity["staffId"] = value
	}
	if value := strings.TrimSpace(self.UnionID); value != "" {
		identity["unionId"] = value
	}
	if value := strings.TrimSpace(self.RobotCode); value != "" {
		identity["robotCode"] = value
	}

	return identity, discoverExternalIdentity(self), nil
}

func (a *Adapter) getClient(cfg Config) *client {
	clientFactory := newClient
	if a != nil && a.newClient != nil {
		clientFactory = a.newClient
	}
	return clientFactory(cfg)
}

func (a *Adapter) ResolveAttachment(ctx context.Context, cfg channel.ChannelConfig, attachment channel.Attachment) (channel.AttachmentPayload, error) {
	parsedCfg, err := parseConfig(cfg.Credentials)
	if err != nil {
		return channel.AttachmentPayload{}, err
	}
	return a.resolveAttachment(ctx, parsedCfg, cfg, attachment)
}
