package l4

import "math"

// NeutralNorm is the normalized value that leaves a component unmoved: the S-curve's inflection.
// An absent sub-signal takes it, so missing data neither lifts nor penalizes.
const NeutralNorm = 0.50

// Breakpoint is one (X, Y) anchor of a piecewise-linear curve.
type Breakpoint struct {
	X, Y float64
}

// Component is one capped S-curve: output = 1 + Cap·(2σ(Steepness·(input − 0.5)) − 1), bounded to
// [1−Cap, 1+Cap]. A Cap of 0 turns the component off: it returns exactly 1.
type Component struct {
	Steepness float64
	Cap       float64
}

// Apply runs input through the S-curve. A non-finite input is unknown and returns 1.
func (c Component) Apply(input float64) float64 {
	if math.IsNaN(input) || math.IsInf(input, 0) {
		return 1.0
	}
	sigma := 1.0 / (1.0 + math.Exp(-c.Steepness*(input-NeutralNorm)))
	return min(max(1.0+c.Cap*(2.0*sigma-1.0), 1.0-c.Cap), 1.0+c.Cap)
}

// interp maps x through anchors sorted by X, flat past either end. An empty curve returns 0.
func interp(c []Breakpoint, x float64) float64 {
	if len(c) == 0 {
		return 0
	}
	last := c[len(c)-1]
	switch {
	case x <= c[0].X:
		return c[0].Y
	case x >= last.X:
		return last.Y
	}
	for i := 1; i < len(c); i++ {
		if hi := c[i]; x <= hi.X {
			lo := c[i-1]
			return lo.Y + (x-lo.X)/(hi.X-lo.X)*(hi.Y-lo.Y)
		}
	}
	return last.Y
}

// curved is a raw sub-signal mapped through its curve, or neutral when absent.
func curved(present bool, c []Breakpoint, raw float64) float64 {
	if !present {
		return NeutralNorm
	}
	return interp(c, raw)
}

// given is an already-normalized sub-signal, or neutral when absent.
func given(present bool, norm float64) float64 {
	if !present {
		return NeutralNorm
	}
	return norm
}

// athleticLift raises a normalized sub-signal toward 1 by the player's RAS:
// base + (1 − base)·strength·RAS/10, with RAS/10 clamped to [0,1]. A strength of 0, an absent RAS
// or a non-finite RAS leaves base alone; a non-finite base is unknown and returns neutral.
func athleticLift(base, ras, strength float64, hasRAS bool) float64 {
	if strength == 0 {
		return base
	}
	if math.IsNaN(base) || math.IsInf(base, 0) {
		return NeutralNorm
	}
	norm := ras / 10.0
	if !hasRAS || math.IsNaN(norm) || math.IsInf(norm, 0) {
		return base
	}
	return base + (1.0-base)*strength*min(max(norm, 0), 1)
}
