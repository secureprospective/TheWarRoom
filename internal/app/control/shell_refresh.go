package control

import (
	"context"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/report"
	"github.com/secureprospective/TheWarRoom/internal/domain"
)

func DailyShellReading[T any](feed report.HeldFeed[[]T], now time.Time) report.HeldFeed[[]T] {
	feed.ReadPending = !feed.FetchedAt.IsZero() && feed.RefreshError == "" &&
		feed.FetchedAt.In(time.Local).Format("2006-01-02") != now.In(time.Local).Format("2006-01-02")
	return feed
}

func ReadDailyShell[T any](ctx context.Context, held report.HeldFeed[[]T], now func() time.Time,
	archive func(context.Context) report.HeldFeed[[]T], merge func(report.HeldFeed[[]T]) report.HeldFeed[[]T]) report.HeldFeed[[]T] {
	if !held.FetchedAt.IsZero() {
		return DailyShellReading(held, now())
	}
	return merge(archive(ctx))
}

func MergeDailyShell[T any](held *report.HeldFeed[[]T], archived report.HeldFeed[[]T], now time.Time) report.HeldFeed[[]T] {
	if held.FetchedAt.IsZero() {
		*held = archived
	}
	return DailyShellReading(*held, now)
}

func ClaimShellDay[T any](days *map[string]string, export string, now time.Time, current report.HeldFeed[[]T]) bool {
	day := now.In(time.Local).Format("2006-01-02")
	if *days == nil {
		*days = map[string]string{}
	}
	if (*days)[export] == day || current.FetchedAt.In(time.Local).Format("2006-01-02") == day {
		return false
	}
	(*days)[export] = day
	return true
}

func RefreshDailyShell[T any](ctx context.Context, claim func() bool,
	read func(context.Context) ([]T, domain.Freshness, error), publish func([]T, domain.Freshness, error)) {
	if !claim() {
		return
	}
	rows, fresh, err := read(ctx)
	publish(rows, fresh, err)
}

func RefreshShell(ctx context.Context, pause func(context.Context, time.Duration) error, calendar, bait func(context.Context)) {
	if err := pause(ctx, time.Second); err != nil {
		return
	}
	calendar(ctx)
	if err := pause(ctx, time.Second); err != nil {
		return
	}
	bait(ctx)
}

func PendingRefreshBackground(parent context.Context, timeout time.Duration, lock, unlock func(), refresh, changed func(context.Context)) {
	ctx, cancel := context.WithTimeout(parent, timeout)
	defer cancel()
	lock()
	defer unlock()
	refresh(ctx)
	changed(ctx)
}
