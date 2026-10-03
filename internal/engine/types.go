// Package engine is the scoring pipeline as pure functions: L1 hygiene, L3 age decay, L4
// scouting (a per-position dispatch), L5 cap scaling and the L6 tiebreaker. L2 arrives as
// BasePoints. It imports no store or I/O (depguard engine-is-pure); composition fills its
// inputs.
//
// Confidence scores never appear in any output that reaches the UI (a Hard Constraint).
package engine

import "github.com/secureprospective/TheWarRoom/internal/domain"

// PlayerInput is one player's scoring input.
type PlayerInput struct {
	Position   domain.Position
	BasePoints float64 // L2 output
	Age        float64
	RAS        float64 // raw Relative Athletic Score
	HasRAS     bool    // false → L1 imputes Calibration.RASFallback
	Salary     float64 // millions
	IsVeteran  bool    // L6 tenure tiebreaker (rookie loses to veteran)
}

// Calibration is the tunable parameter set: globals from the params store and per-position
// values from composition's defaults.
type Calibration struct {
	SalaryFloor float64 // salary is raised to this floor if below it
	RASFallback float64 // imputed RAS when HasRAS is false (spec fallback 5.00)
	PeakLimit   float64 // age past which decay applies
	DecayRate   float64 // annual rate, default 0.03
	// L3 cushion guard (SL-021, DT only; see ApplyCushionGuard)
	CushionRASThreshold  float64 // raw RAS at/above which the guard applies; 0 disables
	CushionDeclineFactor float64 // DT: 0.90 = 10% slower decline
	LeagueCap            float64 // same units as Salary
	ColdCeiling          float64 // salary% below this is Cold
	HotFloor             float64 // salary% above this is Hot
	ScarcityRank         int     // higher wins
}

// ScoutingInput is the raw, position-blind Layer-4 sub-signals (SchoolTierNorm excepted). Each
// has a Has* flag because absent is not zero (a zero breakout age would read as elite); the
// rubric neutralizes an absent one.
type ScoutingInput struct {
	FilmComposite float64 // [0,1], blended upstream
	HasFilm       bool

	// K film arrives as two components that the kicker rubric blends 0.60/0.40 (DECISION-011).
	// Other positions ignore them.
	MaddenFilm       float64 // [0,1] Madden kick power/accuracy
	HasMaddenFilm    bool
	NFLProduction    float64 // [0,1] NFL kicking production
	HasNFLProduction bool

	BreakoutAge    float64 // years
	HasBreakoutAge bool

	SchoolTierNorm float64 // [0,1]
	HasSchoolTier  bool

	CollegeShare    float64 // [0,1]
	HasCollegeShare bool
}

// Layer4Input is what a Layer 4 receives. Age trajectory reads Player.Age.
type Layer4Input struct {
	Player   PlayerInput
	Scouting ScoutingInput
}

// Layer4Output is the scouting result. The score reads only Combined, the product of the three
// capped components (Backend_Architecture:256).
type Layer4Output struct {
	FilmEffective     float64
	FilmRaw           float64 // pre-effective film input, for harness case 3D; never UI
	RASEffective      float64
	BreakoutEffective float64
	Combined          float64
}

// Layer4 is the per-position scouting dispatch; each position's rubric implements it.
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

// TiebreakerKey orders exact AdjustedScore ties: veteran, then RAS, then scarcity (L6).
type TiebreakerKey struct {
	IsVeteran    bool
	RAS          float64
	ScarcityRank int
}

// RanksAbove reports whether k sorts above o among equal AdjustedScores.
func (k TiebreakerKey) RanksAbove(o TiebreakerKey) bool {
	if k.IsVeteran != o.IsVeteran {
		return k.IsVeteran
	}
	if k.RAS != o.RAS {
		return k.RAS > o.RAS
	}
	return k.ScarcityRank > o.ScarcityRank
}

// Result is the final AdjustedScore plus every intermediate, for inspection.
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
