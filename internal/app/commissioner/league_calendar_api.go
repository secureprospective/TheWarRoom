package commissioner

import (
	"context"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/leagueservice"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

type RawLeagueCalendarScope leagueservice.RawLeagueCalendarScope

func (r RawLeagueCalendarScope) Validate() error {
	//nolint:wrapcheck // The shared service supplies the existing user-facing context.
	return leagueservice.RawLeagueCalendarScope(r).Validate()
}

// TargetLeagueCalendar reads only the app league ledger, never an imported calendar.
func (a *API) TargetLeagueCalendar(raw RawLeagueCalendarScope) (commissioner.Calendar, error) {
	//nolint:wrapcheck // The shared service supplies the existing user-facing context.
	return a.reads.TargetLeagueCalendar(leagueservice.RawLeagueCalendarScope(raw))
}

type RawLeagueScheduleScope leagueservice.RawLeagueScheduleScope

func (r RawLeagueScheduleScope) Validate() error {
	//nolint:wrapcheck // The shared service supplies the existing user-facing context.
	return leagueservice.RawLeagueScheduleScope(r).Validate()
}

func (a *API) TargetLeagueSchedule(raw RawLeagueScheduleScope) (commissioner.Schedule, error) {
	//nolint:wrapcheck // The shared service supplies the existing user-facing context.
	return a.reads.TargetLeagueSchedule(leagueservice.RawLeagueScheduleScope(raw))
}

// Ledger is the part of the history store Records reads: one league's events on one stream.
type Ledger interface {
	Events(ctx context.Context, league, stream string, after int64) ([]history.LeagueEvent, error)
}

// Records replays one app league's commissioner ledger. subject names the reading in its errors
// ("calendar", "schedule"), so each binding keeps its own words.
func Records(ctx context.Context, ledger Ledger, league, subject string) ([]commissioner.Record, error) {
	//nolint:wrapcheck // The shared service supplies the existing user-facing context.
	return leagueservice.Records(ctx, ledger, league, subject)
}

// Schedule reads the league schedule from its replayed ledger.
func Schedule(league string, records []commissioner.Record) (commissioner.Schedule, error) {
	//nolint:wrapcheck // The shared service supplies the existing user-facing context.
	return leagueservice.Schedule(league, records)
}

// Calendar reads the league calendar for one franchise and action at the given time, in the owner's zone.
func Calendar(scope CalendarScope, records []commissioner.Record, at time.Time) (commissioner.Calendar, error) {
	//nolint:wrapcheck // The shared service supplies the existing user-facing context.
	return leagueservice.Calendar(leagueservice.CalendarScope(scope), records, at)
}

// CalendarScope is the twin of the root RawLeagueCalendarScope: the same fields in the same order.
type CalendarScope leagueservice.CalendarScope

//nolint:wrapcheck // The service owns validation context; keep desktop refusals identical.
func (r CalendarScope) Validate() error { return leagueservice.CalendarScope(r).Validate() }

// ScheduleScope is the twin of the root RawLeagueScheduleScope.
type ScheduleScope leagueservice.ScheduleScope

//nolint:wrapcheck // The service owns validation context; keep desktop refusals identical.
func (r ScheduleScope) Validate() error { return leagueservice.ScheduleScope(r).Validate() }

// AppLeague admits an app league id: "native." and a name, with no surrounding space.
func AppLeague(league string) error {
	//nolint:wrapcheck // The shared service supplies the existing user-facing context.
	return leagueservice.AppLeague(league)
}

// Blocked is the preview of a body that could not be read: no fields, and the reason as its one blocker.
func Blocked(err error) commissioner.Preview {
	return leagueservice.Blocked(err)
}

// ChangeError words a recorded change's failure: a refusal reads as its reason alone, without the
// context each layer adds for the log; anything else keeps the "league change" context.
func ChangeError(err error) error {
	//nolint:wrapcheck // The shared service supplies the user-facing context.
	return leagueservice.ChangeError(err)
}

type RawNativeScheduleScope struct {
	League    string `json:"league"`
	Franchise string `json:"franchise"`
	Season    int    `json:"season"`
	LocalZone string `json:"localZone"`
}

func (r RawNativeScheduleScope) Validate() error {
	//nolint:wrapcheck // The shared service supplies the existing user-facing context.
	return leagueservice.RawNativeScheduleScope(r).Validate()
}

type RawNativeWeekScope struct {
	RawNativeScheduleScope
	At time.Time `json:"at"`
}

func (r RawNativeWeekScope) Validate() error {
	//nolint:wrapcheck // The shared service supplies the existing user-facing context.
	return leagueservice.RawNativeWeekScope{RawNativeScheduleScope: leagueservice.RawNativeScheduleScope(r.RawNativeScheduleScope), At: r.At}.Validate()
}

func (a *API) TargetNativeFranchiseSchedule(raw RawNativeScheduleScope) (commissioner.FranchiseSchedule,
	error) {
	//nolint:wrapcheck // The shared service supplies the existing user-facing context.
	return a.reads.TargetNativeFranchiseSchedule(leagueservice.RawNativeScheduleScope(raw))
}

func (a *API) TargetNativeWeek(raw RawNativeWeekScope) (commissioner.NativeWeek, error) {
	//nolint:wrapcheck // The shared service supplies the existing user-facing context.
	return a.reads.TargetNativeWeek(leagueservice.RawNativeWeekScope{RawNativeScheduleScope: leagueservice.RawNativeScheduleScope(raw.RawNativeScheduleScope), At: raw.At})
}
