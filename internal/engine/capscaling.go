package engine

import (
	"fmt"
	"math"

	"github.com/secureprospective/TheWarRoom/internal/numeric"
)

// The fixed L5 multipliers (Engine_Specification:399). Only the tier boundaries are tunable.
const (
	coldMultiplier    = 1.15
	neutralMultiplier = 1.00
	hotMultiplier     = 0.85
)

// CapScaling is L5's result.
type CapScaling struct {
	AdjustedScore float64
	Multiplier    float64
	Tier          CapTier
}

// ApplyCapScaling is Layer 5 (Engine_Specification:392):
//
//	salary% = salary / leagueCap × 100
//	salary% < ColdCeiling → Cold ×1.15 · > HotFloor → Hot ×0.85 · else Neutral ×1.00
//
// A zero or non-finite cap would put NaN or Inf into rankings, so it is an error.
func ApplyCapScaling(scoutingAdjusted, salary, leagueCap, coldCeiling, hotFloor float64) (CapScaling, error) {
	if leagueCap <= 0 || math.IsNaN(leagueCap) || math.IsInf(leagueCap, 0) {
		return CapScaling{}, fmt.Errorf("engine: league cap must be positive and finite, got %v", leagueCap)
	}
	// A non-finite input fails every comparison and would silently land in Neutral.
	if !numeric.Finite(salary, coldCeiling, hotFloor) {
		return CapScaling{}, fmt.Errorf("engine: cap inputs must be finite, got salary=%v cold=%v hot=%v", salary, coldCeiling, hotFloor)
	}
	pct := salary / leagueCap * 100
	mult := neutralMultiplier
	tier := CapTierNeutral
	switch {
	case pct < coldCeiling:
		mult, tier = coldMultiplier, CapTierCold
	case pct > hotFloor:
		mult, tier = hotMultiplier, CapTierHot
	}
	return CapScaling{
		AdjustedScore: scoutingAdjusted * mult,
		Multiplier:    mult,
		Tier:          tier,
	}, nil
}
