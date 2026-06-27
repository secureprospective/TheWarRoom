package offense

import (
	"math"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/engine"
	"github.com/secureprospective/TheWarRoom/internal/engine/l4/curve"
)

// NewTE must satisfy the engine.Layer4 contract (compile-time check, kept local so it is not
// a package-level global — gochecknoglobals).
func TestTEImplementsLayer4(_ *testing.T) {
	var _ engine.Layer4 = NewTE()
}

// TestSL019WorkedExamples is the M3 proof for the new mechanic: curve.SL019 at the TE strength
// (0.35) must reproduce the TE_Rubric §4 worked examples EXACTLY. The expected values are the
// spec's literal numbers, hardcoded — so a wrong teSL019Strength (or a broken formula) fails
// here. Breakout age 22 (base 0.50) and age trajectory 32 (base 0.10) are the two §4 tables.
func TestSL019WorkedExamples(t *testing.T) {
	cases := []struct {
		name          string
		base, rasNorm float64
		want          float64
	}{
		{"breakout-age22 RAS9.5", 0.50, 0.95, 0.666},
		{"breakout-age22 RAS7.0", 0.50, 0.70, 0.623},
		{"breakout-age22 RAS4.0", 0.50, 0.40, 0.570},
		{"age-traj32 RAS9.5", 0.10, 0.95, 0.399},
		{"age-traj32 RAS5.4", 0.10, 0.54, 0.270},
	}
	for _, c := range cases {
		got := curve.SL019(c.base, c.rasNorm, teSL019Strength, true)
		if math.Abs(got-c.want) > 1e-3 {
			t.Fatalf("%s: SL019(%.2f,%.2f,0.35) = %.4f, want %.3f (TE_Rubric §4)", c.name, c.base, c.rasNorm, got, c.want)
		}
	}
	// A base already at the ceiling has no headroom: the modulator is inert (1−base = 0).
	if got := curve.SL019(1.00, 0.95, teSL019Strength, true); math.Abs(got-1.00) > 1e-9 {
		t.Fatalf("SL019 on a maxed base should be inert, got %.4f", got)
	}
}

// TestSL019AbsentAndExtremeRAS is the SL-019 Data-Parity + safety contract: an ABSENT RAS
// contributes NOTHING (modulated = base), and an out-of-range / non-finite RAS_normalized
// cannot manufacture an extreme (it is clamped / neutralized to base).
func TestSL019AbsentAndExtremeRAS(t *testing.T) {
	if got := curve.SL019(0.50, 0.95, teSL019Strength, false); got != 0.50 {
		t.Fatalf("absent RAS must leave base unmodulated, got %.4f", got)
	}
	for _, bad := range []float64{math.NaN(), math.Inf(1)} {
		if got := curve.SL019(0.50, bad, teSL019Strength, true); got != 0.50 {
			t.Fatalf("non-finite RAS_normalized must fall back to base, got %.4f (rasNorm=%v)", got, bad)
		}
	}
	// rasNorm clamps to [0,1]: a huge value lifts no more than rasNorm == 1, never above 1.0.
	maxLift := curve.SL019(0.50, 1.0, teSL019Strength, true)
	if got := curve.SL019(0.50, 1e9, teSL019Strength, true); math.Abs(got-maxLift) > 1e-9 {
		t.Fatalf("out-of-range RAS_normalized must clamp to the rasNorm=1 lift %.4f, got %.4f", maxLift, got)
	}
	// Defense-in-depth: a non-finite BASE (a future caller without a Scurve backstop) → neutral.
	for _, bad := range []float64{math.NaN(), math.Inf(1), math.Inf(-1)} {
		if got := curve.SL019(bad, 0.95, teSL019Strength, true); got != curve.NeutralNorm {
			t.Fatalf("non-finite base must collapse to neutral, got %.4f (base=%v)", got, bad)
		}
	}
}

// TestTECurvesReproduceSpecAnchors pins the rubric's ACTUAL breakpoint tables (not just the
// helper) to the TE_Rubric §4 worked examples: age 32 must read base 0.10 and breakout age 22
// base 0.50, and the SL-019 modulation through those tables must reproduce 0.399/0.270/0.666.
// This is what catches a wrong breakpoint (e.g. a naive 29→33 linear read giving 0.125 at 32)
// or a wrong teSL019Strength — the magnitude proof eval3M's ordering check cannot give.
func TestTECurvesReproduceSpecAnchors(t *testing.T) {
	te := NewTE()
	if got := curve.Interp(te.ageTrajectory, 32); math.Abs(got-0.10) > 1e-9 {
		t.Fatalf("age-trajectory base at age 32 = %.4f, want exactly 0.10 (TE_Rubric §4)", got)
	}
	if got := curve.Interp(te.breakoutAge, 22); math.Abs(got-0.50) > 1e-9 {
		t.Fatalf("breakout-age base at age 22 = %.4f, want exactly 0.50 (TE_Rubric §4)", got)
	}
	anchors := []struct {
		name          string
		base, rasNorm float64
		want          float64
	}{
		{"age-traj32 RAS9.5", curve.Interp(te.ageTrajectory, 32), 0.95, 0.399},
		{"age-traj32 RAS5.4", curve.Interp(te.ageTrajectory, 32), 0.54, 0.270},
		{"breakout22 RAS9.5", curve.Interp(te.breakoutAge, 22), 0.95, 0.666},
	}
	for _, a := range anchors {
		if got := curve.SL019(a.base, a.rasNorm, teSL019Strength, true); math.Abs(got-a.want) > 1e-3 {
			t.Fatalf("%s through the rubric curves = %.4f, want %.3f", a.name, got, a.want)
		}
	}
}

