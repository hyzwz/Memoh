package rbac

// Role represents a system-wide user role.
type Role string

const (
	RoleMember     Role = "member"
	RoleDeptAdmin  Role = "dept_admin"
	RoleOrgAdmin   Role = "org_admin"
	RoleSuperAdmin Role = "super_admin"

	// Legacy role mapping: existing "admin" users are treated as super_admin.
	RoleLegacyAdmin Role = "admin"
)

// BotRole represents a user's role within a specific bot.
type BotRole string

const (
	BotRoleOwner  BotRole = "owner"
	BotRoleAdmin  BotRole = "admin"
	BotRoleMember BotRole = "member"
)

// Resource represents a protected resource type.
type Resource string

const (
	ResourceTools    Resource = "tools"
	ResourceSkills   Resource = "skills"
	ResourceSettings Resource = "settings"
	ResourceMembers  Resource = "members"
	ResourceFiles    Resource = "files"
	ResourceChat     Resource = "chat"
)

// Action represents an operation on a resource.
type Action string

const (
	ActionRead    Action = "read"
	ActionWrite   Action = "write"
	ActionExecute Action = "execute"
	ActionAdmin   Action = "admin"
)

// UserContext holds resolved role information for a user.
type UserContext struct {
	UserID        string
	SystemRole    Role
	Departments   []DepartmentMembership
	BotMembership map[string]BotRole // botID -> role
}

// DepartmentMembership represents a user's membership in a department.
type DepartmentMembership struct {
	DepartmentID string
	Role         string // "admin" or "member"
}

// Permission represents a fine-grained permission rule.
type Permission struct {
	BotID    string
	Role     BotRole
	Resource Resource
	Action   Action
	Allowed  bool
}
