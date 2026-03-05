package wecom

import (
	"fmt"
	"strings"

	"github.com/memohai/memoh/internal/channel"
)

// Config holds the WeCom app credentials extracted from a channel configuration.
type Config struct {
	CorpID         string
	AgentID        string
	Secret         string
	Token          string
	EncodingAESKey string
}

func parseConfig(raw map[string]any) (Config, error) {
	corpID := strings.TrimSpace(channel.ReadString(raw, "corpId", "corp_id"))
	agentID := strings.TrimSpace(channel.ReadString(raw, "agentId", "agent_id"))
	secret := strings.TrimSpace(channel.ReadString(raw, "secret"))
	token := strings.TrimSpace(channel.ReadString(raw, "token"))
	encodingAESKey := strings.TrimSpace(channel.ReadString(raw, "encodingAESKey", "encoding_aes_key"))

	if corpID == "" {
		return Config{}, fmt.Errorf("wecom corpId is required")
	}
	if secret == "" {
		return Config{}, fmt.Errorf("wecom secret is required")
	}
	if token == "" {
		return Config{}, fmt.Errorf("wecom token is required")
	}
	if encodingAESKey == "" {
		return Config{}, fmt.Errorf("wecom encodingAESKey is required")
	}

	return Config{
		CorpID:         corpID,
		AgentID:        agentID,
		Secret:         secret,
		Token:          token,
		EncodingAESKey: encodingAESKey,
	}, nil
}

func normalizeConfig(raw map[string]any) (map[string]any, error) {
	cfg, err := parseConfig(raw)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"corpId":         cfg.CorpID,
		"agentId":        cfg.AgentID,
		"secret":         cfg.Secret,
		"token":          cfg.Token,
		"encodingAESKey": cfg.EncodingAESKey,
	}, nil
}
