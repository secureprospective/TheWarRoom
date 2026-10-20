package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
)

type RawLeagueScheduleScope struct {
	League string `json:"league"`
}

func (r RawLeagueScheduleScope) Validate() error {
	if strings.TrimSpace(r.League) != r.League || !strings.HasPrefix(r.League, "native.") ||
		r.League == "native." {
		return fmt.Errorf("choose an app league")
	}
	return nil
}

func (a *App) TargetLeagueSchedule(raw RawLeagueScheduleScope) (commissioner.Schedule, error) {
	if err := raw.Validate(); err != nil {
		return commissioner.Schedule{}, err
	}
	if err := a.ready(); err != nil {
		return commissioner.Schedule{}, fmt.Errorf("league schedule startup: %w", err)
	}
	ctx, cancel := context.WithTimeout(a.ctx, refreshTimeout)
	defer cancel()
	events, err := a.history.Events(ctx, raw.League, commissioner.Stream, 0)
	if err != nil {
		return commissioner.Schedule{}, fmt.Errorf("read league schedule ledger: %w", err)
	}
	records := make([]commissioner.Record, 0, len(events))
	for _, event := range events {
		if event.Payload.Commissioner == nil {
			return commissioner.Schedule{}, fmt.Errorf("the league schedule has an unreadable change")
		}
		record := *event.Payload.Commissioner
		if err := record.Validate(); err != nil {
			return commissioner.Schedule{}, fmt.Errorf("read league schedule change: %w", err)
		}
		records = append(records, record)
	}
	if len(records) == 0 {
		return commissioner.Schedule{}, fmt.Errorf("create this app league before reading its schedule")
	}
	reading, err := commissioner.ScheduleReading(raw.League, records)
	if err != nil {
		return commissioner.Schedule{}, fmt.Errorf("league schedule: %w", err)
	}
	return reading, nil
}
