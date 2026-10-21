package main

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"time"

	poolapp "github.com/secureprospective/TheWarRoom/internal/app/offers/pool"
	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/freeagents"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
)

const poolFreshFor = 5 * time.Minute

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
	var value []playerid.PlayerID
	if err == nil {
		value, err = freeagents.ToPlayerIDs(rows)
		sortFreeAgentIDs(value)
	}
	a.core.SeasonMu.Lock()
	defer a.core.SeasonMu.Unlock()
	if err == nil && !slices.Equal(a.core.SeasonFreeAgents.Value, value) {
		a.core.ModelDirty.Store(true)
	}
	holdSeasonFeed(&a.core.SeasonFreeAgents, value, fresh, err)
}

// restoreFreeAgents makes the first refresh compare with the last held list across launches.
func (a *App) restoreFreeAgents(ctx context.Context) {
	a.core.SeasonMu.Lock()
	held := !a.core.SeasonFreeAgents.FetchedAt.IsZero()
	a.core.SeasonMu.Unlock()
	if held {
		return
	}
	_, err := a.core.History.ArchivedBodies(ctx, "TYPE="+freeagents.Export, func(src string, body []byte, at time.Time) bool {
		if !sameExport(src, freeagents.Export, a.core.Season, false) {
			return false
		}
		rows, err := freeagents.Parse(body)
		if err != nil {
			return false
		}
		ids, err := freeagents.ToPlayerIDs(rows)
		if err != nil {
			return false
		}
		sortFreeAgentIDs(ids)
		a.core.SeasonMu.Lock()
		holdSeasonFeed(&a.core.SeasonFreeAgents, ids, liveFreshness(at), nil)
		a.core.SeasonMu.Unlock()
		return true
	})
	if err != nil {
		a.core.FallbackLog("engine: previous free-agent list unavailable: " + err.Error())
	}
}

// refreshOffersPool fetches the free-agent list from MFL. Without force it skips a copy fetched
// in the last few minutes, so opening the pool twice asks MFL once.
func (a *App) refreshOffersPool(force bool) error {
	if err := a.ready(); err != nil {
		return fmt.Errorf("target pool refresh: startup: %w", err)
	}
	a.core.SeasonMu.Lock()
	held := a.core.SeasonFreeAgents
	a.core.SeasonMu.Unlock()
	if !force && held.RefreshError == "" && time.Since(held.FetchedAt) < poolFreshFor {
		return newMarketAPI(a).TargetMarketReportsRefresh(false) //nolint:wrapcheck // The API preserves the existing refresh error.
	}
	ctx, cancel := context.WithTimeout(a.core.Ctx, refreshTimeout)
	defer cancel()
	a.core.RefreshMu.Lock()
	defer a.core.RefreshMu.Unlock()
	a.refreshFreeAgents(ctx)
	a.refreshMarketFeeds(ctx, force)
	return nil
}

func (a *App) poolAssetValues(ctx context.Context, ids []playerid.PlayerID, at time.Time) (map[string]AssetValue, string) {
	a.hydrateResearchContext(ctx)
	model := a.marketHeld.SnapshotModel()
	also := make([]string, 0, len(ids))
	for _, id := range ids {
		also = append(also, id.String())
	}
	board := buildAssetBoard(researchView{researchInput: a.heldResearchInput()}, model, at, also)
	out := make(map[string]AssetValue, len(board.Values))
	for _, v := range board.Values {
		out[v.ID] = v
	}
	return out, board.NowSource
}

func sortFreeAgentIDs(ids []playerid.PlayerID) { poolapp.SortIDs(ids) }
