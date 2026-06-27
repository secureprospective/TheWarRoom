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
	// decayRate must be in [0,1]. > 1 makes the base negative (sign flip at an integer
	// exponent, NaN at a fractional one); < 0 makes the base > 1, so the "pull" GROWS past
	// peak and inflates the score with age — the opposite of decay. The boundary already
	// range-gates decay ∈ [0,1] (composition); this is the engine's defense-in-depth, and
	// symmetric so neither end slips through (GLM review m3).
	if decayRate < 0 || 1-decayRate < 0 {
		return 0, fmt.Errorf("engine: decay rate must be in [0,1], got %v", decayRate)
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

// ApplyCushionGuard is the SL-021 Late-Career Cushion Guard: a per-position L3 MODULATOR
// layered on the raw age pull (DT_Rubric §1/§3). When a player's raw RAS is at or above
// threshold, the decline past peak is slowed — the cushioned pull moves a fraction
// (declineFactor) of the way the raw pull fell below the 1.0 peak baseline:
//
//	cushioned = 1.0 − (1.0 − rawPull) × declineFactor   (DT: declineFactor 0.90 ⇒ 10% slower)
//
// It is a sibling pure function to ApplyDecay rather than inlined into it: the seam is the
// same (L3, Calibration-driven, depguard-pure), but keeping it separate leaves ApplyDecay's
// signature and tests untouched and gives the planted-failure gate a direct target
// (B5b-DT). The guard is a STRICT no-op unless ALL of: threshold > 0 (DISABLED at every
// position but DT, so the zero value is inert), the player has a real RAS (an imputed
// fallback never earns the cushion), and that RAS ≥ threshold. Below peak rawPull is 1.0,
// so the formula is already a no-op there regardless. declineFactor MUST be in [0,1] to be
// a deceleration: < 0 would push the cushioned pull above the 1.0 peak (boosting the score
// past peak), > 1 would AMPLIFY decline. An out-of-range factor is therefore treated as
// misconfigured and DISABLES the guard (returns the raw pull) rather than corrupting the
// score — consistent with the no-op stance (GLM review m2).
func ApplyCushionGuard(rawPull, ras float64, hasRAS bool, threshold, declineFactor float64) float64 {
	if threshold <= 0 || !hasRAS || ras < threshold {
		return rawPull
	}
	if declineFactor < 0 || declineFactor > 1 {
		return rawPull // misconfigured strength → disabled, never amplify/boost
	}
	return 1.0 - (1.0-rawPull)*declineFactor
}
