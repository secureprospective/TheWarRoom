// Package engine is what is left live of the layer 1-6 scoring engine: Layer 3 age decay and
// the per-position peak ages, which cmd/fit reads. The rest of the engine was removed on
// 2026-10-20 (docs/architecture/removed-code.md). It imports no store or I/O (depguard engine-is-pure).
package engine

import (
	"fmt"
	"math"

	"github.com/secureprospective/TheWarRoom/internal/domain"
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

// PeakLimit is the Layer-3 age past which decay applies (Engine_Specification, "Current Peak
// Limit Defaults"). These are meant to be admin-tunable (SL-017) but are not in the params
// store yet. An unknown position gets the latest peak, so it is never penalized early.
func PeakLimit(p domain.Position) float64 {
	switch p {
	case domain.PosQB:
		return 32
	case domain.PosRB:
		return 25
	case domain.PosWR, domain.PosTE, domain.PosLB:
		return 29
	case domain.PosDE, domain.PosDT, domain.PosK:
		return 30
	case domain.PosCB, domain.PosS:
		return 28
	case domain.PosFlag:
		return 32 // unclassified: most conservative (latest) peak
	default:
		return 32
	}
}
