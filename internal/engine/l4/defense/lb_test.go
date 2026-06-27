package defense

import (
	"math"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/engine"
	"github.com/secureprospective/TheWarRoom/internal/engine/l4/curve"
)

// NewLB must satisfy the engine.Layer4 contract (compile-time check, kept local so it is not
// a package-level global — gochecknoglobals).
func TestLBImplementsLayer4(_ *testing.T) {
	var _ engine.Layer4 = NewLB()
}

// TestLBCurvesReproduceSpecAnchors pins the rubric's ACTUAL breakpoint tables to the LB_Rubric
// §4 anchors: age trajectory peak 29 → 0.50 with the gradual decline to 0.00 at age 33, breakout
// age 20 → 1.00 / 22 → 0.45, and the College-Production-Share curve (≤0.10→0.15, 0.18→0.55,
// ≥0.25→1.00 — the LB-divergent table, elevated weight, so a typo is caught here). Warner's 22%
// share interpolates to 0.807 (§5).
func TestLBCurvesReproduceSpecAnchors(t *testing.T) {
	lb := NewLB()
	for _, c := range []struct {
		name string
		got  float64
		want float64
	}{
		{"age-traj25", curve.Interp(lb.ageTrajectory, 25), 1.00},
		{"age-traj29peak", curve.Interp(lb.ageTrajectory, 29), 0.50},
		{"age-traj33", curve.Interp(lb.ageTrajectory, 33), 0.00},
		{"age-traj36clamp", curve.Interp(lb.ageTrajectory, 36), 0.00}, // past last breakpoint, held flat
		{"breakout-age20", curve.Interp(lb.breakoutAge, 20), 1.00},
		{"breakout-age22", curve.Interp(lb.breakoutAge, 22), 0.45},
		{"breakout-age23", curve.Interp(lb.breakoutAge, 23), 0.15},
		{"college0.10", curve.Interp(lb.collegeShare, 0.10), 0.15},
		{"college0.18", curve.Interp(lb.collegeShare, 0.18), 0.55},
		{"college0.25", curve.Interp(lb.collegeShare, 0.25), 1.00},
		{"college0.22-warner", curve.Interp(lb.collegeShare, 0.22), 0.807143},
	} {
		if math.Abs(c.got-c.want) > 1e-5 {
			t.Fatalf("%s base = %.6f, want %.6f (LB_Rubric §4)", c.name, c.got, c.want)
		}
	}
}

// TestLBRASMediumTier checks the ACTIVE Medium-tier RAS (no SL-020): cap ±4% at the rookie
// weight 0.60, so even elite RAS lands well inside ±4% headroom; a low present RAS pulls below
// 1.000; an ABSENT RAS is Data-Parity neutral (1.000). Medium ±4% is HALF DE's High ±8% band.
func TestLBRASMediumTier(t *testing.T) {
	lb := NewLB()
	elite := lb.Apply(engine.Layer4Input{Player: engine.PlayerInput{Position: domain.PosLB, Age: 24, RAS: 10.0, HasRAS: true}})
	// At rookie weight 0.60 the +4% cap scales to ≈+2.4%: effective ≈ 1 + 0.60×0.04 = 1.024.
	if elite.RASEffective <= 1.0 || elite.RASEffective > 1.024+1e-9 {
		t.Fatalf("elite RAS effective = %v, want (1.0, 1.024] (Medium ±4%% at weight 0.60)", elite.RASEffective)
	}
	low := lb.Apply(engine.Layer4Input{Player: engine.PlayerInput{Position: domain.PosLB, Age: 24, RAS: 0, HasRAS: true}})
	if low.RASEffective >= 1.0 || low.RASEffective < 0.976-1e-9 {
		t.Fatalf("low RAS effective = %v, want [0.976, 1.0) (Medium ±4%% at weight 0.60)", low.RASEffective)
	}
	absent := lb.Apply(engine.Layer4Input{Player: engine.PlayerInput{Position: domain.PosLB, Age: 24, HasRAS: false}})
	if absent.RASEffective != 1.0 {
		t.Fatalf("absent RAS must be Data-Parity neutral 1.000, got %v", absent.RASEffective)
	}
}

