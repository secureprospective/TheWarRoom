package defense

// SL-021 shared mechanic. This file holds the position-AGNOSTIC half of SL-021 (the pass-rush-
// grade EMA blend); the per-position α SCHEDULE lives with each rubric (DT.SL021Alpha dynamic,
// DE.SL021Alpha fixed control). Kept out of dt.go so nothing implies DT owns the blend — DE's
// control uses the same helper.

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
