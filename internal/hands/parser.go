package hands

import (
	"strings"

	"gopkg.in/yaml.v3"
)

// handFrontmatter is the YAML structure parsed from HAND.md frontmatter.
type handFrontmatter struct {
	Name        string         `yaml:"name"`
	Description string         `yaml:"description"`
	Type        string         `yaml:"type"`
	Schedule    *scheduleFM    `yaml:"schedule"`
	Triggers    []triggerFM    `yaml:"triggers"`
	Tools       []string       `yaml:"tools"`
	Model       *modelFM       `yaml:"model"`
	Output      *outputFM      `yaml:"output"`
	Permissions *permissionsFM `yaml:"permissions"`
	Metadata    map[string]any `yaml:"metadata"`
}

type scheduleFM struct {
	Pattern  string `yaml:"pattern"`
	Timezone string `yaml:"timezone"`
	MaxCalls *int   `yaml:"max_calls"`
}

type triggerFM struct {
	Event string `yaml:"event"`
	Match string `yaml:"match"`
	Path  string `yaml:"path"`
}

type modelFM struct {
	ComplexityTier string `yaml:"complexity_tier"`
	Reasoning      *bool  `yaml:"reasoning"`
}

type outputFM struct {
	Channels []string `yaml:"channels"`
	Target   string   `yaml:"target"`
}

type permissionsFM struct {
	RequireApproval bool `yaml:"require_approval"`
	MaxTokensPerRun int  `yaml:"max_tokens_per_run"`
}

// ParseHandMD parses a HAND.md (or SKILL.md) string into a Hand struct.
// fallbackName is used if no name is specified in the frontmatter.
func ParseHandMD(raw string, fallbackName string) Hand {
	trimmed := strings.TrimSpace(raw)

	h := Hand{
		Name: fallbackName,
		Type: "skill",
	}

	if !strings.HasPrefix(trimmed, "---") {
		h.Content = trimmed
		return h
	}

	// Split frontmatter from body.
	rest := trimmed[3:]
	rest = strings.TrimLeft(rest, " \t")
	if len(rest) > 0 && rest[0] == '\n' {
		rest = rest[1:]
	} else if len(rest) > 1 && rest[0] == '\r' && rest[1] == '\n' {
		rest = rest[2:]
	}

	closingIdx := strings.Index(rest, "\n---")
	if closingIdx < 0 {
		h.Content = trimmed
		return h
	}

	fmRaw := rest[:closingIdx]
	body := rest[closingIdx+4:]
	body = strings.TrimLeft(body, "\r\n")
	h.Content = strings.TrimSpace(body)

	var fm handFrontmatter
	if err := yaml.Unmarshal([]byte(fmRaw), &fm); err != nil {
		// Invalid YAML — keep body, return defaults.
		return h
	}

	// Apply parsed values.
	if strings.TrimSpace(fm.Name) != "" {
		h.Name = strings.TrimSpace(fm.Name)
	}
	h.Description = strings.TrimSpace(fm.Description)

	if fm.Type == "hand" {
		h.Type = "hand"
	}

	// Schedule
	if fm.Schedule != nil && fm.Schedule.Pattern != "" {
		tz := fm.Schedule.Timezone
		if tz == "" {
			tz = "UTC"
		}
		h.Schedule = &ScheduleConfig{
			Pattern:  fm.Schedule.Pattern,
			Timezone: tz,
			MaxCalls: fm.Schedule.MaxCalls,
		}
	}

	// Triggers
	for _, tr := range fm.Triggers {
		h.Triggers = append(h.Triggers, TriggerDef{
			Event: tr.Event,
			Match: tr.Match,
			Path:  tr.Path,
		})
	}

	// Tools
	h.AllowedTools = fm.Tools

	// Model
	if fm.Model != nil {
		h.ModelConfig = ModelConfig{
			ComplexityTier: fm.Model.ComplexityTier,
			Reasoning:      fm.Model.Reasoning,
		}
	}

	// Output
	if fm.Output != nil {
		h.OutputConfig = OutputConfig{
			Channels: fm.Output.Channels,
			Target:   fm.Output.Target,
		}
	}

	// Permissions
	if fm.Permissions != nil {
		h.Permissions = PermissionsConfig{
			RequireApproval: fm.Permissions.RequireApproval,
			MaxTokensPerRun: fm.Permissions.MaxTokensPerRun,
		}
	}

	// Metadata
	h.Metadata = fm.Metadata

	return h
}
