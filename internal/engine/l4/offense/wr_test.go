package offense

import (
	"math"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/engine"
)

// NewWR must satisfy the engine.Layer4 contract (compile-time check, kept local so it is not
// a package-level global — gochecknoglobals).
func TestWRImplementsLayer4(_ *testing.T) {
	var _ engine.Layer4 = NewWR()
}

// decliningEliteWR is WR_Rubric §5 (the Lockett structural finding): a declining elite-DRAFT
// profile — early breakout, Power Four, strong college usage — at age 33. The static signals
// offset the age-trajectory crater (0.00), so the breakout component stays ≥ 1.000.
func decliningEliteWR() engine.Layer4Input {
	return engine.Layer4Input{
		Player: engine.PlayerInput{Position: domain.PosWR, Age: 33, RAS: 8.5, HasRAS: true},
		Scouting: engine.ScoutingInput{
			BreakoutAge: 19, HasBreakoutAge: true, // ≤19 → 1.00
			SchoolTierNorm: 1.00, HasSchoolTier: true, // Power Four (template)
			CollegeShare: 0.32, HasCollegeShare: true, // ≈0.85
		},
	}
}

// decliningThinWR is the contrast: the SAME age-33 crater but weak static signals — late
// breakout, FCS, low usage — with no draft-era offset, so the breakout pulls below 1.000.
func decliningThinWR() engine.Layer4Input {
	return engine.Layer4Input{
		Player: engine.PlayerInput{Position: domain.PosWR, Age: 33, RAS: 8.5, HasRAS: true},
		Scouting: engine.ScoutingInput{
			BreakoutAge: 22, HasBreakoutAge: true, // ≥22 → 0.10
			SchoolTierNorm: 0.40, HasSchoolTier: true, // FCS (template)
			CollegeShare: 0.16, HasCollegeShare: true, // ≈0.14
		},
	}
}

// TestWRRASActive checks the High-tier active RAS (no SL-020, AD-09 do-not-force): elite RAS
// pushes toward the +8% cap at the full rookie weight (1.00), a low present RAS pulls below
// 1.000, and an ABSENT RAS is Data-Parity neutral (1.000). Wider band than RB's Medium tier.
func TestWRRASActive(t *testing.T) {
	wr := NewWR()
	base := engine.PlayerInput{Position: domain.PosWR, Age: 23}

	elite := wr.Apply(engine.Layer4Input{Player: withRAS(base, 10.0)})
	if !(elite.RASEffective > 1.0) || elite.RASEffective > 1.0+0.08+1e-9 {
		t.Fatalf("elite RAS effective = %v, want a lift bounded by the High-tier ±8%% band", elite.RASEffective)
	}
	low := wr.Apply(engine.Layer4Input{Player: withRAS(base, 0)})
	if !(low.RASEffective < 1.0) || low.RASEffective < 1.0-0.08-1e-9 {
		t.Fatalf("low RAS effective = %v, want a pull bounded by the High-tier ±8%% band", low.RASEffective)
	}
	absent := wr.Apply(engine.Layer4Input{Player: engine.PlayerInput{Position: domain.PosWR, Age: 23, HasRAS: false}})
	if absent.RASEffective != 1.0 {
		t.Fatalf("absent RAS must be Data-Parity neutral 1.000, got %v", absent.RASEffective)
	}
}

