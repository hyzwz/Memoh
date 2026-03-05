package cost

import (
	"math"
	"testing"
)

func almostEqual(a, b, tolerance float64) bool {
	return math.Abs(a-b) < tolerance
}

func TestCalculate(t *testing.T) {
	tests := []struct {
		name    string
		pricing ModelPricing
		usage   TokenUsage
		wantTotal float64
	}{
		{
			name: "GPT-4o pricing",
			pricing: ModelPricing{
				InputPricePerMTok:      2.50,  // $2.50/1M input
				OutputPricePerMTok:     10.00, // $10.00/1M output
			},
			usage: TokenUsage{
				InputTokens:  1000,
				OutputTokens: 500,
			},
			wantTotal: 0.0025 + 0.005, // 0.0075
		},
		{
			name: "Claude Sonnet with cache",
			pricing: ModelPricing{
				InputPricePerMTok:      3.00,
				OutputPricePerMTok:     15.00,
				CacheReadPricePerMTok:  0.30,
				CacheWritePricePerMTok: 3.75,
			},
			usage: TokenUsage{
				InputTokens:     500,
				OutputTokens:    200,
				CacheReadTokens: 10000,
				CacheWriteTokens: 2000,
			},
			wantTotal: 0.0015 + 0.003 + 0.003 + 0.0075, // 0.015
		},
		{
			name: "Claude Opus with reasoning",
			pricing: ModelPricing{
				InputPricePerMTok:      15.00,
				OutputPricePerMTok:     75.00,
				ReasoningPricePerMTok:  75.00,
			},
			usage: TokenUsage{
				InputTokens:    2000,
				OutputTokens:   1000,
				ReasoningTokens: 5000,
			},
			wantTotal: 0.03 + 0.075 + 0.375, // 0.48
		},
		{
			name: "zero usage",
			pricing: ModelPricing{
				InputPricePerMTok:  10.00,
				OutputPricePerMTok: 30.00,
			},
			usage:     TokenUsage{},
			wantTotal: 0,
		},
		{
			name: "zero pricing (free model)",
			pricing: ModelPricing{},
			usage: TokenUsage{
				InputTokens:  100000,
				OutputTokens: 50000,
			},
			wantTotal: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Calculate(tt.pricing, tt.usage)

			if !almostEqual(result.TotalCost, tt.wantTotal, 0.0001) {
				t.Fatalf("TotalCost = %f, want %f", result.TotalCost, tt.wantTotal)
			}

			// Verify component sum equals total.
			componentSum := result.InputCost + result.OutputCost +
				result.CacheReadCost + result.CacheWriteCost + result.ReasoningCost
			if !almostEqual(componentSum, result.TotalCost, 0.0001) {
				t.Fatalf("component sum %f != TotalCost %f", componentSum, result.TotalCost)
			}
		})
	}
}

func TestBudgetCheck(t *testing.T) {
	tests := []struct {
		name      string
		budget    Budget
		spent     float64
		wantAllow bool
		wantAlert bool
	}{
		{
			name:      "under budget, under threshold",
			budget:    Budget{LimitAmount: 100, AlertThreshold: 0.80, ActionOnExceed: ActionWarn},
			spent:     50,
			wantAllow: true,
			wantAlert: false,
		},
		{
			name:      "over alert threshold but under limit",
			budget:    Budget{LimitAmount: 100, AlertThreshold: 0.80, ActionOnExceed: ActionWarn},
			spent:     85,
			wantAllow: true,
			wantAlert: true,
		},
		{
			name:      "over limit with warn action",
			budget:    Budget{LimitAmount: 100, AlertThreshold: 0.80, ActionOnExceed: ActionWarn},
			spent:     110,
			wantAllow: true,
			wantAlert: true,
		},
		{
			name:      "over limit with block action",
			budget:    Budget{LimitAmount: 100, AlertThreshold: 0.80, ActionOnExceed: ActionBlock},
			spent:     110,
			wantAllow: false,
			wantAlert: true,
		},
		{
			name:      "over limit with downgrade action",
			budget:    Budget{LimitAmount: 100, AlertThreshold: 0.80, ActionOnExceed: ActionDowngrade},
			spent:     110,
			wantAllow: true,
			wantAlert: true,
		},
		{
			name:      "exactly at limit with block",
			budget:    Budget{LimitAmount: 100, AlertThreshold: 0.80, ActionOnExceed: ActionBlock},
			spent:     100,
			wantAllow: false,
			wantAlert: true,
		},
		{
			name:      "zero budget allows everything",
			budget:    Budget{LimitAmount: 0, AlertThreshold: 0.80, ActionOnExceed: ActionBlock},
			spent:     999,
			wantAllow: true,
			wantAlert: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CheckBudget(tt.budget, tt.spent)

			if result.Allowed != tt.wantAllow {
				t.Fatalf("Allowed = %v, want %v", result.Allowed, tt.wantAllow)
			}
			if result.Alert != tt.wantAlert {
				t.Fatalf("Alert = %v, want %v", result.Alert, tt.wantAlert)
			}
		})
	}
}

func TestCalculateNegativeTokens(t *testing.T) {
	pricing := ModelPricing{
		InputPricePerMTok:  10.00,
		OutputPricePerMTok: 30.00,
	}
	usage := TokenUsage{
		InputTokens:  -1000,
		OutputTokens: 500,
	}
	result := Calculate(pricing, usage)
	if result.InputCost != 0 {
		t.Fatalf("negative tokens should produce 0 cost, got %f", result.InputCost)
	}
	if result.OutputCost <= 0 {
		t.Fatalf("positive tokens should produce positive cost, got %f", result.OutputCost)
	}
}

func TestCheckBudgetNegativeSpent(t *testing.T) {
	budget := Budget{LimitAmount: 100, AlertThreshold: 0.80, ActionOnExceed: ActionBlock}
	result := CheckBudget(budget, -50)
	if !result.Allowed {
		t.Fatal("negative spent should be allowed")
	}
	if result.Spent != 0 {
		t.Fatalf("negative spent should be clamped to 0, got %f", result.Spent)
	}
}
