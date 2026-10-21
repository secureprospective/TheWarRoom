package main

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/core"

	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/injuries"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/livescoring"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/pendingtrades"
	feedtransactions "github.com/secureprospective/TheWarRoom/internal/ingestion/transactions"
	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
	"github.com/secureprospective/TheWarRoom/internal/mflkey"
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
	if ctx.Err() != nil {
		return
	}
	defer a.core.SeasonChanged(ctx)
	a.refreshSeasonTransactions(ctx)
	a.core.RefreshCommsBoards(ctx)
	a.refreshShellFeeds(ctx)
	pause := a.core.SeasonPause
	if pause == nil {
		pause = seasonPause
	}
	if err := pause(ctx, time.Second); err != nil {
		return
	}
	a.refreshSeasonLineups(ctx)
	// Lineups are what Franchise HQ opens on: tell the views now, not after the whole refresh.
	a.core.SeasonChanged(ctx)
	if err := pause(ctx, time.Second); err != nil {
		return
	}
	a.refreshPendingTrades(ctx)
	if err := pause(ctx, time.Second); err != nil {
		return
	}
	a.refreshInjuries(ctx)
	if err := pause(ctx, time.Second); err != nil {
		return
	}
	a.core.RefreshDraftReportsFeeds(ctx)
	a.refreshMarketFeeds(ctx, false)
	if err := pause(ctx, time.Second); err != nil {
		return
	}
	a.refreshRace(ctx)
	a.refreshScoreFirsts(ctx)
	// The current NFL week's schedule and the byes are held like the race, so League Pulse › Now
	// never waits on its own refresh (which can run before the week is known).
	if week := a.marketWeek(); week >= 1 {
		if err := pause(ctx, time.Second); err != nil {
			return
		}
		a.refreshNowWeek(ctx, week, false)
	}
	a.revalidateReadyRosterMoves(ctx)
}

