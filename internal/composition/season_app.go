package composition

import (
	"context"
	"fmt"
	"strconv"
	"time"

	controlapp "github.com/secureprospective/TheWarRoom/internal/app/control"
	"github.com/secureprospective/TheWarRoom/internal/app/core"
	"github.com/secureprospective/TheWarRoom/internal/app/report"

	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/injuries"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/livescoring"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/pendingtrades"
	feedtransactions "github.com/secureprospective/TheWarRoom/internal/ingestion/transactions"
	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
)

type heldSeasonFeed[T any] = core.HeldFeed[T]

func holdSeasonFeed[T any](h *heldSeasonFeed[T], value T, fresh Freshness, err error) {
	core.HoldFeed(h, value, fresh, err)
}

func seasonPause(ctx context.Context, duration time.Duration) error { return core.Pause(ctx, duration) } //nolint:wrapcheck // Preserve the shared pause error.

func (a *App) refreshSeason(ctx context.Context) {
	defer a.wakeMoves()
	a.core.RefreshMu.Lock()
	defer a.core.RefreshMu.Unlock()
	(controlapp.SeasonRefresh{
		Changed: a.core.SeasonChanged, Transactions: a.refreshSeasonTransactions, Comms: a.core.RefreshCommsBoards,
		Shell: a.refreshShellFeeds, Lineups: a.refreshSeasonLineups, Trades: a.refreshPendingTrades,
		Injuries: a.refreshInjuries, Draft: a.core.RefreshDraftReportsFeeds,
		Market: func(ctx context.Context) { a.refreshMarketFeeds(ctx, false) }, Race: a.refreshRace,
		Firsts: a.refreshScoreFirsts, Revalidate: a.revalidateReadyRosterMoves,
		Pause: a.seasonPauseFunc, Week: a.marketWeek,
		Schedule: func(ctx context.Context, week int) { a.refreshNowWeek(ctx, week, false) },
	}).Run(ctx)

}

// refreshInjuries pulls MFL's NFL injury designations for the IR rules, live with the archive
// as fallback, and holds the last good copy like the other season feeds.
func (a *App) refreshInjuries(ctx context.Context) {
	controlapp.RefreshInjuries(ctx, func(ctx context.Context) ([]injuries.RawInjury, Freshness, error) {
		return liveOrArchive(ctx, a.fallbackParent(), a.core.Season,
			archivedFeed[injuries.RawInjury]{
				Export: injuries.Export,
				// The injuries export is league-less: fetched without L from the api host, so its
				// archived copies match with or without the league id.
				Leagueless: true,
				Fetch: func(ctx context.Context) ([]injuries.RawInjury, error) {
					raw, ferr := injuries.Fetch(ctx, a.core.MFLClient, strconv.Itoa(a.core.Season))
					if ferr != nil {
						return nil, fmt.Errorf("injuries refresh: %w", ferr)
					}
					return raw, nil
				},
				Parse: func(body []byte) ([]injuries.RawInjury, error) {
					raw, perr := injuries.Parse(ctx, body)
					if perr != nil {
						return nil, fmt.Errorf("injuries refresh: %w", perr)
					}
					return raw, nil
				},
				Bodies: a.core.History.ArchivedBodies,
			}, a.core.FallbackLog)
	}, func(value []leaguefeed.Injury, fresh Freshness, err error) {
		a.core.SeasonMu.Lock()
		defer a.core.SeasonMu.Unlock()
		holdSeasonFeed(&a.core.SeasonInjuries, value, fresh, err)
	})
}

// heldInjuries returns the held injuries feed as the snapshot's eligibility input.
func (a *App) heldInjuries() snapshot.Injuries {
	a.core.SeasonMu.Lock()
	defer a.core.SeasonMu.Unlock()
	return controlapp.HeldInjuries(a.core.SeasonInjuries)
}

func (a *App) refreshSeasonTransactions(ctx context.Context) {
	controlapp.RefreshTransactions(ctx, func(ctx context.Context) ([]feedtransactions.RawTransactions, Freshness, error) {
		return liveOrArchive(ctx, a.fallbackParent(), a.core.Season,
			archivedFeed[feedtransactions.RawTransactions]{
				Export: feedtransactions.Export,
				Fetch: func(ctx context.Context) ([]feedtransactions.RawTransactions, error) {
					raw, err := feedtransactions.Fetch(ctx, a.core.MFLClient, strconv.Itoa(a.core.Season), ingestion.LeagueID)
					if err != nil {
						return nil, fmt.Errorf("season transactions: %w", err)
					}
					return []feedtransactions.RawTransactions{raw}, nil
				},
				Parse: func(body []byte) ([]feedtransactions.RawTransactions, error) {
					raw, err := feedtransactions.Parse(body)
					if err != nil {
						return nil, fmt.Errorf("season transactions: %w", err)
					}
					return []feedtransactions.RawTransactions{raw}, nil
				},
				Bodies: a.core.History.ArchivedBodies,
			}, a.core.FallbackLog)
	}, func(value []leaguefeed.Transaction, fresh Freshness, err error) {
		a.core.SeasonMu.Lock()
		defer a.core.SeasonMu.Unlock()
		holdSeasonFeed(&a.core.SeasonTransactions, value, fresh, err)
	})
}

