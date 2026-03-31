package dingtalk

import "strings"

const sessionCommandReset = "reset"

type SessionCommand struct {
	Name              string
	Raw               string
	Normalized        string
	RouteKey          string
	ReplyTarget       string
	ResetCurrentRoute bool
}

func detectSessionCommand(cfg Config, text, routeKey, replyTarget string) *SessionCommand {
	if !cfg.EnableSessionCommands {
		return nil
	}

	normalizedText := normalizeSessionCommandText(text)
	if normalizedText == "" {
		return nil
	}

	for _, candidate := range effectiveSessionResetCommands(cfg) {
		if normalizedText != candidate {
			continue
		}
		return &SessionCommand{
			Name:              sessionCommandReset,
			Raw:               text,
			Normalized:        normalizedText,
			RouteKey:          strings.TrimSpace(routeKey),
			ReplyTarget:       strings.TrimSpace(replyTarget),
			ResetCurrentRoute: true,
		}
	}

	return nil
}

func effectiveSessionResetCommands(cfg Config) []string {
	rawCommands := cfg.SessionResetCommands
	if len(rawCommands) == 0 {
		rawCommands = defaultSessionResetCommands
	}

	seen := make(map[string]struct{}, len(rawCommands))
	commands := make([]string, 0, len(rawCommands))
	for _, raw := range rawCommands {
		command := normalizeSessionCommandText(raw)
		if command == "" {
			continue
		}
		if _, ok := seen[command]; ok {
			continue
		}
		seen[command] = struct{}{}
		commands = append(commands, command)
	}
	return commands
}

func normalizeSessionCommandText(text string) string {
	return strings.TrimSpace(text)
}
