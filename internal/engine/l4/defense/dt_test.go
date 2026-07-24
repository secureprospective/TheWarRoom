package defense

import (
	"math"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/engine"
)

// NewDT must satisfy the engine.Layer4 contract (compile-time check, kept local so it is
// not a package-level global — gochecknoglobals).
func TestDTImplementsLayer4(_ *testing.T) {
	var _ engine.Layer4 = NewDT()
}

func approx(a, b float64) bool { return math.Abs(a-b) < 5e-4 }

// TestDTRASActive is the headline DT-vs-QB difference: at DT the RAS component is ACTIVE
// (no SL-020). Elite RAS pushes toward the +8% cap, a low (but present) RAS toward −8%, and
// an ABSENT RAS is Data-Parity neutral (1.000) — never a penalty.
func TestDTRASActive(t *testing.T) {
	dt := NewDT()
	elite := dt.Apply(engine.Layer4Input{Player: engine.PlayerInput{Position: domain.PosDT, Age: 22, RAS: 10.0, HasRAS: true}})
	if !(elite.RASEffective > 1.07) || elite.RASEffective > 1.08+1e-9 {
		t.Fatalf("elite RAS effective = %v, want near +8%% cap (≈1.079)", elite.RASEffective)
	}
	low := dt.Apply(engine.Layer4Input{Player: engine.PlayerInput{Position: domain.PosDT, Age: 22, RAS: 0, HasRAS: true}})
	if !(low.RASEffective < 0.93) || low.RASEffective < 0.92-1e-9 {
		t.Fatalf("low RAS effective = %v, want near −8%% cap (≈0.921)", low.RASEffective)
	}
	absent := dt.Apply(engine.Layer4Input{Player: engine.PlayerInput{Position: domain.PosDT, Age: 22, HasRAS: false}})
	if absent.RASEffective != 1.0 {
		t.Fatalf("absent RAS must be Data-Parity neutral 1.000, got %v", absent.RASEffective)
	}
}

// TestDTFilmCompression checks SL-005: the DT film component is bound to ±3%, tighter than
// the ±5% offense band. An elite film composite must land inside [0.97,1.03], and strictly
// below where the same composite would land under the offense ±5% steepness-12 curve.
func TestDTFilmCompression(t *testing.T) {
	dt := NewDT()
	out := dt.Apply(engine.Layer4Input{
		Player:   engine.PlayerInput{Position: domain.PosDT, Age: 22, HasRAS: false},
		Scouting: engine.ScoutingInput{FilmComposite: 1.0, HasFilm: true},
	})
	if out.FilmEffective < 0.97-1e-9 || out.FilmEffective > 1.03+1e-9 {
		t.Fatalf("DT film effective %v escaped the ±3%% band", out.FilmEffective)
	}
	if out.FilmRaw != 1.0 {
		t.Fatalf("FilmRaw should carry the pre-effective composite 1.0, got %v", out.FilmRaw)
	}
	// SL-005 binds tighter than the standard ±5% cap would.
	if !(out.FilmEffective < 1.05) {
		t.Fatalf("DT film %v must be tighter than the ±5%% cap", out.FilmEffective)
	}
}

// TestDTCushionGuardBreakoutTrajectory is the breakout-internal half of SL-021: past peak
// (age > 30) a DT with raw RAS ≥ 8.00 has its Age-Trajectory sub-signal cushioned, so its
// breakout component is HIGHER than an otherwise-identical sub-threshold DT. (The L3 half of
// the cushion is proven in the engine package + harness case 3F.)
func TestDTCushionGuardBreakoutTrajectory(t *testing.T) {
	dt := NewDT()
	base := engine.ScoutingInput{
		BreakoutAge: 22, HasBreakoutAge: true,
		SchoolTierNorm: 0.70, HasSchoolTier: true,
		CollegeShare: 0.15, HasCollegeShare: true,
	}
	hi := dt.Apply(engine.Layer4Input{Player: engine.PlayerInput{Position: domain.PosDT, Age: 32, RAS: 9.0, HasRAS: true}, Scouting: base})
	lo := dt.Apply(engine.Layer4Input{Player: engine.PlayerInput{Position: domain.PosDT, Age: 32, RAS: 7.0, HasRAS: true}, Scouting: base})
	if !(hi.BreakoutEffective > lo.BreakoutEffective) {
		t.Fatalf("cushioned breakout %v should exceed un-cushioned %v at age 32", hi.BreakoutEffective, lo.BreakoutEffective)
	}
	// Below peak the cushion is inert: same RAS split, age 26, must give equal breakout.
	hiYoung := dt.Apply(engine.Layer4Input{Player: engine.PlayerInput{Position: domain.PosDT, Age: 26, RAS: 9.0, HasRAS: true}, Scouting: base})
	loYoung := dt.Apply(engine.Layer4Input{Player: engine.PlayerInput{Position: domain.PosDT, Age: 26, RAS: 7.0, HasRAS: true}, Scouting: base})
	if !approx(hiYoung.BreakoutEffective, loYoung.BreakoutEffective) {
		t.Fatalf("below peak the cushion must be inert: %v vs %v", hiYoung.BreakoutEffective, loYoung.BreakoutEffective)
	}
}

