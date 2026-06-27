package offense

import (
	"math"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/engine"
)

// NewRB must satisfy the engine.Layer4 contract (compile-time check, kept local so it is
// not a package-level global — gochecknoglobals).
func TestRBImplementsLayer4(_ *testing.T) {
	var _ engine.Layer4 = NewRB()
}

// strongRB is RB_Rubric §5 Case 1 (Bijan-pattern): elite athlete, true-freshman breakout,
// Power Four, dominant college workload. age 24 (one shy of peak 25).
func strongRB() engine.Layer4Input {
	return engine.Layer4Input{
		Player: engine.PlayerInput{Position: domain.PosRB, Age: 24, RAS: 9.55, HasRAS: true},
		Scouting: engine.ScoutingInput{
			BreakoutAge: 19, HasBreakoutAge: true, // ≤19.5 → 1.00
			SchoolTierNorm: 1.00, HasSchoolTier: true, // Power Four
			CollegeShare: 0.53, HasCollegeShare: true, // ≥0.40 → 1.00
		},
	}
}

// thinRB is the genuinely-thin-profile vet (RB_Rubric §7 — the Herbert pattern under our
// film-neutral engine): late breakout, smaller school, LOW college workload, post-peak age.
// With film Data-Parity neutral the below-1.000 pull is driven by the breakout component
// (composite < 0.50 inflection), which is the architecture working as specified.
func thinRB() engine.Layer4Input {
	return engine.Layer4Input{
		Player: engine.PlayerInput{Position: domain.PosRB, Age: 28, RAS: 6.0, HasRAS: true},
		Scouting: engine.ScoutingInput{
			BreakoutAge: 22, HasBreakoutAge: true, // ≥21 → 0.20
			SchoolTierNorm: 0.75, HasSchoolTier: true, // RB Group of Five (RB_Rubric §4 softer non-P4)
			CollegeShare: 0.18, HasCollegeShare: true, // ≤0.20 → 0.15
		},
	}
}

// TestRBRASActive checks the Medium-tier active RAS (no SL-020): elite RAS pushes toward the
// +4% cap scaled by the 0.60 rookie weight, a low present RAS pulls below 1.000, and an ABSENT
// RAS is Data-Parity neutral (1.000).
func TestRBRASActive(t *testing.T) {
	rb := NewRB()
	base := engine.PlayerInput{Position: domain.PosRB, Age: 22}

	elite := rb.Apply(engine.Layer4Input{Player: withRAS(base, 10.0)})
	if !(elite.RASEffective > 1.0) || elite.RASEffective > 1.0+0.04*0.60+1e-9 {
		t.Fatalf("elite RAS effective = %v, want a lift bounded by the 0.60-weighted ±4%% band", elite.RASEffective)
	}
	low := rb.Apply(engine.Layer4Input{Player: withRAS(base, 0)})
	if !(low.RASEffective < 1.0) || low.RASEffective < 1.0-0.04*0.60-1e-9 {
		t.Fatalf("low RAS effective = %v, want a pull bounded by the 0.60-weighted ±4%% band", low.RASEffective)
	}
	absent := rb.Apply(engine.Layer4Input{Player: engine.PlayerInput{Position: domain.PosRB, Age: 22, HasRAS: false}})
	if absent.RASEffective != 1.0 {
		t.Fatalf("absent RAS must be Data-Parity neutral 1.000, got %v", absent.RASEffective)
	}
}

func withRAS(p engine.PlayerInput, ras float64) engine.PlayerInput {
	p.RAS = ras
	p.HasRAS = true
	return p
}

// TestRBHerbertPattern is the case-3B mechanic at the unit level: a genuinely-thin-profile
// vet's Combined pulls BELOW 1.000 (driven by breakout, film neutral), while a strong-profile
// RB stays above — Layer 4 separating thin from elite scouting profiles (RB_Rubric §7).
func TestRBHerbertPattern(t *testing.T) {
	rb := NewRB()
	thin := rb.Apply(thinRB())
	strong := rb.Apply(strongRB())
	if !(thin.Combined < 1.0) {
		t.Fatalf("thin-profile RB Combined %v should pull below 1.000", thin.Combined)
	}
	if !(strong.Combined > 1.0) {
		t.Fatalf("strong-profile RB Combined %v should sit above 1.000", strong.Combined)
	}
	if !(strong.Combined > thin.Combined) {
		t.Fatalf("strong RB %v should out-score thin RB %v", strong.Combined, thin.Combined)
	}
	// Film is Data-Parity neutral this session (no source) — the pull is breakout-driven.
	if thin.FilmEffective != 1.0 || strong.FilmEffective != 1.0 {
		t.Fatalf("film must be Data-Parity neutral 1.000 (no source), got thin=%v strong=%v", thin.FilmEffective, strong.FilmEffective)
	}
	if !(thin.BreakoutEffective < 1.0) {
		t.Fatalf("thin RB breakout %v should be the sub-1.000 driver", thin.BreakoutEffective)
	}
}

