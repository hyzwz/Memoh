package handlers

import "testing"

func TestPreferredTerminalShell_PrefersReadlineCapableShells(t *testing.T) {
	for _, tt := range []struct {
		name      string
		available map[string]bool
		want      string
	}{
		{
			name:      "prefers bash",
			available: map[string]bool{"/bin/bash": true, "/bin/sh": true},
			want:      "/bin/bash",
		},
		{
			name:      "falls back to zsh",
			available: map[string]bool{"/usr/bin/zsh": true, "/bin/sh": true},
			want:      "/usr/bin/zsh",
		},
		{
			name:      "falls back to sh when no readline shell exists",
			available: map[string]bool{"/bin/sh": true},
			want:      "/bin/sh",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got := preferredTerminalShell(func(path string) bool { return tt.available[path] })
			if got != tt.want {
				t.Fatalf("preferredTerminalShell() = %q, want %q", got, tt.want)
			}
		})
	}
}
