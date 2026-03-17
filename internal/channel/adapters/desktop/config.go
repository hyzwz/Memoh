package desktop

import (
	"errors"
	"strings"

	"github.com/memohai/memoh/internal/channel"
)

// Config holds desktop channel configuration.
type Config struct {
	Hostname string
	Platform string
}

// UserConfig holds desktop user binding configuration.
type UserConfig struct {
	DeviceID string
}

func normalizeConfig(raw map[string]any) (map[string]any, error) {
	cfg, err := parseConfig(raw)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"hostname": cfg.Hostname,
		"platform": cfg.Platform,
	}, nil
}

func normalizeUserConfig(raw map[string]any) (map[string]any, error) {
	cfg, err := parseUserConfig(raw)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"device_id": cfg.DeviceID,
	}, nil
}

func matchBinding(raw map[string]any, criteria channel.BindingCriteria) bool {
	cfg, err := parseUserConfig(raw)
	if err != nil {
		return false
	}
	if criteria.SubjectID != "" && criteria.SubjectID == cfg.DeviceID {
		return true
	}
	if value := strings.TrimSpace(criteria.Attribute("device_id")); value != "" && value == cfg.DeviceID {
		return true
	}
	return false
}

func normalizeTarget(raw string) string {
	value := strings.TrimSpace(raw)
	if value == "" {
		return ""
	}
	// Strip "desktop:" prefix if present
	value = strings.TrimPrefix(value, "desktop:")
	return strings.TrimSpace(value)
}

func resolveTarget(raw map[string]any) (string, error) {
	cfg, err := parseUserConfig(raw)
	if err != nil {
		return "", err
	}
	return cfg.DeviceID, nil
}

func buildUserConfig(identity channel.Identity) map[string]any {
	result := map[string]any{}
	if value := strings.TrimSpace(identity.Attribute("device_id")); value != "" {
		result["device_id"] = value
	}
	return result
}

func parseConfig(raw map[string]any) (Config, error) {
	hostname := strings.TrimSpace(channel.ReadString(raw, "hostname"))
	platform := strings.TrimSpace(channel.ReadString(raw, "platform"))

	if hostname == "" {
		return Config{}, errors.New("desktop hostname is required")
	}

	switch platform {
	case "darwin", "windows", "linux":
		// valid
	case "":
		platform = "unknown"
	default:
		return Config{}, errors.New("desktop platform must be darwin, windows, or linux")
	}

	return Config{
		Hostname: hostname,
		Platform: platform,
	}, nil
}

func parseUserConfig(raw map[string]any) (UserConfig, error) {
	deviceID := strings.TrimSpace(channel.ReadString(raw, "deviceId", "device_id"))
	if deviceID == "" {
		return UserConfig{}, errors.New("desktop user config requires device_id")
	}
	return UserConfig{DeviceID: deviceID}, nil
}
