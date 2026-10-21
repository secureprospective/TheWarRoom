package commissioner

import (
	"context"
	"fmt"

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
	return CalendarScope(r).Validate()
}

// TargetLeagueCalendar reads only the app league ledger, never an imported calendar.
func (a *API) TargetLeagueCalendar(raw RawLeagueCalendarScope) (commissioner.Calendar, error) {
	if err := raw.Validate(); err != nil {
		return commissioner.Calendar{}, err
	}
	if err := a.host.Ready(); err != nil {
		return commissioner.Calendar{}, fmt.Errorf("league calendar startup: %w", err)
	}
	records, err := a.host.CommissionerRecords(a.host.CommissionerContext(), raw.League)
	if err != nil {
		//nolint:wrapcheck // Preserve the existing refusal text and error identity.
		return commissioner.Calendar{}, err
	}

	return Calendar(CalendarScope(raw), records, a.host.CommissionerNow())
}

type RawLeagueScheduleScope struct {
	League string `json:"league"`
}

func (r RawLeagueScheduleScope) Validate() error {
	return ScheduleScope(r).Validate()
}

func (a *API) TargetLeagueSchedule(raw RawLeagueScheduleScope) (commissioner.Schedule, error) {
	if err := raw.Validate(); err != nil {
		return commissioner.Schedule{}, err
	}
	if err := a.host.Ready(); err != nil {
		return commissioner.Schedule{}, fmt.Errorf("league schedule startup: %w", err)
	}
	ctx, cancel := context.WithTimeout(a.host.CommissionerContext(), refreshTimeout)
	defer cancel()
	records, err := Records(ctx, a.host, raw.League, "schedule")
	if err != nil {
		return commissioner.Schedule{}, err
	}

	return Schedule(raw.League, records)
}
