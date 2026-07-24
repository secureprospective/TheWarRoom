// Package defense holds the B5b defense (IDP) Layer-4 rubrics. DT is the skeleton-setter
// (LB/DE/CB/S reuse its shape); it is deliberately built SECOND, after the QB offense
// skeleton, to STRESS-TEST the architecture on a defensive rubric before the rest are
// mass-produced (AD-15). Each rubric implements engine.Layer4 as a PURE function over the
// shared curve primitives.
//
// PURITY (depguard engine-is-pure, **/internal/engine/**): this package imports only the
// engine (for the Layer4 contract types) and the curve package — no store, db, ingestion,
// or I/O. Every input arrives as a parameter on Layer4Input. The SL-021 L3 cushion guard
// the DT rubric is paired with lives in the pipeline (engine.ApplyCushionGuard) and rides
// on Calibration; this rubric carries only the BREAKOUT-internal half of the cushion (the
// Age-Trajectory sub-signal), which is pure L4.
package defense

import (
	"github.com/secureprospective/TheWarRoom/internal/engine"
	"github.com/secureprospective/TheWarRoom/internal/engine/l4/curve"
)

// DT Layer-4 mechanics (docs/scoring-engine/DT_Rubric.md, locked v1.0). Rubric constants —
// never admin-exposed (Hard Constraint). DT exercises three mechanics QB did not: an ACTIVE
// RAS component (not SL-020-forced), SL-005 film compression (±3%, not ±5%), and the SL-021
// Cushion Guard (whose L3 half is engine.ApplyCushionGuard; the breakout-trajectory half is
// here). NGS Coverage is CB/S-only and MUST NOT appear here (Hard Constraint). SL-019 is
// NOT applied — the Cushion Guard replaces it (Hard Constraint).
const (
	// Film component (DT_Rubric §2): S-curve over the IDP film composite. SL-005 compression
	// is expressed via the cap (±3%) and steepness (10.0), not a position_weight (1.00). FILM
	// Thread C (C-4 step 2) wired that composite: the eliminated subjective sources were
	// replaced by the Madden defense sub-attribute composite (K1), blended UPSTREAM at 0.95 of
	// the DT film budget (rankings.applyScouting) into the single [0,1] FilmComposite this
	// rubric still consumes unchanged; a DT whose Madden record did not resolve keeps
	// HasFilm=false → film returns neutral 1.000 (Data-Parity).
	dtFilmInflection = 0.50
	dtFilmSteepness  = 10.0 // SL-005 compression — vs standard 12.0
	dtFilmCap        = 0.03 // ±3% (SL-005) — vs standard ±5%

	// RAS component (DT_Rubric §3): ACTIVE at DT (no SL-020). High-tier cap; raw RAS / 10.
	dtRASInflection = 0.50
	dtRASSteepness  = 10.0
	dtRASCap        = 0.08 // ±8% (High-tier — RAS is primary at year 0 per SL-021)
	// RAS_position_weight follows the High-tier SL-021 schedule: rookie/pre-NFL 1.00, after
	// 1 NFL season 0.50, year 2+ 0.10. v1.0 ships the ROOKIE weight only — the harness and
	// first real data are rookie cohorts, and the receding schedule needs an NFL-data-stage
	// input that no consumer yet supplies. "Version the boundary": that input (and the
	// 0.50/0.10 tiers) wires in when a case gates on it, not before.
	dtRASRookieWeight = 1.00

	// Breakout component (DT_Rubric §4): standard ±5% cap (NOT compressed), steepness 11.0.
	dtBreakoutInflection = 0.50
	dtBreakoutSteepness  = 11.0
	dtBreakoutCap        = 0.05

	// Breakout sub-signal weights (DT_Rubric §4) — sum to 1.00. College Production Share is
	// elevated to 0.45 (highest of any position); breakout age reduced to 0.20.
	dtWeightBreakoutAge   = 0.20
	dtWeightSchoolTier    = 0.20
	dtWeightCollegeShare  = 0.45
	dtWeightAgeTrajectory = 0.15

	// SL-021 Cushion Guard constants (DT_Rubric §1/§3). The L3 half ships to the engine via
	// Calibration (set by composition); these mirror those numbers for the BREAKOUT
	// Age-Trajectory half, which is pure-L4 and applied here. When raw RAS ≥ threshold AND
	// age > peak, the trajectory's fall below neutral (0.50) is slowed by declineFactor.
	dtCushionRAS     = 8.00
	dtCushionDecline = 0.90 // 10% slower decline
	// dtPeakAge MIRRORS composition.peakLimit(PosDT) — both anchor to DT_Rubric §1 "Layer 3
	// Peak Limit: 30". The engine is pure (cannot import composition), so the value is
	// duplicated by necessity; both cite the same spec line, and drift would desync the L3
	// and L4 cushion windows (GLM review m1 — accepted, single-spec-sourced).
	dtPeakAge = 30.0
)

