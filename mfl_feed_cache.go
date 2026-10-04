package main

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/leagueschedule"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/leaguestandings"
)

// fallbackTimeout bounds a fallback read of the fetch archive.
const fallbackTimeout = 10 * time.Second

// fallbackParent is the context for fallback reads: the app-lifetime one, never the per-call
// one. A fetch usually fails by timing out, which kills the per-call context, and a fallback
// derived from it would fail too.
func (a *App) fallbackParent() context.Context {
	if a.ctx == nil {
		return context.Background()
	}
	return a.ctx
}

// archivedFeed is an MFL export served live, or from its newest good body in the fetch archive
// when the fetch fails. Every live fetch is archived by the transport, so nothing is cached here.
type archivedFeed[T any] struct {
	export string
	fetch  func(context.Context) ([]T, error)
	parse  func([]byte) ([]T, error)
	// bodies offers archived bodies whose URL contains part, newest first, until accept takes one.
	bodies func(ctx context.Context, part string, accept func(string, []byte, time.Time) bool) (bool, error)
}

// liveOrArchive fetches the feed, or falls back to the newest archived body of the same export,
// season and league that still parses. A body that proved bad is passed over, so a glitch MFL
// served once never becomes the fallback. Only a failed fetch with no usable body is an error.
func liveOrArchive[T any](ctx, fallback context.Context, season int, f archivedFeed[T]) ([]T, Freshness, error) {
	rows, ferr := f.fetch(ctx)
	if ferr == nil {
		return rows, liveFreshness(time.Now()), nil
	}
	rctx, cancel := context.WithTimeout(fallback, fallbackTimeout)
	defer cancel()
	var at time.Time
	found, err := f.bodies(rctx, "TYPE="+f.export, func(src string, body []byte, fetched time.Time) bool {
		if !sameExport(src, f.export, season) {
			return false
		}
		parsed, perr := f.parse(body)
		if perr != nil {
			return false
		}
		rows, at = parsed, fetched
		return true
	})
	switch {
	case err != nil:
		return nil, Freshness{}, fmt.Errorf("%s fetch failed (%w) and the archive is unreadable: %v", f.export, ferr, err)
	case !found:
		return nil, Freshness{}, fmt.Errorf("%s fetch failed with no archived copy to fall back on: %w", f.export, ferr)
	}
	return rows, staleFreshness(at, ferr), nil
}

// sameExport reports whether an archived URL is this league's export for season.
func sameExport(raw, export string, season int) bool {
	u, err := url.Parse(raw)
	if err != nil {
		return false
	}
	q := u.Query()
	return q.Get("TYPE") == export && q.Get("L") == ingestion.LeagueID &&
		u.Path == "/"+strconv.Itoa(season)+"/export"
}

func (a *App) standingsOrArchive(ctx context.Context) ([]leaguestandings.RawStanding, Freshness, error) {
	return liveOrArchive(ctx, a.fallbackParent(), a.season, archivedFeed[leaguestandings.RawStanding]{
		export: leaguestandings.Export,
		fetch: func(ctx context.Context) ([]leaguestandings.RawStanding, error) {
			return leaguestandings.Fetch(ctx, a.mflClient, strconv.Itoa(a.season), ingestion.LeagueID)
		},
		parse:  leaguestandings.Parse,
		bodies: a.history.ArchivedBodies,
	})
}

func (a *App) leagueScheduleOrArchive(ctx context.Context) ([]leagueschedule.RawScheduleWeek, Freshness, error) {
	return liveOrArchive(ctx, a.fallbackParent(), a.season, archivedFeed[leagueschedule.RawScheduleWeek]{
		export: leagueschedule.Export,
		fetch: func(ctx context.Context) ([]leagueschedule.RawScheduleWeek, error) {
			return leagueschedule.Fetch(ctx, a.mflClient, strconv.Itoa(a.season), ingestion.LeagueID)
		},
		parse:  leagueschedule.Parse,
		bodies: a.history.ArchivedBodies,
	})
}
