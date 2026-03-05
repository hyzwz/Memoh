package hands

import "testing"

func TestMatchMessageTrigger(t *testing.T) {
	tests := []struct {
		name    string
		hand    Hand
		message string
		want    bool
	}{
		{
			name: "exact keyword match",
			hand: Hand{
				Triggers: []TriggerDef{
					{Event: "message_keyword", Match: "generate report"},
				},
			},
			message: "please generate report for today",
			want:    true,
		},
		{
			name: "case insensitive match",
			hand: Hand{
				Triggers: []TriggerDef{
					{Event: "message_keyword", Match: "Generate Report"},
				},
			},
			message: "can you generate report?",
			want:    true,
		},
		{
			name: "no match",
			hand: Hand{
				Triggers: []TriggerDef{
					{Event: "message_keyword", Match: "generate report"},
				},
			},
			message: "hello world",
			want:    false,
		},
		{
			name: "multiple triggers, second matches",
			hand: Hand{
				Triggers: []TriggerDef{
					{Event: "message_keyword", Match: "daily report"},
					{Event: "message_keyword", Match: "status update"},
				},
			},
			message: "give me a status update",
			want:    true,
		},
		{
			name: "webhook trigger is not a message trigger",
			hand: Hand{
				Triggers: []TriggerDef{
					{Event: "webhook", Path: "/hooks/test"},
				},
			},
			message: "/hooks/test",
			want:    false,
		},
		{
			name: "no triggers means no match",
			hand: Hand{},
			message: "anything",
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MatchMessageTrigger(tt.hand, tt.message)
			if got != tt.want {
				t.Fatalf("MatchMessageTrigger = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMatchWebhookTrigger(t *testing.T) {
	tests := []struct {
		name string
		hand Hand
		path string
		want bool
	}{
		{
			name: "exact path match",
			hand: Hand{
				Triggers: []TriggerDef{
					{Event: "webhook", Path: "/hooks/daily-report"},
				},
			},
			path: "/hooks/daily-report",
			want: true,
		},
		{
			name: "no match",
			hand: Hand{
				Triggers: []TriggerDef{
					{Event: "webhook", Path: "/hooks/daily-report"},
				},
			},
			path: "/hooks/other",
			want: false,
		},
		{
			name: "message_keyword trigger is not webhook",
			hand: Hand{
				Triggers: []TriggerDef{
					{Event: "message_keyword", Match: "test"},
				},
			},
			path: "test",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MatchWebhookTrigger(tt.hand, tt.path)
			if got != tt.want {
				t.Fatalf("MatchWebhookTrigger = %v, want %v", got, tt.want)
			}
		})
	}
}