// DT is the defensive-tackle Layer-4 rubric. Combined = film × RAS × breakout
// (Backend_Architecture:256). Film is Data-Parity neutral in v1.0 (IDP film weights UNSET —
// no clean source yet, DT_Rubric film redesign pending), so DTs differentiate on the ACTIVE
// RAS component and breakout (incl. the Cushion-Guard-protected age trajectory).
//
// Curves are held on the struct (not package globals — gochecknoglobals) and built once.
type DT struct {
	breakoutAge   []curve.Breakpoint
	collegeShare  []curve.Breakpoint
	ageTrajectory []curve.Breakpoint
}

// NewDT builds the DT rubric with its DT_Rubric §4 normalization curves. School tier is
// position-independent and arrives pre-normalized, so it has no curve here.
func NewDT() *DT {
	return &DT{
		// Breakout Age (years → [0,1]): ≤20 → 1.00, 21 → 0.75, 22 → 0.45, ≥23 → 0.15.
		breakoutAge: []curve.Breakpoint{{X: 20, Y: 1.00}, {X: 21, Y: 0.75}, {X: 22, Y: 0.45}, {X: 23, Y: 0.15}},
		// College Production Share (final-year TFL+Sack market share → [0,1]): ≤0.08 → 0.15,
		// 0.15 → 0.55, ≥0.22 → 1.00.
		collegeShare: []curve.Breakpoint{{X: 0.08, Y: 0.15}, {X: 0.15, Y: 0.55}, {X: 0.22, Y: 1.00}},
		// Age Trajectory (player age → [0,1]) relative to DT peak 30; declines past peak,
		// where the Cushion Guard then modulates it for high-RAS DTs.
		ageTrajectory: []curve.Breakpoint{
			{X: 26, Y: 1.00}, {X: 27, Y: 0.85}, {X: 28, Y: 0.70}, {X: 29, Y: 0.55}, {X: 30, Y: 0.50},
			{X: 31, Y: 0.35}, {X: 32, Y: 0.20}, {X: 33, Y: 0.10}, {X: 34, Y: 0.00},
		},
	}
}

