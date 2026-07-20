package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/agetrajectory"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/collegedefense"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/collegeshare"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/crosswalk"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/pfrcoverage"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/ras"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/schooltier"
	"github.com/secureprospective/TheWarRoom/internal/normalize"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
	"github.com/secureprospective/TheWarRoom/internal/rankings"
	"github.com/secureprospective/TheWarRoom/internal/scouting"
	"github.com/secureprospective/TheWarRoom/internal/scouting/assembly"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

// rasFetchTimeout bounds the scouting fetches. The RAS combine feed is ~2 MB and the
// crosswalk ~1 MB, both off the MFL transport on static CDNs; a tight timeout would
// fail on a cold cache.
const rasFetchTimeout = 90 * time.Second

// cfbdEnvVar is the environment variable carrying the CollegeFootballData bearer token
// — the credential the CFBD-sourced signals (SchoolTier / CollegeShare / CollegeDefense
// / BreakoutAge) need. It is read at wire time, never stored in the repo. When it is
// UNSET those signals are skipped rather than failing the board (see below).
const cfbdEnvVar = "CFBD_API_KEY"

// scoutProfiles is the in-progress per-player Profile map the assemblers merge into.
type scoutProfiles = map[playerid.PlayerID]scouting.Profile

// buildScoutingDirectory wires the scouting pipeline and wraps the merged result as a
// rankings ScoutingDirectory. Every signal rides on the SAME cached normalize.Lookup
// the orchestrator already built (position + college are players-DB facts, never
// re-fetched) via one scoutLookupAdapter.
//
//   - S-Phase 0 (RAS): fetch combine + crosswalk, compute per-position RAS-equivalent
//     (§3). A fetch failure surfaces loudly.
//   - S-Phase 1-4 (SchoolTier / CollegeShare / CollegeDefense / BreakoutAge): all need a
//     CFBD API key. When the key is UNCONFIGURED they are SKIPPED wholesale (every player
//     → Data-Parity neutral) rather than failing the board, so an environment without the
//     key still ranks on RAS. A key-PRESENT fetch failure still surfaces loudly — a
//     missing key and a broken fetch are different conditions.
func (a *App) buildScoutingDirectory(ctx context.Context, lk normalize.Lookup) (rankings.MapScoutingDirectory, error) {
	rosterMFLIDs := collectRosterMFLIDs(a.state.Reader())
	client := &http.Client{Timeout: rasFetchTimeout}
	adapter := scoutLookupAdapter{lk: lk}

	// Fetch the MFL→gsis crosswalk ONCE here and thread the resolved Map down into every
	// signal. It was previously re-fetched inside each Build* (four ~1 MB pulls per Score
	// League); a single fetch removes that redundancy and is the one fail-loud crosswalk
	// boundary. Birthdates (breakout-only) are fetched once inside mergeCFBDScouting.
	cw, err := crosswalk.Fetch(ctx, client, crosswalk.SourceURL)
	if err != nil {
		return rankings.MapScoutingDirectory{}, fmt.Errorf("app: fetch scouting crosswalk: %w", err)
	}

	profiles, err := assembly.BuildRAS(ctx, client, ras.SourceURL, cw, rosterMFLIDs, adapter)
	if err != nil {
		return rankings.MapScoutingDirectory{}, fmt.Errorf("app: build RAS scouting directory: %w", err)
	}

	// Coverage (S-Phase FILM C-4 step 1): the CB/S coverage anchor rides on nflverse PFR
	// advanced-defense (NO CFBD key), so it merges here alongside RAS — unconditionally, and
	// before the CFBD-gated block. A fetch failure surfaces loudly (a coverage-less league
	// must be visible), matching BuildRAS.
	if err := mergeCoverage(ctx, client, cw, rosterMFLIDs, adapter, profiles); err != nil {
		return rankings.MapScoutingDirectory{}, err
	}

	if key := strings.TrimSpace(os.Getenv(cfbdEnvVar)); key != "" {
		if err := mergeCFBDScouting(ctx, client, key, cw, rosterMFLIDs, adapter, profiles); err != nil {
			return rankings.MapScoutingDirectory{}, err
		}
	}

	return rankings.NewMapScoutingDirectory(profiles), nil
}

// mergeCFBDScouting merges every CFBD-sourced signal (S-Phase 1-4) into profiles, in
// order. All share one key + season year; each signal's fetch failure surfaces loudly
// (a signal-less league must be visible, not silently neutral).
func mergeCFBDScouting(ctx context.Context, client *http.Client, key string, cw crosswalk.Map,
	rosterMFLIDs []string, adapter scoutLookupAdapter, profiles scoutProfiles) error {
	year, err := strconv.Atoi(ingestion.SeasonYear)
	if err != nil {
		return fmt.Errorf("app: season year %q not numeric: %w", ingestion.SeasonYear, err)
	}
	// Birthdates feed only the breakout scan; fetch them once here (behind the CFBD-key
	// gate) rather than inside BuildBreakoutAge, so a second breakout thread (IDP, S-Phase
	// 4b) reuses the same pull instead of fetching twice.
	ages, err := agetrajectory.Fetch(ctx, client, agetrajectory.SourceURL)
	if err != nil {
		return fmt.Errorf("app: fetch scouting birthdates: %w", err)
	}
	if err := mergeSchoolTier(ctx, client, key, year, rosterMFLIDs, adapter, profiles); err != nil {
		return err
	}
	if err := mergeCollegeShare(ctx, client, key, cw, year, rosterMFLIDs, adapter, profiles); err != nil {
		return err
	}
	if err := mergeCollegeDefense(ctx, client, key, cw, year, rosterMFLIDs, adapter, profiles); err != nil {
		return err
	}
	if err := mergeBreakoutAge(ctx, client, key, cw, ages, year, rosterMFLIDs, adapter, profiles); err != nil {
		return err
	}
	return mergeBreakoutAgeIDP(ctx, client, key, cw, ages, year, rosterMFLIDs, adapter, profiles)
}

