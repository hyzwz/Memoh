package audit

import "time"

// Entry represents a single audit log entry.
type Entry struct {
	UserID       string
	BotID        string
	Action       string
	ResourceType string
	ResourceID   string
	Detail       map[string]any
	IPAddress    string
	UserAgent    string
	CreatedAt    time.Time
}

// Common audit actions.
const (
	ActionLogin        = "login"
	ActionUserCreate   = "user_create"
	ActionUserUpdate   = "user_update"
	ActionBotCreate    = "bot_create"
	ActionBotUpdate    = "bot_update"
	ActionBotDelete    = "bot_delete"
	ActionMemberAdd    = "member_add"
	ActionMemberRemove = "member_remove"
	ActionChat         = "chat"
	ActionToolCall     = "tool_call"
	ActionSkillInvoke  = "skill_invoke"
	ActionSettingChange = "setting_change"
	ActionHandExecute  = "hand_execute"
)
