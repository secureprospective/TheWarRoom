package engine

import (
	"math"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/domain"
)

func approxEq(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

// --- L3 decay (table-driven) ---

func TestApplyDecay(t *testing.T) {
	cases := []struct {
		name                    string
		age, peak, rate, expect float64
	}{
		{"below peak no decay", 25, 32, 0.03, 1.0},
		{"at peak no decay", 32, 32, 0.03, 1.0},
		{"one year past", 33, 32, 0.03, 0.97},
		{"three years past", 35, 32, 0.03, 0.97 * 0.97 * 0.97},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := ApplyDecay(c.age, c.peak, c.rate)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !approxEq(got, c.expect) {
				t.Fatalf("ApplyDecay(%v,%v,%v) = %v, want %v", c.age, c.peak, c.rate, got, c.expect)
			}
		})
	}
}

// PLANTED FAILURE (M3): non-finite decay inputs, and a decayRate>1 that drives the base
// negative (NaN under a fractional exponent), must fail loud — AgePull multiplies
// straight into the score, so a silent NaN would poison rankings. Would FAIL if the L3
// finite guard were removed.
func TestApplyDecayNonFiniteRejected(t *testing.T) {
	cases := []struct {
		name            string
		age, peak, rate float64
	}{
		{"NaN age", math.NaN(), 32, 0.03},
		{"Inf peak", 30, math.Inf(1), 0.03},
		{"NaN rate", 30, 32, math.NaN()},
		{"rate>1 negative base", 35, 32, 1.5},
		{"rate<0 inflates with age", 35, 32, -0.5}, // GLM m3: negative rate grows the pull
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if _, err := ApplyDecay(c.age, c.peak, c.rate); err == nil {
				t.Fatalf("ApplyDecay(%v,%v,%v): expected error, got nil", c.age, c.peak, c.rate)
			}
		})
	}
}

// --- L6 tiebreaker ordering ---

func TestTiebreakerRanksAbove(t *testing.T) {
	vet := TiebreakerKey{IsVeteran: true, RAS: 5, ScarcityRank: 1}
	rookie := TiebreakerKey{IsVeteran: false, RAS: 9, ScarcityRank: 9}
	if !vet.RanksAbove(rookie) {
		t.Fatal("veteran must outrank rookie regardless of RAS/scarcity")
	}
	hi := TiebreakerKey{IsVeteran: true, RAS: 9, ScarcityRank: 1}
	lo := TiebreakerKey{IsVeteran: true, RAS: 7, ScarcityRank: 9}
	if !hi.RanksAbove(lo) {
		t.Fatal("among veterans, higher RAS wins before scarcity")
	}
	scarce := TiebreakerKey{IsVeteran: true, RAS: 7, ScarcityRank: 9}
	common := TiebreakerKey{IsVeteran: true, RAS: 7, ScarcityRank: 1}
	if !scarce.RanksAbove(common) {
		t.Fatal("equal tenure+RAS: higher scarcity rank wins")
	}
}

func TestPeakLimitByPosition(t *testing.T) {
	want := map[domain.Position]float64{
		domain.PosQB: 32, domain.PosRB: 25, domain.PosWR: 29, domain.PosTE: 29,
		domain.PosDE: 30, domain.PosDT: 30, domain.PosLB: 29, domain.PosCB: 28,
		domain.PosS: 28, domain.PosK: 30, domain.PosFlag: 32,
	}
	for pos, w := range want {
		if got := PeakLimit(pos); got != w {
			t.Errorf("PeakLimit(%s) = %v, want %v", pos, got, w)
		}
	}
}
