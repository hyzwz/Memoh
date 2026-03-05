package rbac

import (
	"context"
	"testing"
)

// fakeStore implements the Store interface for testing.
// Each field is a function that can be customized per test case.
type fakeStore struct {
	getUserContextFunc        func(ctx context.Context, userID string) (*UserContext, error)
	getBotDepartmentAccessFunc func(ctx context.Context, botID string) ([]string, error)
	getBotPermissionsFunc     func(ctx context.Context, botID string) ([]Permission, error)
}

func (f *fakeStore) GetUserContext(ctx context.Context, userID string) (*UserContext, error) {
	if f.getUserContextFunc != nil {
		return f.getUserContextFunc(ctx, userID)
	}
	return nil, ErrUserNotFound
}

func (f *fakeStore) GetBotDepartmentAccess(ctx context.Context, botID string) ([]string, error) {
	if f.getBotDepartmentAccessFunc != nil {
		return f.getBotDepartmentAccessFunc(ctx, botID)
	}
	return nil, nil
}

func (f *fakeStore) GetBotPermissions(ctx context.Context, botID string) ([]Permission, error) {
	if f.getBotPermissionsFunc != nil {
		return f.getBotPermissionsFunc(ctx, botID)
	}
	return nil, nil
}

// --- Test: AuthorizeBotAccess ---

func TestAuthorizeBotAccess(t *testing.T) {
	const (
		userAlice   = "user-alice"
		userBob     = "user-bob"
		userCharlie = "user-charlie"
		userDave    = "user-dave"
		userEve     = "user-eve"
		userUnknown = "user-unknown"

		botMarketing = "bot-marketing"
		botFinance   = "bot-finance"

		deptMarketing = "dept-marketing"
		deptFinance   = "dept-finance"
	)

	// Alice: super_admin
	// Bob: org_admin
	// Charlie: dept_admin of marketing, member of marketing department
	// Dave: member, member of marketing department
	// Eve: member, member of finance department (no access to marketing bot)

	makeStore := func() *fakeStore {
		users := map[string]*UserContext{
			userAlice: {
				UserID:     userAlice,
				SystemRole: RoleSuperAdmin,
			},
			userBob: {
				UserID:     userBob,
				SystemRole: RoleOrgAdmin,
			},
			userCharlie: {
				UserID:     userCharlie,
				SystemRole: RoleMember,
				Departments: []DepartmentMembership{
					{DepartmentID: deptMarketing, Role: "admin"},
				},
			},
			userDave: {
				UserID:     userDave,
				SystemRole: RoleMember,
				Departments: []DepartmentMembership{
					{DepartmentID: deptMarketing, Role: "member"},
				},
			},
			userEve: {
				UserID:     userEve,
				SystemRole: RoleMember,
				Departments: []DepartmentMembership{
					{DepartmentID: deptFinance, Role: "member"},
				},
			},
		}

		botDepts := map[string][]string{
			botMarketing: {deptMarketing},
			botFinance:   {deptFinance},
		}

		return &fakeStore{
			getUserContextFunc: func(_ context.Context, userID string) (*UserContext, error) {
				uc, ok := users[userID]
				if !ok {
					return nil, ErrUserNotFound
				}
				return uc, nil
			},
			getBotDepartmentAccessFunc: func(_ context.Context, botID string) ([]string, error) {
				return botDepts[botID], nil
			},
		}
	}

	tests := []struct {
		name    string
		userID  string
		botID   string
		wantErr error
	}{
		{
			name:    "super_admin can access any bot",
			userID:  userAlice,
			botID:   botMarketing,
			wantErr: nil,
		},
		{
			name:    "super_admin can access unrelated bot",
			userID:  userAlice,
			botID:   botFinance,
			wantErr: nil,
		},
		{
			name:    "org_admin can access any bot",
			userID:  userBob,
			botID:   botMarketing,
			wantErr: nil,
		},
		{
			name:    "org_admin can access unrelated bot",
			userID:  userBob,
			botID:   botFinance,
			wantErr: nil,
		},
		{
			name:    "dept_admin can access bot assigned to their department",
			userID:  userCharlie,
			botID:   botMarketing,
			wantErr: nil,
		},
		{
			name:    "dept_admin cannot access bot of other department",
			userID:  userCharlie,
			botID:   botFinance,
			wantErr: ErrAccessDenied,
		},
		{
			name:    "member can access bot assigned to their department",
			userID:  userDave,
			botID:   botMarketing,
			wantErr: nil,
		},
		{
			name:    "member cannot access bot of other department",
			userID:  userDave,
			botID:   botFinance,
			wantErr: ErrAccessDenied,
		},
		{
			name:    "member in finance can access finance bot",
			userID:  userEve,
			botID:   botFinance,
			wantErr: nil,
		},
		{
			name:    "member in finance cannot access marketing bot",
			userID:  userEve,
			botID:   botMarketing,
			wantErr: ErrAccessDenied,
		},
		{
			name:    "unknown user is denied",
			userID:  userUnknown,
			botID:   botMarketing,
			wantErr: ErrUserNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := makeStore()
			resolver := NewResolver(store)

			err := resolver.AuthorizeBotAccess(context.Background(), tt.userID, tt.botID)

			if tt.wantErr == nil {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
			} else {
				if err == nil {
					t.Fatalf("expected error %v, got nil", tt.wantErr)
				}
				if err != tt.wantErr {
					t.Fatalf("expected error %v, got %v", tt.wantErr, err)
				}
			}
		})
	}
}

