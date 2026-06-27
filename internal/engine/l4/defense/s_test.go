package defense

import (
	"math"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/engine"
	"github.com/secureprospective/TheWarRoom/internal/engine/l4/curve"
)

// NewS must satisfy the engine.Layer4 contract (compile-time check, kept local so it is not
// a package-level global — gochecknoglobals).
func TestSImplementsLayer4(_ *testing.T) {
	var _ engine.Layer4 = NewS()
}

// TestSHasNGSAnchor proves the case-3I introspection hook: S reports an NGS coverage anchor
// (S_Rubric §2/§7 — the SECOND NGS-anchor position after CB). CB also reports it; the non-NGS
// defensive rubrics (DT/DE/LB) do NOT implement the method, so a type assertion against them
// fails — the harness reads that as "no anchor" (asserted in harness/eval3I against a non-NGS
// control). Both NGS positions and all three controls are checked here to lock the contract.
func TestSHasNGSAnchor(t *testing.T) {
	type ngsAnchor interface{ HasNGSAnchor() bool }
	if !NewS().HasNGSAnchor() {
		t.Fatal("S must report an NGS coverage anchor")
	}
	for name, r := range map[string]engine.Layer4{"S": NewS(), "CB": NewCB()} {
		a, ok := r.(ngsAnchor)
		if !ok || !a.HasNGSAnchor() {
			t.Fatalf("%s must expose an NGS anchor (NGS is CB/S-only)", name)
		}
	}
	for name, r := range map[string]engine.Layer4{"DT": NewDT(), "DE": NewDE(), "LB": NewLB()} {
		if _, ok := r.(ngsAnchor); ok {
			t.Fatalf("%s must NOT expose an NGS anchor (NGS is CB/S-only)", name)
		}
	}
}

// TestSSL019WorkedExamples is the M3 proof that S reuses curve.SL019 at the 0.30 strength
// (matching CB, NOT TE/DE's 0.35) and reproduces the S_Rubric §4/§5 worked examples EXACTLY.
// The expected values are the spec's literal numbers, hardcoded — so a wrong sSL019Strength
// (a copy-paste of 0.35) fails here. The Mathieu age-34 anchor (§5: base 0.00, RAS 4.5 → 0.135)
// is the canonical S-specific case; the rest are the shared-helper anchors at strength 0.30.
func TestSSL019WorkedExamples(t *testing.T) {
	cases := []struct {
		name          string
		base, rasNorm float64
		want          float64
	}{
		{"age-traj34-Mathieu RAS4.50", 0.00, 0.450, 0.135}, // S_Rubric §5 (Mathieu pull-attempt)
		{"breakout-age RAS9.99", 0.15, 0.999, 0.405},
		{"breakout-age RAS5.00", 0.15, 0.500, 0.278},
		{"age-traj28 RAS9.50", 0.50, 0.950, 0.6425},
		{"age-traj30 RAS7.00", 0.20, 0.700, 0.368},
	}
	for _, c := range cases {
		got := curve.SL019(c.base, c.rasNorm, sSL019Strength, true)
		if math.Abs(got-c.want) > 1e-3 {
			t.Fatalf("%s: SL019(%.2f,%.3f,0.30) = %.4f, want %.4f (S_Rubric §4/§5)", c.name, c.base, c.rasNorm, got, c.want)
		}
	}
	// A base already at the ceiling has no headroom: the modulator is inert (1−base = 0).
	if got := curve.SL019(1.00, 0.999, sSL019Strength, true); math.Abs(got-1.00) > 1e-9 {
		t.Fatalf("SL019 on a maxed base should be inert, got %.4f", got)
	}
}

