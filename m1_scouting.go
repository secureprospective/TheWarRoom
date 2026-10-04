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

// rasFetchTimeout bounds the scouting fetches; the combine and crosswalk files come from static
// CDNs, so a tight budget fails on a cold cache.
const rasFetchTimeout = 90 * time.Second

// cfbdEnvVar holds the CollegeFootballData token, read at wire time. Unset skips the CFBD signals.
const cfbdEnvVar = "CFBD_API_KEY"

// scoutProfiles is the in-progress per-player Profile map the assemblers merge into.
type scoutProfiles = map[playerid.PlayerID]scouting.Profile

// buildScoutingDirectory runs every scouting signal against the board's cached players lookup and
// the crosswalk ScoreLeague fetched, and returns the merged profiles. RAS and coverage always run. The CFBD signals (school tier, college share, breakout age) need a key: without one they are
// skipped and every player is neutral on them, but with a key a failed fetch is an error. A missing
// key and a broken fetch are different conditions.
func (a *App) buildScoutingDirectory(ctx context.Context, lk normalize.Lookup, cw crosswalk.Map) (rankings.MapScoutingDirectory, error) {
	rosterMFLIDs := collectRosterMFLIDs(a.league.Reader())
	client := &http.Client{Timeout: rasFetchTimeout, Transport: a.fetches}
	adapter := scoutLookupAdapter{lk: lk}

	profiles, err := assembly.BuildRAS(ctx, client, ras.SourceURL, cw, rosterMFLIDs, adapter)
	if err != nil {
		return rankings.MapScoutingDirectory{}, fmt.Errorf("app: build RAS scouting directory: %w", err)
	}

	if err := mergeCoverage(ctx, a.season, client, cw, rosterMFLIDs, adapter, profiles); err != nil {
		return rankings.MapScoutingDirectory{}, err
	}

	if key := strings.TrimSpace(os.Getenv(cfbdEnvVar)); key != "" {
		if err := mergeCFBDScouting(ctx, a.season, client, key, cw, rosterMFLIDs, adapter, profiles); err != nil {
			return rankings.MapScoutingDirectory{}, err
		}
	}

	return rankings.NewMapScoutingDirectory(profiles), nil
}

// mergeCFBDScouting merges the CFBD signals in order; any fetch failure is an error.
func mergeCFBDScouting(ctx context.Context, year int, client *http.Client, key string, cw crosswalk.Map,
	rosterMFLIDs []string, adapter scoutLookupAdapter, profiles scoutProfiles) error {
	// Birthdates feed both breakout scans; fetch them once.
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

// mergeCoverage adds the CB/S coverage anchor ([0,1], higher is better) from the prior
// completed season's PFR advanced defense; the current league year has no charting yet. The film
// blend is applied in rankings.applyScouting.
func mergeCoverage(ctx context.Context, year int, client *http.Client, cw crosswalk.Map,
	rosterMFLIDs []string, adapter scoutLookupAdapter, profiles scoutProfiles) error {
	coverageSeason := strconv.Itoa(year - 1)
	cov, err := assembly.BuildCoverage(ctx, client, pfrcoverage.SourceURL, cw, coverageSeason, rosterMFLIDs, adapter)
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

// mergeSchoolTier adds each rostered player's college tier.
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

// mergeCollegeShare adds the offense college production share.
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

// mergeCollegeDefense adds the defense share into the same slot. Offense and defense fill
// disjoint positions, so neither overwrites the other.
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

// mergeBreakoutAge scans the last breakoutSeasonsBack offense seasons for each rostered WR/TE/RB's
// earliest dominator crossing and derives the breakout age from the birthdate.
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

// mergeBreakoutAgeIDP is mergeBreakoutAge for CB/S/LB/DT/DE, against the lower IDP dominator
// line, into the same slot.
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

// breakoutSeasonsBack is the college window the breakout scan covers: enough for rookies and
// recent draftees. Older veterans fall outside it and are neutral. Each season costs one CFBD
// fetch pair.
const breakoutSeasonsBack = 6

// breakoutSeasons returns the ascending window ending at year.
func breakoutSeasons(year int) []int {
	out := make([]int, 0, breakoutSeasonsBack)
	for yr := year - breakoutSeasonsBack + 1; yr <= year; yr++ {
		out = append(out, yr)
	}
	return out
}

// collectRosterMFLIDs returns the set of rostered ids.
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

// scoutLookupAdapter serves assembly's position and school ports from the cached players lookup,
// so assembly never imports normalize. An unknown id or an aggregate is an ordinary miss.
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
