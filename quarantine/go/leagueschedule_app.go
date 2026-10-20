package main

import (
	"context"
	"fmt"
	"strconv"

	"github.com/secureprospective/TheWarRoom/internal/domain"
)

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
	if err := a.ready(); err != nil {
		return LeagueScheduleResult{Detail: err.Error()}
	}
	ctx, cancel := context.WithTimeout(a.ctx, m2Timeout)
	defer cancel()

	weeks, fresh, err := a.leagueScheduleOrArchive(ctx)
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
