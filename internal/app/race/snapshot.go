// Package race builds League Pulse race reports from detached published inputs.
package race

import (
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/report"
	"github.com/secureprospective/TheWarRoom/internal/app/scoring"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/leaguestandings"
	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
)

// Feed is a published payload and its read metadata. Callers must not mutate payloads.
type Feed[T any] struct {
	Value        T
	FetchedAt    time.Time
	RefreshError string
	ReadPending  bool
}

// Held contains detached indexes of immutable published feed payloads.
type Held struct {
	Weeks     Feed[[]leaguefeed.WeekResult]
	Brackets  Feed[[]leaguefeed.Bracket]
	Standings Feed[[]leaguestandings.RawStanding]
	Power     map[string]scoring.PowerRankingsResult
}

func shell[T any](id, title, how string, f Feed[[]T], zone *time.Location, columns ...string) report.Report {
	return report.Shell(id, title, how, report.Feed{FetchedAt: f.FetchedAt, RefreshError: f.RefreshError, ReadPending: f.ReadPending}, len(f.Value), zone, columns...)
}