// TestSCurvesReproduceSpecAnchors pins the rubric's ACTUAL breakpoint tables to the S_Rubric §4
// worked examples: the age-trajectory curve peaks at 28 (same as CB) and craters to 0.00 by 32;
// the breakout-age half-year curve reads 0.15 at 22.5 (same as CB); college share (INT + Tackle,
// S-DIVERGENT) anchors at 0.08/0.14/0.20 — LOWER than CB's 0.08/0.16/0.24. This catches a wrong
// breakpoint (e.g. a copy of CB's college-share thresholds) the magnitude proofs cannot.
func TestSCurvesReproduceSpecAnchors(t *testing.T) {
	s := NewS()
	for _, c := range []struct {
		name string
		got  float64
		want float64
	}{
		{"age-traj24", curve.Interp(s.ageTrajectory, 24), 1.00},
		{"age-traj26", curve.Interp(s.ageTrajectory, 26), 0.70},
		{"age-traj28peak", curve.Interp(s.ageTrajectory, 28), 0.50},
		{"age-traj30", curve.Interp(s.ageTrajectory, 30), 0.20},
		{"age-traj32", curve.Interp(s.ageTrajectory, 32), 0.00},
		{"breakout-age19.5", curve.Interp(s.breakoutAge, 19.5), 1.00},
		{"breakout-age20.5", curve.Interp(s.breakoutAge, 20.5), 0.75},
		{"breakout-age21.5", curve.Interp(s.breakoutAge, 21.5), 0.45},
		{"breakout-age22.5", curve.Interp(s.breakoutAge, 22.5), 0.15},
		// College-share curve anchors (S-specific INT + Tackle bundle): ≤0.08→0.15, 0.14→0.55, ≥0.20→1.00.
		{"college0.08", curve.Interp(s.collegeShare, 0.08), 0.15},
		{"college0.14", curve.Interp(s.collegeShare, 0.14), 0.55},
		{"college0.20", curve.Interp(s.collegeShare, 0.20), 1.00},
	} {
		if math.Abs(c.got-c.want) > 1e-9 {
			t.Fatalf("%s base = %.4f, want exactly %.2f (S_Rubric §4)", c.name, c.got, c.want)
		}
	}
}

// TestSRASActive checks the High-tier active RAS (no SL-020, AD-09 do-not-force) at the S
// steepness 10.0: elite RAS pushes toward the +8% cap at the full rookie weight (1.00), a low
// present RAS pulls below 1.000, and an ABSENT RAS is Data-Parity neutral (1.000).
func TestSRASActive(t *testing.T) {
	s := NewS()
	elite := s.Apply(engine.Layer4Input{Player: engine.PlayerInput{Position: domain.PosS, Age: 24, RAS: 10.0, HasRAS: true}})
	if !(elite.RASEffective > 1.07) || elite.RASEffective > 1.08+1e-9 {
		t.Fatalf("elite RAS effective = %v, want near +8%% cap (≈1.0789)", elite.RASEffective)
	}
	low := s.Apply(engine.Layer4Input{Player: engine.PlayerInput{Position: domain.PosS, Age: 24, RAS: 0, HasRAS: true}})
	if !(low.RASEffective < 0.93) || low.RASEffective < 0.92-1e-9 {
		t.Fatalf("low RAS effective = %v, want near −8%% cap (≈0.9211)", low.RASEffective)
	}
	absent := s.Apply(engine.Layer4Input{Player: engine.PlayerInput{Position: domain.PosS, Age: 24, HasRAS: false}})
	if absent.RASEffective != 1.0 {
		t.Fatalf("absent RAS must be Data-Parity neutral 1.000, got %v", absent.RASEffective)
	}
}

