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
	//
	// HasRAS distinguishes a real assembled RAS from the zero value. RAS is a bare
	// float with no natural "absent" sentinel (0 could be a legitimate score), so —
	// unlike SchoolTier, which has SchoolUnset — presence needs its own flag. Once a
	// Profile can carry more than one signal (S-Phase 1: SchoolTier), "this player is
	// in the directory" no longer implies "this player has a RAS"; the consumer must
	// gate the RAS copy on HasRAS, not on directory presence.
	RAS    float64 // Relative Athletic Score
	HasRAS bool    // a real RAS was assembled (false → RAS field is the zero value, not a signal)

	// --- Breakout (college trajectory) ---
	BreakoutAge float64    // first college-production season vs. signal threshold
	SchoolTier  SchoolTier // competition tier
	// CollegeProductionShare is one player's within-team college production share,
	// collapsed to a single position-defined value upstream (the assembler picks the
	// position-appropriate raw share — S-Phase 2). Like RAS it is a bare float with no
	// natural absent sentinel (0 is a REAL share — the player produced nothing), so
	// presence needs its own flag: gate the copy on HasCollegeProductionShare, never on
	// directory presence or a zero test.
	CollegeProductionShare    float64
	HasCollegeProductionShare bool
	AgeTrajectory             float64 // age vs. position peak limit

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
//
// SOURCE-DRIFT NOTE (IDP Option-D parallel, 2026-06-26): all three fields name sources
// ELIMINATED from the automatable rubric. Recon found no clean Go-reachable feed for any
// IDP film source — The IDP Show, The IDP Guru, and Dynasty Nerds are subjective
// content/paywalled brands, not data feeds (and the rubric film component's other two
// sources, PFF and The Draft Network, were already eliminated on offense). The IDP film
// signal is redesigned around Madden defense sub-attributes (already fetched in the
// madden RawMaddenRating.Attributes map — tackle/manCoverage/zoneCoverage/hitPower/…) +
// NFLProduction + pfrcoverage. These fields are retained pending the Film-component
// redesign (a separate calibration pass — weights UNSET) and are populated by no fetcher.
// See docs/data-layer/Defense_Scouting_Source_Map.md.
type IDPFilm struct {
	IDPShow      float64 // ELIMINATED source (retained pending Film redesign) — was The IDP Show
	IDPGuru      float64 // ELIMINATED source (retained pending Film redesign) — was The IDP Guru
	DynastyNerds float64 // ELIMINATED source (retained pending Film redesign) — was Dynasty Nerds
}

// NGSCoverage holds the defender COVERAGE anchor — the analytical anchor. RESERVED FOR
// CB AND S ONLY: this group is nil at every other position. (Hard constraint: the
// coverage anchor applies at CB/S exclusively; it must not bleed to any other position.)
//
// SOURCE-DRIFT NOTE (rebind, 2026-06-26): named NGSCoverage after NFL Next Gen Stats, but
// recon found nflverse publishes NO defender NGS file (its nextgen_stats release is
// offense-only). Christopher's call: rebind the anchor onto PFR advanced-defense coverage
// (the pfrcoverage fetcher — targets, completion % / yards allowed, passer rating
// allowed). It is coverage-allowed quality, not tracking separation; the field name is
// retained, the source substitution is documented in the Defense source map.
type NGSCoverage struct {
	CoverageMetrics float64 // coverage anchor — PFR coverage-allowed (pfrcoverage), engine-normalized
}
