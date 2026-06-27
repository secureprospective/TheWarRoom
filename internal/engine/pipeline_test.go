package engine

import (
	"math"
	"sync"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/domain"
)

// baseCalibration is a finite, in-range calibration fixture: cap-tier boundaries from
// B4 defaults (Cold 1.2 / Hot 4.8), decay 0.03, generous floors. Tests tweak one field.
func baseCalibration() Calibration {
	return Calibration{
		SalaryFloor:  0.5,
		RASFallback:  5.00,
		PeakLimit:    32,
		DecayRate:    0.03,
		LeagueCap:    200, // millions
		ColdCeiling:  1.2,
		HotFloor:     4.8,
		ScarcityRank: 9,
	}
}

func basePlayer() PlayerInput {
	return PlayerInput{
		Position:   domain.PosQB,
		BasePoints: 100,
		Age:        30, // below peak → no decay
		RAS:        9.0,
		HasRAS:     true,
		Salary:     5, // 5/200 = 2.5% → Neutral
		IsVeteran:  true,
	}
}

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
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if _, err := ApplyDecay(c.age, c.peak, c.rate); err == nil {
				t.Fatalf("ApplyDecay(%v,%v,%v): expected error, got nil", c.age, c.peak, c.rate)
			}
		})
	}
}

// --- L5 cap scaling (table-driven) ---

func TestApplyCapScaling(t *testing.T) {
	cases := []struct {
		name        string
		salary, cap float64
		wantMult    float64
		wantTier    CapTier
	}{
		{"cold below ceiling", 1, 200, 1.15, CapTierCold},                      // 0.5%
		{"neutral mid", 5, 200, 1.00, CapTierNeutral},                          // 2.5%
		{"hot above floor", 12, 200, 0.85, CapTierHot},                         // 6.0%
		{"exactly at cold ceiling is neutral", 2.4, 200, 1.00, CapTierNeutral}, // 1.2% (not < ceiling)
		{"exactly at hot floor is neutral", 9.6, 200, 1.00, CapTierNeutral},    // 4.8% (not > floor)
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := ApplyCapScaling(100, c.salary, c.cap, 1.2, 4.8)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.Multiplier != c.wantMult || got.Tier != c.wantTier {
				t.Fatalf("mult/tier = %v/%v, want %v/%v", got.Multiplier, got.Tier, c.wantMult, c.wantTier)
			}
			if !approxEq(got.AdjustedScore, 100*c.wantMult) {
				t.Fatalf("AdjustedScore = %v, want %v", got.AdjustedScore, 100*c.wantMult)
			}
		})
	}
}

// PLANTED FAILURE (M3): a zero/non-finite league cap must fail loud, never emit a
// NaN/Inf score. Would FAIL if the divide-by-zero guard were removed.
func TestApplyCapScalingInvalidCapRejected(t *testing.T) {
	for _, cap := range []float64{0, -1, math.NaN(), math.Inf(1)} {
		if _, err := ApplyCapScaling(100, 5, cap, 1.2, 4.8); err == nil {
			t.Fatalf("league cap %v: expected error, got nil", cap)
		}
	}
}

// PLANTED FAILURE (M3): a non-finite salary or tier boundary makes every comparison
// false and would silently classify the player Neutral — a wrong CapTier with no error.
// Would FAIL if the L5 finite guard on salary/ceilings were removed.
func TestApplyCapScalingNonFiniteInputsRejected(t *testing.T) {
	cases := []struct {
		name                       string
		salary, coldCeil, hotFloor float64
	}{
		{"NaN salary", math.NaN(), 1.2, 4.8},
		{"Inf salary", math.Inf(1), 1.2, 4.8},
		{"NaN cold ceiling", 5, math.NaN(), 4.8},
		{"NaN hot floor", 5, 1.2, math.NaN()},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if _, err := ApplyCapScaling(100, c.salary, 200, c.coldCeil, c.hotFloor); err == nil {
				t.Fatalf("%s: expected error, got nil", c.name)
			}
		})
	}
}

// --- L1 hygiene ---

func TestApplyHygieneImputesAndFloors(t *testing.T) {
	c := baseCalibration()
	cleaned := ApplyHygiene(PlayerInput{HasRAS: false, Salary: 0.1}, c)
	if !cleaned.RASImputed || cleaned.RAS != c.RASFallback {
		t.Fatalf("expected imputed RAS %v, got %v (imputed=%v)", c.RASFallback, cleaned.RAS, cleaned.RASImputed)
	}
	if cleaned.Salary != c.SalaryFloor {
		t.Fatalf("expected salary raised to floor %v, got %v", c.SalaryFloor, cleaned.Salary)
	}
	kept := ApplyHygiene(PlayerInput{HasRAS: true, RAS: 8.5, Salary: 10}, c)
	if kept.RASImputed || kept.RAS != 8.5 || kept.Salary != 10 {
		t.Fatalf("present values should pass through unchanged, got %+v", kept)
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

// --- end-to-end accumulation with the identity Layer 4 (CLOSE GATE) ---

func TestPipelineEndToEndIdentityLayer4(t *testing.T) {
	pl := NewPipeline(nil) // nil → identity Layer 4
	p := basePlayer()
	c := baseCalibration()

	res, err := pl.Score(p, c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Age 30 < peak 32 → AgePull 1.0; identity L4 Combined 1.0; salary 2.5% → Neutral ×1.0.
	if !approxEq(res.AgePull, 1.0) || !approxEq(res.Layer4Output.Combined, 1.0) {
		t.Fatalf("expected unit AgePull/Combined, got %v / %v", res.AgePull, res.Layer4Output.Combined)
	}
	wantScouting := p.BasePoints * res.AgePull * res.Layer4Output.Combined
	if !approxEq(res.ScoutingAdjusted, wantScouting) {
		t.Fatalf("ScoutingAdjusted = %v, want %v", res.ScoutingAdjusted, wantScouting)
	}
	wantAdjusted := wantScouting * res.CapMultiplier
	if !approxEq(res.AdjustedScore, wantAdjusted) || res.CapTier != CapTierNeutral {
		t.Fatalf("AdjustedScore = %v (tier %v), want %v Neutral", res.AdjustedScore, res.CapTier, wantAdjusted)
	}
	if res.AdjustedScore != 100 { // 100 × 1 × 1 × 1.0
		t.Fatalf("identity neutral path should yield BasePoints unchanged, got %v", res.AdjustedScore)
	}
}

// A decayed, hot-tier player exercises the full accumulation with non-unit factors.
func TestPipelineDecayedHotTier(t *testing.T) {
	pl := NewPipeline(IdentityLayer4())
	p := basePlayer()
	p.Age = 35    // 3 years past peak 32
	p.Salary = 12 // 6% of 200 → Hot
	c := baseCalibration()

	res, err := pl.Score(p, c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	wantAgePull := 0.97 * 0.97 * 0.97
	wantAdjusted := 100 * wantAgePull * 1.0 * 0.85
	if res.CapTier != CapTierHot || !approxEq(res.AdjustedScore, wantAdjusted) {
		t.Fatalf("AdjustedScore = %v (tier %v), want %v Hot", res.AdjustedScore, res.CapTier, wantAdjusted)
	}
}

// Score is pure/stateless; concurrent calls on one Pipeline must be race-free (-race).
func TestPipelineConcurrentScore(t *testing.T) {
	pl := NewPipeline(nil)
	c := baseCalibration()
	var wg sync.WaitGroup
	for i := 0; i < 64; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := pl.Score(basePlayer(), c); err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		}()
	}
	wg.Wait()
}
