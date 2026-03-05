package hands

import "time"

// Hand represents a parsed HAND.md definition.
type Hand struct {
	Name        string
	Description string
	Type        string // "hand" or "skill"
	Content     string // Prompt body (after frontmatter)

	Schedule    *ScheduleConfig
	Triggers    []TriggerDef
	AllowedTools []string
	ModelConfig ModelConfig
	OutputConfig OutputConfig
	Permissions PermissionsConfig
	Metadata    map[string]any
}

// ScheduleConfig defines cron-based scheduling.
type ScheduleConfig struct {
	Pattern  string // 6-field cron expression
	Timezone string
	MaxCalls *int   // nil = unlimited
}

// TriggerDef defines an event-based trigger.
type TriggerDef struct {
	Event string // "message_keyword", "webhook", "schedule"
	Match string // pattern to match for event triggers
	Path  string // webhook path
}

// ModelConfig overrides the bot's default model for this hand.
type ModelConfig struct {
	ComplexityTier string // "simple", "medium", "complex"
	Reasoning      *bool
}

// OutputConfig defines where results go.
type OutputConfig struct {
	Channels []string // e.g. ["wecom"]
	Target   string   // e.g. "department_group"
}

// PermissionsConfig defines execution constraints.
type PermissionsConfig struct {
	RequireApproval bool
	MaxTokensPerRun int
}

// TriggerType constants.
const (
	TriggerSchedule = "schedule"
	TriggerEvent    = "event"
	TriggerManual   = "manual"
	TriggerWebhook  = "webhook"
)

// TriggerContext describes what caused a hand to execute.
type TriggerContext struct {
	Type   string
	Detail map[string]any
}

// ExecutionResult captures the outcome of a hand execution.
type ExecutionResult struct {
	Status       string // "completed", "failed", "cancelled"
	ResultText   string
	ErrorMessage string
	Usage        map[string]any
	StartedAt    time.Time
	CompletedAt  time.Time
}
