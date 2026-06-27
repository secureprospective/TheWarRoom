package harness

import (
	"fmt"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/engine"
	"github.com/secureprospective/TheWarRoom/internal/schema"
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

// validationCases returns the 12 architectural cases in 3A..3L order. The two fully-wired
// exemplars (3C, 3L) run today; 3C auto-flips the moment a QB/K rubric registers. The rest
// are encoded with the rule and the block that will turn them green.
func validationCases() []case3 {
	return []case3{
		gatedPending("3A", "Lockett pattern — L4 near-neutral for declining elite vets",
			"B5b-WR / B5b-QB", "film_composite + static breakout sub-signals on Layer4Input",
			domain.PosWR, domain.PosQB),
		gatedPending("3B", "Herbert pattern — L4 pulls below 1.00 for weak-profile vets",
			"B5b-RB", "film_composite + breakout-age sub-signals on Layer4Input",
			domain.PosRB),
		{id: "3C", name: "SL-020 — QB & K Layer-4 RAS forced to exactly 1.000", b5bBlock: "B5b-QB / B5b-K", eval: eval3C},
		gatedPending("3D", "SL-005 — film compression ±3% at LB/DT vs ±5% elsewhere",
			"B5b-LB / B5b-DT", "film_raw (pre-effective) not on Layer4Output",
			domain.PosLB, domain.PosDT, domain.PosWR),
		gatedPending("3E", "SL-019 — breakout modulator lifts with RAS (DE)",
			"B5b-DE", "modulated breakout intermediate not on Layer4Output",
			domain.PosDE),
		gatedPending("3F", "SL-021 — DT cushion guard (RAS ≥ 8.00 → 10% decel)",
			"B5b-DT", "L3 cushion-guard modulator (per-position strength) not built",
			domain.PosDT),
		gatedPending("3G", "SL-021 — DT dynamic PFF alpha (0.50 Y1 → 0.10 Y2+)",
			"B5b-DT", "PFF blend alpha is a rubric internal, no hook yet",
			domain.PosDT, domain.PosDE),
		subSignalPending("3H", "Confidence floor — all-Unknown component → effective 1.000",
			"B5b (first rubric)", "component confidence inputs not on Layer4Input"),
		gatedPending("3I", "NGS anchor present only at CB & S",
			"B5b-CB / B5b-S", "rubric sub-signal weights not introspectable yet",
			domain.PosCB, domain.PosS),
		subSignalPending("3J", "EDGE classification routing (pass-rush share → DE vs LB)",
			"B5b-DE / B5b-LB + dispatch", "position-resolution dispatch (pass_rush_snap_share) not built"),
		gatedPending("3K", "S-curve boundary safety — output clamped to [1-cap, 1+cap]",
			"B5b (first rubric with S-curve)", "film/breakout extreme inputs not on Layer4Input",
			domain.PosWR),
		{id: "3L", name: "MFL player-ID string enforcement (leading zeros preserved)", b5bBlock: "(none — testable now)", eval: eval3L},
	}
}

// eval3C is the fully-wired exemplar that auto-flips when a QB/K rubric registers. SL-020
// forces Layer-4 RAS to exactly 1.000 for QB and K regardless of the athlete's RAS, so a
// very-low (0.10) and an elite (9.99) QB RAS must both yield RASEffective == 1.0000, and a
// K's Combined must be exactly 1.0000. RAS rides in on PlayerInput, so this needs no new
// Layer4Input field — only the rubric.
func eval3C(reg RubricRegistry) (CaseState, string) {
	if st, why, ok := requireRubrics(reg, domain.PosQB, domain.PosK); !ok {
		return st, why
	}
	qb := reg[domain.PosQB]
	for _, ras := range []float64{0.10, 9.99} {
		out := qb.Apply(engine.Layer4Input{Player: engine.PlayerInput{
			Position: domain.PosQB, RAS: ras, HasRAS: true,
		}})
		if out.RASEffective != 1.0 {
			return StateFail, fmt.Sprintf("QB RAS %.2f → RASEffective %.4f, want exactly 1.0000", ras, out.RASEffective)
		}
	}
	kout := reg[domain.PosK].Apply(engine.Layer4Input{Player: engine.PlayerInput{Position: domain.PosK}})
	if kout.Combined != 1.0 {
		return StateFail, fmt.Sprintf("K Combined %.4f, want exactly 1.0000", kout.Combined)
	}
	return StatePass, "QB RASEffective=1.0000 at RAS 0.10 and 9.99; K Combined=1.0000"
}

// eval3L runs today: MFL player IDs are strings and leading zeros are significant. It
// drives the real schema decode path (the boundary that would convert an ID to an int if
// it were going to) and asserts each ID survives byte-for-byte.
func eval3L(_ RubricRegistry) (CaseState, string) {
	for _, id := range []string{"0001", "0999", "14263"} {
		js := fmt.Sprintf(`{"id":%q,"name":"Test","position":"WR","salary":"1"}`, id)
		rec, err := schema.DecodePlayerRecord(strings.NewReader(js))
		if err != nil {
			return StateFail, fmt.Sprintf("id %q failed decode: %v", id, err)
		}
		if rec.ID != id {
			return StateFail, fmt.Sprintf("id %q mutated to %q (leading zero stripped or int-coerced)", id, rec.ID)
		}
	}
	return StatePass, `IDs "0001","0999","14263" preserved as strings with leading zeros`
}
