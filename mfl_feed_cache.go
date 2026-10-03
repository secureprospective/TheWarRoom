package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/leagueschedule"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/leaguestandings"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

// cacheReadTimeout bounds the local cache reads and writes; they are single-row SQLite calls.
const cacheReadTimeout = 5 * time.Second

// fallbackParent is the context for cache reads and writes: the app-lifetime one, never the
// per-call one. A fetch usually fails by timing out, which kills the per-call context, and a
// fallback derived from it would fail too.
func (a *App) fallbackParent() context.Context {
	if a.ctx == nil {
		return context.Background()
	}
	return a.ctx
}

// feedCache is an MFL feed served live, or from its last good copy when the fetch fails.
type feedCache[T any] struct {
	name     string // for messages: "standings", "schedule"
	fetch    func(context.Context) ([]T, error)
	put      func(context.Context, string, time.Time) error
	cached   func(context.Context) (string, time.Time, error)
	noCached error // the store's "nothing cached yet" sentinel
}

// liveOrCache fetches the feed and caches the result, or falls back to the last good copy. The
// cache is written only after a fetch that validated, so a bad payload never evicts a good one.
// A failed cache write is noted, not returned: live data in hand is still served. Only a failed
// fetch with nothing usable cached is an error.
func liveOrCache[T any](ctx, fallback context.Context, f feedCache[T]) ([]T, Freshness, error) {
	rows, ferr := f.fetch(ctx)
	if ferr == nil {
		now := time.Now()
		fresh := liveFreshness(now)
		payload, err := json.Marshal(rows)
		if err != nil {
			fresh.Note = fmt.Sprintf("%s not cached (encode failed): %v", f.name, err)
			return rows, fresh, nil
		}
		wctx, cancel := context.WithTimeout(fallback, cacheReadTimeout)
		defer cancel()
		if err := f.put(wctx, string(payload), now); err != nil {
			fresh.Note = fmt.Sprintf("%s not cached: %v", f.name, err)
		}
		return rows, fresh, nil
	}

	rctx, cancel := context.WithTimeout(fallback, cacheReadTimeout)
	defer cancel()
	payload, at, err := f.cached(rctx)
	if errors.Is(err, f.noCached) {
		return nil, Freshness{}, fmt.Errorf("%s fetch failed with no cached fallback: %w", f.name, ferr)
	}
	if err != nil {
		// The fetch failure is the actionable cause; the cache fault is appended because the
		// user would otherwise never see it.
		return nil, Freshness{}, fmt.Errorf("%s fetch failed (%w) and the local cache is unreadable: %v", f.name, ferr, err)
	}
	var cached []T
	if err := json.Unmarshal([]byte(payload), &cached); err != nil {
		return nil, Freshness{}, fmt.Errorf("%s fetch failed (%v) and the cached copy could not be decoded: %w", f.name, ferr, err)
	}
	if len(cached) == 0 {
		// The store refuses empty payloads, so an empty cached copy is corruption.
		return nil, Freshness{}, fmt.Errorf("%s fetch failed (%v) and the cached copy was empty", f.name, ferr)
	}
	return cached, staleFreshness(at, ferr), nil
}

func (a *App) standingsOrCache(ctx context.Context) ([]leaguestandings.RawStanding, Freshness, error) {
	return liveOrCache(ctx, a.fallbackParent(), feedCache[leaguestandings.RawStanding]{
		name: "standings",
		fetch: func(ctx context.Context) ([]leaguestandings.RawStanding, error) {
			return leaguestandings.Fetch(ctx, a.mflClient, ingestion.SeasonYear, ingestion.LeagueID)
		},
		put:      a.state.PutStandings,
		cached:   a.state.CachedStandings,
		noCached: state.ErrNoCachedStandings,
	})
}

func (a *App) leagueScheduleOrCache(ctx context.Context) ([]leagueschedule.RawScheduleWeek, Freshness, error) {
	return liveOrCache(ctx, a.fallbackParent(), feedCache[leagueschedule.RawScheduleWeek]{
		name: "schedule",
		fetch: func(ctx context.Context) ([]leagueschedule.RawScheduleWeek, error) {
			return leagueschedule.Fetch(ctx, a.mflClient, ingestion.SeasonYear, ingestion.LeagueID)
		},
		put:      a.state.PutLeagueSchedule,
		cached:   a.state.CachedLeagueSchedule,
		noCached: state.ErrNoCachedLeagueSchedule,
	})
}
