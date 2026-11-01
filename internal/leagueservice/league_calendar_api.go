package leagueservice

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
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

type RawLeagueScheduleScope struct {
	League string `json:"league"`
}

func (r RawLeagueScheduleScope) Validate() error {
	return ScheduleScope(r).Validate()
}

func (a *Service) TargetLeagueCalendar(raw RawLeagueCalendarScope) (commissioner.Calendar, error) {
	if err := raw.Validate(); err != nil {
		return commissioner.Calendar{}, err
	}
	if err := a.admitRead(raw.League, "calendar.read", raw.Franchise); err != nil {
		return commissioner.Calendar{}, err
	}
	if err := a.host.Ready(); err != nil {
		return commissioner.Calendar{}, fmt.Errorf("league calendar startup: %w", err)
	}
	records, err := a.host.CommissionerRecords(a.viewContext(a.host.CommissionerContext()), raw.League)
	if err != nil {
		//nolint:wrapcheck // Preserve the existing refusal text and error identity.
		return commissioner.Calendar{}, err
	}

	reading, err := Calendar(CalendarScope(raw), records, a.host.CommissionerNow())
	reading.Revision = commissioner.ViewRevision(a.viewContext(a.host.CommissionerContext()), records, reading.Revision)
	return reading, err
}

func (a *Service) TargetLeagueSchedule(raw RawLeagueScheduleScope) (commissioner.Schedule, error) {
	if err := raw.Validate(); err != nil {
		return commissioner.Schedule{}, err
	}
	if err := a.admitRead(raw.League, "schedule.read", ""); err != nil {
		return commissioner.Schedule{}, err
	}
	if err := a.host.Ready(); err != nil {
		return commissioner.Schedule{}, fmt.Errorf("league schedule startup: %w", err)
	}
	ctx, cancel := context.WithTimeout(a.viewContext(a.host.CommissionerContext()), refreshTimeout)
	defer cancel()
	records, err := Records(ctx, a.host, raw.League, "schedule")
	if err != nil {
		return commissioner.Schedule{}, err
	}

	reading, err := Schedule(raw.League, records)
	reading.Revision = commissioner.ViewRevision(ctx, records, reading.Revision)
	return reading, err
}

// Ledger is the part of the history store Records reads: one league's events on one stream.
type Ledger interface {
	Events(ctx context.Context, league, stream string, after int64) ([]history.LeagueEvent, error)
}

// Records replays one app league's commissioner ledger. subject names the reading in its errors
// ("calendar", "schedule"), so each binding keeps its own words.
func Records(ctx context.Context, ledger Ledger, league, subject string) ([]commissioner.Record, error) {
	events, err := ledger.Events(ctx, league, commissioner.Stream, 0)
	if err != nil {
		return nil, fmt.Errorf("read league %s ledger: %w", subject, err)
	}
	records := make([]commissioner.Record, 0, len(events))
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
	records, err = commissioner.ResolvePrincipals(ctx, league, records)
	if err != nil {
		return nil, fmt.Errorf("resolve league %s identities: %w", subject, err)
	}
	return records, nil
}

// Schedule reads the league schedule from its replayed ledger.
func Schedule(league string, records []commissioner.Record) (commissioner.Schedule, error) {
	if len(records) == 0 {
		return commissioner.Schedule{}, fmt.Errorf("create this app league before reading its schedule")
	}
	reading, err := commissioner.ScheduleReading(league, records)
	if err != nil {
		return commissioner.Schedule{}, fmt.Errorf("league schedule: %w", err)
	}
	return reading, nil
}

// Calendar reads the league calendar for one franchise and action at the given time, in the owner's zone.
func Calendar(scope CalendarScope, records []commissioner.Record, at time.Time) (commissioner.Calendar, error) {
	zone, err := time.LoadLocation(scope.LocalZone)
	if err != nil {
		return commissioner.Calendar{}, fmt.Errorf("read owner time zone: %w", err)
	}
	reading, err := commissioner.CalendarReading(scope.League, records, at, scope.Season,
		scope.Action, scope.Franchise, zone)
	if err != nil {
		return commissioner.Calendar{}, fmt.Errorf("league calendar: %w", err)
	}
	return reading, nil
}

// CalendarScope is the twin of the root RawLeagueCalendarScope: the same fields in the same order.
type CalendarScope struct {
	League    string             `json:"league"`
	Season    int                `json:"season"`
	LocalZone string             `json:"localZone"`
	Franchise string             `json:"franchise"`
	Action    leaguerules.Action `json:"action"`
}

func (r CalendarScope) Validate() error {
	if AppLeague(r.League) != nil || r.Season < 1900 || r.Season > 9999 ||
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

// ScheduleScope is the twin of the root RawLeagueScheduleScope.
type ScheduleScope struct {
	League string `json:"league"`
}

func (r ScheduleScope) Validate() error { return AppLeague(r.League) }

// AppLeague admits an app league id: "native." and a name, with no surrounding space.
func AppLeague(league string) error {
	if !strings.HasPrefix(league, "native.") || league == "native." || strings.TrimSpace(league) != league {
		return fmt.Errorf("choose an app league")
	}
	return nil
}

// Blocked is the preview of a body that could not be read: no fields, and the reason as its one blocker.
func Blocked(err error) commissioner.Preview {
	var refusal *commissioner.Refusal
	err = commissioner.Refused(err)
	words := err.Error()
	if errors.As(err, &refusal) {
		words = refusal.Reason()
	}
	return commissioner.Preview{Fields: []commissioner.PreviewField{},
		Affects:  commissioner.PreviewAffects{Readings: []string{}, Words: "Not known yet"},
		Blockers: []string{words}}
}
