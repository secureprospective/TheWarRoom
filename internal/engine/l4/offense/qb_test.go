package offense

import (
	"math"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/engine"
)

// NewQB must satisfy the engine.Layer4 contract (compile-time check, kept local so it is
// not a package-level global — gochecknoglobals).
func TestQBImplementsLayer4(_ *testing.T) {
	var _ engine.Layer4 = NewQB()
}

func approx(a, b float64) bool { return math.Abs(a-b) < 5e-4 }

// TestQBSL020 is the SL-020 gate: a QB's RAS contributes NOTHING to Layer 4 — RASEffective
// is forced to exactly 1.000 for any RAS, low or elite. This is the property eval3C asserts.
func TestQBSL020(t *testing.T) {
	qb := NewQB()
	for _, ras := range []float64{0.0, 0.10, 5.0, 9.99, 10.0} {
		out := qb.Apply(engine.Layer4Input{
			Player:   engine.PlayerInput{Position: domain.PosQB, Age: 25, RAS: ras},
			Scouting: engine.ScoutingInput{},
		})
		if out.RASEffective != 1.0 {
			t.Fatalf("RAS %v → RASEffective %v, want exactly 1.0", ras, out.RASEffective)
		}
	}
}

// TestQBDanielsBreakout reproduces QB_Rubric §5 Case 1 (Jayden Daniels): a maxed breakout
// composite (every sub-signal at ceiling) → breakout_effective ≈ 1.050. Film absent (neutral
// 1.000), RAS forced 1.000 → Combined ≈ 1.050.
func TestQBDanielsBreakout(t *testing.T) {
	qb := NewQB()
	out := qb.Apply(engine.Layer4Input{
		Player: engine.PlayerInput{Position: domain.PosQB, Age: 25}, // ≤28 → trajectory 1.00
		Scouting: engine.ScoutingInput{
			BreakoutAge: 19, HasBreakoutAge: true, // ≤20 → 1.00
			SchoolTierNorm: 1.00, HasSchoolTier: true, // Power Four
			CollegeShare: 0.90, HasCollegeShare: true, // ≥0.65 → 1.00
		},
	})
	if !approx(out.BreakoutEffective, 1.050) {
		t.Fatalf("Daniels breakout_effective = %v, want ≈1.050", out.BreakoutEffective)
	}
	if out.FilmEffective != 1.0 {
		t.Fatalf("absent film must be neutral 1.000, got %v", out.FilmEffective)
	}
	if !approx(out.Combined, 1.050) {
		t.Fatalf("Daniels Combined = %v, want ≈1.050", out.Combined)
	}
}

// TestQBCarrBreakout reproduces QB_Rubric §5 Case 2 (Derek Carr): an Average-zone composite
// of 0.6925 → breakout_effective ≈ 1.039. With film absent and RAS forced, Combined ≈ 1.039.
func TestQBCarrBreakout(t *testing.T) {
	qb := NewQB()
	out := qb.Apply(engine.Layer4Input{
		Player: engine.PlayerInput{Position: domain.PosQB, Age: 35}, // 35 → trajectory 0.15
		Scouting: engine.ScoutingInput{
			BreakoutAge: 21, HasBreakoutAge: true, // → 0.80
			SchoolTierNorm: 0.70, HasSchoolTier: true, // Group of Five
			CollegeShare: 0.60, HasCollegeShare: true, // interp(0.50→0.55, 0.65→1.00) → 0.85
		},
	})
	if !approx(out.BreakoutEffective, 1.039) {
		t.Fatalf("Carr breakout_effective = %v, want ≈1.039", out.BreakoutEffective)
	}
	if !approx(out.Combined, 1.039) {
		t.Fatalf("Carr Combined = %v, want ≈1.039", out.Combined)
	}
}

// TestQBFilmActive checks the film S-curve fires when a film source IS populated: an
// elite composite (1.0) pushes film toward the +5% cap; a neutral composite (0.50) → 1.000.
func TestQBFilmActive(t *testing.T) {
	qb := NewQB()
	base := engine.ScoutingInput{BreakoutAge: 22, SchoolTierNorm: 0, CollegeShare: 0} // not under test

	elite := qb.Apply(engine.Layer4Input{
		Player:   engine.PlayerInput{Position: domain.PosQB, Age: 25},
		Scouting: withFilm(base, 1.0),
	})
	if !approx(elite.FilmEffective, 1.050) {
		t.Fatalf("elite film_effective = %v, want ≈1.050", elite.FilmEffective)
	}
	neutral := qb.Apply(engine.Layer4Input{
		Player:   engine.PlayerInput{Position: domain.PosQB, Age: 25},
		Scouting: withFilm(base, 0.50),
	})
	if !approx(neutral.FilmEffective, 1.000) {
		t.Fatalf("neutral film_effective = %v, want ≈1.000", neutral.FilmEffective)
	}
}

