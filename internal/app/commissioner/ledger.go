// Package commissioner is the Commissioner feature's service: the league ledger replay the calendar,
// the schedule and the native gate read, and the readings built on it. It never decides who may act:
// the root adapter reads the session principal, and the commissioner engine checks it.
package commissioner

import (
	"context"
	"fmt"
	"time"

	commish "github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

// Ledger is the part of the history store Records reads: one league's events on one stream.
type Ledger interface {
	Events(ctx context.Context, league, stream string, after int64) ([]history.LeagueEvent, error)
}

// Records replays one app league's commissioner ledger. subject names the reading in its errors
// ("calendar", "schedule"), so each binding keeps its own words.
func Records(ctx context.Context, ledger Ledger, league, subject string) ([]commish.Record, error) {
	events, err := ledger.Events(ctx, league, commish.Stream, 0)
	if err != nil {
		return nil, fmt.Errorf("read league %s ledger: %w", subject, err)
	}
	records := make([]commish.Record, 0, len(events))
	for _, event := range events {
		if event.Payload.Commissioner == nil {
			return nil, fmt.Errorf("the league %s has an unreadable change", subject)
		}
		record := *event.Payload.Commissioner
		if err := record.Validate(); err != nil {
			return nil, fmt.Errorf("read league %s change: %w", subject, err)
		}
		records = append(records, record)
	}
	return records, nil
}

// Schedule reads the league schedule from its replayed ledger.
func Schedule(league string, records []commish.Record) (commish.Schedule, error) {
	if len(records) == 0 {
		return commish.Schedule{}, fmt.Errorf("create this app league before reading its schedule")
	}
	reading, err := commish.ScheduleReading(league, records)
	if err != nil {
		return commish.Schedule{}, fmt.Errorf("league schedule: %w", err)
	}
	return reading, nil
}

// Calendar reads the league calendar for one franchise and action at the given time, in the owner's zone.
func Calendar(scope CalendarScope, records []commish.Record, at time.Time) (commish.Calendar, error) {
	zone, err := time.LoadLocation(scope.LocalZone)
	if err != nil {
		return commish.Calendar{}, fmt.Errorf("read owner time zone: %w", err)
	}
	reading, err := commish.CalendarReading(scope.League, records, at, scope.Season,
		scope.Action, scope.Franchise, zone)
	if err != nil {
		return commish.Calendar{}, fmt.Errorf("league calendar: %w", err)
	}
	return reading, nil
}
