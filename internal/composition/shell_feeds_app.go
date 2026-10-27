package composition

import (
	"context"
	"strconv"
	"time"

	controlapp "github.com/secureprospective/TheWarRoom/internal/app/control"

	"github.com/secureprospective/TheWarRoom/internal/app/report"
	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/leagueclock"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
)

func readShellFeed[T any](ctx context.Context, a *App, held *heldSeasonFeed[[]T], export string,
	parse func([]byte) ([]T, error)) heldSeasonFeed[[]T] {
	a.core.SeasonMu.Lock()
	feed := *held
	a.core.SeasonMu.Unlock()
	return controlapp.ReadDailyShell(ctx, feed, a.timeNow,
		func(ctx context.Context) heldSeasonFeed[[]T] { return archivedOnly(ctx, a, export, false, parse) },
		func(archived heldSeasonFeed[[]T]) heldSeasonFeed[[]T] {
			a.core.SeasonMu.Lock()
			defer a.core.SeasonMu.Unlock()
			return controlapp.MergeDailyShell(held, archived, a.timeNow())
		})
}

func shellFeedTime[T any](feed heldSeasonFeed[[]T], zone *time.Location) string {
	return report.FeedTime(feed.FeedState(), zone)
}

func refreshDailyShellFeed[T any](ctx context.Context, a *App, held *heldSeasonFeed[[]T], now time.Time,
	feed archivedFeed[T]) {
	current := readShellFeed(ctx, a, held, feed.Export, feed.Parse)
	controlapp.RefreshDailyShell(ctx, func() bool {
		a.core.SeasonMu.Lock()
		defer a.core.SeasonMu.Unlock()
		return controlapp.ClaimShellDay(&a.core.ShellFeedDays, feed.Export, now, current)
	}, func(ctx context.Context) ([]T, Freshness, error) {
		return liveOrArchive(ctx, a.fallbackParent(), a.core.Season, feed, a.core.FallbackLog)
	}, func(rows []T, fresh Freshness, err error) {
		a.core.SeasonMu.Lock()
		defer a.core.SeasonMu.Unlock()
		holdSeasonFeed(held, rows, fresh, err)
	})
}

func (a *App) refreshShellFeeds(ctx context.Context) {
	var year string
	var now time.Time
	controlapp.RefreshShell(ctx, a.seasonPauseFunc(), func(ctx context.Context) {
		year, now = strconv.Itoa(a.core.Season), a.timeNow()
		refreshDailyShellFeed(ctx, a, &a.core.SeasonCalendar, now, archivedFeed[ingestion.RawCalendarEvent]{
			Export: "calendar", Parse: ingestion.ParseCalendar, Bodies: a.core.History.ReferenceBodies,
			Fetch: func(ctx context.Context) ([]ingestion.RawCalendarEvent, error) {
				return ingestion.FetchCalendar(ctx, a.core.MFLClient, year, ingestion.LeagueID)
			},
		})
	}, func(ctx context.Context) {
		refreshDailyShellFeed(ctx, a, &a.core.SeasonTradeBait, now, archivedFeed[ingestion.RawTradeBait]{
			Export: "tradeBait", Parse: ingestion.ParseTradeBait, Bodies: a.core.History.ReferenceBodies,
			Fetch: func(ctx context.Context) ([]ingestion.RawTradeBait, error) {
				return ingestion.FetchTradeBait(ctx, a.core.MFLClient, year, ingestion.LeagueID)
			},
		})
	})
}

func (a *App) mergeMFLCalendar(ctx context.Context, now time.Time, clock *snapshot.Sourced[leagueclock.Reading]) {
	held := readShellFeed(ctx, a, &a.core.SeasonCalendar, "calendar", ingestion.ParseCalendar)
	controlapp.MergeMFLCalendar(now, clock, held)
}