// TestLBFilmSL005Compressed checks LB film is SL-005-compressed ±3% — IDENTICAL to DT, and
// STRICTLY BELOW DE's standard ±5% on the same elite composite (the case-3D mechanic at the unit
// level; eval3D asserts it cross-position through the registered rubrics).
func TestLBFilmSL005Compressed(t *testing.T) {
	lb, dt, de := NewLB(), NewDT(), NewDE()
	in := func(pos domain.Position) engine.Layer4Input {
		return engine.Layer4Input{
			Player:   engine.PlayerInput{Position: pos, Age: 24, HasRAS: false},
			Scouting: engine.ScoutingInput{FilmComposite: 1.0, HasFilm: true},
		}
	}
	lbF := lb.Apply(in(domain.PosLB)).FilmEffective
	if lbF < 1.0 || lbF > 1.03+1e-9 {
		t.Fatalf("LB film effective %v escaped the SL-005 ±3%% band", lbF)
	}
	// LB's ±3% must MATCH DT's SL-005 ±3% (same compression idiom) and stay BELOW DE's ±5%.
	dtF := dt.Apply(in(domain.PosDT)).FilmEffective
	if !approx(lbF, dtF) {
		t.Fatalf("LB film %v should equal DT film %v (same SL-005 ±3%% block)", lbF, dtF)
	}
	deF := de.Apply(in(domain.PosDE)).FilmEffective
	if !(lbF < deF) {
		t.Fatalf("LB film %v (±3%% SL-005) should be below DE film %v (±5%% standard)", lbF, deF)
	}
}

// TestLBNoSL019 is the defining negative of the FIRST non-SL-019 IDP rubric: RAS must NOT
// modulate the breakout component at LB (unlike DE/TE). Two otherwise-identical aging LBs with a
// breakout/age-trajectory headroom — one elite-RAS, one absent-RAS — must produce the SAME
// breakout. A regression that wired curve.SL019 into LB would make these diverge.
func TestLBNoSL019(t *testing.T) {
	lb := NewLB()
	withRAS := func(ras float64, has bool) engine.Layer4Input {
		return engine.Layer4Input{
			Player: engine.PlayerInput{Position: domain.PosLB, Age: 31, RAS: ras, HasRAS: has}, // past peak → traj headroom
			Scouting: engine.ScoutingInput{
				BreakoutAge: 22, HasBreakoutAge: true, // base 0.45 — headroom for any modulation
				SchoolTierNorm: 0.70, HasSchoolTier: true,
				CollegeShare: 0.18, HasCollegeShare: true, // base 0.55
			},
		}
	}
	hi := lb.Apply(withRAS(9.9, true))
	absent := lb.Apply(withRAS(0, false))
	if !approx(hi.BreakoutEffective, absent.BreakoutEffective) {
		t.Fatalf("LB breakout must NOT move with RAS (no SL-019): hi-RAS %v vs absent %v", hi.BreakoutEffective, absent.BreakoutEffective)
	}
	// RAS still moves the RAS component itself (it is active), just not breakout.
	if !(hi.RASEffective > absent.RASEffective) {
		t.Fatalf("active RAS component should still differ (%v vs %v)", hi.RASEffective, absent.RASEffective)
	}
}