// mergeCoverage (S-Phase FILM C-4 step 1): join each rostered CB/S id → gsis → the
// engine-ready coverage anchor ([0,1], higher=better — the leaf inverts PFR passer-rating-
// allowed) into a fresh scouting.NGSCoverage group. A coverage-only player gets a fresh
// Profile carrying just Coverage; the film-composite blend (K4 = 0.20 of the film budget)
// is applied downstream in rankings.applyScouting, not here. The season is the same value
// the CFBD signals use (ingestion.SeasonYear).
func mergeCoverage(ctx context.Context, client *http.Client, cw crosswalk.Map,
	rosterMFLIDs []string, adapter scoutLookupAdapter, profiles scoutProfiles) error {
	// pfrcoverage keys the season as a string (the CSV's raw column); pass SeasonYear
	// straight through — no numeric round-trip needed, unlike the CFBD int-year fetchers.
	cov, err := assembly.BuildCoverage(ctx, client, pfrcoverage.SourceURL, cw, ingestion.SeasonYear, rosterMFLIDs, adapter)
	if err != nil {
		return fmt.Errorf("app: build coverage scouting directory: %w", err)
	}
	for pid, norm := range cov {
		p := profiles[pid]
		p.MFLID = pid
		p.Coverage = assembly.CoverageGroup(norm)
		profiles[pid] = p
	}
	return nil
}

// mergeSchoolTier (S-Phase 1): join each rostered id → college → tier. A tier-only
// player (school known, no combine) gets a fresh Profile carrying just SchoolTier — its
// HasRAS stays false, so the RAS rubric still imputes the fallback.
func mergeSchoolTier(ctx context.Context, client *http.Client, key string, year int,
	rosterMFLIDs []string, adapter scoutLookupAdapter, profiles scoutProfiles) error {
	tiers, err := assembly.BuildSchoolTier(ctx, client, schooltier.TeamsURL, key, year, rosterMFLIDs, adapter)
	if err != nil {
		return fmt.Errorf("app: build school-tier scouting directory: %w", err)
	}
	for pid, tier := range tiers {
		p := profiles[pid]
		p.MFLID = pid
		p.SchoolTier = tier
		profiles[pid] = p
	}
	return nil
}

// mergeCollegeShare (S-Phase 2): join each rostered id → gsis → collapsed position-defined
// OFFENSE production share. A share-only player gets a fresh Profile carrying just the share.
func mergeCollegeShare(ctx context.Context, client *http.Client, key string, cw crosswalk.Map, year int,
	rosterMFLIDs []string, adapter scoutLookupAdapter, profiles scoutProfiles) error {
	shares, err := assembly.BuildCollegeShare(ctx, client, collegeshare.SeasonStatsURL, key, cw, year, rosterMFLIDs, adapter)
	if err != nil {
		return fmt.Errorf("app: build college-share scouting directory: %w", err)
	}
	for pid, share := range shares {
		p := profiles[pid]
		p.MFLID = pid
		p.CollegeProductionShare = share
		p.HasCollegeProductionShare = true
		profiles[pid] = p
	}
	return nil
}

// mergeCollegeDefense (S-Phase 3 / IDP): join each rostered defensive id → gsis →
// position-averaged share into the SAME CollegeProductionShare slot. Offense and defense
// populate DISJOINT positions — each collapse returns absent for the other side — so a
// player is filled by at most one feed and the slot never clobbers.
func mergeCollegeDefense(ctx context.Context, client *http.Client, key string, cw crosswalk.Map, year int,
	rosterMFLIDs []string, adapter scoutLookupAdapter, profiles scoutProfiles) error {
	defShares, err := assembly.BuildCollegeDefense(ctx, client, collegedefense.SeasonStatsURL, key, cw, year, rosterMFLIDs, adapter)
	if err != nil {
		return fmt.Errorf("app: build college-defense scouting directory: %w", err)
	}
	for pid, share := range defShares {
		p := profiles[pid]
		p.MFLID = pid
		p.CollegeProductionShare = share
		p.HasCollegeProductionShare = true
		profiles[pid] = p
	}
	return nil
}