// --- Test: Authorize (fine-grained resource+action) ---

func TestAuthorize(t *testing.T) {
	const (
		userAdmin   = "user-admin"
		userBotOwner = "user-owner"
		userBotAdmin = "user-bot-admin"
		userBotMember = "user-bot-member"

		botID = "bot-1"
		deptID = "dept-1"
	)

	makeStore := func(permissions []Permission) *fakeStore {
		users := map[string]*UserContext{
			userAdmin: {
				UserID:     userAdmin,
				SystemRole: RoleSuperAdmin,
			},
			userBotOwner: {
				UserID:     userBotOwner,
				SystemRole: RoleMember,
				Departments: []DepartmentMembership{
					{DepartmentID: deptID, Role: "member"},
				},
				BotMembership: map[string]BotRole{
					botID: BotRoleOwner,
				},
			},
			userBotAdmin: {
				UserID:     userBotAdmin,
				SystemRole: RoleMember,
				Departments: []DepartmentMembership{
					{DepartmentID: deptID, Role: "member"},
				},
				BotMembership: map[string]BotRole{
					botID: BotRoleAdmin,
				},
			},
			userBotMember: {
				UserID:     userBotMember,
				SystemRole: RoleMember,
				Departments: []DepartmentMembership{
					{DepartmentID: deptID, Role: "member"},
				},
				BotMembership: map[string]BotRole{
					botID: BotRoleMember,
				},
			},
		}

		return &fakeStore{
			getUserContextFunc: func(_ context.Context, userID string) (*UserContext, error) {
				uc, ok := users[userID]
				if !ok {
					return nil, ErrUserNotFound
				}
				return uc, nil
			},
			getBotDepartmentAccessFunc: func(_ context.Context, _ string) ([]string, error) {
				return []string{deptID}, nil
			},
			getBotPermissionsFunc: func(_ context.Context, _ string) ([]Permission, error) {
				return permissions, nil
			},
		}
	}

	tests := []struct {
		name        string
		userID      string
		botID       string
		resource    Resource
		action      Action
		permissions []Permission // custom permission overrides
		wantErr     error
	}{
		{
			name:     "super_admin can do anything",
			userID:   userAdmin,
			botID:    botID,
			resource: ResourceSettings,
			action:   ActionAdmin,
			wantErr:  nil,
		},
		{
			name:     "bot owner can admin settings by default",
			userID:   userBotOwner,
			botID:    botID,
			resource: ResourceSettings,
			action:   ActionAdmin,
			wantErr:  nil,
		},
		{
			name:     "bot admin can write settings by default",
			userID:   userBotAdmin,
			botID:    botID,
			resource: ResourceSettings,
			action:   ActionWrite,
			wantErr:  nil,
		},
		{
			name:     "bot member can read chat by default",
			userID:   userBotMember,
			botID:    botID,
			resource: ResourceChat,
			action:   ActionRead,
			wantErr:  nil,
		},
		{
			name:     "bot member can execute chat by default",
			userID:   userBotMember,
			botID:    botID,
			resource: ResourceChat,
			action:   ActionExecute,
			wantErr:  nil,
		},
		{
			name:     "bot member cannot admin settings by default",
			userID:   userBotMember,
			botID:    botID,
			resource: ResourceSettings,
			action:   ActionAdmin,
			wantErr:  ErrAccessDenied,
		},
		{
			name:     "bot member cannot write members by default",
			userID:   userBotMember,
			botID:    botID,
			resource: ResourceMembers,
			action:   ActionWrite,
			wantErr:  ErrAccessDenied,
		},
		{
			name:     "custom permission: deny member execute tools",
			userID:   userBotMember,
			botID:    botID,
			resource: ResourceTools,
			action:   ActionExecute,
			permissions: []Permission{
				{BotID: botID, Role: BotRoleMember, Resource: ResourceTools, Action: ActionExecute, Allowed: false},
			},
			wantErr: ErrAccessDenied,
		},
		{
			name:     "custom permission: allow member admin settings",
			userID:   userBotMember,
			botID:    botID,
			resource: ResourceSettings,
			action:   ActionAdmin,
			permissions: []Permission{
				{BotID: botID, Role: BotRoleMember, Resource: ResourceSettings, Action: ActionAdmin, Allowed: true},
			},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := makeStore(tt.permissions)
			resolver := NewResolver(store)

			err := resolver.Authorize(context.Background(), tt.userID, tt.botID, tt.resource, tt.action)

			if tt.wantErr == nil {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
			} else {
				if err == nil {
					t.Fatalf("expected error %v, got nil", tt.wantErr)
				}
				if err != tt.wantErr {
					t.Fatalf("expected error %v, got %v", tt.wantErr, err)
				}
			}
		})
	}
}

