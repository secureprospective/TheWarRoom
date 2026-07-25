package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/leaguestandings"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

// This file is the M2 DEGRADATION SEAM — the one place that decides whether the Power
// Rankings board is built from live MFL standings or from the last-known-good cache. It
// is split out of m2_app.go (400-line cap, AD-14/AD-17), and the split is a good one on
// its own terms: m2_app.go is about BLENDING numbers, this is about WHERE THEY CAME FROM
// and how honestly we can say so. Keeping the freshness decision in one function means
// there is exactly one place to audit when asking "can this board lie about its age?".

// cacheReadTimeout bounds the local fallback reads (cached standings, season phase). These
// are single-row SQLite reads on a local file — either near-instant or something is badly
// wrong — so the budget is small and deliberately unrelated to the network budget.
const cacheReadTimeout = 5 * time.Second

// fallbackParent returns the context to derive local fallback reads from. It is the
// APP-LIFETIME context, never the per-call one: by the time a fallback is needed the
// per-call context is usually already dead (a fetch that timed out is the most common
// failure), and reusing it would kill the recovery path along with the thing it is
// recovering from. Background when a.ctx is nil keeps this safe at startup and in tests.
func (a *App) fallbackParent() context.Context {
	if a.ctx == nil {
		return context.Background()
	}
	return a.ctx
}

// standingsOrCache is the M2 degradation seam. It tries the live MFL fetch first; on
// success it refreshes the cache and reports live. On failure it falls back to the
// last-known-good cached payload and reports stale, carrying the fetch error as the note
// so the cause stays visible. Only a failed fetch with NOTHING cached returns an error —
// the caller turns that into the fail state.
//
// Ordering matters: the cache is written only AFTER a fetch that both succeeded and
// shape-validated (leaguestandings.Fetch validates before returning), so a garbage or
// empty payload can never evict a good cached copy. That is the property that makes the
// fallback trustworthy — a cache that a bad fetch can poison is worse than no cache.
//
// A cache-write failure is deliberately NOT fatal: we have live data in hand, and refusing
// to serve it because we could not save a copy would turn a disk problem into an outage.
// It degrades the NEXT failed fetch, not this successful one, so it is reported through
// the freshness note rather than as an error.
func (a *App) standingsOrCache(ctx context.Context) ([]leaguestandings.RawStanding, Freshness, error) {
	standings, ferr := leaguestandings.Fetch(ctx, a.mflClient, ingestion.SeasonYear, ingestion.LeagueID)
	if ferr == nil {
		now := time.Now()
		fresh := liveFreshness(now)
		payload, merr := json.Marshal(standings)
		if merr != nil {
			fresh.Note = fmt.Sprintf("standings not cached (encode failed): %v", merr)
			return standings, fresh, nil
		}
		// The WRITE gets a fresh context for the same reason the fallback read does: a
		// fetch that succeeded on the last of its budget would leave ctx nearly expired,
		// and the cache would silently never populate — so the NEXT outage would have
		// nothing to fall back on. The failure would be invisible until it mattered.
		wCtx, wCancel := context.WithTimeout(a.fallbackParent(), cacheReadTimeout)
		defer wCancel()
		//nolint:contextcheck // NOT inheriting ctx is the whole point: a fetch that
		// succeeded on the last of its budget leaves ctx expired, and the cache would
		// silently never populate. See fallbackParent.
		if perr := a.state.PutStandings(wCtx, string(payload), now); perr != nil {
			fresh.Note = fmt.Sprintf("standings not cached: %v", perr)
		}
		return standings, fresh, nil
	}

	// The fallback read MUST NOT inherit the fetch's deadline. The most likely way the
	// fetch fails is by TIMING OUT — at which point ctx is already expired, and reusing it
	// would make the local SQLite read fail instantly too. That would report "no data" while
	// a perfectly good cached board sat in the database, defeating the entire cache in
	// exactly the scenario it exists for (GLM review, B-5, lead 1: a genuine hole).
	// fallbackCtx is derived from the app-lifetime context so it survives the dead one.
	fbCtx, fbCancel := context.WithTimeout(a.fallbackParent(), cacheReadTimeout)
	defer fbCancel()

	//nolint:contextcheck // Deliberately NOT the caller's context — it is typically already
	// expired here (a timed-out fetch is the commonest way to reach this line), and
	// inheriting it would kill the recovery path along with the thing being recovered from.
	payload, at, cerr := a.state.CachedStandings(fbCtx)
	if cerr != nil {
		// Report the ORIGINAL fetch failure as the headline — it is the actionable cause.
		// A plain cache MISS is an expected non-event and adds nothing. Anything else
		// (unreadable row, corrupt timestamp) is a LOCAL fault the user would otherwise
		// never learn about, because it hides behind an MFL error they will blame instead,
		// so that one is appended rather than swallowed.
		if errors.Is(cerr, state.ErrNoCachedStandings) {
			return nil, Freshness{}, fmt.Errorf("standings fetch failed with no cached fallback: %w", ferr)
		}
		return nil, Freshness{}, fmt.Errorf(
			"standings fetch failed (%w) and the local cache is unreadable: %v", ferr, cerr)
	}
	var cached []leaguestandings.RawStanding
	if err := json.Unmarshal([]byte(payload), &cached); err != nil {
		return nil, Freshness{}, fmt.Errorf(
			"live fetch failed (%v) and cached standings could not be decoded: %w", ferr, err)
	}
	if len(cached) == 0 {
		// Mirrors the fetcher's errEmptyStandings guard: a 32-team league never
		// legitimately has zero standings rows, so an empty cache entry is corruption,
		// not data. Serving it would blank the board while claiming to be stale data.
		return nil, Freshness{}, fmt.Errorf("live fetch failed (%v) and cached standings were empty", ferr)
	}
	return cached, staleFreshness(at, ferr), nil
}

// currentPhaseLabel reads the season phase for display labelling only. A failure returns
// "" rather than an error: the phase decorates the board ("final — season complete") and
// must never be able to fail a board that otherwise has every number it needs.
//
// It takes NO caller context on purpose. The phase is a local SQLite read, and the label
// was previously lost precisely when a stale board most needed it — the caller's context
// is typically dead after a failed fetch, so inheriting it dropped the "FINAL" label at
// exactly the moment the user was trying to work out why the board looked odd.
func (a *App) currentPhaseLabel() string {
	ctx, cancel := context.WithTimeout(a.fallbackParent(), cacheReadTimeout)
	defer cancel()
	ph, err := a.state.CurrentPhase(ctx)
	if err != nil {
		return ""
	}
	return string(ph)
}
