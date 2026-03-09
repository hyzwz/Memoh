package wecombot

import (
	"errors"
	"strings"

	"github.com/memohai/memoh/internal/channel"
)

const defaultWebSocketURL = "wss://openws.work.weixin.qq.com"

type Config struct {
	BotID              string
	Secret             string
	WebSocketURL       string
	SendThinkingPrompt bool
}

type UserConfig struct {
	UserID string
	ChatID string
}

func normalizeConfig(raw map[string]any) (map[string]any, error) {
	cfg, err := parseConfig(raw)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"botId":              cfg.BotID,
		"secret":             cfg.Secret,
		"websocketUrl":       cfg.WebSocketURL,
		"sendThinkingPrompt": cfg.SendThinkingPrompt,
	}, nil
}

func normalizeUserConfig(raw map[string]any) (map[string]any, error) {
	cfg, err := parseUserConfig(raw)
	if err != nil {
		return nil, err
	}
	normalized := map[string]any{}
	if cfg.UserID != "" {
		normalized["userid"] = cfg.UserID
	}
	if cfg.ChatID != "" {
		normalized["chatid"] = cfg.ChatID
	}
	return normalized, nil
}

func resolveTarget(raw map[string]any) (string, error) {
	cfg, err := parseUserConfig(raw)
	if err != nil {
		return "", err
	}
	if cfg.ChatID != "" {
		return "chatid:" + cfg.ChatID, nil
	}
	return "userid:" + cfg.UserID, nil
}

func matchBinding(raw map[string]any, criteria channel.BindingCriteria) bool {
	cfg, err := parseUserConfig(raw)
	if err != nil {
		return false
	}
	subjectID := strings.TrimSpace(criteria.SubjectID)
	if subjectID != "" && subjectID == cfg.UserID {
		return true
	}
	for _, key := range []string{"userid", "user_id"} {
		if value := strings.TrimSpace(criteria.Attribute(key)); value != "" && value == cfg.UserID {
			return true
		}
	}
	if cfg.ChatID != "" {
		for _, key := range []string{"chatid", "chat_id"} {
			if value := strings.TrimSpace(criteria.Attribute(key)); value != "" && value == cfg.ChatID {
				return true
			}
		}
	}
	return false
}

func buildUserConfig(identity channel.Identity) map[string]any {
	userID := strings.TrimSpace(identity.Attribute("userid"))
	if userID == "" {
		userID = strings.TrimSpace(identity.Attribute("user_id"))
	}
	if userID == "" {
		userID = strings.TrimSpace(identity.SubjectID)
	}
	if userID == "" {
		return map[string]any{}
	}
	cfg := map[string]any{"userid": userID}
	chatType := strings.ToLower(strings.TrimSpace(identity.Attribute("chat_type")))
	if chatType == "direct" || chatType == "p2p" || chatType == "private" || chatType == "single" {
		if chatID := strings.TrimSpace(identity.Attribute("chatid")); chatID != "" {
			cfg["chatid"] = chatID
		}
	}
	return cfg
}

func parseConfig(raw map[string]any) (Config, error) {
	botID := strings.TrimSpace(channel.ReadString(raw, "botId", "bot_id"))
	secret := strings.TrimSpace(channel.ReadString(raw, "secret"))
	websocketURL := strings.TrimSpace(channel.ReadString(raw, "websocketUrl", "websocket_url", "wsUrl", "ws_url"))
	if websocketURL == "" {
		websocketURL = defaultWebSocketURL
	}
	if botID == "" {
		return Config{}, errors.New("wecom_ai_bot botId is required")
	}
	if secret == "" {
		return Config{}, errors.New("wecom_ai_bot secret is required")
	}
	return Config{
		BotID:              botID,
		Secret:             secret,
		WebSocketURL:       websocketURL,
		SendThinkingPrompt: readBool(raw, true, "sendThinkingPrompt", "send_thinking_prompt"),
	}, nil
}

func parseUserConfig(raw map[string]any) (UserConfig, error) {
	userID := strings.TrimSpace(channel.ReadString(raw, "userid", "userId", "user_id"))
	chatID := strings.TrimSpace(channel.ReadString(raw, "chatid", "chatId", "chat_id"))
	if userID == "" && chatID == "" {
		return UserConfig{}, errors.New("wecom_ai_bot user config requires userid or chatid")
	}
	return UserConfig{UserID: userID, ChatID: chatID}, nil
}

func normalizeTarget(raw string) string {
	value := strings.TrimSpace(raw)
	if value == "" {
		return ""
	}
	for _, prefix := range []string{"wecom_ai_bot:", "wecombot:", "wecom:"} {
		if strings.HasPrefix(strings.ToLower(value), prefix) {
			value = strings.TrimSpace(value[len(prefix):])
			break
		}
	}
	switch {
	case strings.HasPrefix(strings.ToLower(value), "userid:"):
		return "userid:" + strings.TrimSpace(value[len("userid:"):])
	case strings.HasPrefix(strings.ToLower(value), "user:"):
		return "userid:" + strings.TrimSpace(value[len("user:"):])
	case strings.HasPrefix(strings.ToLower(value), "chatid:"):
		return "chatid:" + strings.TrimSpace(value[len("chatid:"):])
	case strings.HasPrefix(strings.ToLower(value), "chat:"):
		return "chatid:" + strings.TrimSpace(value[len("chat:"):])
	default:
		return "userid:" + value
	}
}

func parseTarget(raw string) (targetType, string, error) {
	value := normalizeTarget(raw)
	switch {
	case strings.HasPrefix(value, "userid:"):
		id := strings.TrimSpace(value[len("userid:"):])
		if id == "" {
			return "", "", errors.New("wecom_ai_bot userid target is empty")
		}
		return targetUserID, id, nil
	case strings.HasPrefix(value, "chatid:"):
		id := strings.TrimSpace(value[len("chatid:"):])
		if id == "" {
			return "", "", errors.New("wecom_ai_bot chatid target is empty")
		}
		return targetChatID, id, nil
	default:
		return "", "", errors.New("wecom_ai_bot target must start with userid: or chatid:")
	}
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