// TestSFilmStandardBand checks S film is the STANDARD ±5% (NOT DT/LB's SL-005 ±3%): S is
// rich-data via NGS, not a thin-data IDP. An elite composite lands inside [1.0,1.05] and ABOVE
// where DT's compressed ±3% curve would put it.
func TestSFilmStandardBand(t *testing.T) {
	s, dt := NewS(), NewDT()
	in := engine.Layer4Input{
		Player:   engine.PlayerInput{Position: domain.PosS, Age: 24, HasRAS: false},
		Scouting: engine.ScoutingInput{FilmComposite: 1.0, HasFilm: true},
	}
	out := s.Apply(in)
	if out.FilmEffective < 1.0 || out.FilmEffective > 1.05+1e-9 {
		t.Fatalf("S film effective %v escaped the standard ±5%% band", out.FilmEffective)
	}
	dtOut := dt.Apply(engine.Layer4Input{Player: engine.PlayerInput{Position: domain.PosDT, Age: 24, HasRAS: false}, Scouting: engine.ScoutingInput{FilmComposite: 1.0, HasFilm: true}})
	if !(out.FilmEffective > dtOut.FilmEffective) {
		t.Fatalf("S film %v (±5%%) should exceed DT film %v (±3%% SL-005)", out.FilmEffective, dtOut.FilmEffective)
	}
}

// agingLateS is a late-breakout safety past peak (age 32 → trajectory base 0.00, breakout age
// 21.5 → base 0.45): both modulated sub-signals have headroom, so SL-019 visibly lifts breakout
// with the athletic profile (the §5 Mathieu mechanism, at reduced strength). College share uses
// the S max threshold (0.20 → base 1.00).
func agingLateS(ras float64, hasRAS bool) engine.Layer4Input {
	return engine.Layer4Input{
		Player: engine.PlayerInput{Position: domain.PosS, Age: 32, RAS: ras, HasRAS: hasRAS},
		Scouting: engine.ScoutingInput{
			BreakoutAge: 21.5, HasBreakoutAge: true, // base 0.45
			SchoolTierNorm: 1.00, HasSchoolTier: true, // Power Four (template)
			CollegeShare: 0.20, HasCollegeShare: true, // base 1.00 (S max threshold)
		},
	}
}

// TestSSL019LiftsBreakout is the SL-019 mechanic at the unit level: for the SAME aging-late S,
// a higher RAS lifts the breakout component, and an ABSENT RAS gives the least (the modulator
// contributes nothing — Data-Parity). 3I (already green from CB; S co-asserts) is S's suite hook,
// so this is the unit proof that S's SL-019 reuse is live; no new suite case is added for it.
func TestSSL019LiftsBreakout(t *testing.T) {
	s := NewS()
	hi := s.Apply(agingLateS(9.99, true))
	lo := s.Apply(agingLateS(4.18, true))
	absent := s.Apply(agingLateS(0, false))
	if !(hi.BreakoutEffective > lo.BreakoutEffective) {
		t.Fatalf("SL-019: high-RAS breakout %v should exceed low-RAS %v", hi.BreakoutEffective, lo.BreakoutEffective)
	}
	if !(lo.BreakoutEffective > absent.BreakoutEffective) {
		t.Fatalf("SL-019: any-RAS lift %v should exceed absent-RAS (no modulation) %v", lo.BreakoutEffective, absent.BreakoutEffective)
	}
}

