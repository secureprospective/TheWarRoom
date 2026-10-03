package engine

import (
	"fmt"
	"math"

	"github.com/secureprospective/TheWarRoom/internal/numeric"
)

// ApplyDecay is Layer 3, the age pull (Engine_Specification:113):
//
//	age_pull = (1 - decay_rate) ^ max(0, age - peak_limit)
//
// The pull multiplies straight into the score, so a non-finite input or result is an error.
func ApplyDecay(age, peakLimit, decayRate float64) (float64, error) {
	if !numeric.Finite(age, peakLimit, decayRate) {
		return 0, fmt.Errorf("engine: decay inputs must be finite, got age=%v peak=%v rate=%v", age, peakLimit, decayRate)
	}
	// Above 1 the base goes negative; below 0 the pull grows with age.
	if decayRate < 0 || 1-decayRate < 0 {
		return 0, fmt.Errorf("engine: decay rate must be in [0,1], got %v", decayRate)
	}
	over := age - peakLimit
	if over < 0 {
		over = 0
	}
	pull := math.Pow(1-decayRate, over)
	if !numeric.Finite(pull) {
		return 0, fmt.Errorf("engine: decay produced a non-finite pull (decayRate=%v over=%v)", decayRate, over)
	}
	return pull, nil
}

// ApplyCushionGuard is the SL-021 Late-Career Cushion Guard (DT_Rubric §1/§3): a measured
// RAS at or above threshold slows the decline past peak:
//
//	cushioned = 1.0 − (1.0 − rawPull) × declineFactor   (DT: 0.90 ⇒ 10% slower)
//
// A zero threshold disables it (every position but DT), and an imputed RAS never earns it.
// A declineFactor outside [0,1] would boost or amplify instead of slowing, so it disables
// the guard rather than corrupt the score.
func ApplyCushionGuard(rawPull, ras float64, hasRAS bool, threshold, declineFactor float64) float64 {
	if threshold <= 0 || !hasRAS || ras < threshold {
		return rawPull
	}
	if declineFactor < 0 || declineFactor > 1 {
		return rawPull
	}
	return 1.0 - (1.0-rawPull)*declineFactor
}
