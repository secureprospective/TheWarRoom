package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/leagueschedule"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

// This file is the LEAGUE-SCHEDULE seam: the fantasy league's own weekly matchups (who plays
// whom, real franchise ids), read-only and MFL-sourced — distinct from the commissioner
// calendar (calendar_events), which is a commissioner-AUTHORED intent ledger. Mirrors the M2
// degradation seam (m2_standings_source.go) exactly: live fetch → cache on success → fall back
// to the last-known-good cache on failure, reported through the shared Freshness contract
// (freshness.go) rather than blanking the board on an MFL outage.

// leagueScheduleOrCache is the schedule degradation seam, structured identically to
// standingsOrCache: it tries the live MFL fetch first; on success it refreshes the cache and
// reports live. On failure it falls back to the last-known-good cached payload and reports
// stale, carrying the fetch error as the note. Only a failed fetch with NOTHING cached returns
// an error.
func (a *App) leagueScheduleOrCache(ctx context.Context) ([]leagueschedule.RawScheduleWeek, Freshness, error) {
	weeks, ferr := leagueschedule.Fetch(ctx, a.mflClient, ingestion.SeasonYear, ingestion.LeagueID)
	if ferr == nil {
		now := time.Now()
		fresh := liveFreshness(now)
		payload, merr := json.Marshal(weeks)
		if merr != nil {
			fresh.Note = fmt.Sprintf("schedule not cached (encode failed): %v", merr)
			return weeks, fresh, nil
		}
		// Fresh context on the write, same reason as standingsOrCache: a fetch that succeeded
		// on the last of its budget would leave ctx nearly expired, and the cache would
		// silently never populate — invisible until the next outage needed it.
		wCtx, wCancel := context.WithTimeout(a.fallbackParent(), cacheReadTimeout)
		defer wCancel()
		//nolint:contextcheck // NOT inheriting ctx is the whole point — see fallbackParent.
		if perr := a.state.PutLeagueSchedule(wCtx, string(payload), now); perr != nil {
			fresh.Note = fmt.Sprintf("schedule not cached: %v", perr)
		}
		return weeks, fresh, nil
	}

	// The fallback read must not inherit the fetch's deadline, for the same reason
	// standingsOrCache's fallback doesn't: a timed-out fetch leaves ctx already expired, and
	// reusing it would fail the local SQLite read instantly while good cached data sits there.
	fbCtx, fbCancel := context.WithTimeout(a.fallbackParent(), cacheReadTimeout)
	defer fbCancel()

	//nolint:contextcheck // Deliberately NOT the caller's context — see fallbackParent.
	payload, at, cerr := a.state.CachedLeagueSchedule(fbCtx)
	if cerr != nil {
		if errors.Is(cerr, state.ErrNoCachedLeagueSchedule) {
			return nil, Freshness{}, fmt.Errorf("schedule fetch failed with no cached fallback: %w", ferr)
		}
		return nil, Freshness{}, fmt.Errorf(
			"schedule fetch failed (%w) and the local cache is unreadable: %v", ferr, cerr)
	}
	var cached []leagueschedule.RawScheduleWeek
	if err := json.Unmarshal([]byte(payload), &cached); err != nil {
		return nil, Freshness{}, fmt.Errorf(
			"live fetch failed (%v) and cached schedule could not be decoded: %w", ferr, err)
	}
	if len(cached) == 0 {
		return nil, Freshness{}, fmt.Errorf("live fetch failed (%v) and cached schedule was empty", ferr)
	}
	return cached, staleFreshness(at, ferr), nil
}

// ScheduleMatchupDTO is one matchup as it crosses the IPC boundary: both sides' franchise ids
// resolved to real display names via the rulebook's franchise directory (id fallback when a
// name is unset, matching the m2service/transactions_app convention). Scores are raw strings,
// empty until the week has played.
type ScheduleMatchupDTO struct {
	HomeFranchiseID   string `json:"homeFranchiseID"`
	HomeFranchiseName string `json:"homeFranchiseName"`
	HomeScore         string `json:"homeScore"`
	AwayFranchiseID   string `json:"awayFranchiseID"`
	AwayFranchiseName string `json:"awayFranchiseName"`
	AwayScore         string `json:"awayScore"`
}

// ScheduleWeekDTO is one week's full slate of matchups, Week parsed to an int for the
// frontend's sort/group convenience (the fetcher keeps it a raw string; parsing happens here,
// at the IPC boundary, not upstream).
type ScheduleWeekDTO struct {
	Week     int                  `json:"week"`
	Matchups []ScheduleMatchupDTO `json:"matchups"`
}

// LeagueScheduleResult is a read of the league's full-season matchup schedule. Weeks is never
// null on success (Wails marshals a nil Go slice to JSON null; here it is always non-nil).
type LeagueScheduleResult struct {
	OK        bool              `json:"ok"`
	Weeks     []ScheduleWeekDTO `json:"weeks"`
	Freshness Freshness         `json:"freshness"`
	Detail    string            `json:"detail"`
}

// GetLeagueSchedule is the IPC entry point for the Commissioner Calendar's read-only Schedule
// pane. It is display-only: unlike GetCalendarEvents, nothing here is a commissioner-authored
// transaction, so there is no stage/preview/confirm path — just a read with honest degradation.
func (a *App) GetLeagueSchedule() LeagueScheduleResult {
	if err := a.m1Ready(); err != nil {
		return LeagueScheduleResult{Detail: err.Error()}
	}
	ctx, cancel := context.WithTimeout(a.ctx, m2Timeout)
	defer cancel()

	weeks, fresh, err := a.leagueScheduleOrCache(ctx)
	if err != nil {
		return LeagueScheduleResult{Detail: fmt.Sprintf("league schedule: %v", err)}
	}

	names := a.rulebook.FranchiseNames()
	out := make([]ScheduleWeekDTO, 0, len(weeks))
	for _, w := range weeks {
		wk, perr := strconv.Atoi(w.Week)
		if perr != nil {
			return LeagueScheduleResult{Detail: fmt.Sprintf("league schedule: week %q is not a number: %v", w.Week, perr)}
		}
		matchups := make([]ScheduleMatchupDTO, 0, len(w.Matchups))
		for _, m := range w.Matchups {
			home, away := m.Franchises[0], m.Franchises[1]
			if home.IsHome == "0" {
				home, away = away, home
			}
			matchups = append(matchups, ScheduleMatchupDTO{
				HomeFranchiseID:   home.FranchiseID,
				HomeFranchiseName: franchiseDisplayName(names, home.FranchiseID),
				HomeScore:         home.Score,
				AwayFranchiseID:   away.FranchiseID,
				AwayFranchiseName: franchiseDisplayName(names, away.FranchiseID),
				AwayScore:         away.Score,
			})
		}
		out = append(out, ScheduleWeekDTO{Week: wk, Matchups: matchups})
	}

	return LeagueScheduleResult{OK: true, Weeks: out, Freshness: fresh}
}

// franchiseDisplayName resolves a franchise id to its league name, falling back to the id
// itself when the rulebook has no name on file — mirrors internal/m2service's helper of the
// same name (unexported to each package; no shared dependency to introduce for one line).
func franchiseDisplayName(names map[string]string, fid string) string {
	if n, ok := names[fid]; ok && n != "" {
		return n
	}
	return fid
}
