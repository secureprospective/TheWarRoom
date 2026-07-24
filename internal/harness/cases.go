package harness

import (
	"fmt"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/domain"
)

// requireRubrics gates a case on its B5b rubric(s) being registered. It returns ok=false
// with a PENDING state and a reason when any are missing, so a case body can early-return
// before trying to evaluate against a rubric that does not exist. This is the ONLY thing
// that makes a rubric-gated case pending — there is no "identity ⇒ skip" special case.
func requireRubrics(reg RubricRegistry, positions ...domain.Position) (CaseState, string, bool) {
	missing := make([]string, 0, len(positions))
	for _, p := range positions {
		if _, ok := reg[p]; !ok {
			missing = append(missing, string(p))
		}
	}
	if len(missing) > 0 {
		return StatePending, fmt.Sprintf("awaiting Layer-4 rubric for: %s", strings.Join(missing, ", ")), false
	}
	return "", "", true
}

// validationCases returns the 13 architectural cases in 3A..3M order. The two fully-wired
// exemplars (3C, 3L) run today; 3C auto-flips the moment a QB/K rubric registers. The rest
// are encoded with the rule and the block that will turn them green. 3M is the SL-019
// RAS-modulator gate at TE (the first SL-019 instance); 3E is its canonical DE instance — the
// SECOND SL-019 application, proving the shared curve.SL019 helper transfers (M17).
func validationCases() []case3 {
	return []case3{
		{id: "3A", name: "Lockett pattern — L4 stays non-negative for declining elite-draft vets", b5bBlock: "B5b-WR / B5b-QB", eval: eval3A},
		{id: "3B", name: "Herbert pattern — L4 pulls below 1.00 for weak-profile vets", b5bBlock: "B5b-RB", eval: eval3B},
		{id: "3C", name: "SL-020 — QB & K Layer-4 RAS forced to exactly 1.000", b5bBlock: "B5b-QB / B5b-K", eval: eval3C},
		{id: "3D", name: "SL-005 — film compression ±3% at LB/DT vs ±5% elsewhere", b5bBlock: "B5b-LB / B5b-DT", eval: eval3D},
		{id: "3E", name: "SL-019 — breakout modulator lifts with RAS (DE)", b5bBlock: "B5b-DE", eval: eval3E},
		{id: "3F", name: "SL-021 — DT cushion guard (RAS ≥ 8.00 → 10% decel)", b5bBlock: "B5b-DT", eval: eval3F},
		{id: "3G", name: "SL-021 — DT dynamic pass-rush blend α (0.50 Y1 → 0.10 Y2+; DE control 0.15)", b5bBlock: "B5b-DT / B5b-DE", eval: eval3G},
		{id: "3H", name: "Confidence floor — all-Unknown component → effective 1.000", b5bBlock: "B5b-WR", eval: eval3H},
		{id: "3I", name: "NGS anchor present only at CB & S", b5bBlock: "B5b-CB / B5b-S", eval: eval3I},
		{id: "3J", name: "EDGE classification routing (pass-rush share → DE vs LB)", b5bBlock: "B5b-DE / B5b-LB + dispatch", eval: eval3J},
		{id: "3K", name: "S-curve boundary safety — output clamped to [1-cap, 1+cap]", b5bBlock: "B5b-WR", eval: eval3K},
		{id: "3L", name: "MFL player-ID string enforcement (leading zeros preserved)", b5bBlock: "(none — testable now)", eval: eval3L},
		{id: "3M", name: "SL-019 — RAS-modulator lifts breakout with athletic profile (TE)", b5bBlock: "B5b-TE", eval: eval3M},
	}
}