// TestRBFilmActive checks the film S-curve fires when a source IS populated (the ±5% band):
// an elite composite pushes toward +5%, a neutral composite (0.50) → 1.000.
func TestRBFilmActive(t *testing.T) {
	rb := NewRB()
	base := engine.ScoutingInput{BreakoutAge: 20, SchoolTierNorm: 0, CollegeShare: 0} // not under test
	elite := rb.Apply(engine.Layer4Input{
		Player:   engine.PlayerInput{Position: domain.PosRB, Age: 22},
		Scouting: withFilm(base, 1.0),
	})
	if elite.FilmEffective < 1.0 || elite.FilmEffective > 1.05+1e-9 {
		t.Fatalf("elite film_effective = %v, want a lift bounded by +5%%", elite.FilmEffective)
	}
	neutral := rb.Apply(engine.Layer4Input{
		Player:   engine.PlayerInput{Position: domain.PosRB, Age: 22},
		Scouting: withFilm(base, 0.50),
	})
	if math.Abs(neutral.FilmEffective-1.0) > 5e-4 {
		t.Fatalf("neutral film_effective = %v, want ≈1.000", neutral.FilmEffective)
	}
}

// TestRBAbsentBreakoutIsNeutral is the B1 regression guard (GLM review): an ABSENT breakout
// sub-signal must contribute neutral, never the curve ceiling/floor a raw zero hits.
func TestRBAbsentBreakoutIsNeutral(t *testing.T) {
	rb := NewRB()
	absent := rb.Apply(engine.Layer4Input{
		Player:   engine.PlayerInput{Position: domain.PosRB, Age: 21, HasRAS: false}, // age ≤21 → trajectory ceiling
		Scouting: engine.ScoutingInput{BreakoutAge: 0, SchoolTierNorm: 0, CollegeShare: 0},
	})
	present := rb.Apply(engine.Layer4Input{
		Player: engine.PlayerInput{Position: domain.PosRB, Age: 21, HasRAS: false},
		Scouting: engine.ScoutingInput{
			BreakoutAge: 0, HasBreakoutAge: true, // raw zero PRESENT → ceiling (would be the bug if absent read this way)
			SchoolTierNorm: 1.0, HasSchoolTier: true,
			CollegeShare: 0.50, HasCollegeShare: true,
		},
	})
	if !(present.BreakoutEffective > absent.BreakoutEffective) {
		t.Fatalf("present-elite breakout %v should exceed absent-neutral %v", present.BreakoutEffective, absent.BreakoutEffective)
	}
}

// TestRBComponentsClampedToCap is the planted-failure gate (M3): pathological inputs must
// never push a component past its asymptote (film ±5%, RAS ±4%, breakout ±5%), and a NaN
// must not escape as non-finite.
func TestRBComponentsClampedToCap(t *testing.T) {
	rb := NewRB()
	for _, bad := range []float64{1e9, math.NaN(), math.Inf(1)} {
		out := rb.Apply(engine.Layer4Input{
			Player: engine.PlayerInput{Position: domain.PosRB, Age: 24, RAS: bad, HasRAS: true},
			Scouting: engine.ScoutingInput{
				FilmComposite: bad, HasFilm: true,
				BreakoutAge: bad, HasBreakoutAge: true,
				SchoolTierNorm: bad, HasSchoolTier: true,
				CollegeShare: bad, HasCollegeShare: true,
			},
		})
		assertBand(t, "film", out.FilmEffective, 0.05)
		assertBand(t, "RAS", out.RASEffective, 0.04*0.60) // RAS deviation is scaled by the rookie weight
		assertBand(t, "breakout", out.BreakoutEffective, 0.05)
		if want := out.FilmEffective * out.RASEffective * out.BreakoutEffective; math.Abs(out.Combined-want) > 5e-4 {
			t.Fatalf("Combined %v != film×RAS×breakout %v", out.Combined, want)
		}
	}
}

func assertBand(t *testing.T, name string, v, capBand float64) {
	t.Helper()
	if math.IsNaN(v) || math.IsInf(v, 0) {
		t.Fatalf("%s effect %v is non-finite — a non-finite input escaped the clamp", name, v)
	}
	if v < 1-capBand-1e-9 || v > 1+capBand+1e-9 {
		t.Fatalf("%s effect %v escaped its expected band ±%v", name, v, capBand)
	}
}
