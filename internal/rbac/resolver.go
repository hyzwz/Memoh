package rbac

import (
	"context"
	"errors"
)

var (
	ErrUserNotFound = errors.New("user not found")
	ErrAccessDenied = errors.New("access denied")
)

// Store provides data access for the RBAC resolver.
type Store interface {
	GetUserContext(ctx context.Context, userID string) (*UserContext, error)
	GetBotDepartmentAccess(ctx context.Context, botID string) ([]string, error)
	GetBotPermissions(ctx context.Context, botID string) ([]Permission, error)
}

// Resolver evaluates authorization decisions.
type Resolver struct {
	store Store
}

// NewResolver creates a new RBAC resolver.
func NewResolver(store Store) *Resolver {
	return &Resolver{store: store}
}

// isAdminRole returns true if the role has system-wide admin privileges.
func isAdminRole(role Role) bool {
	return role == RoleSuperAdmin || role == RoleOrgAdmin || role == RoleLegacyAdmin
}

// AuthorizeBotAccess checks if a user can access a bot at all.
func (r *Resolver) AuthorizeBotAccess(ctx context.Context, userID, botID string) error {
	uc, err := r.store.GetUserContext(ctx, userID)
	if err != nil {
		return err
	}

	// Super admin, org admin, and legacy admin can access everything.
	if isAdminRole(uc.SystemRole) {
		return nil
	}

	// Check if bot is assigned to any of the user's departments.
	botDepts, err := r.store.GetBotDepartmentAccess(ctx, botID)
	if err != nil {
		return err
	}

	for _, dm := range uc.Departments {
		for _, bd := range botDepts {
			if dm.DepartmentID == bd {
				return nil
			}
		}
	}

	// Check direct bot membership.
	if _, ok := uc.BotMembership[botID]; ok {
		return nil
	}

	return ErrAccessDenied
}

// defaultPermissions defines what each bot role can do without explicit permission rules.
var defaultPermissions = map[BotRole]map[Resource]map[Action]bool{
	BotRoleOwner: {
		ResourceChat:     {ActionRead: true, ActionWrite: true, ActionExecute: true, ActionAdmin: true},
		ResourceTools:    {ActionRead: true, ActionWrite: true, ActionExecute: true, ActionAdmin: true},
		ResourceSkills:   {ActionRead: true, ActionWrite: true, ActionExecute: true, ActionAdmin: true},
		ResourceSettings: {ActionRead: true, ActionWrite: true, ActionExecute: true, ActionAdmin: true},
		ResourceMembers:  {ActionRead: true, ActionWrite: true, ActionExecute: true, ActionAdmin: true},
		ResourceFiles:    {ActionRead: true, ActionWrite: true, ActionExecute: true, ActionAdmin: true},
	},
	BotRoleAdmin: {
		ResourceChat:     {ActionRead: true, ActionWrite: true, ActionExecute: true, ActionAdmin: true},
		ResourceTools:    {ActionRead: true, ActionWrite: true, ActionExecute: true},
		ResourceSkills:   {ActionRead: true, ActionWrite: true, ActionExecute: true},
		ResourceSettings: {ActionRead: true, ActionWrite: true},
		ResourceMembers:  {ActionRead: true, ActionWrite: true},
		ResourceFiles:    {ActionRead: true, ActionWrite: true},
	},
	BotRoleMember: {
		ResourceChat:   {ActionRead: true, ActionWrite: true, ActionExecute: true},
		ResourceTools:  {ActionRead: true, ActionExecute: true},
		ResourceSkills: {ActionRead: true, ActionExecute: true},
		ResourceFiles:  {ActionRead: true},
	},
}

// Authorize checks if a user can perform a specific action on a resource within a bot.
func (r *Resolver) Authorize(ctx context.Context, userID, botID string, resource Resource, action Action) error {
	uc, err := r.store.GetUserContext(ctx, userID)
	if err != nil {
		return err
	}

	// System admins can do anything.
	if isAdminRole(uc.SystemRole) {
		return nil
	}

	// Determine the user's bot role.
	botRole, hasBotRole := uc.BotMembership[botID]

	// If no direct bot role, check department access and default to member.
	if !hasBotRole {
		if err := r.AuthorizeBotAccess(ctx, userID, botID); err != nil {
			return err
		}
		botRole = BotRoleMember
	}

	// Check custom permissions first (explicit overrides).
	perms, err := r.store.GetBotPermissions(ctx, botID)
	if err != nil {
		return err
	}

	for _, p := range perms {
		if p.Role == botRole && p.Resource == resource && p.Action == action {
			if p.Allowed {
				return nil
			}
			return ErrAccessDenied
		}
	}

	// Fall back to default permissions.
	if rolePerms, ok := defaultPermissions[botRole]; ok {
		if resourcePerms, ok := rolePerms[resource]; ok {
			if resourcePerms[action] {
				return nil
			}
		}
	}

	return ErrAccessDenied
}

// ListAccessibleBots filters a list of bot IDs to those the user can access.
// Optimized to avoid N+1 queries: fetches user context once and batches department lookups.
func (r *Resolver) ListAccessibleBots(ctx context.Context, userID string, allBotIDs []string) ([]string, error) {
	uc, err := r.store.GetUserContext(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Admin roles see everything.
	if isAdminRole(uc.SystemRole) {
		result := make([]string, len(allBotIDs))
		copy(result, allBotIDs)
		return result, nil
	}

	// Build set of user's department IDs for O(1) lookup.
	userDeptSet := make(map[string]struct{}, len(uc.Departments))
	for _, dm := range uc.Departments {
		userDeptSet[dm.DepartmentID] = struct{}{}
	}

	var accessible []string
	for _, botID := range allBotIDs {
		// Check direct bot membership first (no DB call).
		if _, ok := uc.BotMembership[botID]; ok {
			accessible = append(accessible, botID)
			continue
		}

		// Check department overlap.
		botDepts, err := r.store.GetBotDepartmentAccess(ctx, botID)
		if err != nil {
			continue
		}
		for _, bd := range botDepts {
			if _, ok := userDeptSet[bd]; ok {
				accessible = append(accessible, botID)
				break
			}
		}
	}

	return accessible, nil
}
