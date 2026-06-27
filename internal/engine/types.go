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
	// L3 cushion guard (SL-021, DT-only in v1.0; no-op elsewhere). The Late-Career
	// Cushion Guard is a per-position L3 DECAY modulator: when a player's RAW RAS is at or
	// above CushionRASThreshold, the age-decay pull's decline past peak is slowed by
	// CushionDeclineFactor (see ApplyCushionGuard). It rides on Calibration — the strength
	// ships WITH the position's B5b rubric — so the pure engine consumes it as plain
	// parameters and never reaches a store. A zero threshold DISABLES the guard (every
	// position but DT), so the zero value is a safe no-op (B5b-DT gate decision).
	CushionRASThreshold  float64 // raw RAS at/above which the guard applies; 0 disables
	CushionDeclineFactor float64 // decline-velocity multiplier past peak (DT: 0.90 = 10% slower)
	// L5 cap scaling
	LeagueCap   float64 // the cap AMOUNT (B3b rulebook), in the same units as Salary
	ColdCeiling float64 // salary% below this is Cold (B4 global, default 1.2)
	HotFloor    float64 // salary% above this is Hot (B4 global, default 4.8)
	// L6 tiebreaker
	ScarcityRank int // positional scarcity rank, higher wins (per-position; B5b)
}

// ScoutingInput carries the Layer-4 scouting sub-signals the composition boundary
// assembles for a player. They are kept OFF PlayerInput so the L1/L3/L5 hygiene path
// never sees film/breakout signals and the L4 inputs never see cap/salary mechanics —
// the layers stay value-isolated. Every field is position-BLIND raw data; per-position
// normalization (breakout-age / college-share / age-trajectory curves) lives in each
// B5b rubric (Approach A — the engine normalizes). The one exception is SchoolTierNorm,
// which is position-INDEPENDENT and so arrives already normalized from the boundary.
//
// FilmComposite is the upstream-weighted film signal in [0,1]; HasFilm is false until
// the film-source redesign/calibration lands, in which case the rubric applies the QB
// Data-Parity Rule and returns a neutral 1.000 film component (no penalty for absent data).
// Each breakout sub-signal carries a Has* presence flag (mirroring HasRAS/HasFilm). An
// ABSENT sub-signal must NOT be confused with a real zero: e.g. a zero breakout age would
// map to the curve ceiling (elite), a zero college share to the floor, silently corrupting
// the score. The Has* flags let the rubric apply the Data-Parity Rule per sub-signal —
// neutralizing what is absent rather than reading it as an extreme. Presence handling lives
// in the rubric (not this position-blind boundary): K, for one, legitimately has no breakout
// sub-signals at all.
type ScoutingInput struct {
	FilmComposite float64 // normalized [0,1] film composite (weights are a calibration concern, set upstream)
	HasFilm       bool    // false → Data-Parity Rule: film component is forced neutral (1.000)

	BreakoutAge    float64 // raw breakout age in years; the rubric maps it to [0,1]
	HasBreakoutAge bool    // false → the rubric treats breakout age as neutral (no lift/penalty)

	SchoolTierNorm float64 // pre-normalized school-competition tier in [0,1] (position-independent)
	HasSchoolTier  bool    // false → the rubric treats school tier as neutral

	CollegeShare    float64 // raw college production/usage share in [0,1]; the rubric maps it to its index
	HasCollegeShare bool    // false → the rubric treats college share as neutral (0 is a REAL share, not "absent")
}

// Layer4Input is what the pluggable Layer 4 receives: the per-player facts (Player) plus
// the scouting sub-signals (Scouting). The identity default reads nothing from it; B5b's
// real implementations read the sub-signals. Age trajectory reads Player.Age — there is no
// duplicate age field on Scouting.
type Layer4Input struct {
	Player   PlayerInput
	Scouting ScoutingInput
}

// Layer4Output is the scouting layer's result. Combined is the product of the three
// component multipliers and is the only field the accumulation reads; the components
// are retained for the M9a debug surface. No overall Layer 4 cap exists — component
// caps are the natural bounds (Backend_Architecture:256).
type Layer4Output struct {
	FilmEffective     float64
	FilmRaw           float64 // pre-effective normalized film input — DEBUG/sandbox only, never UI (case 3D hook)
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