// TestTERASActive checks the High-tier active RAS (no SL-020, AD-09 do-not-force) at the
// STEEPER TE steepness: elite RAS pushes toward the +8% cap at the full rookie weight (1.00),
// a low present RAS pulls below 1.000, and an ABSENT RAS is Data-Parity neutral (1.000).
func TestTERASActive(t *testing.T) {
	te := NewTE()
	base := engine.PlayerInput{Position: domain.PosTE, Age: 24}

	elite := te.Apply(engine.Layer4Input{Player: withRAS(base, 10.0)})
	if !(elite.RASEffective > 1.0) || elite.RASEffective > 1.0+0.08+1e-9 {
		t.Fatalf("elite RAS effective = %v, want a lift bounded by the High-tier ±8%% band", elite.RASEffective)
	}
	low := te.Apply(engine.Layer4Input{Player: withRAS(base, 0)})
	if !(low.RASEffective < 1.0) || low.RASEffective < 1.0-0.08-1e-9 {
		t.Fatalf("low RAS effective = %v, want a pull bounded by the High-tier ±8%% band", low.RASEffective)
	}
	absent := te.Apply(engine.Layer4Input{Player: engine.PlayerInput{Position: domain.PosTE, Age: 24, HasRAS: false}})
	if absent.RASEffective != 1.0 {
		t.Fatalf("absent RAS must be Data-Parity neutral 1.000, got %v", absent.RASEffective)
	}
}

// agingLateTE is a late-breakout TE past peak (age 32 → trajectory base 0.10, breakout age 22
// → base 0.50): both modulated sub-signals have headroom, so SL-019 visibly lifts breakout
// with the athletic profile (the §5 Higbee/Henry mechanism).
func agingLateTE(ras float64, hasRAS bool) engine.Layer4Input {
	return engine.Layer4Input{
		Player: engine.PlayerInput{Position: domain.PosTE, Age: 32, RAS: ras, HasRAS: hasRAS},
		Scouting: engine.ScoutingInput{
			BreakoutAge: 22, HasBreakoutAge: true, // base 0.50
			SchoolTierNorm: 0.70, HasSchoolTier: true, // Group of Five (template)
			CollegeShare: 0.15, HasCollegeShare: true, // base 0.50
		},
	}
}

// TestTESL019LiftsBreakout is the case-3M mechanic at the unit level: for the SAME aging-late
// TE, a higher RAS lifts the breakout component (more athletic → more credit on breakout-age +
// age-trajectory), and an ABSENT RAS gives the least (the modulator contributes nothing).
func TestTESL019LiftsBreakout(t *testing.T) {
	te := NewTE()
	hi := te.Apply(agingLateTE(9.5, true))
	lo := te.Apply(agingLateTE(4.0, true))
	absent := te.Apply(agingLateTE(0, false))
	if !(hi.BreakoutEffective > lo.BreakoutEffective) {
		t.Fatalf("SL-019: high-RAS breakout %v should exceed low-RAS %v", hi.BreakoutEffective, lo.BreakoutEffective)
	}
	if !(lo.BreakoutEffective > absent.BreakoutEffective) {
		t.Fatalf("SL-019: any-RAS lift %v should exceed absent-RAS (no modulation) %v", lo.BreakoutEffective, absent.BreakoutEffective)
	}
}

// TestTEFilmActive checks the film S-curve fires when a source IS populated (±5%): an elite
// composite pushes toward +5%, a neutral composite (0.50) → 1.000.
func TestTEFilmActive(t *testing.T) {
	te := NewTE()
	base := engine.ScoutingInput{BreakoutAge: 21, SchoolTierNorm: 0, CollegeShare: 0} // not under test
	elite := te.Apply(engine.Layer4Input{
		Player:   engine.PlayerInput{Position: domain.PosTE, Age: 24},
		Scouting: withFilm(base, 1.0),
	})
	if elite.FilmEffective < 1.0 || elite.FilmEffective > 1.05+1e-9 {
		t.Fatalf("elite film_effective = %v, want a lift bounded by +5%%", elite.FilmEffective)
	}
	neutral := te.Apply(engine.Layer4Input{
		Player:   engine.PlayerInput{Position: domain.PosTE, Age: 24},
		Scouting: withFilm(base, 0.50),
	})
	if math.Abs(neutral.FilmEffective-1.0) > 5e-4 {
		t.Fatalf("neutral film_effective = %v, want ≈1.000", neutral.FilmEffective)
	}
}

