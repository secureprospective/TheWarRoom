package main

import (
	"context"
	"fmt"

	commissionerapp "github.com/secureprospective/TheWarRoom/internal/app/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

type RawLeagueCalendarScope struct {
	League    string             `json:"league"`
	Season    int                `json:"season"`
	LocalZone string             `json:"localZone"`
	Franchise string             `json:"franchise"`
	Action    leaguerules.Action `json:"action"`
}

func (r RawLeagueCalendarScope) Validate() error {
	return commissionerapp.CalendarScope(r).Validate() //nolint:wrapcheck // the owner reads the words as they are
}

// TargetLeagueCalendar reads only the app league ledger, never an imported calendar.
func (a *App) TargetLeagueCalendar(raw RawLeagueCalendarScope) (commissioner.Calendar, error) {
	if err := raw.Validate(); err != nil {
		return commissioner.Calendar{}, err
	}
	if err := a.ready(); err != nil {
		return commissioner.Calendar{}, fmt.Errorf("league calendar startup: %w", err)
	}
	records, err := a.leagueRecords(a.ctx, raw.League)
	if err != nil {
		return commissioner.Calendar{}, err
	}
	//nolint:wrapcheck // the service's errors already start "league calendar:" or "read owner time zone:"
	return commissionerapp.Calendar(commissionerapp.CalendarScope(raw), records, a.timeNow())
}

// leagueRecords replays one app league's commissioner ledger; the calendar and the gate read the same one.
func (a *App) leagueRecords(parent context.Context, league string) ([]commissioner.Record, error) {
	ctx, cancel := context.WithTimeout(parent, refreshTimeout)
	defer cancel()
	//nolint:wrapcheck // the service's errors already name the league calendar
	return commissionerapp.Records(ctx, a.history, league, "calendar")
}
