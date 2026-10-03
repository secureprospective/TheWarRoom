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

// cacheReadTimeout bounds the local fallback reads: single-row SQLite reads, so the budget is
// small and unrelated to the network budget.
const cacheReadTimeout = 5 * time.Second

// fallbackParent is the context for local fallback reads and cache writes: the app-lifetime
// context, never the per-call one. A fetch most often fails by timing out, which kills the
// per-call context, and a fallback derived from it would fail too.
func (a *App) fallbackParent() context.Context {
	if a.ctx == nil {
		return context.Background()
	}
	return a.ctx
}

// standingsOrCache decides whether M2 is built from live MFL standings or the last good cache.
// The cache is written only after a fetch that succeeded and validated, so a bad payload can
// never evict a good copy. A failed cache write is reported in the note, not as an error: live
// data in hand is still served. Only a failed fetch with nothing cached is an error.
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
		wCtx, wCancel := context.WithTimeout(a.fallbackParent(), cacheReadTimeout)
		defer wCancel()
		//nolint:contextcheck // NOT inheriting ctx is the whole point: a fetch that
		if perr := a.state.PutStandings(wCtx, string(payload), now); perr != nil {
			fresh.Note = fmt.Sprintf("standings not cached: %v", perr)
		}
		return standings, fresh, nil
	}

	fbCtx, fbCancel := context.WithTimeout(a.fallbackParent(), cacheReadTimeout)
	defer fbCancel()

	//nolint:contextcheck // Deliberately NOT the caller's context — it is typically already
	payload, at, cerr := a.state.CachedStandings(fbCtx)
	if cerr != nil {
		// Headline the fetch failure: it is the actionable cause. A cache miss adds nothing; any other
		// cache error is a local fault the user would never see otherwise, so it is appended.
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
		// A 32-team league never has zero standings rows: an empty cache entry is corruption.
		return nil, Freshness{}, fmt.Errorf("live fetch failed (%v) and cached standings were empty", ferr)
	}
	return cached, staleFreshness(at, ferr), nil
}

// currentPhaseLabel reads the season phase for labelling only; a failure returns "". It uses
// its own context because the caller's is usually dead after a failed fetch, which is exactly
// when the "final" label matters.
func (a *App) currentPhaseLabel() string {
	ctx, cancel := context.WithTimeout(a.fallbackParent(), cacheReadTimeout)
	defer cancel()
	ph, err := a.state.CurrentPhase(ctx)
	if err != nil {
		return ""
	}
	return string(ph)
}
