package kicker

import (
	"math"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/engine"
)

func approx(a, b float64) bool { return math.Abs(a-b) < 5e-4 }

// TestKImplementsLayer4 is the compile-time contract check (kept local so it is not a
// package-level global — gochecknoglobals).
func TestKImplementsLayer4(_ *testing.T) {
	var _ engine.Layer4 = NewK()
}

// TestKRASForcedToOne is the SL-020 (partial) gate: K's RAS contributes NOTHING to Layer 4 —
// RASEffective is forced to exactly 1.000 regardless of the athlete's RAS AND regardless of an
// active film path. This is the property eval3C asserts and what keeps case 3C green.
func TestKRASForcedToOne(t *testing.T) {
	k := NewK()
	for _, ras := range []float64{0.0, 0.10, 5.0, 9.99, 10.0} {
		out := k.Apply(engine.Layer4Input{
			Player: engine.PlayerInput{Position: domain.PosK, Age: 27, RAS: ras, HasRAS: true},
			// Film populated, to prove the active-film path does not leak into RAS.
			Scouting: engine.ScoutingInput{MaddenFilm: 0.90, HasMaddenFilm: true, NFLProduction: 0.80, HasNFLProduction: true},
		})
		if out.RASEffective != 1.0 {
			t.Fatalf("RAS %v → RASEffective %v, want exactly 1.0", ras, out.RASEffective)
		}
	}
}

// TestKBreakoutForcedToOne proves there is no breakout framework at K: BreakoutEffective is
// forced to exactly 1.000 even with an active film composite.
func TestKBreakoutForcedToOne(t *testing.T) {
	k := NewK()
	out := k.Apply(engine.Layer4Input{
		Player:   engine.PlayerInput{Position: domain.PosK, Age: 27},
		Scouting: engine.ScoutingInput{MaddenFilm: 1.0, HasMaddenFilm: true, NFLProduction: 1.0, HasNFLProduction: true},
	})
	if out.BreakoutEffective != 1.0 {
		t.Fatalf("BreakoutEffective %v, want exactly 1.0", out.BreakoutEffective)
	}
}

// TestKFilmActiveMaddenMajority is the magnitude gate (DECISION-011): K's film is the S-curve
// over a Madden 0.60 / NFLProduction 0.40 composite at the pinned ±3% cap / steepness 10.0 /
// inflection 0.50. The worked example uses Madden 1.0 / NFLProduction 0.0 so the composite is
// EXACTLY the Madden weight (0.60) — a swapped 0.40/0.60 split would give composite 0.40 and a
// visibly different film (≈0.98614, the symmetric case below), so this pins the split, the cap,
// AND the steepness through Apply, not just inside the helper. Combined == film (RAS+breakout 1.000).
func TestKFilmActiveMaddenMajority(t *testing.T) {
	k := NewK()

	// Madden 1.0 / NFLProduction 0.0 → composite 0.60 → film ≈ 1.013864.
	maddenOnly := k.Apply(engine.Layer4Input{
		Player:   engine.PlayerInput{Position: domain.PosK, Age: 27},
		Scouting: engine.ScoutingInput{MaddenFilm: 1.0, HasMaddenFilm: true, NFLProduction: 0.0, HasNFLProduction: true},
	})
	if !approx(maddenOnly.FilmEffective, 1.013864) {
		t.Fatalf("Madden-1.0/NFLProd-0.0 film_effective = %v, want ≈1.013864 (composite 0.60, cap ±3%%, steep 10)", maddenOnly.FilmEffective)
	}
	if maddenOnly.Combined != maddenOnly.FilmEffective {
		t.Fatalf("Combined %v must equal film %v (RAS+breakout forced 1.000)", maddenOnly.Combined, maddenOnly.FilmEffective)
	}

	// Madden 0.0 / NFLProduction 1.0 → composite 0.40 → film ≈ 0.986136. The two cases sit at
	// equal-magnitude but OPPOSITE-direction offsets around 1.000 (+0.013864 vs −0.013864): film
	// lifts when the Madden-weighted (0.60) input is high and drops when the NFLProduction-weighted
	// (0.40) input is high. That direction is the direct proof Madden carries the MAJORITY weight;
	// an equal or inverted split would not produce it.
	nflOnly := k.Apply(engine.Layer4Input{
		Player:   engine.PlayerInput{Position: domain.PosK, Age: 27},
		Scouting: engine.ScoutingInput{MaddenFilm: 0.0, HasMaddenFilm: true, NFLProduction: 1.0, HasNFLProduction: true},
	})
	if !approx(nflOnly.FilmEffective, 0.986136) {
		t.Fatalf("Madden-0.0/NFLProd-1.0 film_effective = %v, want ≈0.986136 (composite 0.40)", nflOnly.FilmEffective)
	}
	if !(maddenOnly.FilmEffective > nflOnly.FilmEffective) {
		t.Fatalf("Madden-weighted film %v should exceed NFLProd-weighted film %v (0.60 > 0.40)", maddenOnly.FilmEffective, nflOnly.FilmEffective)
	}
}

