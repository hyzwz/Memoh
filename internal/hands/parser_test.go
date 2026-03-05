package hands

import (
	"testing"
)

func TestParseHandMD(t *testing.T) {
	tests := []struct {
		name     string
		raw      string
		wantName string
		wantType string
		check    func(t *testing.T, h Hand)
	}{
		{
			name: "full HAND.md with all fields",
			raw: `---
name: daily-report
description: Generate daily team activity report
type: hand
schedule:
  pattern: "0 0 18 * * 1-5"
  timezone: "Asia/Shanghai"
  max_calls: 100
triggers:
  - event: message_keyword
    match: "generate report"
  - event: webhook
    path: "/hooks/daily-report"
tools:
  - web_fetch
  - search_inbox
  - send
model:
  complexity_tier: medium
  reasoning: false
output:
  channels: ["wecom"]
  target: department_group
permissions:
  require_approval: false
  max_tokens_per_run: 50000
metadata:
  author: enterprise-admin
  version: "1.0"
---

## Role
You are a daily report generator.
`,
			wantName: "daily-report",
			wantType: "hand",
			check: func(t *testing.T, h Hand) {
				t.Helper()
				if h.Description != "Generate daily team activity report" {
					t.Fatalf("description = %q", h.Description)
				}
				if h.Schedule == nil {
					t.Fatal("expected schedule")
				}
				if h.Schedule.Pattern != "0 0 18 * * 1-5" {
					t.Fatalf("schedule.pattern = %q", h.Schedule.Pattern)
				}
				if h.Schedule.Timezone != "Asia/Shanghai" {
					t.Fatalf("schedule.timezone = %q", h.Schedule.Timezone)
				}
				if h.Schedule.MaxCalls == nil || *h.Schedule.MaxCalls != 100 {
					t.Fatalf("schedule.max_calls = %v", h.Schedule.MaxCalls)
				}
				if len(h.Triggers) != 2 {
					t.Fatalf("expected 2 triggers, got %d", len(h.Triggers))
				}
				if h.Triggers[0].Event != "message_keyword" || h.Triggers[0].Match != "generate report" {
					t.Fatalf("trigger[0] = %+v", h.Triggers[0])
				}
				if h.Triggers[1].Event != "webhook" || h.Triggers[1].Path != "/hooks/daily-report" {
					t.Fatalf("trigger[1] = %+v", h.Triggers[1])
				}
				if len(h.AllowedTools) != 3 || h.AllowedTools[0] != "web_fetch" {
					t.Fatalf("tools = %v", h.AllowedTools)
				}
				if h.ModelConfig.ComplexityTier != "medium" {
					t.Fatalf("model.complexity_tier = %q", h.ModelConfig.ComplexityTier)
				}
				if h.ModelConfig.Reasoning == nil || *h.ModelConfig.Reasoning != false {
					t.Fatalf("model.reasoning = %v", h.ModelConfig.Reasoning)
				}
				if len(h.OutputConfig.Channels) != 1 || h.OutputConfig.Channels[0] != "wecom" {
					t.Fatalf("output.channels = %v", h.OutputConfig.Channels)
				}
				if h.OutputConfig.Target != "department_group" {
					t.Fatalf("output.target = %q", h.OutputConfig.Target)
				}
				if h.Permissions.RequireApproval {
					t.Fatal("expected require_approval=false")
				}
				if h.Permissions.MaxTokensPerRun != 50000 {
					t.Fatalf("max_tokens_per_run = %d", h.Permissions.MaxTokensPerRun)
				}
				if h.Content != "## Role\nYou are a daily report generator." {
					t.Fatalf("content = %q", h.Content)
				}
				if h.Metadata["author"] != "enterprise-admin" {
					t.Fatalf("metadata.author = %v", h.Metadata["author"])
				}
			},
		},
		{
			name: "minimal skill (no hand-specific fields)",
			raw: `---
name: greet
description: A simple greeting skill
---

Hello! How can I help you?
`,
			wantName: "greet",
			wantType: "skill",
			check: func(t *testing.T, h Hand) {
				t.Helper()
				if h.Schedule != nil {
					t.Fatal("expected no schedule")
				}
				if len(h.Triggers) != 0 {
					t.Fatal("expected no triggers")
				}
				if len(h.AllowedTools) != 0 {
					t.Fatal("expected no tools")
				}
				if h.Content != "Hello! How can I help you?" {
					t.Fatalf("content = %q", h.Content)
				}
			},
		},
		{
			name: "no frontmatter at all",
			raw:  "Just a plain prompt with no YAML.",
			wantName: "fallback",
			wantType: "skill",
			check: func(t *testing.T, h Hand) {
				t.Helper()
				if h.Content != "Just a plain prompt with no YAML." {
					t.Fatalf("content = %q", h.Content)
				}
			},
		},
		{
			name: "schedule without max_calls means unlimited",
			raw: `---
name: hourly-check
type: hand
schedule:
  pattern: "0 0 * * * *"
---

Check system status.
`,
			wantName: "hourly-check",
			wantType: "hand",
			check: func(t *testing.T, h Hand) {
				t.Helper()
				if h.Schedule == nil {
					t.Fatal("expected schedule")
				}
				if h.Schedule.MaxCalls != nil {
					t.Fatalf("expected nil max_calls, got %d", *h.Schedule.MaxCalls)
				}
				if h.Schedule.Timezone != "UTC" {
					t.Fatalf("expected default timezone UTC, got %q", h.Schedule.Timezone)
				}
			},
		},
		{
			name: "invalid YAML frontmatter falls back gracefully",
			raw: `---
name: [invalid yaml
  !!!bad
---

Some content here.
`,
			wantName: "fallback",
			wantType: "skill",
			check: func(t *testing.T, h Hand) {
				t.Helper()
				if h.Content != "Some content here." {
					t.Fatalf("content = %q", h.Content)
				}
			},
		},
		{
			name: "hand with only triggers no schedule",
			raw: `---
name: alert-handler
type: hand
triggers:
  - event: message_keyword
    match: "urgent alert"
permissions:
  require_approval: true
  max_tokens_per_run: 10000
---

Handle urgent alerts immediately.
`,
			wantName: "alert-handler",
			wantType: "hand",
			check: func(t *testing.T, h Hand) {
				t.Helper()
				if h.Schedule != nil {
					t.Fatal("expected no schedule")
				}
				if len(h.Triggers) != 1 {
					t.Fatalf("expected 1 trigger, got %d", len(h.Triggers))
				}
				if !h.Permissions.RequireApproval {
					t.Fatal("expected require_approval=true")
				}
				if h.Permissions.MaxTokensPerRun != 10000 {
					t.Fatalf("max_tokens = %d", h.Permissions.MaxTokensPerRun)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := ParseHandMD(tt.raw, "fallback")

			if h.Name != tt.wantName {
				t.Fatalf("name = %q, want %q", h.Name, tt.wantName)
			}
			if h.Type != tt.wantType {
				t.Fatalf("type = %q, want %q", h.Type, tt.wantType)
			}
			if tt.check != nil {
				tt.check(t, h)
			}
		})
	}
}