func (a *App) refreshSeasonLineups(ctx context.Context) {
	a.core.WeekMu.Lock()
	week := a.core.Week.Number
	a.core.WeekMu.Unlock()
	controlapp.RefreshLineups(ctx, func(ctx context.Context) ([]livescoring.RawLineups, Freshness, error) {
		return liveOrArchive(ctx, a.fallbackParent(), a.core.Season, archivedFeed[livescoring.RawLineups]{
			Export: livescoring.Export,
			Fetch: func(ctx context.Context) ([]livescoring.RawLineups, error) {
				raw, err := livescoring.Fetch(ctx, a.core.MFLClient, strconv.Itoa(a.core.Season), ingestion.LeagueID, week)
				if err != nil {
					return nil, fmt.Errorf("season lineups: %w", err)
				}
				if err := controlapp.CheckLineupWeek(raw, week); err != nil {
					return nil, err //nolint:wrapcheck // Preserve the requested-week refusal.
				}
				return []livescoring.RawLineups{raw}, nil
			},
			Parse: func(body []byte) ([]livescoring.RawLineups, error) {
				raw, err := livescoring.Parse(body)
				if err != nil {
					return nil, fmt.Errorf("season lineups: %w", err)
				}
				if err := controlapp.CheckLineupWeek(raw, week); err != nil {
					return nil, err //nolint:wrapcheck // Preserve the requested-week refusal.
				}
				return []livescoring.RawLineups{raw}, nil
			},
			Bodies: report.ScopedBodies(a.core.History.ArchivedBodies, "W", lineupWeekParam(week)),
		}, a.core.FallbackLog)
	}, func(value leaguefeed.Lineups, fresh Freshness, err error) {
		a.core.SeasonMu.Lock()
		defer a.core.SeasonMu.Unlock()
		controlapp.PublishLineups(&a.core.SeasonLineups, &a.core.LineupsWeek, week, value, fresh, err)
	})
}

func lineupWeekParam(week int) string {
	return controlapp.LineupWeekParam(week)
}

func (a *App) refreshPendingTrades(ctx context.Context) {
	// Read the keyring first: with no key, not even the unkeyed host discovery goes out.
	a.core.KeyMu.Lock()
	generation := a.core.KeyGeneration
	err := controlapp.PendingKey(ctx, a.core.KeyStore)
	a.core.KeyMu.Unlock()
	controlapp.RefreshPending(ctx, err, func(ctx context.Context) (pendingtrades.RawPendingTrades, error) {
		return pendingtrades.Fetch(ctx, a.core.MFLClient, strconv.Itoa(a.core.Season), ingestion.LeagueID)
	}, func(value []leaguefeed.PendingTrade, err error) {
		a.core.KeyMu.Lock()
		defer a.core.KeyMu.Unlock()
		controlapp.CurrentPending(generation, a.core.KeyGeneration, func() {
			a.core.SeasonMu.Lock()
			defer a.core.SeasonMu.Unlock()
			controlapp.PublishPending(ctx, &a.core.SeasonPendingTrades, value, err, a.timeNow, a.recordOffers, a.core.FallbackLog)
		})
	})
}

// The caller holds seasonMu; keyGeneration prevents an older fetch from republishing after deletion.
func (a *App) clearPendingTrades() {
	controlapp.ClearPending(&a.core.SeasonPendingTrades)
}

// refreshPendingTradesInBackground follows a key change. Shutdown cancels workCtx, so it never
// waits out a slow fetch; tests that skip startup run it under the app context.
func (a *App) refreshPendingTradesInBackground() {
	parent := controlapp.PendingParent(a.core.WorkCtx, a.fallbackParent)
	a.startWorker(func() {
		controlapp.PendingRefreshBackground(parent, refreshTimeout, a.core.RefreshMu.Lock, a.core.RefreshMu.Unlock, a.refreshPendingTrades, a.core.SeasonChanged)
	})
}

func (a *App) seasonPauseFunc() func(context.Context, time.Duration) error {
	return report.RefreshPause(a.core.SeasonPause)
}
