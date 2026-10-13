package main

import (
	"context"
	"strconv"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/ingestion"
)

func readShellFeed[T any](ctx context.Context, a *App, held *heldSeasonFeed[[]T], export string,
	parse func([]byte) ([]T, error)) heldSeasonFeed[[]T] {
	a.seasonMu.Lock()
	feed := *held
	a.seasonMu.Unlock()
	if !feed.fetchedAt.IsZero() {
		return dailyShellReading(feed, a.timeNow())
	}
	feed = archivedOnly(ctx, a, export, false, parse)
	a.seasonMu.Lock()
	defer a.seasonMu.Unlock()
	if held.fetchedAt.IsZero() {
		*held = feed
	}
	return dailyShellReading(*held, a.timeNow())
}

func dailyShellReading[T any](feed heldSeasonFeed[[]T], now time.Time) heldSeasonFeed[[]T] {
	feed.readPending = !feed.fetchedAt.IsZero() && feed.refreshError == "" &&
		feed.fetchedAt.In(time.Local).Format("2006-01-02") != now.In(time.Local).Format("2006-01-02")
	return feed
}

func shellFeedTime[T any](feed heldSeasonFeed[[]T], zone *time.Location) string {
	if feed.fetchedAt.IsZero() {
		return "unavailable · no held copy"
	}
	kind := "live"
	if feed.refreshError != "" || feed.readPending {
		kind = "archived copy"
	}
	text := kind + " · fetched " + reportTime(feed.fetchedAt, zone)
	if feed.readPending {
		text += " · " + pendingShellRead(feed.fetchedAt, zone)
	} else if feed.refreshError != "" {
		text += " · Refresh failed"
	}
	return text
}

func refreshDailyShellFeed[T any](ctx context.Context, a *App, held *heldSeasonFeed[[]T], now time.Time,
	feed archivedFeed[T]) {
	day := now.In(time.Local).Format("2006-01-02")
	current := readShellFeed(ctx, a, held, feed.export, feed.parse)
	a.seasonMu.Lock()
	if a.shellFeedDays == nil {
		a.shellFeedDays = map[string]string{}
	}
	if a.shellFeedDays[feed.export] == day || current.fetchedAt.In(time.Local).Format("2006-01-02") == day {
		a.seasonMu.Unlock()
		return
	}
	a.shellFeedDays[feed.export] = day
	a.seasonMu.Unlock()
	rows, fresh, err := liveOrArchive(ctx, a.fallbackParent(), a.season, feed, a.fallbackLog)
	a.seasonMu.Lock()
	defer a.seasonMu.Unlock()
	holdSeasonFeed(held, rows, fresh, err)
}

func (a *App) refreshShellFeeds(ctx context.Context) {
	pause := a.seasonPause
	if pause == nil {
		pause = seasonPause
	}
	if err := pause(ctx, time.Second); err != nil {
		return
	}
	year, now := strconv.Itoa(a.season), a.timeNow()
	refreshDailyShellFeed(ctx, a, &a.seasonCalendar, now, archivedFeed[ingestion.RawCalendarEvent]{
		export: "calendar", parse: ingestion.ParseCalendar, bodies: a.history.ReferenceBodies,
		fetch: func(ctx context.Context) ([]ingestion.RawCalendarEvent, error) {
			return ingestion.FetchCalendar(ctx, a.mflClient, year, ingestion.LeagueID)
		},
	})
	if err := pause(ctx, time.Second); err != nil {
		return
	}
	refreshDailyShellFeed(ctx, a, &a.seasonTradeBait, now, archivedFeed[ingestion.RawTradeBait]{
		export: "tradeBait", parse: ingestion.ParseTradeBait, bodies: a.history.ReferenceBodies,
		fetch: func(ctx context.Context) ([]ingestion.RawTradeBait, error) {
			return ingestion.FetchTradeBait(ctx, a.mflClient, year, ingestion.LeagueID)
		},
	})
}
