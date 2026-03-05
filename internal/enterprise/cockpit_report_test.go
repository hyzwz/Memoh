package enterprise

import (
	"testing"
)

func TestEstimateSavedHours(t *testing.T) {
	tests := []struct {
		name            string
		conversations   int64
		handExecs       int64
		wantMinHours    float64
		wantMaxHours    float64
	}{
		{"no activity", 0, 0, 0, 0},
		{"conversations only", 10, 0, 1.5, 2.5},           // 10 * ~0.2 hours each
		{"hand executions only", 0, 5, 1.5, 3.0},            // 5 * ~0.5 hours each
		{"mixed activity", 10, 5, 3.0, 5.5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hours := estimateSavedHours(tt.conversations, tt.handExecs)
			if hours < tt.wantMinHours || hours > tt.wantMaxHours {
				t.Errorf("estimateSavedHours(%d, %d) = %.2f, want [%.2f, %.2f]",
					tt.conversations, tt.handExecs, hours, tt.wantMinHours, tt.wantMaxHours)
			}
		})
	}
}

func TestEstimateAITimeMinutes(t *testing.T) {
	// 1M tokens at roughly 100 tok/sec → ~10000 sec → ~166 min
	// But we're estimating AI interaction time, not raw processing.
	minutes := estimateAITimeMinutes(500_000, 200_000)
	if minutes <= 0 {
		t.Errorf("expected positive AI time, got %d", minutes)
	}
}

func TestComputeEfficiencyMultiplier(t *testing.T) {
	// If AI took 30 min and saved 2 hours (120 min), multiplier = 120/30 = 4x
	mult := computeEfficiencyMultiplier(2.0, 30)
	if mult < 3.5 || mult > 4.5 {
		t.Errorf("efficiency = %.2f, want ~4.0", mult)
	}

	// Zero AI time → multiplier 1.0 (no division by zero)
	mult = computeEfficiencyMultiplier(1.0, 0)
	if mult != 1.0 {
		t.Errorf("efficiency with zero AI time = %.2f, want 1.0", mult)
	}

	// Zero saved hours → multiplier 0
	mult = computeEfficiencyMultiplier(0, 30)
	if mult != 0 {
		t.Errorf("efficiency with zero saved = %.2f, want 0", mult)
	}
}