// --- Test: ListAccessibleBots ---

func TestListAccessibleBots(t *testing.T) {
	const (
		userAdmin  = "user-admin"
		userMember = "user-member"
		deptA      = "dept-a"
		deptB      = "dept-b"
		botA1      = "bot-a1"
		botA2      = "bot-a2"
		botB1      = "bot-b1"
	)

	store := &fakeStore{
		getUserContextFunc: func(_ context.Context, userID string) (*UserContext, error) {
			switch userID {
			case userAdmin:
				return &UserContext{
					UserID:     userAdmin,
					SystemRole: RoleSuperAdmin,
				}, nil
			case userMember:
				return &UserContext{
					UserID:     userMember,
					SystemRole: RoleMember,
					Departments: []DepartmentMembership{
						{DepartmentID: deptA, Role: "member"},
					},
				}, nil
			default:
				return nil, ErrUserNotFound
			}
		},
		getBotDepartmentAccessFunc: func(_ context.Context, botID string) ([]string, error) {
			mapping := map[string][]string{
				botA1: {deptA},
				botA2: {deptA},
				botB1: {deptB},
			}
			return mapping[botID], nil
		},
	}

	resolver := NewResolver(store)

	t.Run("super_admin gets all bots", func(t *testing.T) {
		bots, err := resolver.ListAccessibleBots(context.Background(), userAdmin, []string{botA1, botA2, botB1})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(bots) != 3 {
			t.Fatalf("expected 3 bots, got %d: %v", len(bots), bots)
		}
	})

	t.Run("member gets only department bots", func(t *testing.T) {
		bots, err := resolver.ListAccessibleBots(context.Background(), userMember, []string{botA1, botA2, botB1})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(bots) != 2 {
			t.Fatalf("expected 2 bots, got %d: %v", len(bots), bots)
		}
		for _, b := range bots {
			if b != botA1 && b != botA2 {
				t.Fatalf("unexpected bot %s, expected only %s or %s", b, botA1, botA2)
			}
		}
	})
}

// --- Test: Legacy role mapping ---

func TestLegacyAdminRole(t *testing.T) {
	store := &fakeStore{
		getUserContextFunc: func(_ context.Context, _ string) (*UserContext, error) {
			return &UserContext{
				UserID:     "legacy-admin",
				SystemRole: RoleLegacyAdmin,
			}, nil
		},
	}

	resolver := NewResolver(store)

	err := resolver.AuthorizeBotAccess(context.Background(), "legacy-admin", "any-bot")
	if err != nil {
		t.Fatalf("legacy admin should be treated as super_admin, got error: %v", err)
	}
}