// TestSMagnitudeThroughApply closes the standing GLM-review wiring lead: the worked-example
// tests call curve.SL019 directly and the ranking test only checks ORDERING — neither proves
// S.Apply passes the right constants AT THE CALL SITE. This drives the real Apply and pins two
// magnitudes computed by hand from the §3/§4 spec to 1e-3 — catching a RAS steepness copied from
// CB's 11.0, a breakout steepness copied from CB's 10.0, or an SL-019 strength copied from DE's 0.35.
func TestSMagnitudeThroughApply(t *testing.T) {
	s := NewS()

	// RAS component at RAS 6.5 (rasNorm 0.65): Scurve(0.65,0.50,10.0,0.08) = 1 + 0.08×(2σ(1.5)−1)
	// = 1.05081. This pins steepness 10.0 (a copy of CB's 11.0 gives ≈1.05422), inflection 0.50,
	// cap ±8%, rookie weight 1.00, and the raw/10 divisor — all through Apply.
	ras := s.Apply(engine.Layer4Input{Player: engine.PlayerInput{Position: domain.PosS, Age: 24, RAS: 6.5, HasRAS: true}})
	if math.Abs(ras.RASEffective-1.05081) > 1e-3 {
		t.Fatalf("RASEffective(RAS 6.5) = %.5f, want ≈1.05081 (steepness 10.0, not CB's 11.0; raw/10)", ras.RASEffective)
	}

	// Breakout via SL-019 through Apply: breakout age 22.5 (base 0.15) at RAS 9.99 modulates at
	// strength 0.30 to 0.40474; age 24 → trajectory base 1.00 (inert); school + college absent →
	// neutral 0.50. composite = 0.20×0.40474 + 0.25×0.50 + 0.40×0.50 + 0.15×1.00 = 0.55595;
	// breakout = Scurve(0.55595,0.50,11.0,0.05) ≈ 1.01492. This pins breakout steepness 11.0 (a
	// copy of CB's 10.0 gives ≈1.01364) AND SL-019 strength 0.30 (a 0.35 → modulated 0.447 →
	// breakout ≈1.01683).
	bk := s.Apply(engine.Layer4Input{
		Player:   engine.PlayerInput{Position: domain.PosS, Age: 24, RAS: 9.99, HasRAS: true},
		Scouting: engine.ScoutingInput{BreakoutAge: 22.5, HasBreakoutAge: true},
	})
	if math.Abs(bk.BreakoutEffective-1.01492) > 1e-3 {
		t.Fatalf("BreakoutEffective (BO-age 22.5 @ RAS 9.99, else neutral) = %.5f, want ≈1.01492 (steepness 11.0, SL-019 strength 0.30, raw/10)", bk.BreakoutEffective)
	}

	// Pin the school (0.25) and college (0.40) sub-signal WEIGHTS individually — the other tests
	// leave these two resolving to equal values, so a constant swap would pass them invisibly.
	// Here SchoolTierNorm 1.00 and CollegeShare 0.08→0.15 resolve DIFFERENTLY: with no RAS, age 24
	// (traj 1.00, SL-019 inert), breakout-age absent (neutral 0.50), composite = 0.20×0.50 +
	// 0.25×1.00 + 0.40×0.15 + 0.15×1.00 = 0.56 → Scurve(0.56,0.50,11.0,0.05) ≈ 1.01593. A
	// school↔college weight swap gives composite 0.6875 → ≈1.03868, failing here.
	wt := s.Apply(engine.Layer4Input{
		Player: engine.PlayerInput{Position: domain.PosS, Age: 24, HasRAS: false},
		Scouting: engine.ScoutingInput{
			SchoolTierNorm: 1.00, HasSchoolTier: true,
			CollegeShare: 0.08, HasCollegeShare: true,
		},
	})
	if math.Abs(wt.BreakoutEffective-1.01593) > 1e-3 {
		t.Fatalf("BreakoutEffective (school 1.00 vs college 0.15) = %.5f, want ≈1.01593 (school weight 0.25, college weight 0.40)", wt.BreakoutEffective)
	}
}

// TestSComponentsClampedToCap is the planted-failure gate (M3): pathological inputs must never
// push an EFFECTIVE component (film/RAS/breakout) past its asymptote (±5%/±8%/±5%), and none may
// escape as non-finite — even through the SL-019 modulator path (bad RAS feeds rasNorm). FilmRaw
// is deliberately NOT asserted: it is the raw pre-effective composite echo (types.go — DEBUG/
// sandbox only, never UI), so it passes the input through unclamped by design (matching CB/DE/LB);
// the production boundary (composition.validateScouting) fail-louds on a non-finite FilmComposite
// before it can ever reach Apply, so FilmRaw is always finite in the live path.
func TestSComponentsClampedToCap(t *testing.T) {
	s := NewS()
	for _, bad := range []float64{1e9, math.NaN(), math.Inf(1)} {
		out := s.Apply(engine.Layer4Input{
			Player: engine.PlayerInput{Position: domain.PosS, Age: 32, RAS: bad, HasRAS: true},
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
