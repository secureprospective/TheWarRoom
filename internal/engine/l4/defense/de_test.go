package defense

import (
	"math"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/engine"
	"github.com/secureprospective/TheWarRoom/internal/engine/l4/curve"
)

// NewDE must satisfy the engine.Layer4 contract (compile-time check, kept local so it is not
// a package-level global — gochecknoglobals).
func TestDEImplementsLayer4(_ *testing.T) {
	var _ engine.Layer4 = NewDE()
}

// TestDESL019WorkedExamples is the M3 proof that DE reuses curve.SL019 at the DE strength
// (0.35) and reproduces the DE_Rubric §4 worked examples EXACTLY. The expected values are the
// spec's literal numbers, hardcoded — so a wrong deSL019Strength (or a broken formula) fails
// here. Breakout age 21 (base 0.15) and age trajectory 30/32/34 (base 0.50/0.20/0.00) are the
// two §4 tables. This is the SAME helper TE proved — the point is that DE inherits it verbatim.
func TestDESL019WorkedExamples(t *testing.T) {
	cases := []struct {
		name          string
		base, rasNorm float64
		want          float64
	}{
		{"breakout-age21 RAS9.99", 0.15, 0.999, 0.447},
		{"breakout-age21 RAS7.50", 0.15, 0.750, 0.373},
		{"breakout-age21 RAS4.18", 0.15, 0.418, 0.274},
		{"age-traj30 RAS9.99", 0.50, 0.999, 0.675},
		{"age-traj32 RAS7.50", 0.20, 0.750, 0.410},
		{"age-traj34 RAS4.18", 0.00, 0.418, 0.146},
	}
	for _, c := range cases {
		got := curve.SL019(c.base, c.rasNorm, deSL019Strength, true)
		if math.Abs(got-c.want) > 1e-3 {
			t.Fatalf("%s: SL019(%.2f,%.3f,0.35) = %.4f, want %.3f (DE_Rubric §4)", c.name, c.base, c.rasNorm, got, c.want)
		}
	}
	// A base already at the ceiling has no headroom: the modulator is inert (1−base = 0).
	if got := curve.SL019(1.00, 0.999, deSL019Strength, true); math.Abs(got-1.00) > 1e-9 {
		t.Fatalf("SL019 on a maxed base should be inert, got %.4f", got)
	}
}

// TestDECurvesReproduceSpecAnchors pins the rubric's ACTUAL breakpoint tables (not just the
// helper) to the DE_Rubric §4 worked examples: age 30 reads base 0.50, age 32 → 0.20, age 34 →
// 0.00, and breakout age 21 → 0.15; the SL-019 modulation through those tables must reproduce
// 0.675/0.410/0.146/0.447. This catches a wrong breakpoint (the RB-shaped half-year curve has
// 0.50 at 20.5, not at 21) or a wrong deSL019Strength — the magnitude proof eval3E cannot give.
func TestDECurvesReproduceSpecAnchors(t *testing.T) {
	de := NewDE()
	for _, c := range []struct {
		name string
		got  float64
		want float64
	}{
		{"age-traj30", curve.Interp(de.ageTrajectory, 30), 0.50},
		{"age-traj32", curve.Interp(de.ageTrajectory, 32), 0.20},
		{"age-traj34", curve.Interp(de.ageTrajectory, 34), 0.00},
		{"breakout-age21", curve.Interp(de.breakoutAge, 21), 0.15},
		{"breakout-age20.5", curve.Interp(de.breakoutAge, 20.5), 0.50},
		// College-share curve anchors (DE-divergent from TE — pin them so a typo is caught here,
		// not only indirectly via fixtures). DE_Rubric §4: ≤0.12→0.15, 0.20→0.55, ≥0.28→1.00.
		{"college0.12", curve.Interp(de.collegeShare, 0.12), 0.15},
		{"college0.20", curve.Interp(de.collegeShare, 0.20), 0.55},
		{"college0.28", curve.Interp(de.collegeShare, 0.28), 1.00},
	} {
		if math.Abs(c.got-c.want) > 1e-9 {
			t.Fatalf("%s base = %.4f, want exactly %.2f (DE_Rubric §4)", c.name, c.got, c.want)
		}
	}
	anchors := []struct {
		name          string
		base, rasNorm float64
		want          float64
	}{
		{"age-traj30 RAS9.99", curve.Interp(de.ageTrajectory, 30), 0.999, 0.675},
		{"age-traj32 RAS7.50", curve.Interp(de.ageTrajectory, 32), 0.750, 0.410},
		{"age-traj34 RAS4.18", curve.Interp(de.ageTrajectory, 34), 0.418, 0.146},
		{"breakout21 RAS9.99", curve.Interp(de.breakoutAge, 21), 0.999, 0.447},
	}
	for _, a := range anchors {
		if got := curve.SL019(a.base, a.rasNorm, deSL019Strength, true); math.Abs(got-a.want) > 1e-3 {
			t.Fatalf("%s through the rubric curves = %.4f, want %.3f", a.name, got, a.want)
		}
	}
}

