package report

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/offline"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
)

// FallbackTimeout bounds a fallback read of the fetch archive.
const FallbackTimeout = 10 * time.Second

// HeldFeed is a feed's last good copy and why the latest refresh failed, if it did.
type HeldFeed[T any] struct {
	Value        T
	FetchedAt    time.Time
	RefreshError string
	ReadPending  bool
}

// state is the feed's held read state, without its value.
func (h HeldFeed[T]) FeedState() Feed {
	return Feed{FetchedAt: h.FetchedAt, RefreshError: h.RefreshError, ReadPending: h.ReadPending}
}

func (h HeldFeed[T]) Provenance(source, feed string) snapshot.Provenance {
	fresh := domain.Freshness{State: domain.FreshFail, Note: feed + " not fetched yet"}
	if !h.FetchedAt.IsZero() {
		fresh = domain.Freshness{State: domain.FreshLive, FetchedAt: h.FetchedAt.UTC().Format(time.RFC3339)}
	}
	if h.ReadPending {
		fresh.State, fresh.Note = domain.FreshStale, PendingRead(h.FetchedAt, time.Local)
	}
	if h.RefreshError != "" {
		fresh.Note = h.RefreshError
		if !h.FetchedAt.IsZero() {
			fresh.State = domain.FreshStale
		}
	}
	return snapshot.Provenance{Kind: "live", Source: source, Freshness: fresh}
}

func HoldFeed[T any](h *HeldFeed[T], value T, fresh domain.Freshness, err error) {
	if err != nil {
		h.RefreshError = FailureText(err)
		return
	}
	at, parseErr := time.Parse(time.RFC3339, fresh.FetchedAt)
	if parseErr != nil {
		h.RefreshError = "season refresh time: " + parseErr.Error()
		return
	}
	h.Value, h.FetchedAt, h.RefreshError, h.ReadPending = value, at, fresh.Note, false
}

// LiveFreshness marks a result as served from a successful fetch at time t.
func LiveFreshness(t time.Time) domain.Freshness {
	return domain.Freshness{State: domain.FreshLive, FetchedAt: t.UTC().Format(time.RFC3339)}
}

// StaleFreshness marks a cached result after a failed fetch. The cause is shown so an MFL
// outage reads differently from a local network fault, which reads as the offline sentence.
func StaleFreshness(fetchedAt time.Time, cause error) domain.Freshness {
	note := fmt.Sprintf("live fetch failed, showing last known data: %v", cause)
	if offline.Is(cause) {
		note = "Showing the last copy saved. " + FailureText(cause)
	}
	return domain.Freshness{State: domain.FreshStale, FetchedAt: fetchedAt.UTC().Format(time.RFC3339), Note: note}
}

// Offline is what a person sees when a fetch never reached MFL. offline.Sentence words the others.
const Offline = offline.League

// FailureText is PlainFailure's text, for the held-feed notes a screen shows.
func FailureText(err error) string { return offline.Text(err) }

// ScopedBodies admits only bodies whose archive URL carries the requested parameter value.
func ScopedBodies(bodies func(context.Context, string, func(string, []byte, time.Time) bool) (bool, error), param, value string) func(context.Context, string, func(string, []byte, time.Time) bool) (bool, error) {
	return func(ctx context.Context, part string, accept func(string, []byte, time.Time) bool) (bool, error) {
		return bodies(ctx, part, func(src string, body []byte, at time.Time) bool {
			u, err := url.Parse(src)
			if err != nil || u.Query().Get(param) != value {
				return false
			}
			return accept(src, body, at)
		})
	}
}

func RefreshPause(pause func(context.Context, time.Duration) error) func(context.Context, time.Duration) error {
	if pause != nil {
		return pause
	}
	return Pause
}

func Pause(ctx context.Context, duration time.Duration) error {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return fmt.Errorf("season refresh pause: %w", ctx.Err())
	case <-timer.C:
		return nil
	}
}

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

// ArchivedOnly reads held reference exports without contacting MFL.
func ReadHeldArchive[T any](ctx context.Context, available bool, season int, source ArchivedFeed[T]) HeldFeed[[]T] {
	feed := HeldFeed[[]T]{}
	if !available {
		feed.RefreshError = "No held archive"
		return feed
	}
	found, err := source.Bodies(ctx, "TYPE="+source.Export, func(src string, body []byte, at time.Time) bool {
		if !SameExport(src, source.Export, season, source.Leagueless) {
			return false
		}
		rows, err := source.Parse(body)
		if err != nil {
			return false
		}
		feed.Value, feed.FetchedAt = rows, at
		return true
	})
	if err != nil {
		feed.RefreshError = "Held archive unreadable: " + err.Error()
	} else if !found {
		feed.RefreshError = "No usable held " + source.Export + " export"
	}
	return feed
}
