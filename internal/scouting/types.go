package scouting

import "github.com/secureprospective/TheWarRoom/internal/playerid"

// Profile is one player's scouting inputs, keyed by MFLID. It carries no position: the
// engine already knows it, and holding it would break this package's leaf status.
// Position-specific groups are pointers, nil where the position does not use them; Coverage
// is non-nil only at CB and S.
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

	OffenseFilm *OffenseFilm // QB / RB / WR / TE
	IDPFilm     *IDPFilm     // DT / DE / LB / CB / S
	Coverage    *NGSCoverage // CB / S only (hard constraint)

	SafetyRole SafetyRole
}

// OffenseFilm is the offense film signal, present at QB, RB, WR and TE.
type OffenseFilm struct {
	// Composite is in [0,1], higher is better: the mean of the position's curated Madden
	// attributes (/99), plus an FTN charting overlay (percentile-ranked, 15% discounted,
	// clamped to ±0.10) where the charting sample is large enough. A player without a
	// Madden record has a nil OffenseFilm.
	Composite float64
}

// IDPFilm is the IDP film signal, present at DT, DE, LB, CB and S.
type IDPFilm struct {
	// MaddenComposite is in [0,1]: the mean of the position's curated Madden defense
	// attributes, man and zone coverage averaged into one. A player without a Madden record
	// has a nil IDPFilm.
	MaddenComposite float64
}

// NGSCoverage is the CB/S coverage anchor. The name predates the source: nflverse has no
// defender NGS file, so it is PFR advanced-defense coverage allowed (pfrcoverage).
type NGSCoverage struct {
	CoverageMetrics float64
}