// mergeBreakoutAge (S-Phase 4, offense v1): the first MULTI-season signal. Scan the last
// breakoutSeasonsBack seasons of the offense college-share feed for each rostered WR/TE/RB's
// earliest dominator crossing, join a birthdate, and derive the raw breakout age. A
// breakout-only player gets a fresh Profile carrying just the age; HasBreakoutAge (not a
// zero test — a young age is a real signal) gates the copy downstream.
func mergeBreakoutAge(ctx context.Context, client *http.Client, key string, cw crosswalk.Map,
	ages map[string]agetrajectory.RawAge, year int,
	rosterMFLIDs []string, adapter scoutLookupAdapter, profiles scoutProfiles) error {
	breakouts, err := assembly.BuildBreakoutAge(ctx, client, collegeshare.SeasonStatsURL, key,
		cw, ages, breakoutSeasons(year), rosterMFLIDs, adapter)
	if err != nil {
		return fmt.Errorf("app: build breakout-age scouting directory: %w", err)
	}
	for pid, age := range breakouts {
		p := profiles[pid]
		p.MFLID = pid
		p.BreakoutAge = age
		p.HasBreakoutAge = true
		profiles[pid] = p
	}
	return nil
}

// mergeBreakoutAgeIDP (S-Phase 4b, defense): the DEFENSIVE clone of mergeBreakoutAge. Scan
// the last breakoutSeasonsBack seasons of the defensive college feed for each rostered
// CB/S/LB/DT/DE's earliest crossing of the (lower, calibrated) IDP dominator line, join the
// SAME already-fetched birthdate + crosswalk maps, and derive the raw breakout age into the
// SAME Profile.BreakoutAge slot. Offense and defense populate DISJOINT positions (the
// defensive collapse returns absent for offense positions), so the slot never clobbers.
func mergeBreakoutAgeIDP(ctx context.Context, client *http.Client, key string, cw crosswalk.Map,
	ages map[string]agetrajectory.RawAge, year int,
	rosterMFLIDs []string, adapter scoutLookupAdapter, profiles scoutProfiles) error {
	breakouts, err := assembly.BuildBreakoutAgeIDP(ctx, client, collegedefense.SeasonStatsURL, key,
		cw, ages, breakoutSeasons(year), rosterMFLIDs, adapter)
	if err != nil {
		return fmt.Errorf("app: build IDP breakout-age scouting directory: %w", err)
	}
	for pid, age := range breakouts {
		p := profiles[pid]
		p.MFLID = pid
		p.BreakoutAge = age
		p.HasBreakoutAge = true
		profiles[pid] = p
	}
	return nil
}

// breakoutSeasonsBack is how many college seasons (ending at the scoring year) the
// breakout-age scan pulls — wide enough to cover incoming rookies and recent draftees
// (the signal's live targets), whose earliest college season sits a few years back.
// Older veterans fall outside the window → absent breakout → neutral (their other
// scouting signals carry them). Tunable; each season is one extra CFBD fetch pair.
const breakoutSeasonsBack = 6

// breakoutSeasons returns the ascending season window [year-(N-1) .. year] the breakout
// scan covers. Kept tiny and pure so the window is one obvious, testable place.
func breakoutSeasons(year int) []int {
	out := make([]int, 0, breakoutSeasonsBack)
	for yr := year - breakoutSeasonsBack + 1; yr <= year; yr++ {
		out = append(out, yr)
	}
	return out
}

// collectRosterMFLIDs walks every franchise roster and returns the set of rostered MFL
// ids — the population the RAS pipeline scores against. Order-independent: BuildRAS treats
// the slice as a set, and the cohort math is order-independent (see ras_math.go).
// Duplicates across franchises are deduped defensively; in practice a player is on
// exactly one roster (state's invariant).
func collectRosterMFLIDs(st state.Reader) []string {
	seen := make(map[string]struct{})
	out := make([]string, 0, 64)
	for _, fid := range st.Franchises() {
		roster, ok := st.Roster(fid)
		if !ok {
			continue
		}
		for _, p := range roster {
			if _, dup := seen[p.MFLID]; dup {
				continue
			}
			seen[p.MFLID] = struct{}{}
			out = append(out, p.MFLID)
		}
	}
	return out
}

// scoutLookupAdapter adapts the existing normalize.Lookup to assembly's narrow scouting
// ports — PositionLookup (RAS) and SchoolLookup (SchoolTier) — so every signal reads the
// SAME cached players-DB facts and the assembly package stays free of any normalize
// import. Facts returns ok=false on an unknown id OR an aggregate (collapsed there), which
// both ports treat as an ordinary miss.
type scoutLookupAdapter struct {
	lk normalize.Lookup
}

func (a scoutLookupAdapter) Position(mflID string) (domain.Position, bool) {
	facts, ok := a.lk.Facts(mflID)
	if !ok {
		return "", false
	}
	return facts.Position, true
}

// College returns the player's raw MFL college name. ok=false when the player is
// unknown/aggregate OR MFL carries no college for them (team-D rows, some deep database
// players) — the school-tier join treats an absent college as a neutral miss.
func (a scoutLookupAdapter) College(mflID string) (string, bool) {
	facts, ok := a.lk.Facts(mflID)
	if !ok || facts.College == "" {
		return "", false
	}
	return facts.College, true
}