// TestKFilmDataParityNeutral is the Data-Parity gate: with BOTH film sub-signals absent the film
// component is exactly 1.000 (Apply short-circuits to the neutral 1.000 — equivalently the
// composite is the neutral 0.50, whose S-curve is exactly 1.000), so a missing film source is
// never a penalty and Combined is exactly 1.000.
func TestKFilmDataParityNeutral(t *testing.T) {
	k := NewK()
	out := k.Apply(engine.Layer4Input{
		Player:   engine.PlayerInput{Position: domain.PosK, Age: 27},
		Scouting: engine.ScoutingInput{}, // no Madden, no NFLProduction
	})
	if out.FilmEffective != 1.0 {
		t.Fatalf("absent film must be exactly neutral 1.000, got %v", out.FilmEffective)
	}
	if out.Combined != 1.0 {
		t.Fatalf("Combined with no film must be exactly 1.000, got %v", out.Combined)
	}
	// A real-zero sub-signal must NOT be confused with absent: Madden present at 0.0 pulls the
	// composite below neutral (0.60·0.0 + 0.40·0.50 = 0.20) → film below 1.000.
	zero := k.Apply(engine.Layer4Input{
		Player:   engine.PlayerInput{Position: domain.PosK, Age: 27},
		Scouting: engine.ScoutingInput{MaddenFilm: 0.0, HasMaddenFilm: true},
	})
	if !(zero.FilmEffective < 1.0) {
		t.Fatalf("a present zero-Madden rating should pull film below 1.000 (not be treated as absent), got %v", zero.FilmEffective)
	}
}

// TestKComponentsClampedToCap is the planted-failure gate (M3): even a pathological, out-of-range
// or non-finite film input must never push the film component past its ±3% asymptote, and must
// leave RAS + breakout at exactly 1.000. A NaN must not escape (it defeats min/max comparisons —
// the curve.Scurve guard catches it).
func TestKComponentsClampedToCap(t *testing.T) {
	k := NewK()
	for _, v := range []float64{1e9, -1e9, math.NaN(), math.Inf(1)} {
		out := k.Apply(engine.Layer4Input{
			Player:   engine.PlayerInput{Position: domain.PosK, Age: 27},
			Scouting: engine.ScoutingInput{MaddenFilm: v, HasMaddenFilm: true, NFLProduction: v, HasNFLProduction: true},
		})
		if math.IsNaN(out.FilmEffective) || math.IsInf(out.FilmEffective, 0) {
			t.Fatalf("film %v is non-finite — a non-finite input escaped the clamp", out.FilmEffective)
		}
		if out.FilmEffective < 0.97-1e-9 || out.FilmEffective > 1.03+1e-9 {
			t.Fatalf("film %v escaped the ±3%% cap band [0.97,1.03]", out.FilmEffective)
		}
		if out.RASEffective != 1.0 || out.BreakoutEffective != 1.0 {
			t.Fatalf("RAS/breakout must stay 1.000; got RAS=%v breakout=%v", out.RASEffective, out.BreakoutEffective)
		}
		if want := out.FilmEffective * out.RASEffective * out.BreakoutEffective; out.Combined != want {
			t.Fatalf("Combined %v != film×RAS×breakout %v", out.Combined, want)
		}
	}
}