// TestWRLockettPattern is the case-3A mechanic at the unit level: a declining elite-draft WR's
// breakout stays ≥ 1.000 (static signals offset the age crater) while a declining thin WR's
// breakout pulls below — the deliberate contrast to RB's Herbert (WR_Rubric §5).
func TestWRLockettPattern(t *testing.T) {
	wr := NewWR()
	elite := wr.Apply(decliningEliteWR())
	thin := wr.Apply(decliningThinWR())
	if !(elite.BreakoutEffective >= 1.0) {
		t.Fatalf("declining elite WR breakout %v should stay ≥ 1.000 (static offset)", elite.BreakoutEffective)
	}
	if !(thin.BreakoutEffective < 1.0) {
		t.Fatalf("declining thin WR breakout %v should pull below 1.000 (no offset)", thin.BreakoutEffective)
	}
	if !(elite.BreakoutEffective > thin.BreakoutEffective) {
		t.Fatalf("elite WR breakout %v should exceed thin %v", elite.BreakoutEffective, thin.BreakoutEffective)
	}
	// Film is Data-Parity neutral this session (no offense source).
	if elite.FilmEffective != 1.0 || thin.FilmEffective != 1.0 {
		t.Fatalf("film must be Data-Parity neutral 1.000, got elite=%v thin=%v", elite.FilmEffective, thin.FilmEffective)
	}
}

// TestWRFilmActive checks the film S-curve fires when a source IS populated (the ±5% band):
// an elite composite pushes toward +5%, a neutral composite (0.50) → 1.000.
func TestWRFilmActive(t *testing.T) {
	wr := NewWR()
	base := engine.ScoutingInput{BreakoutAge: 20, SchoolTierNorm: 0, CollegeShare: 0} // not under test
	elite := wr.Apply(engine.Layer4Input{
		Player:   engine.PlayerInput{Position: domain.PosWR, Age: 23},
		Scouting: withFilm(base, 1.0),
	})
	if elite.FilmEffective < 1.0 || elite.FilmEffective > 1.05+1e-9 {
		t.Fatalf("elite film_effective = %v, want a lift bounded by +5%%", elite.FilmEffective)
	}
	neutral := wr.Apply(engine.Layer4Input{
		Player:   engine.PlayerInput{Position: domain.PosWR, Age: 23},
		Scouting: withFilm(base, 0.50),
	})
	if math.Abs(neutral.FilmEffective-1.0) > 5e-4 {
		t.Fatalf("neutral film_effective = %v, want ≈1.000", neutral.FilmEffective)
	}
}

// TestWRAbsentBreakoutIsNeutral is the B1 regression guard: an ABSENT breakout sub-signal must
// contribute neutral, never the curve ceiling/floor a raw zero hits.
func TestWRAbsentBreakoutIsNeutral(t *testing.T) {
	wr := NewWR()
	absent := wr.Apply(engine.Layer4Input{
		Player:   engine.PlayerInput{Position: domain.PosWR, Age: 25, HasRAS: false}, // age ≤25 → trajectory ceiling
		Scouting: engine.ScoutingInput{BreakoutAge: 0, SchoolTierNorm: 0, CollegeShare: 0},
	})
	present := wr.Apply(engine.Layer4Input{
		Player: engine.PlayerInput{Position: domain.PosWR, Age: 25, HasRAS: false},
		Scouting: engine.ScoutingInput{
			BreakoutAge: 0, HasBreakoutAge: true, // raw zero PRESENT → ceiling (the bug if absent read this way)
			SchoolTierNorm: 1.0, HasSchoolTier: true,
			CollegeShare: 0.50, HasCollegeShare: true,
		},
	})
	if !(present.BreakoutEffective > absent.BreakoutEffective) {
		t.Fatalf("present-elite breakout %v should exceed absent-neutral %v", present.BreakoutEffective, absent.BreakoutEffective)
	}
}

// TestWRComponentsClampedToCap is the planted-failure gate (M3): pathological inputs must never
// push a component past its asymptote (film ±5%, RAS ±8%, breakout ±5%), and a NaN must not
// escape as non-finite. This is the unit-level template eval3K asserts through the registry.
func TestWRComponentsClampedToCap(t *testing.T) {
	wr := NewWR()
	for _, bad := range []float64{1e9, math.NaN(), math.Inf(1)} {
		out := wr.Apply(engine.Layer4Input{
			Player: engine.PlayerInput{Position: domain.PosWR, Age: 24, RAS: bad, HasRAS: true},
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
