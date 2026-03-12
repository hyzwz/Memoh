package enterprise

import (
	"context"
	"testing"
)

func TestUpdateDirectoryTemplates_PathValidation(t *testing.T) {
	// Use a valid UUID format so UUID parsing passes and we reach path validation.
	// DB call will panic (nil queries) but path validation happens first.
	const validUUID = "00000000-0000-0000-0000-000000000001"
	svc := &DepartmentService{}

	tests := []struct {
		name    string
		paths   []string
		wantErr string
	}{
		{
			name:    "path without /data/ prefix",
			paths:   []string{"/etc/passwd"},
			wantErr: "path must start with /data/",
		},
		{
			name:    "path with traversal",
			paths:   []string{"/data/../etc/passwd"},
			wantErr: "path must not contain '..'",
		},
		{
			name:    "mixed valid and invalid",
			paths:   []string{"/data/ok", "/tmp/bad"},
			wantErr: "path must start with /data/",
		},
		{
			name:    "root path rejected",
			paths:   []string{"/"},
			wantErr: "path must start with /data/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := svc.UpdateDirectoryTemplates(context.Background(), validUUID, tt.paths)
			if err == nil {
				t.Fatal("expected error for invalid path")
			}
			if got := err.Error(); !contains(got, tt.wantErr) {
				t.Errorf("error = %q, want containing %q", got, tt.wantErr)
			}
		})
	}
}

func TestUpdateDirectoryTemplates_PathValidation_ValidUUID(t *testing.T) {
	// Test that invalid paths are rejected even with a valid UUID format.
	svc := &DepartmentService{}

	// Valid UUID format but no DB — path validation should still reject bad paths before DB call.
	err := svc.UpdateDirectoryTemplates(context.Background(), "00000000-0000-0000-0000-000000000001", []string{"/etc/shadow"})
	if err == nil {
		t.Fatal("expected error for invalid path")
	}
	if got := err.Error(); !contains(got, "path must start with /data/") {
		t.Errorf("error = %q, want path validation error", got)
	}

	err = svc.UpdateDirectoryTemplates(context.Background(), "00000000-0000-0000-0000-000000000001", []string{"/data/../../etc/passwd"})
	if err == nil {
		t.Fatal("expected error for traversal path")
	}
	if got := err.Error(); !contains(got, "path must not contain '..'") {
		t.Errorf("error = %q, want traversal error", got)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
