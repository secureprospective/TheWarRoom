package defense

import (
	"math"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/engine"
	"github.com/secureprospective/TheWarRoom/internal/engine/l4/curve"
)

// NewCB must satisfy the engine.Layer4 contract (compile-time check, kept local so it is not
// a package-level global — gochecknoglobals).
func TestCBImplementsLayer4(_ *testing.T) {
	var _ engine.Layer4 = NewCB()
}

// TestCBHasNGSAnchor proves the case-3I introspection hook: CB reports an NGS coverage anchor
// (CB_Rubric §2/§7). The other defensive rubrics built so far do NOT implement the method, so
// a type assertion against them fails — the harness reads that as "no anchor" (asserted in
// harness/eval3I against a non-NGS control). DT/DE/LB are checked here to lock that contract.
func TestCBHasNGSAnchor(t *testing.T) {
	if !NewCB().HasNGSAnchor() {
		t.Fatal("CB must report an NGS coverage anchor")
	}
	type ngsAnchor interface{ HasNGSAnchor() bool }
	for name, r := range map[string]engine.Layer4{"DT": NewDT(), "DE": NewDE(), "LB": NewLB()} {
		if _, ok := r.(ngsAnchor); ok {
			t.Fatalf("%s must NOT expose an NGS anchor (NGS is CB/S-only)", name)
		}
	}
}

// TestCBSL019WorkedExamples is the M3 proof that CB reuses curve.SL019 at the REDUCED CB
// strength (0.30, not TE/DE's 0.35) and reproduces the CB_Rubric §4 worked examples EXACTLY.
// The expected values are the spec's literal numbers, hardcoded — so a wrong cbSL019Strength
// (e.g. a copy-paste of 0.35) fails here. This is the SAME helper TE/DE proved; the point is
// that CB inherits it verbatim at a different strength passed as a parameter.
func TestCBSL019WorkedExamples(t *testing.T) {
	cases := []struct {
		name          string
		base, rasNorm float64
		want          float64
	}{
		{"breakout-age RAS9.99", 0.15, 0.999, 0.405},
		{"breakout-age RAS5.00", 0.15, 0.500, 0.278},
		{"age-traj28 RAS9.50", 0.50, 0.950, 0.6425},
		{"age-traj30 RAS7.00", 0.20, 0.700, 0.368},
		{"age-traj32 RAS9.30", 0.00, 0.930, 0.279},
	}
	for _, c := range cases {
		got := curve.SL019(c.base, c.rasNorm, cbSL019Strength, true)
		if math.Abs(got-c.want) > 1e-3 {
			t.Fatalf("%s: SL019(%.2f,%.3f,0.30) = %.4f, want %.4f (CB_Rubric §4)", c.name, c.base, c.rasNorm, got, c.want)
		}
	}
	// A base already at the ceiling has no headroom: the modulator is inert (1−base = 0).
	if got := curve.SL019(1.00, 0.999, cbSL019Strength, true); math.Abs(got-1.00) > 1e-9 {
		t.Fatalf("SL019 on a maxed base should be inert, got %.4f", got)
	}
}

// TestCBCurvesReproduceSpecAnchors pins the rubric's ACTUAL breakpoint tables to the CB_Rubric
// §4 worked examples: the age-trajectory curve peaks at 28 (one earlier than WR/TE/LB's 29) and
// craters to 0.00 by 32; the breakout-age half-year curve reads 0.15 at 22.5; college share
// (PD + INT) anchors at 0.08/0.16/0.24. This catches a wrong breakpoint (e.g. a peak-29 copy
// from DE) the magnitude proofs cannot.
func TestCBCurvesReproduceSpecAnchors(t *testing.T) {
	cb := NewCB()
	for _, c := range []struct {
		name string
		got  float64
		want float64
	}{
		{"age-traj24", curve.Interp(cb.ageTrajectory, 24), 1.00},
		{"age-traj26", curve.Interp(cb.ageTrajectory, 26), 0.70},
		{"age-traj28peak", curve.Interp(cb.ageTrajectory, 28), 0.50},
		{"age-traj30", curve.Interp(cb.ageTrajectory, 30), 0.20},
		{"age-traj32", curve.Interp(cb.ageTrajectory, 32), 0.00},
		{"breakout-age19.5", curve.Interp(cb.breakoutAge, 19.5), 1.00},
		{"breakout-age20.5", curve.Interp(cb.breakoutAge, 20.5), 0.75},
		{"breakout-age21.5", curve.Interp(cb.breakoutAge, 21.5), 0.45},
		{"breakout-age22.5", curve.Interp(cb.breakoutAge, 22.5), 0.15},
		// College-share curve anchors (CB-specific PD + INT bundle): ≤0.08→0.15, 0.16→0.55, ≥0.24→1.00.
		{"college0.08", curve.Interp(cb.collegeShare, 0.08), 0.15},
		{"college0.16", curve.Interp(cb.collegeShare, 0.16), 0.55},
		{"college0.24", curve.Interp(cb.collegeShare, 0.24), 1.00},
	} {
		if math.Abs(c.got-c.want) > 1e-9 {
			t.Fatalf("%s base = %.4f, want exactly %.2f (CB_Rubric §4)", c.name, c.got, c.want)
		}
	}
}

