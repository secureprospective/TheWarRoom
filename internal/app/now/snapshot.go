// Package now constructs Now and Live game readings from detached published inputs.
package now

import (
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/season"

	"github.com/secureprospective/TheWarRoom/internal/app/report"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/livegame"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/marketfeeds"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/nflschedule"
	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
)

// Feed borrows an immutable published payload with its captured read metadata.
type Feed[T any] struct {
	Value        T
	FetchedAt    time.Time
	RefreshError string
	ReadPending  bool
}

// Feeds are Now's detached held feeds; refresh may replace payloads, never mutate them.
type Feeds struct {
	Schedules map[int]Feed[[]nflschedule.RawWeek]
	Byes      Feed[[]nflschedule.RawBye]
	Firsts    FirstScores
}

// FirstScores is, per week, the oldest archived weekly results that had the week scored.
// Through is the newest copy already read, so a later read only looks at newer copies.
type FirstScores struct {
	Weeks   map[int]FirstScore
	Through time.Time
	Copies  int
}
type FirstScore struct {
	At   time.Time
	Week leaguefeed.WeekResult
}

// Snapshot has no references to root locks, refreshers or scheduling state.
type Snapshot struct {
	Race        season.Snapshot
	Held        Feeds
	Live        Feed[leaguefeed.Lineups]
	Projections Feed[[]marketfeeds.Feed]
	At          time.Time
	ReportWeek  int
	Current     int
	Week        int
}

// LiveSnapshot excludes the polling adapter's transport, archive and backoff state.
type LiveSnapshot struct {
	Board   livegame.Held
	Details map[string]livegame.Detail
	Follow  string
	Active  bool
	Zone    *time.Location
}

func reportShell[T any](id, title, how string, f Feed[[]T], zone *time.Location, columns ...string) report.Report {
	return report.Shell(id, title, how, report.Feed{FetchedAt: f.FetchedAt, RefreshError: f.RefreshError, ReadPending: f.ReadPending}, len(f.Value), zone, columns...)
}
func reportTime(at time.Time, zone *time.Location) string { return report.Time(at, zone) }