// Apply computes the DT Layer-4 output. Film is the SL-005-compressed S-curve over the IDP
// composite, or neutral 1.000 when no source is populated (Data-Parity). RAS is ACTIVE: the
// raw RAS / 10 through a High-tier S-curve, scaled by the rookie position weight, or neutral
// 1.000 when RAS is absent (Data-Parity — an absent signal never penalizes). Breakout weights
// the four sub-signals; its Age-Trajectory sub-signal is Cushion-Guard-protected past peak
// for high-RAS DTs (the breakout half of SL-021; the L3 half is engine.ApplyCushionGuard).
func (d *DT) Apply(in engine.Layer4Input) engine.Layer4Output {
	sc := in.Scouting
	p := in.Player

	filmRaw := curve.NeutralNorm // Data-Parity neutral default (IDP film weights unset)
	film := 1.0
	if sc.HasFilm {
		filmRaw = sc.FilmComposite
		film = curve.Scurve(sc.FilmComposite, dtFilmInflection, dtFilmSteepness, dtFilmCap)
	}

	// RAS component (Data-Parity neutral when absent). position_weight scales the S-curve's
	// deviation from neutral, so at the rookie weight (1.00) the effect is the full S-curve;
	// a later receding weight pulls it toward 1.000 as NFL data accrues.
	rasEffective := 1.0
	if p.HasRAS {
		rasCurve := curve.Scurve(p.RAS/10.0, dtRASInflection, dtRASSteepness, dtRASCap)
		rasEffective = 1.0 + dtRASRookieWeight*(rasCurve-1.0)
	}

	// Breakout composite. Age trajectory always has a value (age is known); for a qualifying
	// high-RAS DT past peak the Cushion Guard slows its decline below neutral.
	ageTraj := curve.Interp(d.ageTrajectory, p.Age)
	if p.HasRAS && p.RAS >= dtCushionRAS && p.Age > dtPeakAge {
		ageTraj = curve.NeutralNorm - (curve.NeutralNorm-ageTraj)*dtCushionDecline
	}
	composite := dtWeightBreakoutAge*curve.SubSignal(sc.HasBreakoutAge, d.breakoutAge, sc.BreakoutAge) +
		dtWeightSchoolTier*curve.Present(sc.HasSchoolTier, sc.SchoolTierNorm) +
		dtWeightCollegeShare*curve.SubSignal(sc.HasCollegeShare, d.collegeShare, sc.CollegeShare) +
		dtWeightAgeTrajectory*ageTraj
	breakout := curve.Scurve(composite, dtBreakoutInflection, dtBreakoutSteepness, dtBreakoutCap)

	return engine.Layer4Output{
		FilmEffective:     film,
		FilmRaw:           filmRaw,
		RASEffective:      rasEffective,
		BreakoutEffective: breakout,
		Combined:          film * rasEffective * breakout,
	}
}

// SL021Alpha is the case-3G introspection hook: the DT-unique dynamic SL-021 EMA blend rate
// over the pass-rush grade — 0.50 in NFL Year 1 (aggressive blend to displace rookie-era RAS
// dominance), 0.10 in Year 2+ (stable vet signal). (Formerly "PFFAlpha": PFF is RETIRED —
// TOS-restricted + paywalled — and the graded input is now the pfrpassrush pressure composite,
// not a PFF grade; only the SL-021 α schedule survives, unchanged.) It is a RUBRIC INTERNAL
// exposed for the harness only; it deliberately does NOT live on Layer4Output (the production
// surface). nflYear is 1-based; year ≤ 1 is the rookie/first NFL season.
func (d *DT) SL021Alpha(nflYear int) float64 {
	if nflYear <= 1 {
		return 0.50
	}
	return 0.10
}

// SL021Blend is the SL-021 pass-rush-grade EMA step (case-3G introspection hook): it folds this
// season's observed grade into the prior smoothed value at rate alpha —
// new = (1-alpha)·previous + alpha·observation. It is a PURE, position-agnostic mechanic; the
// per-position α SCHEDULE is what differs (DT dynamic via SL021Alpha, DE fixed via its control),
// so this helper takes alpha as a parameter and carries no schedule of its own. Both `previous`
// and `observation` are [0,1] grades.
//
// SCOPE (2026-07-24, α-schedule-only wiring): this is the SL-021 blend MECHANIC exposed for the
// harness. It is NOT yet fed a live pass-rush observation into production scoring — the pressure
// composite that would supply `observation` (pfrpassrush) proved largely redundant with the
// locked Madden IDP film anchor at DT (r≈0.75) and DE (r≈0.82) in the C-1 evidence
// (docs/data-layer/PassRush_C1_Distributions.md), so a live DT/DE pressure weight is DEFERRED to
// the expert-panel gate. This helper closes case 3G by proving the α schedule + blend math on
// the spec's synthetic inputs; it sets no weight and does not touch the locked film budget.
func SL021Blend(previous, observation, alpha float64) float64 {
	return (1-alpha)*previous + alpha*observation
}
