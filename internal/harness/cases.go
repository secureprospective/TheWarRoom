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

// pendingSubSignals marks a case that cannot evaluate yet because it asserts on rubric
// INTERNALS (film_raw, modulated breakout, PFF alpha, NGS weight, route) that the current
// Layer4Output does not expose and Layer4Input does not yet carry. B5b adds those fields
// AND the assertion body; until then the encoded case documents the rule it will gate.
func pendingSubSignals(detail string) (CaseState, string) {
	return StatePending, "needs B5b rubric inputs/hooks: " + detail
}

// gatedPending builds a case that is PENDING on its rubric(s) and, once those register,
// still PENDING until B5b adds the sub-signal inputs/hooks it asserts on. The detail names
// what B5b must expose. This collapses the many same-shaped cases into one builder (M17).
func gatedPending(id, name, b5b, detail string, pos ...domain.Position) case3 {
	return case3{id: id, name: name, b5bBlock: b5b, eval: func(reg RubricRegistry) (CaseState, string) {
		if st, why, ok := requireRubrics(reg, pos...); !ok {
			return st, why
		}
		return pendingSubSignals(detail)
	}}
}

// subSignalPending builds a case that has no single rubric gate (it tests dispatch or a
// cross-component floor) and is simply PENDING on B5b inputs/hooks.
func subSignalPending(id, name, b5b, detail string) case3 {
	return case3{id: id, name: name, b5bBlock: b5b, eval: func(_ RubricRegistry) (CaseState, string) {
		return pendingSubSignals(detail)
	}}
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
		gatedPending("3G", "SL-021 — DT dynamic PFF alpha (0.50 Y1 → 0.10 Y2+)",
			"B5b-DT", "DT.PFFAlpha introspection hook + DE rubric both built; 3G assertion-wiring deferred",
			domain.PosDT, domain.PosDE),
		subSignalPending("3H", "Confidence floor — all-Unknown component → effective 1.000",
			"B5b (first rubric)", "component confidence inputs not on Layer4Input"),
		gatedPending("3I", "NGS anchor present only at CB & S",
			"B5b-CB / B5b-S", "rubric sub-signal weights not introspectable yet",
			domain.PosCB, domain.PosS),
		{id: "3J", name: "EDGE classification routing (pass-rush share → DE vs LB)", b5bBlock: "B5b-DE / B5b-LB + dispatch", eval: eval3J},
		{id: "3K", name: "S-curve boundary safety — output clamped to [1-cap, 1+cap]", b5bBlock: "B5b-WR", eval: eval3K},
		{id: "3L", name: "MFL player-ID string enforcement (leading zeros preserved)", b5bBlock: "(none — testable now)", eval: eval3L},
		{id: "3M", name: "SL-019 — RAS-modulator lifts breakout with athletic profile (TE)", b5bBlock: "B5b-TE", eval: eval3M},
	}
}
