// Package now constructs Now and Live game readings from detached published inputs.
package now

import (
	"time"

	raceapp "github.com/secureprospective/TheWarRoom/internal/app/race"
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

// Held owns detached indexes; refresh may replace payloads, never mutate them.
type Held struct {
	Schedules map[int]Feed[[]nflschedule.RawWeek]
	Byes      Feed[[]nflschedule.RawBye]
	Firsts    FirstScores
}
type FirstScores struct {
	Weeks  map[int]FirstScore
	Copies int
}
type FirstScore struct {
	At   time.Time
	Week leaguefeed.WeekResult
}

// Snapshot has no references to root locks, refreshers or scheduling state.
type Snapshot struct {
	Race        raceapp.Snapshot
	Held        Held
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
