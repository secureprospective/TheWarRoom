package assembly

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"sort"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/agetrajectory"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/collegeshare"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/crosswalk"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
)

// BreakoutThreshold is the within-team production share a player must cross for a
// college season to count as his "breakout" — the classic dynasty dominator line
// (v1 provisional, Christopher 2026-07-20; mirror [[lesson_inference_vs_calibration]]
// — recalibrate against live data before treating as final). Offense-only in v1:
// WR/TE break out on receiving-yard share, RB on rushing-yard share. IDP breakout is
// a SEPARATE calibrated phase (S-Phase 4b): the averaged component-share distribution
// collapseCollegeDefense emits does not share this offense yardage threshold, so
// reusing 0.20 there would mark ~nobody a breakout — it needs its own number.
const BreakoutThreshold = 0.20

// daysPerYear converts a (reference-date − birthdate) day span into fractional years
// for the engine's per-position breakout-age curves (which anchor on half-year
// granularity). 365.25 folds in leap years without a calendar-diff dependency.
const daysPerYear = 365.25

// BuildBreakoutAge derives each rostered OFFENSE prospect's raw breakout age — the age
// (in years) at which he first crossed the college-production dominator threshold — and
// returns it keyed by playerid.PlayerID. It is the FIRST scouting signal that is not a
// single-season clone: breakout age is inherently multi-season (the EARLIEST crossing),
// so this assembler fetches several seasons of the offense college-share feed and scans
// them ascending. A player NOT in the returned map has no breakout signal (clean miss —
// the rankings side treats "absent from map" and "HasBreakoutAge=false" identically, and
// each position's §4 curve runs the Data-Parity neutral path).
//
// THE JOIN (three feeds, one gsis bridge):
//   - crosswalk (supplied by the caller): rostered MFL id → gsis, and CFBD/espn id → gsis.
//     The app fetches it once and threads the Map into every signal, so this assembler
//     never re-fetches it.
//   - agetrajectory (supplied by the caller): gsis → raw birth DATE (age is an as-of
//     derivation — zero-leak; the fetcher never emits an age). Also fetched once upstream.
//   - collegeshare (fetched PER SEASON, here): gsis-keyed within-team production shares.
//
// Then, per rostered player: pick the position-appropriate within-team share (WR/TE →
// receiving-yard share, RB → rushing-yard share; every other position has NO offense
// breakout source in v1 — QB's passing is never fetched (zero-leak), K has no breakout
// framework, and defense is the separate S-Phase 4b feed), scan seasons earliest-first
// for the first crossing of BreakoutThreshold, and compute age = (Sept 1 of that season −
// birthdate) in years. The engine consumes the RAW age and applies the §4 curve — the
// same raw-in / normalize-in-engine posture as RAS and CollegeShare.
//
// The multi-season loop lives HERE (assembly), not in the fetcher: collegeshare.Fetch
// stays a pure single-year Layer-1 call, and this composition leaf orchestrates the
// several calls — keeping Layer 1 unaware of the breakout concept.
//
// Failures: a genuine fetch failure on ANY scanned season's college feed (network/HTTP/
// parse or a feed that resolved zero records) is surfaced loudly, matching the sibling
// assemblers — a breakout-less league should be visible, not silently neutral. (The
// crosswalk and birthdate fetch failures now surface upstream at the caller's single fetch.) A player-level miss (no gsis, no birthdate, a non-offense
// position, no season crossed the line, a non-finite/negative derived age) is ordinary
// and never an error.
//
// ZERO-LEAK (hard constraint): every feed carries only clean college box-score shares
// and a birth date — no fantasy points / projected volume / MFL scoring / CFBD PPA
// exists on any of them to bind.
func BuildBreakoutAge(
	ctx context.Context,
	client *http.Client,
	statsBaseURL, apiKey string,
	cw crosswalk.Map,
	ages map[string]agetrajectory.RawAge,
	seasons []int,
	rosterMFLIDs []string,
	pos PositionLookup,
) (map[playerid.PlayerID]float64, error) {
	if client == nil {
		return nil, fmt.Errorf("assembly: BuildBreakoutAge requires a non-nil *http.Client")
	}
	if pos == nil {
		return nil, fmt.Errorf("assembly: BuildBreakoutAge requires a non-nil PositionLookup")
	}
	if len(seasons) == 0 {
		return nil, fmt.Errorf("assembly: BuildBreakoutAge requires at least one season to scan")
	}

	scan := append([]int(nil), seasons...)
	sort.Ints(scan)
	byseason, err := fetchSeasonShares(ctx, client, statsBaseURL, apiKey, scan, cw.GSISForESPN)
	if err != nil {
		return nil, err
	}

	return deriveBreakoutAges(rosterMFLIDs, cw, ages, scan, byseason, pos), nil
}