// TestDERASActive checks the High-tier active RAS (no SL-020, AD-09 do-not-force) at the
// STANDARD DE steepness 10.0: elite RAS pushes toward the +8% cap at the full rookie weight
// (1.00), a low present RAS pulls below 1.000, and an ABSENT RAS is Data-Parity neutral (1.000).
func TestDERASActive(t *testing.T) {
	de := NewDE()
	elite := de.Apply(engine.Layer4Input{Player: engine.PlayerInput{Position: domain.PosDE, Age: 24, RAS: 10.0, HasRAS: true}})
	if !(elite.RASEffective > 1.07) || elite.RASEffective > 1.08+1e-9 {
		t.Fatalf("elite RAS effective = %v, want near +8%% cap (≈1.079)", elite.RASEffective)
	}
	low := de.Apply(engine.Layer4Input{Player: engine.PlayerInput{Position: domain.PosDE, Age: 24, RAS: 0, HasRAS: true}})
	if !(low.RASEffective < 0.93) || low.RASEffective < 0.92-1e-9 {
		t.Fatalf("low RAS effective = %v, want near −8%% cap (≈0.921)", low.RASEffective)
	}
	absent := de.Apply(engine.Layer4Input{Player: engine.PlayerInput{Position: domain.PosDE, Age: 24, HasRAS: false}})
	if absent.RASEffective != 1.0 {
		t.Fatalf("absent RAS must be Data-Parity neutral 1.000, got %v", absent.RASEffective)
	}
}

// TestDEFilmStandardBand checks DE film is the STANDARD ±5% (NOT DT's SL-005 ±3%): an elite
// composite lands inside [1.0,1.05] and ABOVE where DT's compressed ±3% curve would put it.
func TestDEFilmStandardBand(t *testing.T) {
	de, dt := NewDE(), NewDT()
	in := engine.Layer4Input{
		Player:   engine.PlayerInput{Position: domain.PosDE, Age: 24, HasRAS: false},
		Scouting: engine.ScoutingInput{FilmComposite: 1.0, HasFilm: true},
	}
	out := de.Apply(in)
	if out.FilmEffective < 1.0 || out.FilmEffective > 1.05+1e-9 {
		t.Fatalf("DE film effective %v escaped the standard ±5%% band", out.FilmEffective)
	}
	// DE's ±5% must reach higher than DT's SL-005-compressed ±3% on the same elite composite.
	dtOut := dt.Apply(engine.Layer4Input{Player: engine.PlayerInput{Position: domain.PosDT, Age: 24, HasRAS: false}, Scouting: engine.ScoutingInput{FilmComposite: 1.0, HasFilm: true}})
	if !(out.FilmEffective > dtOut.FilmEffective) {
		t.Fatalf("DE film %v (±5%%) should exceed DT film %v (±3%% SL-005)", out.FilmEffective, dtOut.FilmEffective)
	}
}

