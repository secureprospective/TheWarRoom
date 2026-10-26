package composition

import (
	"context"
	"fmt"
	"strconv"
	"time"

	marketapp "github.com/secureprospective/TheWarRoom/internal/app/market"
	poolapp "github.com/secureprospective/TheWarRoom/internal/app/offers/pool"
	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/freeagents"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
)

// refreshFreeAgents pulls MFL's free-agent list live, with the archive as fallback, and holds
// the last good copy like the other season feeds. The caller holds refreshMu.
func (a *App) refreshFreeAgents(ctx context.Context) {
	a.restoreFreeAgents(ctx)
	rows, fresh, err := liveOrArchive(ctx, a.fallbackParent(), a.core.Season,
		archivedFeed[freeagents.RawFreeAgent]{
			Export: freeagents.Export,
			Fetch: func(ctx context.Context) ([]freeagents.RawFreeAgent, error) {
				raw, ferr := freeagents.Fetch(ctx, a.core.MFLClient, strconv.Itoa(a.core.Season), ingestion.LeagueID)
				if ferr != nil {
					return nil, fmt.Errorf("free agents refresh: %w", ferr)
				}
				return raw, nil
			},
			Parse: func(body []byte) ([]freeagents.RawFreeAgent, error) {
				raw, perr := freeagents.Parse(body)
				if perr != nil {
					return nil, fmt.Errorf("free agents refresh: %w", perr)
				}
				return raw, nil
			},
			Bodies: a.core.History.ArchivedBodies,
		}, a.core.FallbackLog)
	value, err := poolapp.FreeAgentIDs(rows, err)
	a.core.SeasonMu.Lock()
	defer a.core.SeasonMu.Unlock()
	poolapp.HoldFreeAgents(&a.core.SeasonFreeAgents, value, fresh, err, func() { a.core.ModelDirty.Store(true) })
}

// restoreFreeAgents makes the first refresh compare with the last held list across launches.
func (a *App) restoreFreeAgents(ctx context.Context) {
	a.core.SeasonMu.Lock()
	held := !a.core.SeasonFreeAgents.FetchedAt.IsZero()
	a.core.SeasonMu.Unlock()
	poolapp.RestoreFreeAgents(ctx, held, a.core.Season, a.core.History.ArchivedBodies,
		func(ids []playerid.PlayerID, at time.Time) {
			a.core.SeasonMu.Lock()
			holdSeasonFeed(&a.core.SeasonFreeAgents, ids, liveFreshness(at), nil)
			a.core.SeasonMu.Unlock()
		}, a.core.FallbackLog)
}

// refreshOffersPool fetches the free-agent list from MFL. Without force it skips a copy fetched
// in the last few minutes, so opening the pool twice asks MFL once.
func (a *App) refreshOffersPool(force bool) error {
	//nolint:wrapcheck // Preserve the pool refresh refusal.
	return poolapp.RefreshPool(force, a.ready, func() (time.Time, string) {
		a.core.SeasonMu.Lock()
		held := a.core.SeasonFreeAgents
		a.core.SeasonMu.Unlock()
		return held.FetchedAt, held.RefreshError
	}, time.Since, func() error { return newMarketAPI(a).TargetMarketReportsRefresh(false) }, func() {
		ctx, cancel := context.WithTimeout(a.core.Ctx, refreshTimeout)
		defer cancel()
		a.core.RefreshMu.Lock()
		defer a.core.RefreshMu.Unlock()
		a.refreshFreeAgents(ctx)
		a.refreshMarketFeeds(ctx, force)
	})
}

func (a *App) poolAssetValues(ctx context.Context, ids []playerid.PlayerID, at time.Time) (map[string]AssetValue, string) {
	a.hydrateResearchContext(ctx)
	model := a.marketHeld.SnapshotModel()
	return marketapp.PoolAssetValues(researchView{researchInput: a.heldResearchInput()}.onField(), model, ids, at)
}
