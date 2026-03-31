package ctripcs

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strconv"
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
	if accountLabel == "" {
		return Config{}, errors.New("ctrip_cs accountLabel is required")
	}
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
	if isLocalSimulatorHost(host) {
		return true
	}

	switch {
	case host == "ctrip.com", host == "ctrip.cn":
		return true
	case strings.HasSuffix(host, ".ctrip.com"), strings.HasSuffix(host, ".ctrip.cn"):
		return true
	default:
		return false
	}
}

func isLocalSimulatorHost(host string) bool {
	host = strings.ToLower(strings.TrimSpace(host))
	if net.ParseIP(host) != nil {
		return true
	}

	switch host {
	case "localhost", "127.0.0.1", "::1":
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
		return intFromInt64(int64(v))
	case int8:
		return intFromInt64(int64(v))
	case int16:
		return intFromInt64(int64(v))
	case int32:
		return intFromInt64(int64(v))
	case int64:
		return intFromInt64(v)
	case float32:
		if float32(int(v)) != v {
			return 0, errors.New("fractional value is not allowed")
		}
		return intFromInt64(int64(v))
	case float64:
		if float64(int(v)) != v {
			return 0, errors.New("fractional value is not allowed")
		}
		return intFromInt64(int64(v))
	case json.Number:
		return parseStrictIntString(v.String())
	case string:
		return parseStrictIntString(v)
	default:
		return 0, fmt.Errorf("unsupported type %T", value)
	}
}

func parseStrictIntString(raw string) (int, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return 0, errors.New("empty value is not allowed")
	}
	if strings.Contains(value, ".") {
		return 0, errors.New("fractional value is not allowed")
	}
	n, err := strconv.ParseInt(value, 10, strconv.IntSize)
	if err != nil {
		return 0, err
	}
	return intFromInt64(n)
}

func intFromInt64(v int64) (int, error) {
	result := int(v)
	if int64(result) != v {
		return 0, errors.New("integer value is out of range")
	}
	return result, nil
}
