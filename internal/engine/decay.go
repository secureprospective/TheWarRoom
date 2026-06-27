package engine

import (
	"fmt"
	"math"
)

// ApplyDecay is Layer 3: the age pull. Below the position peak limit there is no decay
// (AgePull = 1.0); past it the pull compounds annually (Engine_Specification:113):
//
//	age_pull = (1 - decay_rate) ^ max(0, age - peak_limit)
//
// The SL-018 RAS buffer and SL-021 cushion guard are per-position MODULATORS layered on
// this raw pull; they consume per-position modulator strengths that ship with B5b, so
// B5a computes the raw pull only. decayRate and peakLimit arrive as parameters.
//
// AgePull multiplies straight into the score, so a poisoned value would round-trip
// silently into rankings — the same hazard L5 guards its denominator against. Hence
// ApplyDecay fails loud on non-finite inputs and on any input that yields a non-finite
// pull (e.g. decayRate > 1 makes the base negative, and a negative base raised to a
// fractional exponent is NaN).
func ApplyDecay(age, peakLimit, decayRate float64) (float64, error) {
	if !finite(age, peakLimit, decayRate) {
		return 0, fmt.Errorf("engine: decay inputs must be finite, got age=%v peak=%v rate=%v", age, peakLimit, decayRate)
	}
	// decayRate > 1 makes the base negative: a negative base is invalid for an age
	// multiplier (it would flip the score's sign at an integer exponent, or be NaN at a
	// fractional one). Reject it directly rather than depend on the result check below.
	if 1-decayRate < 0 {
		return 0, fmt.Errorf("engine: decay rate must be <= 1, got %v", decayRate)
	}
	over := age - peakLimit
	if over < 0 {
		over = 0
	}
	pull := math.Pow(1-decayRate, over)
	if !finite(pull) {
		return 0, fmt.Errorf("engine: decay produced a non-finite pull (decayRate=%v over=%v)", decayRate, over)
	}
	return pull, nil
}