func withFilm(sc engine.ScoutingInput, composite float64) engine.ScoutingInput {
	sc.FilmComposite = composite
	sc.HasFilm = true
	return sc
}

// TestQBComponentsClampedToCap is the planted-failure gate (M3): even a pathological,
// out-of-range input must never push a component past its ±5% asymptote. Feeding a wildly
// out-of-band film composite and breakout share must still yield effects within [0.95,1.05].
func TestQBComponentsClampedToCap(t *testing.T) {
	qb := NewQB()
	for _, film := range []float64{1e9, math.NaN(), math.Inf(1)} {
		out := qb.Apply(engine.Layer4Input{
			Player: engine.PlayerInput{Position: domain.PosQB, Age: 25},
			Scouting: engine.ScoutingInput{
				FilmComposite: film, HasFilm: true,
				BreakoutAge: -1e9, HasBreakoutAge: true,
				SchoolTierNorm: 1e9, HasSchoolTier: true,
				CollegeShare: 1e9, HasCollegeShare: true,
			},
		})
		assertClamped(t, out)
	}
}

// assertClamped checks the M3 invariant: each component stays in its ±5% band, the result
// is finite, and Combined is exactly the product (no overall L4 cap).
func assertClamped(t *testing.T, out engine.Layer4Output) {
	t.Helper()
	// Each COMPONENT is bounded to its ±5% asymptote AND finite (a NaN input must NOT escape
	// — the M1 regression: NaN defeats min/max comparisons, so scurve must guard it).
	for name, v := range map[string]float64{
		"film": out.FilmEffective, "breakout": out.BreakoutEffective,
	} {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			t.Fatalf("%s effect %v is non-finite — a non-finite input escaped the clamp", name, v)
		}
		if v < 0.95-1e-9 || v > 1.05+1e-9 {
			t.Fatalf("%s effect %v escaped the ±5%% cap band [0.95,1.05]", name, v)
		}
	}
	if out.RASEffective != 1.0 {
		t.Fatalf("RASEffective must stay 1.000, got %v", out.RASEffective)
	}
	// Combined is the product of the components — it has NO overall L4 cap; it must equal
	// exactly film × RAS × breakout.
	if want := out.FilmEffective * out.RASEffective * out.BreakoutEffective; !approx(out.Combined, want) {
		t.Fatalf("Combined %v != film×RAS×breakout %v", out.Combined, want)
	}
}

// TestQBAbsentBreakoutIsNeutral is the B1 regression guard (GLM review): an ABSENT breakout
// sub-signal (Has* false) must contribute NEUTRAL, never the curve ceiling/floor a raw zero
// hits. A QB with all breakout sub-signals absent and an age at the trajectory ceiling must
// score breakout ≈ 1.000 (not the elite ~1.05 a zero breakout age would fake via interp).
func TestQBAbsentBreakoutIsNeutral(t *testing.T) {
	qb := NewQB()
	// All breakout sub-signals absent; raw zeros present to prove they are IGNORED.
	out := qb.Apply(engine.Layer4Input{
		Player:   engine.PlayerInput{Position: domain.PosQB, Age: 25}, // age ≤28 → trajectory ceiling 1.00
		Scouting: engine.ScoutingInput{BreakoutAge: 0, SchoolTierNorm: 0, CollegeShare: 0},
	})
	// composite = 0.30·0.50 + 0.25·0.50 + 0.30·0.50 + 0.15·1.00 = 0.575 (age still real).
	// A raw-zero bug would give 0.30·1.00 (breakout age ceiling) + 0.30·0.15 (share floor) + …
	// and a clearly different, inflated breakout. Assert we are near neutral, not inflated.
	if out.BreakoutEffective <= 1.0 || out.BreakoutEffective > 1.02 {
		t.Fatalf("absent breakout should be ~neutral (small lift from age only), got %v", out.BreakoutEffective)
	}
	// Sanity: had breakout age been read as a raw zero it would hit the ≤20 ceiling and push
	// the composite far higher; confirm the absent path is materially below that.
	rawZeroElite := qb.Apply(engine.Layer4Input{
		Player: engine.PlayerInput{Position: domain.PosQB, Age: 25},
		Scouting: engine.ScoutingInput{
			BreakoutAge: 0, HasBreakoutAge: true, // raw zero TREATED AS PRESENT → ceiling (the bug)
			SchoolTierNorm: 1.0, HasSchoolTier: true,
			CollegeShare: 0.90, HasCollegeShare: true,
		},
	})
	if !(rawZeroElite.BreakoutEffective > out.BreakoutEffective) {
		t.Fatalf("present-elite breakout %v should exceed absent-neutral %v",
			rawZeroElite.BreakoutEffective, out.BreakoutEffective)
	}
}
