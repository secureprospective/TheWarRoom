// Package offense holds the B5b offense Layer-4 rubrics (QB first; RB/WR/TE reuse this
// shape). Each rubric implements engine.Layer4 as a PURE function: it reads the player's
// scouting sub-signals from the Layer4Input, normalizes the position-specific ones with
// its own curves, and returns the three component multipliers plus their product.
//
// PURITY (depguard engine-is-pure, **/internal/engine/**): this package imports only the
// engine (for the Layer4 contract types) and math — no store, db, ingestion, or I/O. Every
// input arrives as a parameter on Layer4Input. The composition boundary fills those
// inputs; this package never reaches for them.
package offense

import "math"

// scurve is the Shape-B sigmoid the rubrics apply to every component composite
// (Engine_Specification / *_Rubric §5):
//
//	output = 1 + cap × (2·σ(steepness·(input − inflection)) − 1),  σ(x) = 1/(1+e^(−x))
//
// The output is naturally bounded to [1−cap, 1+cap]: σ ∈ (0,1) ⇒ (2σ−1) ∈ (−1,1). A
// neutral input (input == inflection) returns exactly 1.0. capBand is the component
// asymptote (±0.05 at offense). A non-finite input is treated as "unknown" and returns the
// neutral 1.0 (the Data-Parity stance) — a plain min/max clamp does NOT catch NaN, since
// every comparison with NaN is false, so it is guarded explicitly here. The finite result
// is then clamped to the cap band so an extreme input can never escape the documented bound
// (M3 — both the NaN guard and the clamp are proven by test).
func scurve(input, inflection, steepness, capBand float64) float64 {
	if math.IsNaN(input) || math.IsInf(input, 0) {
		return 1.0
	}
	sigma := 1.0 / (1.0 + math.Exp(-steepness*(input-inflection)))
	out := 1.0 + capBand*(2.0*sigma-1.0)
	lo, hi := 1.0-capBand, 1.0+capBand
	if out < lo {
		return lo
	}
	if out > hi {
		return hi
	}
	return out
}

// neutralNorm is the normalized sub-signal value that leaves a component unmoved: it is the
// S-curve inflection, so a composite made entirely of neutral sub-signals yields exactly
// 1.000. It is what the Data-Parity Rule substitutes for an ABSENT sub-signal.
const neutralNorm = 0.50

// subSignal returns a sub-signal's normalized value via its curve when present, or the
// neutral value when absent (Data-Parity Rule). Used for raw sub-signals the rubric maps
// through a curve (breakout age, college share).
func subSignal(present bool, curve []breakpoint, raw float64) float64 {
	if !present {
		return neutralNorm
	}
	return interp(curve, raw)
}

// present returns an already-normalized sub-signal when present, or the neutral value when
// absent. Used for pre-normalized inputs the rubric does not curve (school tier).
func present(has bool, norm float64) float64 {
	if !has {
		return neutralNorm
	}
	return norm
}

// breakpoint is one (x, y) anchor of a piecewise-linear normalization curve.
type breakpoint struct {
	x, y float64
}

// interp maps a raw sub-signal x onto its normalized value using a piecewise-linear curve
// defined by anchors sorted ascending in x. Below the first anchor it returns the first
// y; above the last it returns the last y (the curves are flat past their endpoints, e.g.
// QB breakout age ≤20 → 1.00 and ≥23 → 0.10). Between anchors it interpolates linearly.
// The anchor table is a rubric constant, so it is never empty in practice; an empty table
// returns 0 rather than panicking.
func interp(curve []breakpoint, x float64) float64 {
	if len(curve) == 0 {
		return 0
	}
	if x <= curve[0].x {
		return curve[0].y
	}
	last := curve[len(curve)-1]
	if x >= last.x {
		return last.y
	}
	for i := 1; i < len(curve); i++ {
		hi := curve[i]
		if x <= hi.x {
			lo := curve[i-1]
			frac := (x - lo.x) / (hi.x - lo.x)
			return lo.y + frac*(hi.y-lo.y)
		}
	}
	return last.y
}
