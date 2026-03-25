package ctripcs

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/memohai/memoh/internal/channel"
)

const (
	defaultPollIntervalMS = 1500
	defaultEntryURL       = "https://m.ctrip.com/"
)

type Config struct {
	BrowserContextID string
	EntryURL         string
	AccountLabel     string
	PollIntervalMS   int
	InboxPageURL     string
}

func normalizeConfig(raw map[string]any) (map[string]any, error) {
	cfg, err := parseConfig(raw)
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"browserContextId": cfg.BrowserContextID,
		"entryUrl":         cfg.EntryURL,
		"accountLabel":     cfg.AccountLabel,
		"pollIntervalMs":   cfg.PollIntervalMS,
		"inboxPageUrl":     cfg.InboxPageURL,
	}, nil
}

func parseConfig(raw map[string]any) (Config, error) {
	browserContextID := strings.TrimSpace(channel.ReadString(raw, "browserContextId", "browser_context_id"))
	if browserContextID == "" {
		return Config{}, errors.New("ctrip_cs browserContextId is required")
	}

	entryURLRaw := strings.TrimSpace(channel.ReadString(raw, "entryUrl", "entry_url"))
	if entryURLRaw == "" {
		entryURLRaw = defaultEntryURL
	}
	entryURL, err := validateCtripURL(entryURLRaw, "entryUrl")
	if err != nil {
		return Config{}, err
	}

	accountLabel := strings.TrimSpace(channel.ReadString(raw, "accountLabel", "account_label"))
	pollIntervalMS, err := parsePollIntervalMS(raw)
	if err != nil {
		return Config{}, err
	}

	inboxPageURL := strings.TrimSpace(channel.ReadString(raw, "inboxPageUrl", "inbox_page_url", "messagePageUrl", "message_page_url"))
	if inboxPageURL == "" {
		inboxPageURL = entryURL
	} else {
		inboxPageURL, err = validateCtripURL(inboxPageURL, "inboxPageUrl")
		if err != nil {
			return Config{}, err
		}
	}

	return Config{
		BrowserContextID: browserContextID,
		EntryURL:         entryURL,
		AccountLabel:     accountLabel,
		PollIntervalMS:   pollIntervalMS,
		InboxPageURL:     inboxPageURL,
	}, nil
}

func validateCtripURL(raw, fieldName string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return "", fmt.Errorf("ctrip_cs %s must be a valid URL: %w", fieldName, err)
	}
	scheme := strings.ToLower(strings.TrimSpace(parsed.Scheme))
	if scheme != "http" && scheme != "https" {
		return "", fmt.Errorf("ctrip_cs %s must use http or https", fieldName)
	}
	host := strings.ToLower(strings.TrimSpace(parsed.Hostname()))
	if !isCtripHost(host) {
		return "", fmt.Errorf("ctrip_cs %s must be under a ctrip host", fieldName)
	}
	return parsed.String(), nil
}

func isCtripHost(host string) bool {
	switch {
	case host == "ctrip.com", host == "ctrip.cn":
		return true
	case strings.HasSuffix(host, ".ctrip.com"), strings.HasSuffix(host, ".ctrip.cn"):
		return true
	default:
		return false
	}
}

func parsePollIntervalMS(raw map[string]any) (int, error) {
	value, found := lookupRawValue(raw, "pollIntervalMs", "poll_interval_ms")
	if !found {
		return defaultPollIntervalMS, nil
	}

	interval, err := strictInt(value)
	if err != nil {
		return 0, fmt.Errorf("ctrip_cs pollIntervalMs must be a positive integer: %w", err)
	}
	if interval <= 0 {
		return 0, errors.New("ctrip_cs pollIntervalMs must be a positive integer")
	}
	return interval, nil
}

func lookupRawValue(raw map[string]any, keys ...string) (any, bool) {
	for _, key := range keys {
		value, ok := raw[key]
		if !ok {
			continue
		}
		return value, true
	}
	return nil, false
}

func strictInt(value any) (int, error) {
	switch v := value.(type) {
	case int:
		return v, nil
	case int8:
		return int(v), nil
	case int16:
		return int(v), nil
	case int32:
		return int(v), nil
	case int64:
		return int(v), nil
	case float32:
		if float32(int(v)) != v {
			return 0, errors.New("fractional value is not allowed")
		}
		return int(v), nil
	case float64:
		if float64(int(v)) != v {
			return 0, errors.New("fractional value is not allowed")
		}
		return int(v), nil
	case json.Number:
		if strings.Contains(v.String(), ".") {
			return 0, errors.New("fractional value is not allowed")
		}
		return strictInt(v.String())
	case string:
		value := strings.TrimSpace(v)
		if value == "" {
			return 0, errors.New("empty value is not allowed")
		}
		if strings.Contains(value, ".") {
			return 0, errors.New("fractional value is not allowed")
		}
		return parseDecimalInt(value)
	default:
		return 0, fmt.Errorf("unsupported type %T", value)
	}
}

func parseDecimalInt(raw string) (int, error) {
	if raw == "" {
		return 0, errors.New("empty value is not allowed")
	}

	sign := 1
	start := 0
	if raw[0] == '-' {
		sign = -1
		start = 1
		if len(raw) == 1 {
			return 0, errors.New("invalid integer value")
		}
	}

	n := 0
	for _, r := range raw[start:] {
		if r < '0' || r > '9' {
			return 0, errors.New("invalid integer value")
		}
		n = n*10 + int(r-'0')
	}
	return sign * n, nil
}