// TestCBRASActive checks the High-tier active RAS (no SL-020, AD-09 do-not-force) at the CB
// steepness 11.0: elite RAS pushes toward the +8% cap at the full rookie weight (1.00), a low
// present RAS pulls below 1.000, and an ABSENT RAS is Data-Parity neutral (1.000).
func TestCBRASActive(t *testing.T) {
	cb := NewCB()
	elite := cb.Apply(engine.Layer4Input{Player: engine.PlayerInput{Position: domain.PosCB, Age: 24, RAS: 10.0, HasRAS: true}})
	if !(elite.RASEffective > 1.07) || elite.RASEffective > 1.08+1e-9 {
		t.Fatalf("elite RAS effective = %v, want near +8%% cap (≈1.0793)", elite.RASEffective)
	}
	low := cb.Apply(engine.Layer4Input{Player: engine.PlayerInput{Position: domain.PosCB, Age: 24, RAS: 0, HasRAS: true}})
	if !(low.RASEffective < 0.93) || low.RASEffective < 0.92-1e-9 {
		t.Fatalf("low RAS effective = %v, want near −8%% cap (≈0.9207)", low.RASEffective)
	}
	absent := cb.Apply(engine.Layer4Input{Player: engine.PlayerInput{Position: domain.PosCB, Age: 24, HasRAS: false}})
	if absent.RASEffective != 1.0 {
		t.Fatalf("absent RAS must be Data-Parity neutral 1.000, got %v", absent.RASEffective)
	}
}

// TestCBFilmStandardBand checks CB film is the STANDARD ±5% (NOT DT/LB's SL-005 ±3%): CB is
// rich-data via NGS, not a thin-data IDP. An elite composite lands inside [1.0,1.05] and ABOVE
// where DT's compressed ±3% curve would put it.
func TestCBFilmStandardBand(t *testing.T) {
	cb, dt := NewCB(), NewDT()
	in := engine.Layer4Input{
		Player:   engine.PlayerInput{Position: domain.PosCB, Age: 24, HasRAS: false},
		Scouting: engine.ScoutingInput{FilmComposite: 1.0, HasFilm: true},
	}
	out := cb.Apply(in)
	if out.FilmEffective < 1.0 || out.FilmEffective > 1.05+1e-9 {
		t.Fatalf("CB film effective %v escaped the standard ±5%% band", out.FilmEffective)
	}
	dtOut := dt.Apply(engine.Layer4Input{Player: engine.PlayerInput{Position: domain.PosDT, Age: 24, HasRAS: false}, Scouting: engine.ScoutingInput{FilmComposite: 1.0, HasFilm: true}})
	if !(out.FilmEffective > dtOut.FilmEffective) {
		t.Fatalf("CB film %v (±5%%) should exceed DT film %v (±3%% SL-005)", out.FilmEffective, dtOut.FilmEffective)
	}
}

// agingLateCB is a late-breakout CB past peak (age 32 → trajectory base 0.00, breakout age 21.5
// → base 0.45): both modulated sub-signals have headroom, so SL-019 visibly lifts breakout with
// the athletic profile (the §5 Surtain/Peterson mechanism, at reduced strength).
func agingLateCB(ras float64, hasRAS bool) engine.Layer4Input {
	return engine.Layer4Input{
		Player: engine.PlayerInput{Position: domain.PosCB, Age: 32, RAS: ras, HasRAS: hasRAS},
		Scouting: engine.ScoutingInput{
			BreakoutAge: 21.5, HasBreakoutAge: true, // base 0.45
			SchoolTierNorm: 1.00, HasSchoolTier: true, // Power Four (template)
			CollegeShare: 0.24, HasCollegeShare: true, // base 1.00
		},
	}
}

