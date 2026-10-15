package main

import (
	"context"
	"fmt"
	"strings"
	"time"

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
	if !strings.HasPrefix(r.League, "native.") || r.League == "native." ||
		strings.TrimSpace(r.League) != r.League || r.Season < 1900 || r.Season > 9999 ||
		strings.TrimSpace(r.Franchise) == "" || strings.TrimSpace(string(r.Action)) == "" {
		return fmt.Errorf("choose an app league, season, franchise and action")
	}
	if r.LocalZone == "" || r.LocalZone == "Local" {
		return fmt.Errorf("choose your local time zone")
	}
	if _, err := time.LoadLocation(r.LocalZone); err != nil {
		return fmt.Errorf("read owner time zone: %w", err)
	}
	return nil
}

// TargetLeagueCalendar reads only the app league ledger, never an imported calendar.
func (a *App) TargetLeagueCalendar(raw RawLeagueCalendarScope) (commissioner.Calendar, error) {
	if err := raw.Validate(); err != nil {
		return commissioner.Calendar{}, err
	}
	if err := a.ready(); err != nil {
		return commissioner.Calendar{}, fmt.Errorf("league calendar startup: %w", err)
	}
	records, err := a.leagueRecords(raw.League)
	if err != nil {
		return commissioner.Calendar{}, err
	}
	zone, err := time.LoadLocation(raw.LocalZone)
	if err != nil {
		return commissioner.Calendar{}, fmt.Errorf("read owner time zone: %w", err)
	}
	reading, err := commissioner.CalendarReading(raw.League, records, a.timeNow(), raw.Season,
		raw.Action, raw.Franchise, zone)
	if err != nil {
		return commissioner.Calendar{}, fmt.Errorf("league calendar: %w", err)
	}
	return reading, nil
}

// leagueRecords replays one app league's commissioner ledger; the calendar and the gate read the same one.
func (a *App) leagueRecords(league string) ([]commissioner.Record, error) {
	return a.leagueRecordsIn(a.ctx, league)
}

// leagueRecordsIn is leagueRecords for a caller that already holds a context, as an owner action does.
func (a *App) leagueRecordsIn(parent context.Context, league string) ([]commissioner.Record, error) {
	ctx, cancel := context.WithTimeout(parent, refreshTimeout)
	defer cancel()
	events, err := a.history.Events(ctx, league, commissioner.Stream, 0)
	if err != nil {
		return nil, fmt.Errorf("read league calendar ledger: %w", err)
	}
	records := make([]commissioner.Record, 0, len(events))
	for _, event := range events {
		if event.Payload.Commissioner == nil {
			return nil, fmt.Errorf("the league calendar has an unreadable change")
		}
		record := *event.Payload.Commissioner
		if err := record.Validate(); err != nil {
			return nil, fmt.Errorf("read league calendar change: %w", err)
		}
		records = append(records, record)
	}
	return records, nil
}
