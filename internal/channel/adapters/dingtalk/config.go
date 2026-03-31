package dingtalk

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/memohai/memoh/internal/channel"
)

const (
	defaultGroupSessionScope     = "group"
	groupSessionScopeGroupSender = "group_sender"
	defaultRequireMentionInGroup = true
	defaultEnableSessionCommands = true
	defaultEnableAICard          = false
	defaultCardUpdateThrottleMS  = 250
)

var defaultSessionResetCommands = []string{"/new", "新会话"}

type Config struct {
	AppKey                string
	AppSecret             string
	RobotCode             string
	StreamEndpoint        string
	APIBaseURL            string
	GroupSessionScope     string
	RequireMentionInGroup bool
	EnableSessionCommands bool
	SessionResetCommands  []string
	EnableAICard          bool
	CardUpdateThrottleMS  int
}

type UserConfig struct {
	ConversationID string
	StaffID        string
	UnionID        string
}

func normalizeConfig(raw map[string]any) (map[string]any, error) {
	cfg, err := parseConfig(raw)
	if err != nil {
		return nil, err
	}

	result := map[string]any{
		"appKey":                cfg.AppKey,
		"appSecret":             cfg.AppSecret,
		"groupSessionScope":     cfg.GroupSessionScope,
		"requireMentionInGroup": cfg.RequireMentionInGroup,
		"enableSessionCommands": cfg.EnableSessionCommands,
		"sessionResetCommands":  append([]string(nil), cfg.SessionResetCommands...),
		"enableAICard":          cfg.EnableAICard,
		"cardUpdateThrottleMs":  cfg.CardUpdateThrottleMS,
	}
	if cfg.RobotCode != "" {
		result["robotCode"] = cfg.RobotCode
	}
	if cfg.StreamEndpoint != "" {
		result["streamEndpoint"] = cfg.StreamEndpoint
	}
	if cfg.APIBaseURL != "" {
		result["apiBaseURL"] = cfg.APIBaseURL
	}

	return result, nil
}

func normalizeUserConfig(raw map[string]any) (map[string]any, error) {
	cfg, err := parseUserConfig(raw)
	if err != nil {
		return nil, err
	}

	result := map[string]any{}
	if cfg.ConversationID != "" {
		result["conversationId"] = cfg.ConversationID
	}
	if cfg.StaffID != "" {
		result["staffId"] = cfg.StaffID
	}
	if cfg.UnionID != "" {
		result["unionId"] = cfg.UnionID
	}
	return result, nil
}

func parseConfig(raw map[string]any) (Config, error) {
	appKey := strings.TrimSpace(channel.ReadString(raw, "appKey", "app_key", "clientId", "client_id"))
	if appKey == "" {
		return Config{}, errors.New("dingtalk appKey is required")
	}

	appSecret := strings.TrimSpace(channel.ReadString(raw, "appSecret", "app_secret", "clientSecret", "client_secret"))
	if appSecret == "" {
		return Config{}, errors.New("dingtalk appSecret is required")
	}

	groupSessionScope, err := normalizeGroupSessionScope(channel.ReadString(raw, "groupSessionScope", "group_session_scope"))
	if err != nil {
		return Config{}, err
	}

	cardUpdateThrottleMS, err := readInt(raw, defaultCardUpdateThrottleMS, "cardUpdateThrottleMs", "card_update_throttle_ms")
	if err != nil {
		return Config{}, err
	}
	if cardUpdateThrottleMS < 0 {
		return Config{}, errors.New("dingtalk cardUpdateThrottleMs must be greater than or equal to 0")
	}

	return Config{
		AppKey:                appKey,
		AppSecret:             appSecret,
		RobotCode:             strings.TrimSpace(channel.ReadString(raw, "robotCode", "robot_code")),
		StreamEndpoint:        strings.TrimSpace(channel.ReadString(raw, "streamEndpoint", "stream_endpoint")),
		APIBaseURL:            strings.TrimRight(strings.TrimSpace(channel.ReadString(raw, "apiBaseURL", "api_base_url")), "/"),
		GroupSessionScope:     groupSessionScope,
		RequireMentionInGroup: readBool(raw, defaultRequireMentionInGroup, "requireMentionInGroup", "require_mention_in_group"),
		EnableSessionCommands: readBool(raw, defaultEnableSessionCommands, "enableSessionCommands", "enable_session_commands"),
		SessionResetCommands:  normalizeResetCommands(channel.ReadString(raw, "sessionResetCommands", "session_reset_commands")),
		EnableAICard:          readBool(raw, defaultEnableAICard, "enableAICard", "enable_ai_card"),
		CardUpdateThrottleMS:  cardUpdateThrottleMS,
	}, nil
}

func parseUserConfig(raw map[string]any) (UserConfig, error) {
	cfg := UserConfig{
		ConversationID: strings.TrimSpace(channel.ReadString(raw, "conversationId", "conversation_id")),
		StaffID:        strings.TrimSpace(channel.ReadString(raw, "staffId", "staff_id")),
		UnionID:        strings.TrimSpace(channel.ReadString(raw, "unionId", "union_id")),
	}
	if cfg.ConversationID == "" && cfg.StaffID == "" && cfg.UnionID == "" {
		return UserConfig{}, errors.New("dingtalk user config requires conversationId, staffId, or unionId")
	}
	return cfg, nil
}

func normalizeGroupSessionScope(raw string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", defaultGroupSessionScope:
		return defaultGroupSessionScope, nil
	case groupSessionScopeGroupSender:
		return groupSessionScopeGroupSender, nil
	default:
		return "", errors.New("dingtalk groupSessionScope must be group or group_sender")
	}
}

func normalizeResetCommands(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return append([]string(nil), defaultSessionResetCommands...)
	}

	seen := make(map[string]struct{})
	commands := make([]string, 0, len(defaultSessionResetCommands))
	for _, part := range strings.Split(raw, ",") {
		command := strings.TrimSpace(part)
		if command == "" {
			continue
		}
		if _, ok := seen[command]; ok {
			continue
		}
		seen[command] = struct{}{}
		commands = append(commands, command)
	}
	if len(commands) == 0 {
		return append([]string(nil), defaultSessionResetCommands...)
	}
	return commands
}

func readBool(raw map[string]any, fallback bool, keys ...string) bool {
	for _, key := range keys {
		value, ok := raw[key]
		if !ok {
			continue
		}
		switch v := value.(type) {
		case bool:
			return v
		case string:
			switch strings.ToLower(strings.TrimSpace(v)) {
			case "true", "1", "yes", "on":
				return true
			case "false", "0", "no", "off":
				return false
			}
		}
	}
	return fallback
}

func readInt(raw map[string]any, fallback int, keys ...string) (int, error) {
	for _, key := range keys {
		value, ok := raw[key]
		if !ok {
			continue
		}
		switch v := value.(type) {
		case int:
			return v, nil
		case int32:
			return int(v), nil
		case int64:
			return int(v), nil
		case float64:
			if v != float64(int(v)) {
				return 0, fmt.Errorf("dingtalk %s must be a whole number", key)
			}
			return int(v), nil
		case string:
			parsed, err := strconv.Atoi(strings.TrimSpace(v))
			if err != nil {
				return 0, fmt.Errorf("dingtalk %s must be a number", key)
			}
			return parsed, nil
		default:
			return 0, fmt.Errorf("dingtalk %s must be a number", key)
		}
	}
	return fallback, nil
}
