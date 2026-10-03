// Package leagueschedule fetches the fantasy league's own weekly matchups from MFL's
// `schedule` export (not the NFL schedule). Without W, one call returns the whole season. The
// app resolves franchise names.
package leagueschedule

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/mfl"
)

// errEmptySchedule: once a schedule exists in MFL it is never empty, so zero weeks is a failed
// fetch.
var errEmptySchedule = errors.New("leagueschedule: response contained zero weeks")

// RawMatchupSide is one franchise's side. IsHome stays "0"/"1"; Score is empty until played.
type RawMatchupSide struct {
	FranchiseID string
	IsHome      string
	Score       string
}

// RawMatchup is one scheduled game: exactly two franchise sides.
type RawMatchup struct {
	Franchises [2]RawMatchupSide
}

// RawScheduleWeek is one week's matchups.
type RawScheduleWeek struct {
	Week     string
	Matchups []RawMatchup
}

// Validate requires a week number and, for each matchup, two sides with ids and exactly one
// home side. The app decides home/away with a single swap on isHome == "0", so two home or two
// away sides would mislabel silently.
//
// A week with no matchups is valid: an unseeded playoff week has none (seen live 2026-07-27),
// and rejecting it would fail the whole season.
func (w RawScheduleWeek) Validate() error {
	if strings.TrimSpace(w.Week) == "" {
		return fmt.Errorf("leagueschedule: week missing its number")
	}
	for _, m := range w.Matchups {
		homeCount := 0
		for _, side := range m.Franchises {
			if strings.TrimSpace(side.FranchiseID) == "" {
				return fmt.Errorf("leagueschedule: week %s has a matchup with an empty franchise id", w.Week)
			}
			h := strings.TrimSpace(side.IsHome)
			if h != "0" && h != "1" {
				return fmt.Errorf("leagueschedule: week %s franchise %s isHome %q is not 0/1",
					w.Week, side.FranchiseID, side.IsHome)
			}
			if h == "1" {
				homeCount++
			}
		}
		if homeCount != 1 {
			return fmt.Errorf(
				"leagueschedule: week %s has a matchup where %d of 2 sides are marked home (want exactly 1)",
				w.Week, homeCount)
		}
	}
	return nil
}

type scheduleEnvelope struct {
	Schedule struct {
		WeeklySchedule ingestion.MFLList[weekBlock] `json:"weeklySchedule"`
	} `json:"schedule"`
}

type weekBlock struct {
	Week    string                          `json:"week"`
	Matchup ingestion.MFLList[matchupBlock] `json:"matchup"`
}

type matchupBlock struct {
	Franchise ingestion.MFLList[franchiseBlock] `json:"franchise"`
}

type franchiseBlock struct {
	ID     string `json:"id"`
	IsHome string `json:"isHome"`
	Score  string `json:"score"`
}

// Fetch discovers the league host and returns every week of the season, validated.
func Fetch(ctx context.Context, c *mfl.Client, year, leagueID string) ([]RawScheduleWeek, error) {
	env, err := ingestion.FetchLeagueExport[scheduleEnvelope](ctx, c, "schedule", year, leagueID, nil)
	if err != nil {
		return nil, fmt.Errorf("leagueschedule: %w", err)
	}
	if len(env.Schedule.WeeklySchedule) == 0 {
		return nil, errEmptySchedule
	}
	return flatten(ctx, env)
}

// flatten validates each week; a malformed one fails the fetch.
func flatten(ctx context.Context, env scheduleEnvelope) ([]RawScheduleWeek, error) {
	out := make([]RawScheduleWeek, 0, len(env.Schedule.WeeklySchedule))
	for _, wb := range env.Schedule.WeeklySchedule {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("leagueschedule: flatten cancelled: %w", ctx.Err())
		default:
		}

		matchups := make([]RawMatchup, 0, len(wb.Matchup))
		for _, mb := range wb.Matchup {
			if len(mb.Franchise) != 2 {
				return nil, fmt.Errorf("leagueschedule: week %s has a matchup with %d franchises, want 2",
					wb.Week, len(mb.Franchise))
			}
			matchups = append(matchups, RawMatchup{
				Franchises: [2]RawMatchupSide{
					{FranchiseID: mb.Franchise[0].ID, IsHome: mb.Franchise[0].IsHome, Score: mb.Franchise[0].Score},
					{FranchiseID: mb.Franchise[1].ID, IsHome: mb.Franchise[1].IsHome, Score: mb.Franchise[1].Score},
				},
			})
		}

		w := RawScheduleWeek{Week: wb.Week, Matchups: matchups}
		if err := w.Validate(); err != nil {
			return nil, err
		}
		out = append(out, w)
	}
	return out, nil
}
