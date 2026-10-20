package main

import (
	"context"
	"fmt"

	commissionerapp "github.com/secureprospective/TheWarRoom/internal/app/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/commissioner"
)

type RawLeagueScheduleScope struct {
	League string `json:"league"`
}

func (r RawLeagueScheduleScope) Validate() error {
	return commissionerapp.ScheduleScope(r).Validate() //nolint:wrapcheck // the owner reads the words as they are
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
	records, err := commissionerapp.Records(ctx, a.history, raw.League, "schedule")
	if err != nil {
		return commissioner.Schedule{}, err //nolint:wrapcheck // the service's errors already name the league schedule
	}
	//nolint:wrapcheck // the service's errors already start "league schedule:"
	return commissionerapp.Schedule(raw.League, records)
}
