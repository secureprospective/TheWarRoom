package main

import (
	"context"
	"slices"
	"strconv"
	"strings"
	"time"

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
	return report.FeedTime(feed.FeedState(), zone)
}

func refreshDailyShellFeed[T any](ctx context.Context, a *App, held *heldSeasonFeed[[]T], now time.Time,
	feed archivedFeed[T]) {
	day := now.In(time.Local).Format("2006-01-02")
	current := readShellFeed(ctx, a, held, feed.Export, feed.Parse)
	a.core.SeasonMu.Lock()
	if a.core.ShellFeedDays == nil {
		a.core.ShellFeedDays = map[string]string{}
	}
	if a.core.ShellFeedDays[feed.Export] == day || current.FetchedAt.In(time.Local).Format("2006-01-02") == day {
		a.core.SeasonMu.Unlock()
		return
	}
	a.core.ShellFeedDays[feed.Export] = day
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
		Export: "calendar", Parse: ingestion.ParseCalendar, Bodies: a.core.History.ReferenceBodies,
		Fetch: func(ctx context.Context) ([]ingestion.RawCalendarEvent, error) {
			return ingestion.FetchCalendar(ctx, a.core.MFLClient, year, ingestion.LeagueID)
		},
	})
	if err := pause(ctx, time.Second); err != nil {
		return
	}
	refreshDailyShellFeed(ctx, a, &a.core.SeasonTradeBait, now, archivedFeed[ingestion.RawTradeBait]{
		Export: "tradeBait", Parse: ingestion.ParseTradeBait, Bodies: a.core.History.ReferenceBodies,
		Fetch: func(ctx context.Context) ([]ingestion.RawTradeBait, error) {
			return ingestion.FetchTradeBait(ctx, a.core.MFLClient, year, ingestion.LeagueID)
		},
	})
}

func (a *App) mergeMFLCalendar(ctx context.Context, now time.Time, clock *snapshot.Sourced[leagueclock.Reading]) {
	held := readShellFeed(ctx, a, &a.core.SeasonCalendar, "calendar", ingestion.ParseCalendar)
	events := make([]leagueclock.Event, 0, len(held.Value))
	for _, r := range held.Value {
		var at time.Time
		if r.Start != "" {
			seconds, _ := strconv.ParseInt(r.Start, 10, 64)
			at = time.Unix(seconds, 0).UTC()
		}
		label := r.Title
		if label == "" {
			label = strings.ToLower(strings.ReplaceAll(r.Type, "_", " "))
		}
		events = append(events, leagueclock.Event{ID: "mfl-calendar:" + r.ID, Kind: "MFL · " + label, Status: "PLANNED", At: at})
	}
	added := leagueclock.Clock(now, leagueclock.Inputs{Events: events})
	// A calendar event that has passed is history, not a deadline: the clock lists what is coming.
	upcoming := slices.DeleteFunc(added.Deadlines, func(d leagueclock.Deadline) bool { return d.At != nil && d.At.Before(now) })
	clock.Value.Deadlines = append(clock.Value.Deadlines, upcoming...)
	slices.SortFunc(clock.Value.Deadlines, func(a, b leagueclock.Deadline) int {
		if a.Urgency != b.Urgency {
			if a.Urgency > b.Urgency {
				return -1
			}
			return 1
		}
		if a.At != nil && b.At != nil && !a.At.Equal(*b.At) {
			return a.At.Compare(*b.At)
		}
		if a.ID < b.ID {
			return -1
		}
		if a.ID > b.ID {
			return 1
		}
		return 0
	})
	clock.Provenance.Source += "+held-mfl-calendar"
	if held.FetchedAt.IsZero() {
		clock.Provenance.Freshness.Note += "; MFL calendar unavailable; showing app clocks only"
	} else {
		clock.Provenance.Freshness.Note += "; MFL calendar " + shellFeedTime(held, time.Local)
	}
}
