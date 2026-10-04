package scouting

import "github.com/secureprospective/TheWarRoom/internal/playerid"

// Profile is one player's scouting inputs, keyed by MFLID. Position-specific groups are
// pointers, nil where the position does not use them; Coverage is non-nil only at CB and S.
type Profile struct {
	MFLID playerid.PlayerID

	// RAS is excluded at K and forced to 1.000 at QB (SL-020); the raw value is still held.
	RAS    float64
	HasRAS bool

	// BreakoutAge is the age in years at the first college season whose within-team share
	// crossed the breakout threshold; each rubric curves it per position.
	BreakoutAge    float64
	HasBreakoutAge bool
	SchoolTier     SchoolTier

	// CollegeProductionShare is the within-team college share, collapsed to the
	// position's measure by the assembler.
	CollegeProductionShare    float64
	HasCollegeProductionShare bool

	Coverage *NGSCoverage // CB / S only (hard constraint)

	SafetyRole SafetyRole
}

// NGSCoverage is the CB/S coverage anchor. The name predates the source: nflverse has no
// defender NGS file, so it is PFR advanced-defense coverage allowed (pfrcoverage).
type NGSCoverage struct {
	CoverageMetrics float64
}