// TestCBSL019LiftsBreakout is the SL-019 mechanic at the unit level: for the SAME aging-late CB,
// a higher RAS lifts the breakout component, and an ABSENT RAS gives the least (the modulator
// contributes nothing — Data-Parity). 3I (not 3E/3M) is CB's suite case, so this is the unit
// proof that CB's SL-019 reuse is live; no new suite case is added for it.
func TestCBSL019LiftsBreakout(t *testing.T) {
	cb := NewCB()
	hi := cb.Apply(agingLateCB(9.99, true))
	lo := cb.Apply(agingLateCB(4.18, true))
	absent := cb.Apply(agingLateCB(0, false))
	if !(hi.BreakoutEffective > lo.BreakoutEffective) {
		t.Fatalf("SL-019: high-RAS breakout %v should exceed low-RAS %v", hi.BreakoutEffective, lo.BreakoutEffective)
	}
	if !(lo.BreakoutEffective > absent.BreakoutEffective) {
		t.Fatalf("SL-019: any-RAS lift %v should exceed absent-RAS (no modulation) %v", lo.BreakoutEffective, absent.BreakoutEffective)
	}
}

// TestCBMagnitudeThroughApply closes the standing GLM-review wiring lead: the worked-example
// tests call curve.SL019 directly and the ranking test only checks ORDERING — neither proves
// CB.Apply passes the right constants AT THE CALL SITE. This drives the real Apply and pins two
// magnitudes computed by hand from the §3/§4 spec to 1e-3 — catching a RAS steepness regressed
// to DE's 10.0 or an SL-019 strength copied from DE's 0.35.
func TestCBMagnitudeThroughApply(t *testing.T) {
	cb := NewCB()

	// RAS component at RAS 6.5 (rasNorm 0.65): Scurve(0.65,0.50,11.0,0.08) = 1 + 0.08×(2σ(1.65)−1)
	// = 1.05422. This pins steepness 11.0 (a regression to DE's 10.0 gives ≈1.05081), inflection
	// 0.50, cap ±8%, rookie weight 1.00, and the raw/10 divisor — all through Apply.
	ras := cb.Apply(engine.Layer4Input{Player: engine.PlayerInput{Position: domain.PosCB, Age: 24, RAS: 6.5, HasRAS: true}})
	if math.Abs(ras.RASEffective-1.05422) > 1e-3 {
		t.Fatalf("RASEffective(RAS 6.5) = %.5f, want ≈1.05422 (steepness 11.0, not 10.0; raw/10)", ras.RASEffective)
	}

	// Breakout via SL-019 through Apply: breakout age 22.5 (base 0.15) at RAS 9.99 modulates at
	// strength 0.30 to 0.40474; age 24 → trajectory base 1.00 (inert); school + college absent →
	// neutral 0.50. composite = 0.20×0.40474 + 0.25×0.50 + 0.40×0.50 + 0.15×1.00 = 0.55595;
	// breakout = Scurve(0.55595,0.50,10.0,0.05) ≈ 1.01364. A wrong SL-019 strength (0.35 →
	// modulated 0.447 → breakout ≈1.01556) fails here.
	bk := cb.Apply(engine.Layer4Input{
		Player:   engine.PlayerInput{Position: domain.PosCB, Age: 24, RAS: 9.99, HasRAS: true},
		Scouting: engine.ScoutingInput{BreakoutAge: 22.5, HasBreakoutAge: true},
	})
	if math.Abs(bk.BreakoutEffective-1.01364) > 1e-3 {
		t.Fatalf("BreakoutEffective (BO-age 22.5 @ RAS 9.99, else neutral) = %.5f, want ≈1.01364 (SL-019 strength 0.30, raw/10)", bk.BreakoutEffective)
	}
}

// TestCBComponentsClampedToCap is the planted-failure gate (M3): pathological inputs must never
// push an EFFECTIVE component (film/RAS/breakout) past its asymptote (±5%/±8%/±5%), and none may
// escape as non-finite — even through the SL-019 modulator path (bad RAS feeds rasNorm). FilmRaw
// is deliberately NOT asserted: it is the raw pre-effective composite echo (types.go — DEBUG/
// sandbox only, never UI), so it passes the input through unclamped by design (matching DT/DE/LB);
// the production boundary (composition.validateScouting) fail-louds on a non-finite FilmComposite
// before it can ever reach Apply, so FilmRaw is always finite in the live path.
func TestCBComponentsClampedToCap(t *testing.T) {
	cb := NewCB()
	for _, bad := range []float64{1e9, math.NaN(), math.Inf(1)} {
		out := cb.Apply(engine.Layer4Input{
			Player: engine.PlayerInput{Position: domain.PosCB, Age: 32, RAS: bad, HasRAS: true},
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
