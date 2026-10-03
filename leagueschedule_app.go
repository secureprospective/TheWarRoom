package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/leagueschedule"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

// leagueScheduleOrCache returns the league's MFL matchup schedule, live or from the last good
// cache, the same way standingsOrCache does.
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
		wCtx, wCancel := context.WithTimeout(a.fallbackParent(), cacheReadTimeout)
		defer wCancel()
		//nolint:contextcheck // NOT inheriting ctx is the whole point — see fallbackParent.
		if perr := a.state.PutLeagueSchedule(wCtx, string(payload), now); perr != nil {
			fresh.Note = fmt.Sprintf("schedule not cached: %v", perr)
		}
		return weeks, fresh, nil
	}

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

// ScheduleMatchupDTO is one matchup with both franchises' display names. Scores are empty until
// the week is played.
type ScheduleMatchupDTO struct {
	HomeFranchiseID   string `json:"homeFranchiseID"`
	HomeFranchiseName string `json:"homeFranchiseName"`
	HomeScore         string `json:"homeScore"`
	AwayFranchiseID   string `json:"awayFranchiseID"`
	AwayFranchiseName string `json:"awayFranchiseName"`
	AwayScore         string `json:"awayScore"`
}

// ScheduleWeekDTO is one week's matchups; Week is parsed to an int here, at the boundary.
type ScheduleWeekDTO struct {
	Week     int                  `json:"week"`
	Matchups []ScheduleMatchupDTO `json:"matchups"`
}

// LeagueScheduleResult is the full-season schedule. Weeks is never nil on success.
type LeagueScheduleResult struct {
	OK        bool              `json:"ok"`
	Weeks     []ScheduleWeekDTO `json:"weeks"`
	Freshness Freshness         `json:"freshness"`
	Detail    string            `json:"detail"`
}

// GetLeagueSchedule is the read behind the calendar's schedule pane: display-only.
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
				HomeFranchiseName: domain.FranchiseLabel(names, home.FranchiseID),
				HomeScore:         home.Score,
				AwayFranchiseID:   away.FranchiseID,
				AwayFranchiseName: domain.FranchiseLabel(names, away.FranchiseID),
				AwayScore:         away.Score,
			})
		}
		out = append(out, ScheduleWeekDTO{Week: wk, Matchups: matchups})
	}

	return LeagueScheduleResult{OK: true, Weeks: out, Freshness: fresh}
}