// fetchSeasonShares pulls each season's offense college-share feed ONCE, ascending, and
// indexes it by season so the earliest-crossing scan is a pure memory walk (not repeated
// network I/O). ANY season's fetch failure surfaces loudly (a signal-less league must be
// visible). collegeshare.Fetch stays a pure single-year Layer-1 call — the multi-season
// orchestration lives here in the composition leaf.
func fetchSeasonShares(ctx context.Context, client *http.Client, statsBaseURL, apiKey string,
	seasons []int, resolve collegeshare.GSISResolver) (map[int]map[string]collegeshare.RawCollegeShare, error) {
	byseason := make(map[int]map[string]collegeshare.RawCollegeShare, len(seasons))
	for _, yr := range seasons {
		raw, err := collegeshare.Fetch(ctx, client, statsBaseURL, apiKey, yr, resolve)
		if err != nil {
			return nil, fmt.Errorf("assembly: fetch college share season %d: %w", yr, err)
		}
		byseason[yr] = raw
	}
	return byseason, nil
}

// deriveBreakoutAges walks the roster and, per player, joins gsis + birthdate + position
// and returns the raw breakout age keyed by PlayerID. Every miss (no gsis, no birthdate,
// a non-offense position, no season crossed the line) is ordinary — the player is simply
// absent from the returned map.
func deriveBreakoutAges(rosterMFLIDs []string, cw crosswalk.Map, ages map[string]agetrajectory.RawAge,
	seasons []int, byseason map[int]map[string]collegeshare.RawCollegeShare, pos PositionLookup) map[playerid.PlayerID]float64 {
	out := make(map[playerid.PlayerID]float64, len(rosterMFLIDs))
	for _, mfl := range rosterMFLIDs {
		pid, err := playerid.New(mfl)
		if err != nil {
			continue // malformed — an upstream layer should have caught this; skip
		}
		gsis, ok := cw.Lookup(pid)
		if !ok {
			continue // no MFL→gsis mapping — ordinary miss
		}
		age, ok := ages[gsis]
		if !ok {
			continue // no birthdate — cannot derive a breakout age (ordinary miss)
		}
		position, ok := pos.Position(mfl)
		if !ok {
			continue // no resolved position → cannot pick a share — ordinary miss
		}
		share, ok := breakoutShare(position)
		if !ok {
			continue // position has no offense breakout source (QB/K/def/FLAG) — neutral
		}
		breakoutAge, ok := earliestBreakoutAge(seasons, byseason, gsis, age.BirthDate, share)
		if !ok {
			continue // never crossed the line / non-finite age — absent, neutral
		}
		out[pid] = breakoutAge
	}
	return out
}

// shareSelector picks the within-team production share whose crossing defines a
// breakout for a given position — an offense yardage share, not the engine's job.
type shareSelector func(collegeshare.RawCollegeShare) float64

func receivingShare(rc collegeshare.RawCollegeShare) float64 { return rc.ReceivingYardShare }
func rushingShare(rc collegeshare.RawCollegeShare) float64   { return rc.RushingYardShare }

// breakoutShare returns the position-appropriate share selector and ok=false for a
// position with no offense breakout source in v1. WR/TE break out on receiving-yard
// share; RB on rushing-yard share. QB has no share in this feed (passing is never
// fetched — zero-leak), K has no breakout framework, and every defensive position
// draws from the separate IDP feed (S-Phase 4b) — all absent here. FLAG is unclassified.
func breakoutShare(pos domain.Position) (shareSelector, bool) {
	switch pos {
	case domain.PosWR, domain.PosTE:
		return receivingShare, true
	case domain.PosRB:
		return rushingShare, true
	case domain.PosQB, domain.PosK, domain.PosCB, domain.PosS,
		domain.PosLB, domain.PosDT, domain.PosDE, domain.PosFlag:
		return nil, false
	}
	return nil, false
}

// earliestBreakoutAge scans seasons (which MUST be ascending) and returns the player's
// age at the reference date of the FIRST season whose selected within-team share crossed
// BreakoutThreshold. ok=false when he never crossed it in the scanned window, when a
// season carries no row for him, or when the derived age is non-finite or negative (a
// birthdate AFTER the season is a corrupt join, not a real breakout age — never emit it).
func earliestBreakoutAge(
	seasons []int,
	byseason map[int]map[string]collegeshare.RawCollegeShare,
	gsis string,
	birth time.Time,
	share shareSelector,
) (float64, bool) {
	for _, yr := range seasons {
		rc, ok := byseason[yr][gsis]
		if !ok {
			continue // no college row this season — did not play / not captured
		}
		if share(rc) < BreakoutThreshold {
			continue // produced, but below the breakout line
		}
		age := breakoutAgeYears(birth, yr)
		if math.IsNaN(age) || math.IsInf(age, 0) || age < 0 {
			return 0, false
		}
		return age, true
	}
	return 0, false
}

// breakoutAgeYears is the player's age in fractional years at the season reference date
// — Sept 1 of the season year (v1, Christopher 2026-07-20), which approximates the
// college season's start. The engine's per-position curves anchor on half-year steps.
func breakoutAgeYears(birth time.Time, season int) float64 {
	ref := time.Date(season, time.September, 1, 0, 0, 0, 0, time.UTC)
	return ref.Sub(birth).Hours() / 24 / daysPerYear
}
