package dingtalk

import (
	"errors"
	"net/url"
	"strings"
)

const (
	targetFamilyWebhook      = "webhook"
	targetFamilyConversation = "conversation"
	targetFamilyUser         = "user"
	targetUserSubtypeStaff   = "staff"
	targetUserSubtypeUnion   = "union"
)

type Target struct {
	Family      string
	UserSubtype string
	Value       string
}

func resolveTarget(raw map[string]any) (string, error) {
	cfg, err := parseUserConfig(raw)
	if err != nil {
		return "", err
	}
	switch {
	case cfg.ConversationID != "":
		return targetFamilyConversation + ":" + cfg.ConversationID, nil
	case cfg.StaffID != "":
		return targetFamilyUser + ":" + targetUserSubtypeStaff + ":" + cfg.StaffID, nil
	case cfg.UnionID != "":
		return targetFamilyUser + ":" + targetUserSubtypeUnion + ":" + cfg.UnionID, nil
	default:
		return "", errors.New("dingtalk binding is incomplete")
	}
}

func normalizeTarget(raw string) string {
	target, err := parseTarget(raw)
	if err != nil {
		return ""
	}
	if target.Family == targetFamilyUser {
		return target.Family + ":" + target.UserSubtype + ":" + target.Value
	}
	return target.Family + ":" + target.Value
}

func parseTarget(raw string) (Target, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return Target{}, errors.New("dingtalk target is required")
	}

	family, payload, ok := strings.Cut(value, ":")
	if !ok {
		return Target{}, errors.New("dingtalk target must include a family prefix")
	}

	family = strings.ToLower(strings.TrimSpace(family))
	payload = strings.TrimSpace(payload)
	if payload == "" {
		return Target{}, errors.New("dingtalk target payload is required")
	}

	switch family {
	case targetFamilyWebhook:
		return parseWebhookTarget(payload)
	case targetFamilyConversation:
		return parseConversationTarget(payload)
	case targetFamilyUser:
		return parseUserTarget(payload)
	default:
		return Target{}, errors.New("dingtalk target family must be webhook, conversation, or user")
	}
}

func parseWebhookTarget(raw string) (Target, error) {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return Target{}, errors.New("dingtalk webhook target must be an absolute URL")
	}
	switch strings.ToLower(parsed.Scheme) {
	case "http", "https":
	default:
		return Target{}, errors.New("dingtalk webhook target must use http or https")
	}
	return Target{Family: targetFamilyWebhook, Value: parsed.String()}, nil
}

func parseConversationTarget(raw string) (Target, error) {
	if strings.TrimSpace(raw) == "" {
		return Target{}, errors.New("dingtalk conversation target requires conversationId")
	}
	return Target{Family: targetFamilyConversation, Value: raw}, nil
}

func parseUserTarget(raw string) (Target, error) {
	subtype, identifier, ok := strings.Cut(raw, ":")
	if !ok {
		return Target{}, errors.New("dingtalk user target requires subtype and identifier")
	}

	subtype = strings.ToLower(strings.TrimSpace(subtype))
	identifier = strings.TrimSpace(identifier)
	if identifier == "" || strings.Contains(identifier, ":") || strings.ContainsAny(identifier, " \t\r\n") {
		return Target{}, errors.New("dingtalk user target requires a single identifier")
	}

	switch subtype {
	case targetUserSubtypeStaff, targetUserSubtypeUnion:
		return Target{Family: targetFamilyUser, UserSubtype: subtype, Value: identifier}, nil
	default:
		return Target{}, errors.New("dingtalk user target subtype must be staff or union")
	}
}
