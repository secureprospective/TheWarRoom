// Package engine is the B5a scoring pipeline: a PURE-FUNCTION spine that turns a
// player's inputs into a final AdjustedScore plus a tiebreaker key. It runs the
// fixed layer order from Backend_Architecture — L1 hygiene, (L2 base scoring is
// supplied as BasePoints by an earlier block), L3 age decay, L4 scouting (a
// pluggable per-position dispatch, identity by default until B5b), L5 cap scaling,
// L6 tiebreaker — accumulating at the two fixed points:
//
//	ScoutingAdjusted = BasePoints × AgePull × Layer4Output.Combined   (Backend_Architecture:259)
//	AdjustedScore    = ScoutingAdjusted × CapMultiplier               (Backend_Architecture:266)
//
// PURITY (depguard engine-is-pure): the engine imports NO store, db, normalize, or
// I/O. Every value arrives as a PARAMETER. A composition boundary OUTSIDE the engine
// (a later session) reads B3b/B3c/B4 and fills PlayerInput + Calibration; the engine
// only computes. domain is imported solely for the Position enum (a pure value type).
//
// Confidence scores are an INTERNAL engine concern and never appear in any output
// struct that reaches the UI (a Hard Constraint) — the identity Layer 4 carries none,
// and B5b's real Layer 4 keeps them off Result.
package engine

import "github.com/secureprospective/TheWarRoom/internal/domain"

// PlayerInput is the per-player scoring input the composition boundary fills from the
// normalized record (B3c) plus the supplied L2 BasePoints. The engine reads these as
// plain values; it never touches a store. L2 base scoring is a separate block, so
// BasePoints arrives already computed (the handoff's accumulation example treats it
// as a given).
type PlayerInput struct {
	Position   domain.Position
	BasePoints float64 // L2 output, supplied by the composition boundary
	Age        float64
	RAS        float64 // raw Relative Athletic Score; see HasRAS
	HasRAS     bool    // false → L1 imputes Calibration.RASFallback
	Salary     float64 // millions
	IsVeteran  bool    // L6 tenure tiebreaker (rookie loses to veteran)
}

// Calibration is the tunable parameter set the composition boundary fills: GLOBALS
// from B4 (decay rate, cap-tier percentages) and PER-POSITION values that ship WITH
// their B5b layer (peak limit, scarcity rank, salary floor). All plain values — the
// engine never references the params store. Adding a per-position table later changes
// what fills these fields, not the engine.
type Calibration struct {
	// L1 hygiene
	SalaryFloor float64 // salary is raised to this floor if below it
	RASFallback float64 // imputed RAS when HasRAS is false (spec fallback 5.00)
	// L3 decay
	PeakLimit float64 // age past which decay applies (per-position; B5b)
	DecayRate float64 // annual age-decay rate (B4 global, default 0.03)
	// L5 cap scaling
	LeagueCap   float64 // the cap AMOUNT (B3b rulebook), in the same units as Salary
	ColdCeiling float64 // salary% below this is Cold (B4 global, default 1.2)
	HotFloor    float64 // salary% above this is Hot (B4 global, default 4.8)
	// L6 tiebreaker
	ScarcityRank int // positional scarcity rank, higher wins (per-position; B5b)
}

// Layer4Input is what the pluggable Layer 4 receives. The identity default needs
// nothing from it; B5b's real implementations read the player's sub-signals through
// the composition-filled fields added here when those layers land.
type Layer4Input struct {
	Player PlayerInput
}

// Layer4Output is the scouting layer's result. Combined is the product of the three
// component multipliers and is the only field the accumulation reads; the components
// are retained for the M9a debug surface. No overall Layer 4 cap exists — component
// caps are the natural bounds (Backend_Architecture:256).
type Layer4Output struct {
	FilmEffective     float64
	RASEffective      float64
	BreakoutEffective float64
	Combined          float64
}

// Layer4 is the pluggable per-position scouting dispatch — the seam B5b fills with a
// real rubric per position. The pipeline calls it through this interface so the spine
// is testable end to end before any rubric exists (mirrors B3c's Writer/Reader DI).
type Layer4 interface {
	Apply(in Layer4Input) Layer4Output
}

// CapTier is the L5 salary-tier classification.
type CapTier string

// The three cap tiers (Engine_Specification L5).
const (
	CapTierCold    CapTier = "Cold"
	CapTierNeutral CapTier = "Neutral"
	CapTierHot     CapTier = "Hot"
)

// TiebreakerKey orders players with identical AdjustedScores (Backend_Architecture:270):
// veteran status first, then RAS, then positional scarcity. It affects sort order only.
type TiebreakerKey struct {
	IsVeteran    bool
	RAS          float64
	ScarcityRank int
}

// RanksAbove reports whether k should sort ABOVE o among equal AdjustedScores, applying
// the L6 protocol in order: tenure, then athletic score, then positional scarcity.
func (k TiebreakerKey) RanksAbove(o TiebreakerKey) bool {
	if k.IsVeteran != o.IsVeteran {
		return k.IsVeteran
	}
	if k.RAS != o.RAS {
		return k.RAS > o.RAS
	}
	return k.ScarcityRank > o.ScarcityRank
}

// Result is the engine's full per-player output: the final AdjustedScore plus every
// intermediate the M9a calibration surface inspects. No confidence score appears here.
type Result struct {
	BasePoints       float64
	AgePull          float64
	Layer4Output     Layer4Output
	ScoutingAdjusted float64
	CapMultiplier    float64
	CapTier          CapTier
	AdjustedScore    float64
	Tiebreaker       TiebreakerKey
}
