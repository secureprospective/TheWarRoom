package core

import (
	"context"
	"fmt"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/report"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/ingestion"
)

// FallbackTimeout bounds a fallback read of the fetch archive.
const FallbackTimeout = report.FallbackTimeout

// ArchivedFeed is an MFL export served live, or from its newest good body in the fetch archive
// when the fetch fails. Every live fetch is archived by the transport, so nothing is cached here.
type ArchivedFeed[T any] struct {
	Export string
	// leagueless marks one of MFL's league-less exports, fetched without L from the api host:
	// its archived copies match with or without the league id, since MFL serves the same
	// NFL-wide feed to every league.
	Leagueless bool
	Fetch      func(context.Context) ([]T, error)
	Parse      func([]byte) ([]T, error)
	// bodies offers archived bodies whose URL contains part, newest first, until accept takes one.
	Bodies func(ctx context.Context, part string, accept func(string, []byte, time.Time) bool) (bool, error)
}

// LiveOrArchive fetches the feed, or falls back to the newest archived body of the same export,
// season and league that still parses. A body that proved bad is passed over, so a glitch MFL
// served once never becomes the fallback. Only a failed fetch with no usable body is an error.
// logLine names the feed and the error once whenever the live fetch fails, so run.log shows why
// a feed went stale or empty; it is the app's fallbackLog seam, log.Print in production.
func LiveOrArchive[T any](
	ctx, fallback context.Context, season int, f ArchivedFeed[T], logLine func(string),
) ([]T, domain.Freshness, error) {
	rows, ferr := f.Fetch(ctx)
	if ferr == nil {
		return rows, LiveFreshness(time.Now()), nil
	}
	rctx, cancel := context.WithTimeout(fallback, FallbackTimeout)
	defer cancel()
	rows, at, found, err := ReadArchivedFeed(rctx, season, f)
	switch {
	case err != nil:
		logLine(fmt.Sprintf("the war room: %s: live fetch failed and the archive is unreadable: %v", f.Export, ferr))
		return nil, domain.Freshness{}, fmt.Errorf("%s fetch failed (%w) and the archive is unreadable: %v", f.Export, ferr, err)
	case !found:
		logLine(fmt.Sprintf("the war room: %s: live fetch failed with no archived copy to serve: %v", f.Export, ferr))
		return nil, domain.Freshness{}, fmt.Errorf("%s fetch failed with no archived copy to fall back on: %w", f.Export, ferr)
	}
	logLine(fmt.Sprintf("the war room: %s: live fetch failed, serving the archived copy fetched %s: %v",
		f.Export, at.UTC().Format(time.RFC3339), ferr))
	return rows, StaleFreshness(at, ferr), nil
}

func ReadArchivedFeed[T any](ctx context.Context, season int, f ArchivedFeed[T]) ([]T, time.Time, bool, error) {
	var rows []T
	var at time.Time
	found, err := f.Bodies(ctx, "TYPE="+f.Export, func(src string, body []byte, fetched time.Time) bool {
		if !SameExport(src, f.Export, season, f.Leagueless) {
			return false
		}
		parsed, perr := f.Parse(body)
		if perr != nil {
			return false
		}
		rows, at = parsed, fetched
		return true
	})
	if err != nil {
		return nil, at, false, fmt.Errorf("%s archive: %w", f.Export, err)
	}
	return rows, at, found, nil
}

// SameExport reports whether an archived URL is this league's export for season; a league-less
// export's copies match with or without the league id.
func SameExport(raw, export string, season int, leagueless bool) bool {
	if leagueless {
		return ingestion.SameLeaguelessExport(raw, export, season)
	}
	return ingestion.SameExport(raw, export, season)
}

// LiveFreshness marks a successful fetch without changing its wire provenance.
func LiveFreshness(t time.Time) domain.Freshness { return report.LiveFreshness(t) }

func StaleFreshness(at time.Time, cause error) domain.Freshness {
	return report.StaleFreshness(at, cause)
}

// ArchivedOnly reads held reference exports without contacting MFL.
func ArchivedOnly[T any](ctx context.Context, s *State, export string, leagueless bool,
	parse func([]byte) ([]T, error)) HeldFeed[[]T] {
	feed := HeldFeed[[]T]{}
	if s.History == nil {
		feed.RefreshError = "No held archive"
		return feed
	}
	found, err := s.History.ReferenceBodies(ctx, "TYPE="+export, func(src string, body []byte, at time.Time) bool {
		if !SameExport(src, export, s.Season, leagueless) {
			return false
		}
		rows, err := parse(body)
		if err != nil {
			return false
		}
		feed.Value, feed.FetchedAt = rows, at
		return true
	})
	if err != nil {
		feed.RefreshError = "Held archive unreadable: " + err.Error()
	} else if !found {
		feed.RefreshError = "No usable held " + export + " export"
	}
	return feed
}
