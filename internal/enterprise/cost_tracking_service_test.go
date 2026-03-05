package enterprise

import (
	"math"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

func TestFloat64ToNumericAndBack(t *testing.T) {
	cases := []float64{0, 1.0, 99.99, 0.5, 100}
	for _, f := range cases {
		n := float64ToNumeric(f)
		if !n.Valid {
			t.Fatalf("numeric should be valid for %v", f)
		}
		back := numericToFloat64(n)
		if math.Abs(back-f) > 0.01 {
			t.Fatalf("roundtrip %v -> %v", f, back)
		}
	}
}

func TestNumericToFloat64Invalid(t *testing.T) {
	n := pgtype.Numeric{Valid: false}
	if v := numericToFloat64(n); v != 0 {
		t.Fatalf("invalid numeric should return 0, got %v", v)
	}
}

func TestPeriodToTimeRange(t *testing.T) {
	now := time.Date(2026, 3, 15, 14, 30, 0, 0, time.UTC)

	tests := []struct {
		period    string
		wantStart time.Time
		wantEnd   time.Time
	}{
		{
			period:    "daily",
			wantStart: time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC),
			wantEnd:   time.Date(2026, 3, 16, 0, 0, 0, 0, time.UTC),
		},
		{
			period:    "weekly",
			wantStart: time.Date(2026, 3, 9, 0, 0, 0, 0, time.UTC), // Monday of that week
			wantEnd:   time.Date(2026, 3, 16, 0, 0, 0, 0, time.UTC),
		},
		{
			period:    "monthly",
			wantStart: time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
			wantEnd:   time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
		},
	}

	for _, tt := range tests {
		t.Run(tt.period, func(t *testing.T) {
			start, end := periodToTimeRange(tt.period, now)
			if !start.Equal(tt.wantStart) {
				t.Errorf("start = %v, want %v", start, tt.wantStart)
			}
			if !end.Equal(tt.wantEnd) {
				t.Errorf("end = %v, want %v", end, tt.wantEnd)
			}
		})
	}
}

func TestComputeTokenCost(t *testing.T) {
	// 1M input tokens at $3/Mtok + 500K output tokens at $15/Mtok = $3 + $7.5 = $10.5
	cost := computeTokenCost(1_000_000, 500_000, 0, 0, 0, 3.0, 15.0, 0, 0, 0)
	if math.Abs(cost-10.5) > 0.001 {
		t.Fatalf("cost = %v, want 10.5", cost)
	}
}

func TestComputeTokenCostWithCache(t *testing.T) {
	// 100K input at $3/Mtok + 200K output at $15/Mtok + 50K cacheRead at $0.3/Mtok
	// + 20K cacheWrite at $3.75/Mtok + 10K reasoning at $15/Mtok
	// = 0.3 + 3.0 + 0.015 + 0.075 + 0.15 = 3.54
	cost := computeTokenCost(100_000, 200_000, 50_000, 20_000, 10_000, 3.0, 15.0, 0.3, 3.75, 15.0)
	if math.Abs(cost-3.54) > 0.001 {
		t.Fatalf("cost = %v, want 3.54", cost)
	}
}

func TestComputeTokenCostWithCacheWrite(t *testing.T) {
	// Verify cacheWriteTokens are counted: 1M cache write at $3.75/Mtok = $3.75
	cost := computeTokenCost(0, 0, 0, 1_000_000, 0, 0, 0, 0, 3.75, 0)
	if math.Abs(cost-3.75) > 0.001 {
		t.Fatalf("cost = %v, want 3.75", cost)
	}
}

func TestComputeTokenCostFromUsage(t *testing.T) {
	// Uses default pricing: input=3, output=15, cacheRead=0.3, cacheWrite=3.75, reasoning=15
	// 1M input = $3, 1M output = $15
	cost := computeTokenCostFromUsage(1_000_000, 1_000_000, 0, 0, 0)
	if math.Abs(cost-18.0) > 0.001 {
		t.Fatalf("cost = %v, want 18.0", cost)
	}
}
