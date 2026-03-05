package enterprise

import "testing"

func TestRoleLevel(t *testing.T) {
	tests := []struct {
		role  string
		level int
	}{
		{"member", 0},
		{"dept_admin", 1},
		{"admin", 2},
		{"org_admin", 3},
		{"super_admin", 4},
		{"unknown", -1},
	}

	for _, tt := range tests {
		t.Run(tt.role, func(t *testing.T) {
			got := roleLevel(tt.role)
			if got != tt.level {
				t.Errorf("roleLevel(%q) = %d, want %d", tt.role, got, tt.level)
			}
		})
	}
}

func TestHasMinRole(t *testing.T) {
	tests := []struct {
		userRole    string
		minRequired string
		want        bool
	}{
		{"super_admin", "admin", true},
		{"admin", "admin", true},
		{"member", "admin", false},
		{"dept_admin", "member", true},
		{"member", "member", true},
		{"org_admin", "dept_admin", true},
	}

	for _, tt := range tests {
		got := hasMinRole(tt.userRole, tt.minRequired)
		if got != tt.want {
			t.Errorf("hasMinRole(%q, %q) = %v, want %v", tt.userRole, tt.minRequired, got, tt.want)
		}
	}
}

func TestHasMinRole_UnknownMinRole(t *testing.T) {
	// Unknown minRole should reject all users (even super_admin cannot match unknown role).
	roles := []string{"member", "dept_admin", "admin", "org_admin", "super_admin"}
	for _, r := range roles {
		if hasMinRole(r, "nonexistent") {
			t.Errorf("hasMinRole(%q, %q) should be false for unknown minRole", r, "nonexistent")
		}
	}
}

func TestHasMinRole_UnknownUserRole(t *testing.T) {
	// Unknown user role should be denied access to any valid role.
	if hasMinRole("unknown_role", "member") {
		t.Error("unknown user role should not meet member requirement")
	}
}
