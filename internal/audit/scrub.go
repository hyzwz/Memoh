package audit

import (
	"strings"
)

// sensitiveKeys are detail keys whose values should be fully redacted.
var sensitiveKeys = map[string]bool{
	"password":      true,
	"secret":        true,
	"token":         true,
	"access_token":  true,
	"refresh_token": true,
}

// maskKeys are detail keys whose values should be partially masked.
var maskKeys = map[string]bool{
	"api_key": true,
	"apikey":  true,
}

const maxContentLength = 1024

// ScrubDetail returns a copy of detail with sensitive values redacted.
// Recursively scrubs nested maps.
func ScrubDetail(detail map[string]any) map[string]any {
	if detail == nil {
		return nil
	}
	return scrubMap(detail)
}

func scrubMap(m map[string]any) map[string]any {
	scrubbed := make(map[string]any, len(m))
	for k, v := range m {
		scrubbed[k] = scrubValue(k, v)
	}
	return scrubbed
}

func scrubValue(key string, v any) any {
	keyLower := strings.ToLower(key)

	if sensitiveKeys[keyLower] {
		return "[REDACTED]"
	}

	switch val := v.(type) {
	case string:
		if maskKeys[keyLower] {
			return maskAPIKey(val)
		}
		if len(val) > maxContentLength {
			return val[:maxContentLength] + "[truncated]"
		}
		return val
	case map[string]any:
		return scrubMap(val)
	case []any:
		return scrubSlice(val)
	default:
		return v
	}
}

func scrubSlice(s []any) []any {
	result := make([]any, len(s))
	for i, v := range s {
		switch val := v.(type) {
		case map[string]any:
			result[i] = scrubMap(val)
		default:
			result[i] = val
		}
	}
	return result
}

func maskAPIKey(key string) string {
	if len(key) <= 8 {
		return strings.Repeat("*", len(key))
	}
	return key[:8] + strings.Repeat("*", len(key)-8)
}
