package ctripcs

import (
	"encoding/json"
	"errors"
	"fmt"
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

	result := map[string]any{
		"browserContextId": cfg.BrowserContextID,
		"entryUrl":         cfg.EntryURL,
		"accountLabel":     cfg.AccountLabel,
		"pollIntervalMs":   cfg.PollIntervalMS,
		"inboxPageUrl":     cfg.InboxPageURL,
	}
	return result, nil
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
	entryURL, err := validateCtripEntryURL(entryURLRaw)
	if err != nil {
		return Config{}, err
	}

	accountLabel := strings.TrimSpace(channel.ReadString(raw, "accountLabel", "account_label"))
	pollIntervalMS := readInt(raw, defaultPollIntervalMS, "pollIntervalMs", "poll_interval_ms")
	if pollIntervalMS <= 0 {
		pollIntervalMS = defaultPollIntervalMS
	}

	inboxPageURL := strings.TrimSpace(channel.ReadString(raw, "inboxPageUrl", "inbox_page_url", "messagePageUrl", "message_page_url"))
	if inboxPageURL == "" {
		inboxPageURL = entryURL
	} else {
		inboxPageURL, err = validateCtripEntryURL(inboxPageURL)
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

func validateCtripEntryURL(raw string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return "", fmt.Errorf("ctrip_cs entryUrl must be a valid URL: %w", err)
	}
	scheme := strings.ToLower(strings.TrimSpace(parsed.Scheme))
	if scheme != "http" && scheme != "https" {
		return "", errors.New("ctrip_cs entryUrl must use http or https")
	}
	host := strings.ToLower(strings.TrimSpace(parsed.Hostname()))
	if !isCtripHost(host) {
		return "", errors.New("ctrip_cs entryUrl must be under a ctrip host")
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

func readInt(raw map[string]any, fallback int, keys ...string) int {
	for _, key := range keys {
		value, ok := raw[key]
		if !ok {
			continue
		}
		switch v := value.(type) {
		case int:
			return v
		case int8:
			return int(v)
		case int16:
			return int(v)
		case int32:
			return int(v)
		case int64:
			return int(v)
		case float32:
			if float32(int(v)) == v {
				return int(v)
			}
		case float64:
			if float64(int(v)) == v {
				return int(v)
			}
		case json.Number:
			if n, err := strconv.Atoi(v.String()); err == nil {
				return n
			}
		case string:
			if n, err := strconv.Atoi(strings.TrimSpace(v)); err == nil {
				return n
			}
		}
	}
	return fallback
}
