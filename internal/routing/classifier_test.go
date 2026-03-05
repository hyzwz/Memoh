package routing

import (
	"testing"
)

func TestClassifyComplexity(t *testing.T) {
	tests := []struct {
		name        string
		message     string
		historyLen  int
		want        ComplexityTier
	}{
		// Simple tier
		{
			name:    "greeting is simple",
			message: "hi",
			want:    TierSimple,
		},
		{
			name:    "thanks is simple",
			message: "thanks",
			want:    TierSimple,
		},
		{
			name:    "yes is simple",
			message: "yes",
			want:    TierSimple,
		},
		{
			name:    "short question is simple",
			message: "what time is it?",
			want:    TierSimple,
		},

		// Medium tier
		{
			name:    "moderate question is medium",
			message: "Can you help me summarize the key points from our last meeting about the marketing strategy?",
			want:    TierMedium,
		},
		{
			name:    "long conversation bumps to medium",
			message:    "please continue",
			historyLen: 25,
			want:       TierMedium,
		},

		// Complex tier
		{
			name:    "code request is complex",
			message: "Please analyze this codebase and implement a new feature that refactors the authentication module to support OAuth2 with PKCE flow",
			want:    TierComplex,
		},
		{
			name:    "architecture question is complex",
			message: "Design a microservice architecture for our e-commerce platform that handles 10k concurrent users with event-driven communication",
			want:    TierComplex,
		},
		{
			name:    "multi-keyword complex request",
			message: "Analyze the performance data, compare the results across all regions, evaluate the trends, and implement the algorithm to optimize our supply chain",
			want:    TierComplex,
		},
		{
			name:    "very long message is complex",
			message: string(make([]byte, 2500)),
			want:    TierComplex,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ClassifyComplexity(tt.message, tt.historyLen)
			if got != tt.want {
				t.Fatalf("ClassifyComplexity(%q, historyLen=%d) = %q, want %q",
					truncate(tt.message, 60), tt.historyLen, got, tt.want)
			}
		})
	}
}

func truncate(s string, maxLen int) string {
	if len(s) > maxLen {
		return s[:maxLen] + "..."
	}
	return s
}
