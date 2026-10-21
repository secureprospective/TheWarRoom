package main

import (
	"context"
	"strconv"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/ingestion"
)

func readShellFeed[T any](ctx context.Context, a *App, held *heldSeasonFeed[[]T], export string,
	parse func([]byte) ([]T, error)) heldSeasonFeed[[]T] {
	a.core.SeasonMu.Lock()
	feed := *held
	a.core.SeasonMu.Unlock()
	if !feed.FetchedAt.IsZero() {
		return dailyShellReading(feed, a.timeNow())
	}
	feed = archivedOnly(ctx, a, export, false, parse)
	a.core.SeasonMu.Lock()
	defer a.core.SeasonMu.Unlock()
	if held.FetchedAt.IsZero() {
		*held = feed
	}
	return dailyShellReading(*held, a.timeNow())
}

func dailyShellReading[T any](feed heldSeasonFeed[[]T], now time.Time) heldSeasonFeed[[]T] {
	feed.ReadPending = !feed.FetchedAt.IsZero() && feed.RefreshError == "" &&
		feed.FetchedAt.In(time.Local).Format("2006-01-02") != now.In(time.Local).Format("2006-01-02")
	return feed
}

func shellFeedTime[T any](feed heldSeasonFeed[[]T], zone *time.Location) string {
	if feed.FetchedAt.IsZero() {
		return "unavailable · no held copy"
	}
	kind := "live"
	if feed.RefreshError != "" || feed.ReadPending {
		kind = "archived copy"
	}
	text := kind + " · fetched " + reportTime(feed.FetchedAt, zone)
	if feed.ReadPending {
		text += " · " + pendingShellRead(feed.FetchedAt, zone)
	} else if feed.RefreshError != "" {
		text += " · Refresh failed"
	}
	return text
}

func refreshDailyShellFeed[T any](ctx context.Context, a *App, held *heldSeasonFeed[[]T], now time.Time,
	feed archivedFeed[T]) {
	day := now.In(time.Local).Format("2006-01-02")
	current := readShellFeed(ctx, a, held, feed.export, feed.parse)
	a.core.SeasonMu.Lock()
	if a.core.ShellFeedDays == nil {
		a.core.ShellFeedDays = map[string]string{}
	}
	if a.core.ShellFeedDays[feed.export] == day || current.FetchedAt.In(time.Local).Format("2006-01-02") == day {
		a.core.SeasonMu.Unlock()
		return
	}
	a.core.ShellFeedDays[feed.export] = day
	a.core.SeasonMu.Unlock()
	rows, fresh, err := liveOrArchive(ctx, a.fallbackParent(), a.core.Season, feed, a.core.FallbackLog)
	a.core.SeasonMu.Lock()
	defer a.core.SeasonMu.Unlock()
	holdSeasonFeed(held, rows, fresh, err)
}

func (a *App) refreshShellFeeds(ctx context.Context) {
	pause := a.core.SeasonPause
	if pause == nil {
		pause = seasonPause
	}
	if err := pause(ctx, time.Second); err != nil {
		return
	}
	year, now := strconv.Itoa(a.core.Season), a.timeNow()
	refreshDailyShellFeed(ctx, a, &a.core.SeasonCalendar, now, archivedFeed[ingestion.RawCalendarEvent]{
		export: "calendar", parse: ingestion.ParseCalendar, bodies: a.core.History.ReferenceBodies,
		fetch: func(ctx context.Context) ([]ingestion.RawCalendarEvent, error) {
			return ingestion.FetchCalendar(ctx, a.core.MFLClient, year, ingestion.LeagueID)
		},
	})
	if err := pause(ctx, time.Second); err != nil {
		return
	}
	refreshDailyShellFeed(ctx, a, &a.core.SeasonTradeBait, now, archivedFeed[ingestion.RawTradeBait]{
		export: "tradeBait", parse: ingestion.ParseTradeBait, bodies: a.core.History.ReferenceBodies,
		fetch: func(ctx context.Context) ([]ingestion.RawTradeBait, error) {
			return ingestion.FetchTradeBait(ctx, a.core.MFLClient, year, ingestion.LeagueID)
		},
	})
}