// refreshInjuries pulls MFL's NFL injury designations for the IR rules, live with the archive
// as fallback, and holds the last good copy like the other season feeds.
func (a *App) refreshInjuries(ctx context.Context) {
	rows, fresh, err := liveOrArchive(ctx, a.fallbackParent(), a.core.Season,
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
	var value []leaguefeed.Injury
	if err == nil {
		value, err = injuries.ToInjuries(rows)
	}
	a.core.SeasonMu.Lock()
	defer a.core.SeasonMu.Unlock()
	holdSeasonFeed(&a.core.SeasonInjuries, value, fresh, err)
}

// heldInjuries returns the held injuries feed as the snapshot's eligibility input.
func (a *App) heldInjuries() snapshot.Injuries {
	a.core.SeasonMu.Lock()
	defer a.core.SeasonMu.Unlock()
	statuses := make(map[string]string, len(a.core.SeasonInjuries.Value))
	for _, row := range a.core.SeasonInjuries.Value {
		statuses[row.ID.String()] = row.Status
	}
	return snapshot.Injuries{
		Status:     statuses,
		Provenance: a.core.SeasonInjuries.Provenance("injuries", "injuries"),
	}
}

func (a *App) refreshSeasonTransactions(ctx context.Context) {
	rows, fresh, err := liveOrArchive(ctx, a.fallbackParent(), a.core.Season,
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
	var value []leaguefeed.Transaction
	if err == nil {
		value, err = feedtransactions.ToTransactions(rows[0])
	}
	a.core.SeasonMu.Lock()
	defer a.core.SeasonMu.Unlock()
	holdSeasonFeed(&a.core.SeasonTransactions, value, fresh, err)
}

func (a *App) refreshSeasonLineups(ctx context.Context) {
	a.core.WeekMu.Lock()
	week := a.core.Week.Number
	a.core.WeekMu.Unlock()
	rows, fresh, err := liveOrArchive(ctx, a.fallbackParent(), a.core.Season, archivedFeed[livescoring.RawLineups]{
		Export: livescoring.Export,
		Fetch: func(ctx context.Context) ([]livescoring.RawLineups, error) {
			raw, err := livescoring.Fetch(ctx, a.core.MFLClient, strconv.Itoa(a.core.Season), ingestion.LeagueID, week)
			if err != nil {
				return nil, fmt.Errorf("season lineups: %w", err)
			}
			if week != 0 && raw.Week != strconv.Itoa(week) {
				return nil, fmt.Errorf("season lineups: requested week %d, received %s", week, raw.Week)
			}
			return []livescoring.RawLineups{raw}, nil
		},
		Parse: func(body []byte) ([]livescoring.RawLineups, error) {
			raw, err := livescoring.Parse(body)
			if err != nil {
				return nil, fmt.Errorf("season lineups: %w", err)
			}
			if week != 0 && raw.Week != strconv.Itoa(week) {
				return nil, fmt.Errorf("season lineups: requested week %d, received %s", week, raw.Week)
			}
			return []livescoring.RawLineups{raw}, nil
		},
		Bodies: func(ctx context.Context, part string, accept func(string, []byte, time.Time) bool) (bool, error) {
			return a.core.History.ArchivedBodies(ctx, part, func(src string, body []byte, at time.Time) bool {
				u, err := url.Parse(src)
				if err != nil || u.Query().Get("W") != lineupWeekParam(week) {
					return false
				}
				return accept(src, body, at)
			})
		},
	}, a.core.FallbackLog)
	var value leaguefeed.Lineups
	if err == nil {
		value, err = livescoring.ToLineups(rows[0])
	}
	a.core.SeasonMu.Lock()
	defer a.core.SeasonMu.Unlock()
	// A kept copy must not masquerade as the newly requested week's lineups either.
	if a.core.LineupsWeek != week {
		a.core.SeasonLineups = heldSeasonFeed[leaguefeed.Lineups]{}
	}
	a.core.LineupsWeek = week
	holdSeasonFeed(&a.core.SeasonLineups, value, fresh, err)
}

func lineupWeekParam(week int) string {
	if week == 0 {
		return ""
	}
	return strconv.Itoa(week)
}

func (a *App) refreshPendingTrades(ctx context.Context) {
	// Read the keyring first: with no key, not even the unkeyed host discovery goes out.
	a.core.KeyMu.Lock()
	generation := a.core.KeyGeneration
	var err error
	if a.core.KeyStore == nil {
		err = mflkey.ErrUnavailable
	} else {
		_, err = a.core.KeyStore.Get(ctx)
	}
	a.core.KeyMu.Unlock()
	var value []leaguefeed.PendingTrade
	if err == nil {
		raw, ferr := pendingtrades.Fetch(ctx, a.core.MFLClient, strconv.Itoa(a.core.Season), ingestion.LeagueID)
		err = ferr
		if err == nil {
			value, err = pendingtrades.ToPendingTrades(raw)
		}
	}
	a.core.KeyMu.Lock()
	defer a.core.KeyMu.Unlock()
	if generation != a.core.KeyGeneration {
		return
	}
	a.core.SeasonMu.Lock()
	defer a.core.SeasonMu.Unlock()
	switch {
	case errors.Is(err, mflkey.ErrNoKey):
		a.clearPendingTrades()
	case errors.Is(err, mflkey.ErrUnavailable), errors.Is(err, mflkey.ErrTimeout):
		a.core.SeasonPendingTrades.RefreshError = "MFL keyring is locked or unavailable"
	default:
		at := a.timeNow()
		// The desk's read stands on its own: failing to save it into Offers history is logged, never
		// reported as an MFL failure.
		if rerr := a.recordOffers(ctx, value, at, err); rerr != nil && err == nil {
			a.core.FallbackLog(rerr.Error())
		}
		holdSeasonFeed(&a.core.SeasonPendingTrades, value, liveFreshness(at), err)
	}
}

// The caller holds seasonMu; keyGeneration prevents an older fetch from republishing after deletion.
func (a *App) clearPendingTrades() {
	a.core.SeasonPendingTrades = heldSeasonFeed[[]leaguefeed.PendingTrade]{RefreshError: "MFL not connected"}
}

// refreshPendingTradesInBackground follows a key change. Shutdown cancels workCtx, so it never
// waits out a slow fetch; tests that skip startup run it under the app context.
func (a *App) refreshPendingTradesInBackground() {
	parent := a.core.WorkCtx
	if parent == nil {
		parent = a.fallbackParent()
	}
	a.startWorker(func() {
		ctx, cancel := context.WithTimeout(parent, refreshTimeout)
		defer cancel()
		a.core.RefreshMu.Lock()
		defer a.core.RefreshMu.Unlock()
		a.refreshPendingTrades(ctx)
		a.core.SeasonChanged(ctx)
	})
}
