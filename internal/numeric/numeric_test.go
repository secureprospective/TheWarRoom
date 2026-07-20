package numeric

import (
	"math"
	"testing"
)

func TestFinite(t *testing.T) {
	cases := []struct {
		name string
		vs   []float64
		want bool
	}{
		{"empty is vacuously finite", nil, true},
		{"all real", []float64{-3.5, 0, 42}, true},
		{"one NaN", []float64{1, math.NaN(), 3}, false},
		{"one +Inf", []float64{math.Inf(1)}, false},
		{"one -Inf", []float64{0, math.Inf(-1)}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := Finite(c.vs...); got != c.want {
				t.Errorf("Finite(%v) = %v, want %v", c.vs, got, c.want)
			}
		})
	}
}

func TestBoolToInt(t *testing.T) {
	if got := BoolToInt(true); got != 1 {
		t.Errorf("BoolToInt(true) = %d, want 1", got)
	}
	if got := BoolToInt(false); got != 0 {
		t.Errorf("BoolToInt(false) = %d, want 0", got)
	}
}