// TestDTAbsentBreakoutIsNeutral mirrors the QB B1 regression: an ABSENT breakout sub-signal
// must contribute neutral, never the curve ceiling/floor a raw zero hits.
func TestDTAbsentBreakoutIsNeutral(t *testing.T) {
	dt := NewDT()
	absent := dt.Apply(engine.Layer4Input{
		Player:   engine.PlayerInput{Position: domain.PosDT, Age: 26, HasRAS: false},
		Scouting: engine.ScoutingInput{BreakoutAge: 0, SchoolTierNorm: 0, CollegeShare: 0}, // all absent
	})
	present := dt.Apply(engine.Layer4Input{
		Player: engine.PlayerInput{Position: domain.PosDT, Age: 26, HasRAS: false},
		Scouting: engine.ScoutingInput{
			BreakoutAge: 0, HasBreakoutAge: true, // raw zero PRESENT → ceiling (would be the bug if absent read this way)
			SchoolTierNorm: 1.0, HasSchoolTier: true,
			CollegeShare: 0.30, HasCollegeShare: true,
		},
	})
	if !(present.BreakoutEffective > absent.BreakoutEffective) {
		t.Fatalf("present-elite breakout %v should exceed absent-neutral %v", present.BreakoutEffective, absent.BreakoutEffective)
	}
}

// TestDTComponentsClampedToCap is the planted-failure gate (M3): pathological inputs must
// never push a component past its asymptote (film ±3%, RAS ±8%, breakout ±5%), and a NaN
// must not escape as non-finite.
func TestDTComponentsClampedToCap(t *testing.T) {
	dt := NewDT()
	for _, bad := range []float64{1e9, math.NaN(), math.Inf(1)} {
		out := dt.Apply(engine.Layer4Input{
			Player: engine.PlayerInput{Position: domain.PosDT, Age: 32, RAS: bad, HasRAS: true},
			Scouting: engine.ScoutingInput{
				FilmComposite: bad, HasFilm: true,
				BreakoutAge: bad, HasBreakoutAge: true,
				SchoolTierNorm: bad, HasSchoolTier: true,
				CollegeShare: bad, HasCollegeShare: true,
			},
		})
		assertBand(t, "film", out.FilmEffective, 0.03)
		assertBand(t, "RAS", out.RASEffective, 0.08)
		assertBand(t, "breakout", out.BreakoutEffective, 0.05)
		if want := out.FilmEffective * out.RASEffective * out.BreakoutEffective; !approx(out.Combined, want) {
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
		t.Fatalf("%s effect %v escaped its ±%.0f%% band", name, v, capBand*100)
	}
}

// TestDTSL021Alpha is the case-3G introspection hook: the dynamic SL-021 EMA alpha schedule
// (over the pass-rush grade; PFF is retired).
func TestDTSL021Alpha(t *testing.T) {
	dt := NewDT()
	for _, year := range []int{0, 1} {
		if got := dt.SL021Alpha(year); got != 0.50 {
			t.Fatalf("SL021Alpha(%d) = %v, want 0.50 (Year 1)", year, got)
		}
	}
	for _, year := range []int{2, 5} {
		if got := dt.SL021Alpha(year); got != 0.10 {
			t.Fatalf("SL021Alpha(%d) = %v, want 0.10 (Year 2+)", year, got)
		}
	}
}