// TestLBMagnitudeThroughApply pins the §3/§4 constants AT THE CALL SITE (the standing GLM lead —
// curve-level tests and eval3D ordering don't prove DE.Apply/LB.Apply pass the right numbers).
// Two magnitudes hand-computed from the spec: RAS@6.5 distinguishes Medium steepness 11.0 from
// RB's 8.0 and pins the 0.60 rookie weight + raw/10 divisor; the breakout reproduces Warner's
// §5 composite (0.788 → 1.046), pinning the four weights, the college curve, and steepness 11.0.
func TestLBMagnitudeThroughApply(t *testing.T) {
	lb := NewLB()

	// RAS 6.5 (rasNorm 0.65): Scurve(0.65,0.50,11.0,0.04)=1.027116; effective=1+0.60×0.027116=1.01627.
	// Steepness regressed to RB's 8.0 would give ≈1.01289 — outside 1e-3, so this pins 11.0.
	ras := lb.Apply(engine.Layer4Input{Player: engine.PlayerInput{Position: domain.PosLB, Age: 26, RAS: 6.5, HasRAS: true}})
	if math.Abs(ras.RASEffective-1.01627) > 1e-3 {
		t.Fatalf("RASEffective(RAS 6.5) = %.5f, want ≈1.01627 (Medium steepness 11.0 not 8.0; weight 0.60; raw/10)", ras.RASEffective)
	}

	// Warner breakout (§5): breakout age 20 → 1.00, school G5 0.70, college 22% → 0.807143, age 29
	// (peak) → 0.50. composite = 0.25×1.00 + 0.20×0.70 + 0.40×0.807143 + 0.15×0.50 = 0.787857;
	// breakout = Scurve(0.787857,0.50,11.0,0.05) ≈ 1.04596. A wrong weight or curve fails here.
	bk := lb.Apply(engine.Layer4Input{
		Player: engine.PlayerInput{Position: domain.PosLB, Age: 29, HasRAS: false},
		Scouting: engine.ScoutingInput{
			BreakoutAge: 20, HasBreakoutAge: true,
			SchoolTierNorm: 0.70, HasSchoolTier: true,
			CollegeShare: 0.22, HasCollegeShare: true,
		},
	})
	if math.Abs(bk.BreakoutEffective-1.04596) > 1e-3 {
		t.Fatalf("BreakoutEffective (Warner §5 composite) = %.5f, want ≈1.04596 (weights 0.25/0.20/0.40/0.15, steepness 11.0)", bk.BreakoutEffective)
	}

	// Warner film (§5): composite 0.914 → Scurve(0.914,0.50,10.0,0.03) ≈ 1.029. Reproduces the
	// spec's named film anchor at the call site and pins the SL-005 cap ±3% (a standard ±5% cap
	// gives ≈1.048). Steepness 10-vs-12 is indistinguishable this near the asymptote, so the
	// steepness is locked instead by TestLBFilmSL005Compressed (LB-equals-DT's ±3%/10.0 block).
	fm := lb.Apply(engine.Layer4Input{
		Player:   engine.PlayerInput{Position: domain.PosLB, Age: 24, HasRAS: false},
		Scouting: engine.ScoutingInput{FilmComposite: 0.914, HasFilm: true},
	})
	if math.Abs(fm.FilmEffective-1.029) > 1e-3 {
		t.Fatalf("FilmEffective(composite 0.914) = %.5f, want ≈1.029 (SL-005 cap ±3%%, LB_Rubric §5)", fm.FilmEffective)
	}
}

// TestLBComponentsClampedToCap is the planted-failure gate (M3): pathological inputs must never
// push a component past its asymptote (film ±3%, RAS ±4% at weight 0.60, breakout ±5%), and no
// *Effective/Combined field escapes as non-finite. (FilmRaw is a debug passthrough of the raw
// composite and is intentionally not asserted here; Validate rejects a non-finite film upstream.)
func TestLBComponentsClampedToCap(t *testing.T) {
	lb := NewLB()
	for _, bad := range []float64{1e9, math.NaN(), math.Inf(1)} {
		out := lb.Apply(engine.Layer4Input{
			Player: engine.PlayerInput{Position: domain.PosLB, Age: 31, RAS: bad, HasRAS: true},
			Scouting: engine.ScoutingInput{
				FilmComposite: bad, HasFilm: true,
				BreakoutAge: bad, HasBreakoutAge: true,
				SchoolTierNorm: bad, HasSchoolTier: true,
				CollegeShare: bad, HasCollegeShare: true,
			},
		})
		assertBand(t, "film", out.FilmEffective, 0.03)
		assertBand(t, "RAS", out.RASEffective, 0.04*lbRASRookieWeight+1e-9) // Medium band scaled by rookie weight
		assertBand(t, "breakout", out.BreakoutEffective, 0.05)
		if want := out.FilmEffective * out.RASEffective * out.BreakoutEffective; !approx(out.Combined, want) {
			t.Fatalf("Combined %v != film×RAS×breakout %v", out.Combined, want)
		}
	}
}
