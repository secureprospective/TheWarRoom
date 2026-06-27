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
		{id: "3B", name: "Herbert pattern — L4 pulls below 1.00 for weak-profile vets", b5bBlock: "B5b-RB", eval: eval3B},
		{id: "3C", name: "SL-020 — QB & K Layer-4 RAS forced to exactly 1.000", b5bBlock: "B5b-QB / B5b-K", eval: eval3C},
		gatedPending("3D", "SL-005 — film compression ±3% at LB/DT vs ±5% elsewhere",
			"B5b-LB / B5b-DT", "FilmRaw hook on Layer4Output (built); awaiting LB + WR rubrics",
			domain.PosLB, domain.PosDT, domain.PosWR),
		gatedPending("3E", "SL-019 — breakout modulator lifts with RAS (DE)",
			"B5b-DE", "modulated breakout intermediate not on Layer4Output",
			domain.PosDE),
		{id: "3F", name: "SL-021 — DT cushion guard (RAS ≥ 8.00 → 10% decel)", b5bBlock: "B5b-DT", eval: eval3F},
		gatedPending("3G", "SL-021 — DT dynamic PFF alpha (0.50 Y1 → 0.10 Y2+)",
			"B5b-DT", "DT.PFFAlpha introspection hook (built); awaiting DE rubric",
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

// eval3B is the B5b-RB close gate (the Herbert pattern): Layer 4 PULLS BELOW 1.000 for a
// genuinely-thin-profile vet, while staying above 1.000 for a strong-profile RB. With film
// Data-Parity neutral this session (no offense film source), the sub-1.000 pull is driven by
// the BREAKOUT component — a late breakout age, low college workload, smaller school, and a
// post-peak age push the breakout composite below the 0.50 inflection (RB_Rubric §7). The
// spec's literal Khalil Herbert is a borderline profile whose sub-1.000 result in §5 is
// film-driven; that half reproduces when the film source lands. RAS rides on PlayerInput, so
// this needs no new Layer4Input field — only the registered RB rubric.
func eval3B(reg RubricRegistry) (CaseState, string) {
	if st, why, ok := requireRubrics(reg, domain.PosRB); !ok {
		return st, why
	}
	rb := reg[domain.PosRB]
	// Genuinely-thin vet: breakout age 22 (≥21 → 0.20), Group of Five (0.70), college workload
	// 18% (≤20% → 0.15), age 28 (trajectory 0.05) → breakout composite well below 0.50.
	thin := rb.Apply(engine.Layer4Input{
		Player: engine.PlayerInput{Position: domain.PosRB, Age: 28, RAS: 6.0, HasRAS: true},
		Scouting: engine.ScoutingInput{
			BreakoutAge: 22, HasBreakoutAge: true,
			SchoolTierNorm: 0.75, HasSchoolTier: true, // RB G5 norm (RB_Rubric §4 softer non-P4)
			CollegeShare: 0.18, HasCollegeShare: true,
		},
	})
	// Strong profile (Bijan pattern): true-freshman breakout, Power Four, dominant workload.
	strong := rb.Apply(engine.Layer4Input{
		Player: engine.PlayerInput{Position: domain.PosRB, Age: 24, RAS: 9.55, HasRAS: true},
		Scouting: engine.ScoutingInput{
			BreakoutAge: 19, HasBreakoutAge: true,
			SchoolTierNorm: 1.00, HasSchoolTier: true,
			CollegeShare: 0.53, HasCollegeShare: true,
		},
	})
	if !(thin.Combined < 1.0) {
		return StateFail, fmt.Sprintf("thin-profile RB Combined %.4f, want < 1.0000 (the Herbert pull)", thin.Combined)
	}
	if !(strong.Combined > 1.0) {
		return StateFail, fmt.Sprintf("strong-profile RB Combined %.4f, want > 1.0000", strong.Combined)
	}
	if !(strong.Combined > thin.Combined) {
		return StateFail, fmt.Sprintf("strong RB %.4f should out-score thin RB %.4f", strong.Combined, thin.Combined)
	}
	if thin.FilmEffective != 1.0 {
		return StateFail, fmt.Sprintf("film must be Data-Parity neutral (no source), got %.4f", thin.FilmEffective)
	}
	return StatePass, fmt.Sprintf("thin RB Combined=%.4f (<1, breakout-driven) vs strong RB %.4f (>1); film neutral", thin.Combined, strong.Combined)
}

// eval3F is the B5b-DT close gate (SL-021 Late-Career Cushion Guard). The guard has TWO
// halves and this case closes BOTH (GLM review M4): (1) the L3 decay modulator at the
// engine level — past peak, a DT with raw RAS ≥ 8.00 decays SLOWER than an identical DT
// below threshold, and a sub-threshold RAS leaves the raw pull unchanged; (2) the L4
// breakout Age-Trajectory cushion inside the REGISTERED DT rubric — past peak a qualifying
// DT's breakout component is lifted vs an identical sub-threshold DT. The cushion strength
// is the per-position value the DT rubric ships via Calibration (8.00 / 0.90).
func eval3F(reg RubricRegistry) (CaseState, string) {
	if st, why, ok := requireRubrics(reg, domain.PosDT); !ok {
		return st, why
	}
	// --- Half 1: L3 decay modulator ---
	const peak, rate, age = 30.0, 0.03, 33.0 // three years past the DT peak
	const threshold, decline = 8.00, 0.90
	raw, err := engine.ApplyDecay(age, peak, rate)
	if err != nil {
		return StateFail, fmt.Sprintf("raw decay errored: %v", err)
	}
	cushioned := engine.ApplyCushionGuard(raw, 8.00, true, threshold, decline) // qualifying RAS
	below := engine.ApplyCushionGuard(raw, 7.99, true, threshold, decline)     // just under threshold
	if !(cushioned > raw) {
		return StateFail, fmt.Sprintf("L3: cushion did not slow decay: cushioned %.4f not > raw %.4f", cushioned, raw)
	}
	if below != raw {
		return StateFail, fmt.Sprintf("L3: sub-threshold RAS must not be cushioned: got %.4f, want raw %.4f", below, raw)
	}
	// --- Half 2: L4 breakout Age-Trajectory cushion through the registered DT rubric ---
	dt := reg[domain.PosDT]
	bk := engine.ScoutingInput{BreakoutAge: 22, HasBreakoutAge: true, SchoolTierNorm: 0.70, HasSchoolTier: true, CollegeShare: 0.15, HasCollegeShare: true}
	hi := dt.Apply(engine.Layer4Input{Player: engine.PlayerInput{Position: domain.PosDT, Age: 32, RAS: 9.0, HasRAS: true}, Scouting: bk})
	lo := dt.Apply(engine.Layer4Input{Player: engine.PlayerInput{Position: domain.PosDT, Age: 32, RAS: 7.0, HasRAS: true}, Scouting: bk})
	if !(hi.BreakoutEffective > lo.BreakoutEffective) {
		return StateFail, fmt.Sprintf("L4: age-32 cushioned breakout %.5f not > un-cushioned %.5f", hi.BreakoutEffective, lo.BreakoutEffective)
	}
	return StatePass, fmt.Sprintf("L3 age33 pull %.4f→%.4f at RAS 8.00 (RAS 7.99 stays %.4f); L4 age32 breakout %.5f (RAS 9) > %.5f (RAS 7)",
		raw, cushioned, below, hi.BreakoutEffective, lo.BreakoutEffective)
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
