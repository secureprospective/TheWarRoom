package main

import (
	"context"
	"fmt"
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
	// leagueless marks one of MFL's league-less exports, fetched without L from the api host:
	// its archived copies match with or without the league id, since MFL serves the same
	// NFL-wide feed to every league.
	leagueless bool
	fetch      func(context.Context) ([]T, error)
	parse      func([]byte) ([]T, error)
	// bodies offers archived bodies whose URL contains part, newest first, until accept takes one.
	bodies func(ctx context.Context, part string, accept func(string, []byte, time.Time) bool) (bool, error)
}

// liveOrArchive fetches the feed, or falls back to the newest archived body of the same export,
// season and league that still parses. A body that proved bad is passed over, so a glitch MFL
// served once never becomes the fallback. Only a failed fetch with no usable body is an error.
// logLine names the feed and the error once whenever the live fetch fails, so run.log shows why
// a feed went stale or empty; it is the app's fallbackLog seam, log.Print in production.
func liveOrArchive[T any](
	ctx, fallback context.Context, season int, f archivedFeed[T], logLine func(string),
) ([]T, Freshness, error) {
	rows, ferr := f.fetch(ctx)
	if ferr == nil {
		return rows, liveFreshness(time.Now()), nil
	}
	rctx, cancel := context.WithTimeout(fallback, fallbackTimeout)
	defer cancel()
	var at time.Time
	found, err := f.bodies(rctx, "TYPE="+f.export, func(src string, body []byte, fetched time.Time) bool {
		if !sameExport(src, f.export, season, f.leagueless) {
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
		logLine(fmt.Sprintf("the war room: %s: live fetch failed and the archive is unreadable: %v", f.export, ferr))
		return nil, Freshness{}, fmt.Errorf("%s fetch failed (%w) and the archive is unreadable: %v", f.export, ferr, err)
	case !found:
		logLine(fmt.Sprintf("the war room: %s: live fetch failed with no archived copy to serve: %v", f.export, ferr))
		return nil, Freshness{}, fmt.Errorf("%s fetch failed with no archived copy to fall back on: %w", f.export, ferr)
	}
	logLine(fmt.Sprintf("the war room: %s: live fetch failed, serving the archived copy fetched %s: %v",
		f.export, at.UTC().Format(time.RFC3339), ferr))
	return rows, staleFreshness(at, ferr), nil
}

// sameExport reports whether an archived URL is this league's export for season; a league-less
// export's copies match with or without the league id.
func sameExport(raw, export string, season int, leagueless bool) bool {
	if leagueless {
		return ingestion.SameLeaguelessExport(raw, export, season)
	}
	return ingestion.SameExport(raw, export, season)
}

func (a *App) standingsOrArchive(ctx context.Context) ([]leaguestandings.RawStanding, Freshness, error) {
	return liveOrArchive(ctx, a.fallbackParent(), a.season, archivedFeed[leaguestandings.RawStanding]{
		export: leaguestandings.Export,
		fetch: func(ctx context.Context) ([]leaguestandings.RawStanding, error) {
			return leaguestandings.Fetch(ctx, a.mflClient, strconv.Itoa(a.season), ingestion.LeagueID)
		},
		parse:  leaguestandings.Parse,
		bodies: a.history.ArchivedBodies,
	}, a.fallbackLog)
}

func (a *App) leagueScheduleOrArchive(ctx context.Context) ([]leagueschedule.RawScheduleWeek, Freshness, error) {
	return liveOrArchive(ctx, a.fallbackParent(), a.season, archivedFeed[leagueschedule.RawScheduleWeek]{
		export: leagueschedule.Export,
		fetch: func(ctx context.Context) ([]leagueschedule.RawScheduleWeek, error) {
			return leagueschedule.Fetch(ctx, a.mflClient, strconv.Itoa(a.season), ingestion.LeagueID)
		},
		parse:  leagueschedule.Parse,
		bodies: a.history.ArchivedBodies,
	}, a.fallbackLog)
}
