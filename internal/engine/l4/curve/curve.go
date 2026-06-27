// Package curve holds the shared Layer-4 normalization primitives every B5b rubric
// uses: the Shape-B S-curve, the piecewise-linear interpolator, and the Data-Parity
// presence helpers. It was extracted from the offense package on the SECOND caller
// (B5b-DT, the defense skeleton-setter) so the two rubric families share one tested
// implementation rather than diverging copies (M17 — shared logic extracted on the
// second use).
//
// PURITY (depguard engine-is-pure, **/internal/engine/**): this package imports only
// math — no store, db, ingestion, or I/O. Every input arrives as a parameter.
package curve

import "math"

// NeutralNorm is the normalized sub-signal value that leaves a component unmoved: it is
// the S-curve inflection, so a composite made entirely of neutral sub-signals yields
// exactly 1.000. It is what the Data-Parity Rule substitutes for an ABSENT sub-signal.
const NeutralNorm = 0.50

// Scurve is the Shape-B sigmoid the rubrics apply to every component composite
// (Engine_Specification / *_Rubric §5):
//
//	output = 1 + cap × (2·σ(steepness·(input − inflection)) − 1),  σ(x) = 1/(1+e^(−x))
//
// The output is naturally bounded to [1−cap, 1+cap]: σ ∈ (0,1) ⇒ (2σ−1) ∈ (−1,1). A
// neutral input (input == inflection) returns exactly 1.0. capBand is the component
// asymptote (±0.05 at offense; ±0.03 for film at DT/LB per SL-005; ±0.08 for RAS at DT).
// A non-finite input is treated as "unknown" and returns the neutral 1.0 (the Data-Parity
// stance) — a plain min/max clamp does NOT catch NaN, since every comparison with NaN is
// false, so it is guarded explicitly here. The finite result is then clamped to the cap
// band so an extreme input can never escape the documented bound (M3 — both the NaN guard
// and the clamp are proven by test).
func Scurve(input, inflection, steepness, capBand float64) float64 {
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

// SubSignal returns a sub-signal's normalized value via its curve when present, or the
// neutral value when absent (Data-Parity Rule). Used for raw sub-signals the rubric maps
// through a curve (breakout age, college share).
func SubSignal(present bool, c []Breakpoint, raw float64) float64 {
	if !present {
		return NeutralNorm
	}
	return Interp(c, raw)
}

// Present returns an already-normalized sub-signal when present, or the neutral value
// when absent. Used for pre-normalized inputs the rubric does not curve (school tier).
func Present(has bool, norm float64) float64 {
	if !has {
		return NeutralNorm
	}
	return norm
}

// Breakpoint is one (X, Y) anchor of a piecewise-linear normalization curve.
type Breakpoint struct {
	X, Y float64
}

// Interp maps a raw sub-signal x onto its normalized value using a piecewise-linear curve
// defined by anchors sorted ascending in X. Below the first anchor it returns the first
// Y; above the last it returns the last Y (the curves are flat past their endpoints, e.g.
// QB breakout age ≤20 → 1.00 and ≥23 → 0.10). Between anchors it interpolates linearly.
// The anchor table is a rubric constant, so it is never empty in practice; an empty table
// returns 0 rather than panicking.
func Interp(c []Breakpoint, x float64) float64 {
	if len(c) == 0 {
		return 0
	}
	if x <= c[0].X {
		return c[0].Y
	}
	last := c[len(c)-1]
	if x >= last.X {
		return last.Y
	}
	for i := 1; i < len(c); i++ {
		hi := c[i]
		if x <= hi.X {
			lo := c[i-1]
			frac := (x - lo.X) / (hi.X - lo.X)
			return lo.Y + frac*(hi.Y-lo.Y)
		}
	}
	return last.Y
}
