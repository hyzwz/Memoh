package dingtalk

import "testing"

func TestNormalizeTarget(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  string
	}{
		{
			input: " webhook:https://oapi.dingtalk.com/robot/send?access_token=abc ",
			want:  "webhook:https://oapi.dingtalk.com/robot/send?access_token=abc",
		},
		{
			input: "CONVERSATION:cid123",
			want:  "conversation:cid123",
		},
		{
			input: " user:staff:staff123 ",
			want:  "user:staff:staff123",
		},
		{
			input: "USER:UNION:union123",
			want:  "user:union:union123",
		},
	}

	for _, tt := range tests {
		if got := normalizeTarget(tt.input); got != tt.want {
			t.Fatalf("normalizeTarget(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestNormalizeTargetRejectsMalformedOrUnknownValues(t *testing.T) {
	t.Parallel()

	inputs := []string{
		"",
		"https://oapi.dingtalk.com/robot/send?access_token=abc",
		"unknown:value",
		"webhook:",
		"webhook:ftp://oapi.dingtalk.com/robot/send?access_token=abc",
		"conversation:",
		"user:",
		"user:staff123",
		"user:staff:",
		"user:union:",
		"user:other:extra",
		"user:staff:extra:more",
	}

	for _, input := range inputs {
		if got := normalizeTarget(input); got != "" {
			t.Fatalf("normalizeTarget(%q) = %q, want empty string", input, got)
		}
	}
}

func TestResolveTarget(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		raw  map[string]any
		want string
	}{
		{
			name: "conversation",
			raw: map[string]any{
				"conversationId": "cid-1",
			},
			want: "conversation:cid-1",
		},
		{
			name: "staff user",
			raw: map[string]any{
				"staffId": "staff-1",
			},
			want: "user:staff:staff-1",
		},
		{
			name: "union user",
			raw: map[string]any{
				"unionId": "union-1",
			},
			want: "user:union:union-1",
		},
		{
			name: "conversation preferred",
			raw: map[string]any{
				"conversationId": "cid-2",
				"staffId":        "staff-2",
				"unionId":        "union-2",
			},
			want: "conversation:cid-2",
		},
		{
			name: "staff preferred over union",
			raw: map[string]any{
				"staffId": "staff-3",
				"unionId": "union-3",
			},
			want: "user:staff:staff-3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := resolveTarget(tt.raw)
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if got != tt.want {
				t.Fatalf("unexpected target: %q", got)
			}
		})
	}
}

func TestResolveTargetRejectsIncompleteUserConfig(t *testing.T) {
	t.Parallel()

	if _, err := resolveTarget(map[string]any{}); err == nil {
		t.Fatal("expected target resolution error")
	}
}
