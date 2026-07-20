// Package scouting holds the leaf scouting-input types the engine's Layer 4
// (Film / RAS / Breakout) consumes. It is the single unified field set every
// scouting fetcher (B2b-Fetch-*) populates and the engine reads — one shape for
// all ten positions, with position-conditional groups present only where a
// position uses them.
//
// WIRING STATUS (updated 2026-07-20, S-Phase 3): the scouting data-integration phase is
// LANDING signal by signal through internal/scouting/assembly + m1_app.buildScoutingDirectory.
// WIRED (fetch → crosswalk join → Profile → engine Layer 4): RAS (S-Phase 0), SchoolTier
// (S-Phase 1), offense CollegeProductionShare (S-Phase 2, collegeshare), and IDP
// CollegeProductionShare (S-Phase 3, collegedefense). STILL-DEFERRED scaffolding (shape +
// fetchers exist, unit-tested, NOT yet switched on): BreakoutAge/AgeTrajectory, the Film
// components (OffenseFilm/IDPFilm via veteranfilm/madden/nflproduction/touchshare) and
// NGSCoverage (pfrcoverage/SafetyRole) + K film (kicking) — these stay Data-Parity NEUTRAL,
// and the FILM reweight in particular is a separate decision-gated CALIBRATION pass (do NOT
// ship blind film weights). For an un-wired sub-signal the rubric runs the identity/neutral
// path. The plan (Option D hybrid, source maps in
// docs/data-layer/{Offense,Defense}_Scouting_Source_Map.md) and the retained-field
// rationale (the SOURCE-DRIFT notes on each type in types.go) are the answer to
// "why is this in main?" — kept by ruling 2026-07-20 rather than carved out. See
// SYSTEM_MAP.md → "Deferred / unwired scaffolding".
//
// It is a LEAF — it imports only internal/playerid (the shared id value type) and
// nothing else internal. It does not fetch (fetchers populate it) and it does not
// score (the engine consumes it). All raw→typed mapping lives in the fetchers.
//
// ZERO-LEAK INVARIANT (hard constraint): no field in this package may hold or
// reference fantasy points, projected volume, MFL scoring config, or
// format-dependent volume stats. Every field is a film, athletic, or college
// signal. This is the entire reason scouting is its own schema and not roster
// fields — Layer 2/Layer 4 must never see each other's signals.
//
// SCALE CONVENTION: every numeric input is the source's normalized sub-signal in
// [0,1]. For a position that uses a core field, 0 is a legitimate floor grade; the
// fetcher always populates it. K is sparse but NOT empty: its Layer-4 valuation is
// driven by MaddenFilm as the MAJORITY weight (DECISION-011, 2026-06-19), with
// NFLProduction secondary. RAS is still excluded at K (SL-020); no film/breakout
// sub-signals are scouted for K. (DECISION-011 reverses the K rubric's prior
// structural-exclusion model and AD-10 — see docs/scoring-engine/Scouting_Schema.md.)
package scouting

// SchoolTier is the normalized college-competition tier feeding the Breakout
// component. SchoolUnset marks a profile whose tier the fetcher has not set.
type SchoolTier string

const (
	SchoolUnset       SchoolTier = ""
	SchoolPowerFour   SchoolTier = "POWER_FOUR"
	SchoolGroupOfFive SchoolTier = "GROUP_OF_FIVE"
	SchoolFCS         SchoolTier = "FCS"
	SchoolNonFCS      SchoolTier = "NON_FCS"
)

// SafetyRole is RESERVED for SL-OQ-035/036 (safety box/deep branching vs.
// role-conditional weighting — both resolve together, post-live-data). It is
// deliberately reserved now (AD-16) and left SafetyRoleUnset for v1.0: S is
// scored monolithically until the empirical decision lands. College production
// share stays a single monolithic field regardless of which way it resolves —
// this enum is the only structure reserved. Applies to S only.
type SafetyRole string

const (
	SafetyRoleUnset  SafetyRole = "" // v1.0 default — S scored monolithically
	SafetyRoleBox    SafetyRole = "box"
	SafetyRoleDeep   SafetyRole = "deep"
	SafetyRoleHybrid SafetyRole = "hybrid"
)
