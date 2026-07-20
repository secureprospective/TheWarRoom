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
	"github.com/secureprospective/TheWarRoom/internal/ingestion/collegedefense"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/crosswalk"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
)

// BreakoutThresholdIDP is the dominator line for the DEFENSIVE breakout scan — the
// fraction of his team's collapsed defensive production (the equal-weight average of the
// position's component shares, per collapseCollegeDefense) a defender must reach in a
// season for it to count as a breakout. It is DELIBERATELY lower than the offense's 0.20
// (BreakoutThreshold): a live sample of 25k college defensive player-seasons (2021-23)
// showed the averaged collapse shares sit far below offense yardage shares — 0.20 marked
// only ~1-3% of players. 0.12 is the "featured defender" bar (the shoulder of every
// position blend's distribution, ~top 4-7% of the population — the defensive analog to what
// 0.20 means on offense). Single cross-position line: the CB/S/LB/edge collapse blends run
// within ~0.02 of each other at the top, so one threshold avoids overfitting thin data.
// v1, Christopher-decided 2026-07-20.
const BreakoutThresholdIDP = 0.12

// BuildBreakoutAgeIDP is the DEFENSIVE clone of BuildBreakoutAge: it derives each rostered
// IDP prospect's raw breakout age — the age (in years, at the Sept 1 reference) at which he
// first crossed BreakoutThresholdIDP on the averaged defensive collapse share — and returns
// it keyed by playerid.PlayerID. It shares the offense path's structure and its birthdate
// join, reference date, season window, and Profile.BreakoutAge slot; it differs only in the
// feed (collegedefense, not collegeshare), the per-position collapse, and the threshold.
//
// DISJOINT from offense by construction: collapseCollegeDefense returns ok=false for every
// offense position (and K/FLAG), so an offense player scanned here is simply skipped. A
// roster is therefore filled by at most one breakout feed and the shared Profile.BreakoutAge
// slot never clobbers (the same disjointness proof as CollegeShare vs CollegeDefense).
//
// Inputs mirror BuildBreakoutAge: the crosswalk Map and birthdate map are SUPPLIED by the
// caller (fetched once upstream and threaded into every signal), so this assembler fetches
// only the several seasons of the defensive college feed. Any season's fetch failure
// surfaces loudly; a player-level miss (no gsis, no birthdate, an offense position, no
// season crossed the line, a non-finite/negative derived age) is ordinary and never an error.
//
// ZERO-LEAK (hard constraint): RawCollegeDefense carries only clean college defensive
// box-score counts and their within-team shares (CFBD PPA is never fetched) plus a birth
// date — nothing fantasy/projected/MFL-scored exists on it to bind.
func BuildBreakoutAgeIDP(
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
		return nil, fmt.Errorf("assembly: BuildBreakoutAgeIDP requires a non-nil *http.Client")
	}
	if pos == nil {
		return nil, fmt.Errorf("assembly: BuildBreakoutAgeIDP requires a non-nil PositionLookup")
	}
	if len(seasons) == 0 {
		return nil, fmt.Errorf("assembly: BuildBreakoutAgeIDP requires at least one season to scan")
	}

	scan := append([]int(nil), seasons...)
	sort.Ints(scan)
	byseason, err := fetchSeasonDefenseShares(ctx, client, statsBaseURL, apiKey, scan, cw.GSISForESPN)
	if err != nil {
		return nil, err
	}

	return deriveBreakoutAgesIDP(rosterMFLIDs, cw, ages, scan, byseason, pos), nil
}

// fetchSeasonDefenseShares pulls each season's defensive college feed ONCE, ascending, and
// indexes it by season so the earliest-crossing scan is a pure memory walk. ANY season's
// fetch failure surfaces loudly. collegedefense.Fetch stays a pure single-year Layer-1 call
// — the multi-season orchestration lives here in the composition leaf (mirrors the offense
// fetchSeasonShares).
func fetchSeasonDefenseShares(ctx context.Context, client *http.Client, statsBaseURL, apiKey string,
	seasons []int, resolve collegedefense.GSISResolver) (map[int]map[string]collegedefense.RawCollegeDefense, error) {
	byseason := make(map[int]map[string]collegedefense.RawCollegeDefense, len(seasons))
	for _, yr := range seasons {
		raw, err := collegedefense.Fetch(ctx, client, statsBaseURL, apiKey, yr, resolve)
		if err != nil {
			return nil, fmt.Errorf("assembly: fetch college defense season %d: %w", yr, err)
		}
		byseason[yr] = raw
	}
	return byseason, nil
}

// deriveBreakoutAgesIDP walks the roster and, per player, joins gsis + birthdate + position
// and returns the raw defensive breakout age keyed by PlayerID. The position collapse is the
// position gate: an offense/K/FLAG player yields ok=false from collapseCollegeDefense and is
// skipped. Every other miss (no gsis, no birthdate, no season crossed the line) is ordinary.
func deriveBreakoutAgesIDP(rosterMFLIDs []string, cw crosswalk.Map, ages map[string]agetrajectory.RawAge,
	seasons []int, byseason map[int]map[string]collegedefense.RawCollegeDefense, pos PositionLookup) map[playerid.PlayerID]float64 {
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
			continue // no resolved position → cannot collapse — ordinary miss
		}
		breakoutAge, ok := earliestBreakoutAgeIDP(seasons, byseason, gsis, age.BirthDate, position)
		if !ok {
			continue // offense position / never crossed / non-finite age — absent, neutral
		}
		out[pid] = breakoutAge
	}
	return out
}

// earliestBreakoutAgeIDP scans seasons (which MUST be ascending) and returns the player's
// age at the reference date of the FIRST season whose averaged defensive collapse share
// crossed BreakoutThresholdIDP. ok=false when his position has no defensive source (offense/
// K/FLAG), when he never crossed the line in the window, when a season carries no row for
// him, or when the derived age is non-finite or negative (a corrupt birthdate join — never
// emit it). Reuses the shared breakoutAgeYears (Sept 1 reference).
func earliestBreakoutAgeIDP(
	seasons []int,
	byseason map[int]map[string]collegedefense.RawCollegeDefense,
	gsis string,
	birth time.Time,
	pos domain.Position,
) (float64, bool) {
	for _, yr := range seasons {
		rc, ok := byseason[yr][gsis]
		if !ok {
			continue // no college row this season — did not play / not captured
		}
		share, ok := collapseCollegeDefense(rc, pos)
		if !ok {
			return 0, false // no defensive source for this position — constant across seasons
		}
		if share < BreakoutThresholdIDP {
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