// TestTEAbsentBreakoutIsNeutral guards the Data-Parity Rule AND the SL-019 interaction with it:
// an ABSENT breakout age must contribute neutral and NOT be modulated (modulating the neutral
// would manufacture an athletic lift for unknown data). A present-elite breakout still exceeds
// the absent-neutral one.
func TestTEAbsentBreakoutIsNeutral(t *testing.T) {
	te := NewTE()
	// Absent breakout age at high RAS: if SL-019 wrongly modulated the neutral, this would lift.
	absent := te.Apply(engine.Layer4Input{
		Player:   engine.PlayerInput{Position: domain.PosTE, Age: 27, RAS: 9.5, HasRAS: true},
		Scouting: engine.ScoutingInput{BreakoutAge: 0, SchoolTierNorm: 0, CollegeShare: 0},
	})
	// Same player but breakout age absent at LOW RAS: the breakout-age contribution must be
	// identical (neutral, un-modulated) — proving the absent sub-signal ignores RAS entirely.
	absentLowRAS := te.Apply(engine.Layer4Input{
		Player:   engine.PlayerInput{Position: domain.PosTE, Age: 27, RAS: 0, HasRAS: true},
		Scouting: engine.ScoutingInput{BreakoutAge: 0, SchoolTierNorm: 0, CollegeShare: 0},
	})
	// The breakout composite differs only via the (present, age-trajectory) modulation, never via
	// the absent breakout age. With age 27 the trajectory base is 0.70 (headroom), so high RAS
	// lifts the age-trajectory term — absent breakout age stays neutral in BOTH.
	if !(absent.BreakoutEffective > absentLowRAS.BreakoutEffective) {
		t.Fatalf("age-trajectory should still modulate (%v vs %v) while absent breakout age stays neutral", absent.BreakoutEffective, absentLowRAS.BreakoutEffective)
	}
	present := te.Apply(engine.Layer4Input{
		Player: engine.PlayerInput{Position: domain.PosTE, Age: 25, HasRAS: false},
		Scouting: engine.ScoutingInput{
			BreakoutAge: 20, HasBreakoutAge: true, // ≤20 → 1.00
			SchoolTierNorm: 1.0, HasSchoolTier: true,
			CollegeShare: 0.30, HasCollegeShare: true,
		},
	})
	absentAll := te.Apply(engine.Layer4Input{
		Player:   engine.PlayerInput{Position: domain.PosTE, Age: 25, HasRAS: false},
		Scouting: engine.ScoutingInput{BreakoutAge: 0, SchoolTierNorm: 0, CollegeShare: 0},
	})
	if !(present.BreakoutEffective > absentAll.BreakoutEffective) {
		t.Fatalf("present-elite breakout %v should exceed absent-neutral %v", present.BreakoutEffective, absentAll.BreakoutEffective)
	}
}

// TestTEComponentsClampedToCap is the planted-failure gate (M3): pathological inputs must never
// push a component past its asymptote (film ±5%, RAS ±8%, breakout ±5%), and a NaN must not
// escape as non-finite — even through the SL-019 modulator path (bad RAS feeds rasNorm).
func TestTEComponentsClampedToCap(t *testing.T) {
	te := NewTE()
	for _, bad := range []float64{1e9, math.NaN(), math.Inf(1)} {
		out := te.Apply(engine.Layer4Input{
			Player: engine.PlayerInput{Position: domain.PosTE, Age: 24, RAS: bad, HasRAS: true},
			Scouting: engine.ScoutingInput{
				FilmComposite: bad, HasFilm: true,
				BreakoutAge: bad, HasBreakoutAge: true,
				SchoolTierNorm: bad, HasSchoolTier: true,
				CollegeShare: bad, HasCollegeShare: true,
			},
		})
		assertBand(t, "film", out.FilmEffective, 0.05)
		assertBand(t, "RAS", out.RASEffective, 0.08) // High-tier band, rookie weight 1.00
		assertBand(t, "breakout", out.BreakoutEffective, 0.05)
		if want := out.FilmEffective * out.RASEffective * out.BreakoutEffective; math.Abs(out.Combined-want) > 5e-4 {
			t.Fatalf("Combined %v != film×RAS×breakout %v", out.Combined, want)
		}
	}
}