// agingLateDE is a late-breakout DE past peak (age 32 → trajectory base 0.20, breakout age 21
// → base 0.15): both modulated sub-signals have headroom, so SL-019 visibly lifts breakout with
// the athletic profile (the §5 Garrett/Lawrence mechanism).
func agingLateDE(ras float64, hasRAS bool) engine.Layer4Input {
	return engine.Layer4Input{
		Player: engine.PlayerInput{Position: domain.PosDE, Age: 32, RAS: ras, HasRAS: hasRAS},
		Scouting: engine.ScoutingInput{
			BreakoutAge: 21, HasBreakoutAge: true, // base 0.15
			SchoolTierNorm: 0.70, HasSchoolTier: true, // Group of Five (template)
			CollegeShare: 0.20, HasCollegeShare: true, // base 0.55
		},
	}
}

// TestDESL019LiftsBreakout is the case-3E mechanic at the unit level: for the SAME aging-late
// DE, a higher RAS lifts the breakout component (more athletic → more credit on breakout-age +
// age-trajectory), and an ABSENT RAS gives the least (the modulator contributes nothing).
func TestDESL019LiftsBreakout(t *testing.T) {
	de := NewDE()
	hi := de.Apply(agingLateDE(9.99, true))
	lo := de.Apply(agingLateDE(4.18, true))
	absent := de.Apply(agingLateDE(0, false))
	if !(hi.BreakoutEffective > lo.BreakoutEffective) {
		t.Fatalf("SL-019: high-RAS breakout %v should exceed low-RAS %v", hi.BreakoutEffective, lo.BreakoutEffective)
	}
	if !(lo.BreakoutEffective > absent.BreakoutEffective) {
		t.Fatalf("SL-019: any-RAS lift %v should exceed absent-RAS (no modulation) %v", lo.BreakoutEffective, absent.BreakoutEffective)
	}
}

// TestDEAbsentBreakoutIsNeutral guards the Data-Parity Rule AND its SL-019 interaction: an
// ABSENT breakout age must contribute neutral and NOT be modulated (modulating the neutral
// would manufacture an athletic lift for unknown data). A present-elite breakout still exceeds
// the absent-neutral one.
func TestDEAbsentBreakoutIsNeutral(t *testing.T) {
	de := NewDE()
	// Absent breakout age at high vs low RAS, age 27 (trajectory base 0.85 — headroom): the
	// breakout differs ONLY via age-trajectory modulation; the absent breakout age stays neutral.
	hiRAS := de.Apply(engine.Layer4Input{
		Player:   engine.PlayerInput{Position: domain.PosDE, Age: 27, RAS: 9.5, HasRAS: true},
		Scouting: engine.ScoutingInput{BreakoutAge: 0, SchoolTierNorm: 0, CollegeShare: 0},
	})
	loRAS := de.Apply(engine.Layer4Input{
		Player:   engine.PlayerInput{Position: domain.PosDE, Age: 27, RAS: 0, HasRAS: true},
		Scouting: engine.ScoutingInput{BreakoutAge: 0, SchoolTierNorm: 0, CollegeShare: 0},
	})
	if !(hiRAS.BreakoutEffective > loRAS.BreakoutEffective) {
		t.Fatalf("age-trajectory should still modulate (%v vs %v) while absent breakout age stays neutral", hiRAS.BreakoutEffective, loRAS.BreakoutEffective)
	}
	present := de.Apply(engine.Layer4Input{
		Player: engine.PlayerInput{Position: domain.PosDE, Age: 26, HasRAS: false},
		Scouting: engine.ScoutingInput{
			BreakoutAge: 19, HasBreakoutAge: true, // ≤19.5 → 1.00
			SchoolTierNorm: 1.0, HasSchoolTier: true,
			CollegeShare: 0.28, HasCollegeShare: true,
		},
	})
	absentAll := de.Apply(engine.Layer4Input{
		Player:   engine.PlayerInput{Position: domain.PosDE, Age: 26, HasRAS: false},
		Scouting: engine.ScoutingInput{BreakoutAge: 0, SchoolTierNorm: 0, CollegeShare: 0},
	})
	if !(present.BreakoutEffective > absentAll.BreakoutEffective) {
		t.Fatalf("present-elite breakout %v should exceed absent-neutral %v", present.BreakoutEffective, absentAll.BreakoutEffective)
	}
}

