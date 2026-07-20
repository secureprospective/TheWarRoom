package engine

import (
	"fmt"
	"math"

	"github.com/secureprospective/TheWarRoom/internal/numeric"
)

// The fixed L5 cap multipliers (Engine_Specification:399). These are NOT calibration —
// only the tier BOUNDARY percentages (ColdCeiling/HotFloor) are admin-tunable (B4).
const (
	coldMultiplier    = 1.15
	neutralMultiplier = 1.00
	hotMultiplier     = 0.85
)

// CapScaling is L5's result: the final AdjustedScore, the multiplier applied, and the
// tier it fell in.
type CapScaling struct {
	AdjustedScore float64
	Multiplier    float64
	Tier          CapTier
}

// ApplyCapScaling is Layer 5: salary-as-a-percent-of-cap places the player in a tier,
// and the tier's fixed multiplier scales the scouting-adjusted score
// (Engine_Specification:392):
//
//	salary% = salary / leagueCap × 100
//	salary% < ColdCeiling → Cold ×1.15 · > HotFloor → Hot ×0.85 · else Neutral ×1.00
//
// leagueCap is the denominator, so it must be a positive finite number — a zero or
// non-finite cap would silently produce a NaN/Inf score that round-trips into rankings.
// Rather than emit a poisoned score, L5 fails loud (the same class of bug B4's review
// caught at the override gate).
func ApplyCapScaling(scoutingAdjusted, salary, leagueCap, coldCeiling, hotFloor float64) (CapScaling, error) {
	if leagueCap <= 0 || math.IsNaN(leagueCap) || math.IsInf(leagueCap, 0) {
		return CapScaling{}, fmt.Errorf("engine: league cap must be positive and finite, got %v", leagueCap)
	}
	// A non-finite salary or boundary makes every tier comparison false and silently
	// classifies the player Neutral — a wrong tier label with no error. Reject it.
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
