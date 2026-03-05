package enterprise

import (
	"math"
	"testing"

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