// TestDEMagnitudeThroughApply closes the GLM-review wiring lead: the worked-example tests call
// curve.SL019 directly, and eval3E only checks ORDERING — neither proves DE.Apply passes the
// right constants AT THE CALL SITE. A hardcoded wrong SL-019 strength, a wrong rasNorm divisor,
// or a RAS steepness regressed to TE's 11.0 would survive all of those. This drives the real
// Apply and pins two magnitudes computed by hand from the §3/§4 spec to 1e-3.
func TestDEMagnitudeThroughApply(t *testing.T) {
	de := NewDE()

	// RAS component at RAS 6.5 (rasNorm 0.65): Scurve(0.65,0.50,10.0,0.08) = 1 + 0.08×(2σ(1.5)−1)
	// = 1.05081. This pins steepness 10.0 (a regression to 11.0 gives ≈1.05423), inflection 0.50,
	// cap ±8%, rookie weight 1.00, and the raw/10 divisor — all through Apply.
	ras := de.Apply(engine.Layer4Input{Player: engine.PlayerInput{Position: domain.PosDE, Age: 26, RAS: 6.5, HasRAS: true}})
	if math.Abs(ras.RASEffective-1.05081) > 1e-3 {
		t.Fatalf("RASEffective(RAS 6.5) = %.5f, want ≈1.05081 (steepness 10.0, not 11.0; raw/10)", ras.RASEffective)
	}

	// Breakout via SL-019 through Apply: breakout age 21 (base 0.15) at RAS 9.99 modulates to
	// 0.447; age 26 → trajectory base 1.00 (inert); school + college absent → neutral 0.50.
	// composite = 0.30×0.447 + 0.20×0.50 + 0.35×0.50 + 0.15×1.00 = 0.55916;
	// breakout = Scurve(0.55916,0.50,11.0,0.05) ≈ 1.01572. A wrong SL-019 strength (e.g. 0.25 →
	// modulated 0.362 → breakout ≈1.00916) or a wrong rasNorm divisor fails here.
	bk := de.Apply(engine.Layer4Input{
		Player:   engine.PlayerInput{Position: domain.PosDE, Age: 26, RAS: 9.99, HasRAS: true},
		Scouting: engine.ScoutingInput{BreakoutAge: 21, HasBreakoutAge: true},
	})
	if math.Abs(bk.BreakoutEffective-1.01572) > 1e-3 {
		t.Fatalf("BreakoutEffective (BO-age 21 @ RAS 9.99, else neutral) = %.5f, want ≈1.01572 (SL-019 strength 0.35, raw/10)", bk.BreakoutEffective)
	}
}

// TestDEComponentsClampedToCap is the planted-failure gate (M3): pathological inputs must never
// push a component past its asymptote (film ±5%, RAS ±8%, breakout ±5%), and a NaN must not
// escape as non-finite — even through the SL-019 modulator path (bad RAS feeds rasNorm).
func TestDEComponentsClampedToCap(t *testing.T) {
	de := NewDE()
	for _, bad := range []float64{1e9, math.NaN(), math.Inf(1)} {
		out := de.Apply(engine.Layer4Input{
			Player: engine.PlayerInput{Position: domain.PosDE, Age: 32, RAS: bad, HasRAS: true},
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
		if want := out.FilmEffective * out.RASEffective * out.BreakoutEffective; !approx(out.Combined, want) {
			t.Fatalf("Combined %v != film×RAS×breakout %v", out.Combined, want)
		}
	}
}
