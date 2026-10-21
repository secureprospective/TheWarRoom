package main

import (
	"context"
	"strconv"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/core"

	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/leagueschedule"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/leaguestandings"
)

// fallbackParent uses the application lifetime so a timed-out live request can still read history.
func (a *App) fallbackParent() context.Context { return a.core.CommsContext() }

type archivedFeed[T any] = core.ArchivedFeed[T]

func liveOrArchive[T any](ctx, fallback context.Context, season int, f archivedFeed[T], logLine func(string)) ([]T, Freshness, error) {
	return core.LiveOrArchive(ctx, fallback, season, f, logLine) //nolint:wrapcheck // Preserve the shared fallback error text.
}
func readArchivedFeed[T any](ctx context.Context, season int, f archivedFeed[T]) ([]T, time.Time, bool, error) {
	return core.ReadArchivedFeed(ctx, season, f) //nolint:wrapcheck // Preserve the shared fallback error text.
}
func sameExport(raw, export string, season int, leagueless bool) bool {
	return core.SameExport(raw, export, season, leagueless)
}

func (a *App) standingsOrArchive(ctx context.Context) ([]leaguestandings.RawStanding, Freshness, error) {
	return liveOrArchive(ctx, a.fallbackParent(), a.core.Season, archivedFeed[leaguestandings.RawStanding]{
		Export: leaguestandings.Export,
		Fetch: func(ctx context.Context) ([]leaguestandings.RawStanding, error) {
			return leaguestandings.Fetch(ctx, a.core.MFLClient, strconv.Itoa(a.core.Season), ingestion.LeagueID)
		},
		Parse:  leaguestandings.Parse,
		Bodies: a.core.History.ArchivedBodies,
	}, a.core.FallbackLog)
}

func (a *App) leagueScheduleOrArchive(ctx context.Context) ([]leagueschedule.RawScheduleWeek, Freshness, error) {
	return liveOrArchive(ctx, a.fallbackParent(), a.core.Season, archivedFeed[leagueschedule.RawScheduleWeek]{
		Export: leagueschedule.Export,
		Fetch: func(ctx context.Context) ([]leagueschedule.RawScheduleWeek, error) {
			return leagueschedule.Fetch(ctx, a.core.MFLClient, strconv.Itoa(a.core.Season), ingestion.LeagueID)
		},
		Parse:  leagueschedule.Parse,
		Bodies: a.core.History.ArchivedBodies,
	}, a.core.FallbackLog)
}

const fallbackTimeout = core.FallbackTimeout
