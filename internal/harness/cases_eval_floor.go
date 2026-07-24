package harness

import (
	"fmt"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/engine"
)

// eval3H is the confidence-floor close gate (Testing_App_Specification Test 3H): when a
// component carries no live signal, its effective output is EXACTLY 1.000 and any raw value
// is ignored — "the deviation collapses to zero, no distortion regardless of what the raw
// component calculated." The current engine implements this via the Data-Parity design rather
// than an explicit confidence multiplier, so the case asserts the property where that design
// realizes it:
//
//   - Film + RAS (universal): the presence FLAG is the confidence gate. For EVERY registered
//     rubric, a stray non-neutral raw (FilmComposite 1.04, RAS 9.99) flagged ABSENT must yield
//     FilmEffective / RASEffective == exactly 1.000 — the raw never leaks (wr.go film/RAS guards;
//     mirrors the spec's "film_raw: any value → film_effective 1.000"). QB/K also force RAS via
//     SL-020, so their RAS floor is exercised through the Data-Parity path, not the forced one.
//   - Breakout + full Layer-4 (at WR, the canonical standard rubric): breakout has no whole-
//     component flag — age-trajectory is always live — so "all breakout fields Unknown" is reached
//     when the three flagged sub-signals are absent (→ neutral 0.50) AND age sits at the WR peak
//     (29, WR_Rubric §4 ageTrajectory 0.50), driving the composite to the 0.50 inflection. Then
//     breakout == 1.000 exactly; with film + RAS also absent, all three components floor and
//     Combined == 1.000 (the spec's Full Layer-4 Floor Test). WR gates the case.
func eval3H(reg RubricRegistry) (CaseState, string) {
	if st, why, ok := requireRubrics(reg, domain.PosWR); !ok {
		return st, why
	}
	// Film + RAS floor across every registered rubric, with stray raws flagged absent.
	for pos, rb := range reg {
		out := rb.Apply(engine.Layer4Input{
			Player:   engine.PlayerInput{Position: pos, RAS: 9.99, HasRAS: false},
			Scouting: engine.ScoutingInput{FilmComposite: 1.04, HasFilm: false},
		})
		if out.FilmEffective != 1.0 {
			return StateFail, fmt.Sprintf("%s: absent film with stray raw 1.04 → FilmEffective %.6f, want exactly 1.0000", pos, out.FilmEffective)
		}
		if out.RASEffective != 1.0 {
			return StateFail, fmt.Sprintf("%s: absent RAS with stray raw 9.99 → RASEffective %.6f, want exactly 1.0000", pos, out.RASEffective)
		}
	}
	// Breakout + full Layer-4 floor at WR: all breakout sub-signals unknown (three flagged
	// absent, age at the WR peak 29 → age-trajectory neutral) with film + RAS also absent.
	const wrPeakAge = 29.0
	floor := reg[domain.PosWR].Apply(engine.Layer4Input{Player: engine.PlayerInput{Position: domain.PosWR, Age: wrPeakAge}})
	if floor.BreakoutEffective != 1.0 {
		return StateFail, fmt.Sprintf("WR all-unknown breakout (age %.0f) → BreakoutEffective %.6f, want exactly 1.0000", wrPeakAge, floor.BreakoutEffective)
	}
	if floor.Combined != 1.0 {
		return StateFail, fmt.Sprintf("WR all-component-absent → Combined %.6f, want exactly 1.0000", floor.Combined)
	}
	return StatePass, fmt.Sprintf("absent film(raw 1.04)+RAS(raw 9.99) floor to 1.0000 across %d rubrics; WR all-unknown breakout & Combined = 1.0000 (age %.0f neutral)", len(reg), wrPeakAge)
}
