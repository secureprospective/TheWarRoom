package scouting

import "github.com/secureprospective/TheWarRoom/internal/playerid"

// Profile is one player's full set of scouting inputs — the unified shape
// covering all ten positions. It is keyed by MFLID and joined to a
// domain.PlayerRecord by the engine, which already knows the player's position;
// the profile deliberately does NOT carry the position (that would duplicate
// domain and break this package's leaf status).
//
// The universal core fields are flat and present at every scored position. The
// position-conditional groups are pointers: nil means the position does not use
// that group. This encodes the position boundaries structurally — most importantly
// the NGS coverage boundary (Coverage is non-nil at CB and S ONLY).
type Profile struct {
	MFLID playerid.PlayerID // canonical, validated id (RISK-003)

	// --- Universal core (Film anchors present at every scored position) ---
	// SOURCE-DRIFT NOTE (Option D, 2026-06-19): PFFGrade and DraftNetwork name sources
	// ELIMINATED from the automatable rubric (no clean source — source map §2). The
	// fields are retained pending the Film-component redesign (a separate calibration
	// pass — Offense_Scouting_Source_Map §5/§6); no fetcher populates them today.
	PFFGrade      float64 // ELIMINATED source (retained pending Film redesign) — was PFF position grade
	DraftNetwork  float64 // ELIMINATED source (retained pending Film redesign) — was The Draft Network qualitative
	MaddenFilm    float64 // Madden sub-attribute composite (Approach D); K's MAJORITY Layer-4 signal (DECISION-011)
	NFLProduction float64 // accumulating NFL production signal

	// --- RAS (athletic testing) ---
	// Excluded at K (SL-020). The engine forces QB's RAS contribution to 1.000
	// (SL-020 Low-tier); the raw value is still held here.
	RAS float64 // Relative Athletic Score

	// --- Breakout (college trajectory) ---
	BreakoutAge            float64    // first college-production season vs. signal threshold
	SchoolTier             SchoolTier // competition tier
	CollegeProductionShare float64    // monolithic, position-defined upstream (one slot)
	AgeTrajectory          float64    // age vs. position peak limit

	// --- Position-conditional groups (nil when the position does not use them) ---
	OffenseFilm *OffenseFilm // QB / RB / WR / TE
	IDPFilm     *IDPFilm     // DT / DE / LB / CB / S
	Coverage    *NGSCoverage // CB / S ONLY (hard boundary)
	TouchShare  *float64     // RB ONLY (snap-count workload share — Option D; was FantasyPros touch share)

	// --- Reserved (SL-OQ-035/036; S only; unset in v1.0) ---
	SafetyRole SafetyRole
}

// OffenseFilm holds the offense-only qualitative film sources. Present at QB, RB,
// WR, and TE; nil at every defensive position and K.
//
// SOURCE-DRIFT NOTE (Option D, 2026-06-19): both fields name sources ELIMINATED from
// the automatable rubric (no clean source — source map §2). The new offense film signal
// is FTN-charting (primary) + Madden (fallback); these fields are retained pending the
// Film-component redesign (a separate calibration pass, source map §5/§6) and are not
// populated by any fetcher today.
type OffenseFilm struct {
	RSPQualitative float64 // ELIMINATED source (retained pending Film redesign) — was Matt Waldman RSP
	SharpFootball  float64 // ELIMINATED source (retained pending Film redesign) — was Sharp Football Analysis
}

// IDPFilm holds the IDP-only film sources. Present at DT, DE, LB, CB, and S; nil at
// every offensive position and K.
type IDPFilm struct {
	IDPShow      float64 // The IDP Show
	IDPGuru      float64 // IDP Guru
	DynastyNerds float64 // Dynasty Nerds
}

// NGSCoverage holds NFL Next Gen Stats coverage metrics — the analytical anchor.
// RESERVED FOR CB AND S ONLY: this group is nil at every other position. (Hard
// constraint: the NGS coverage anchor applies at CB/S exclusively; it must not
// bleed to any other position.)
type NGSCoverage struct {
	CoverageMetrics float64 // NGS coverage/range, analytical anchor
}
