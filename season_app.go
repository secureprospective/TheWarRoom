package main

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/livescoring"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/pendingtrades"
	feedtransactions "github.com/secureprospective/TheWarRoom/internal/ingestion/transactions"
	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
	"github.com/secureprospective/TheWarRoom/internal/mflkey"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
)

// heldSeasonFeed is a feed's last good copy and why the latest refresh failed, if it did.
type heldSeasonFeed[T any] struct {
	value        T
	fetchedAt    time.Time
	refreshError string
}

func (h heldSeasonFeed[T]) provenance(source, feed string) snapshot.Provenance {
	fresh := Freshness{State: FreshFail, Note: feed + " not fetched yet"}
	if !h.fetchedAt.IsZero() {
		fresh = liveFreshness(h.fetchedAt)
	}
	if h.refreshError != "" {
		fresh.Note = h.refreshError
		if !h.fetchedAt.IsZero() {
			fresh.State = FreshStale
		}
	}
	return snapshot.Provenance{Kind: "live", Source: source, Freshness: fresh}
}

func holdSeasonFeed[T any](h *heldSeasonFeed[T], value T, fresh Freshness, err error) {
	if err != nil {
		h.refreshError = err.Error()
		return
	}
	at, parseErr := time.Parse(time.RFC3339, fresh.FetchedAt)
	if parseErr != nil {
		h.refreshError = "season refresh time: " + parseErr.Error()
		return
	}
	h.value, h.fetchedAt, h.refreshError = value, at, fresh.Note
}

// SeasonReading is the three season feeds as held, each with its own provenance.
type SeasonReading struct {
	Transactions  snapshot.Sourced[[]leaguefeed.Transaction]  `json:"transactions"`
	Lineups       snapshot.Sourced[leaguefeed.Lineups]        `json:"lineups"`
	PendingTrades snapshot.Sourced[[]leaguefeed.PendingTrade] `json:"pendingTrades"`
}

// TargetSeason reads held state only: no network and never refreshMu, so it answers while a
// refresh runs. Its only error is "not ready".
func (a *App) TargetSeason() (SeasonReading, error) {
	if err := a.ready(); err != nil {
		return SeasonReading{}, fmt.Errorf("target season: startup: %w", err)
	}
	a.seasonMu.Lock()
	defer a.seasonMu.Unlock()
	source := "liveScoring default week"
	if a.lineupsWeek != 0 {
		source = "liveScoring W=" + strconv.Itoa(a.lineupsWeek)
	}
	// Copies serialize as [] rather than null before the first fetch.
	txs := append([]leaguefeed.Transaction{}, a.seasonTransactions.value...)
	trades := append([]leaguefeed.PendingTrade{}, a.seasonPendingTrades.value...)
	lineups := a.seasonLineups.value
	lineups.Franchises = append([]leaguefeed.Lineup{}, lineups.Franchises...)
	lineups.Matchups = append([]leaguefeed.Matchup{}, lineups.Matchups...)
	for i := range lineups.Franchises {
		row := &lineups.Franchises[i]
		row.Players = append([]leaguefeed.PlayerScore{}, row.Players...)
	}
	return SeasonReading{
		Transactions: snapshot.Sourced[[]leaguefeed.Transaction]{
			Value: txs, Provenance: a.seasonTransactions.provenance("transactions", "transactions"),
		},
		Lineups: snapshot.Sourced[leaguefeed.Lineups]{
			Value: lineups, Provenance: a.seasonLineups.provenance(source, "liveScoring"),
		},
		PendingTrades: snapshot.Sourced[[]leaguefeed.PendingTrade]{
			Value: trades, Provenance: a.seasonPendingTrades.provenance("pendingTrades", "pendingTrades"),
		},
	}, nil
}

func seasonPause(ctx context.Context, duration time.Duration) error {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return fmt.Errorf("season refresh pause: %w", ctx.Err())
	case <-timer.C:
		return nil
	}
}

func (a *App) refreshSeason(ctx context.Context) {
	defer a.wakeMoves()
	a.refreshMu.Lock()
	defer a.refreshMu.Unlock()
	if ctx.Err() != nil {
		return
	}
	defer a.seasonChanged(ctx)
	a.refreshSeasonTransactions(ctx)
	pause := a.seasonPause
	if pause == nil {
		pause = seasonPause
	}
	if err := pause(ctx, time.Second); err != nil {
		return
	}
	a.refreshSeasonLineups(ctx)
	if err := pause(ctx, time.Second); err != nil {
		return
	}
	a.refreshPendingTrades(ctx)
}

func (a *App) refreshSeasonTransactions(ctx context.Context) {
	rows, fresh, err := liveOrArchive(ctx, a.fallbackParent(), a.season,
		archivedFeed[feedtransactions.RawTransactions]{
			export: feedtransactions.Export,
			fetch: func(ctx context.Context) ([]feedtransactions.RawTransactions, error) {
				raw, err := feedtransactions.Fetch(ctx, a.mflClient, strconv.Itoa(a.season), ingestion.LeagueID)
				if err != nil {
					return nil, fmt.Errorf("season transactions: %w", err)
				}
				return []feedtransactions.RawTransactions{raw}, nil
			},
			parse: func(body []byte) ([]feedtransactions.RawTransactions, error) {
				raw, err := feedtransactions.Parse(body)
				if err != nil {
					return nil, fmt.Errorf("season transactions: %w", err)
				}
				return []feedtransactions.RawTransactions{raw}, nil
			},
			bodies: a.history.ArchivedBodies,
		})
	var value []leaguefeed.Transaction
	if err == nil {
		value, err = feedtransactions.ToTransactions(rows[0])
	}
	a.seasonMu.Lock()
	defer a.seasonMu.Unlock()
	holdSeasonFeed(&a.seasonTransactions, value, fresh, err)
}

func (a *App) refreshSeasonLineups(ctx context.Context) {
	a.weekMu.Lock()
	week := a.week.Number
	a.weekMu.Unlock()
	rows, fresh, err := liveOrArchive(ctx, a.fallbackParent(), a.season, archivedFeed[livescoring.RawLineups]{
		export: livescoring.Export,
		fetch: func(ctx context.Context) ([]livescoring.RawLineups, error) {
			raw, err := livescoring.Fetch(ctx, a.mflClient, strconv.Itoa(a.season), ingestion.LeagueID, week)
			if err != nil {
				return nil, fmt.Errorf("season lineups: %w", err)
			}
			if week != 0 && raw.Week != strconv.Itoa(week) {
				return nil, fmt.Errorf("season lineups: requested week %d, received %s", week, raw.Week)
			}
			return []livescoring.RawLineups{raw}, nil
		},
		parse: func(body []byte) ([]livescoring.RawLineups, error) {
			raw, err := livescoring.Parse(body)
			if err != nil {
				return nil, fmt.Errorf("season lineups: %w", err)
			}
			if week != 0 && raw.Week != strconv.Itoa(week) {
				return nil, fmt.Errorf("season lineups: requested week %d, received %s", week, raw.Week)
			}
			return []livescoring.RawLineups{raw}, nil
		},
		bodies: func(ctx context.Context, part string, accept func(string, []byte, time.Time) bool) (bool, error) {
			return a.history.ArchivedBodies(ctx, part, func(src string, body []byte, at time.Time) bool {
				u, err := url.Parse(src)
				if err != nil || u.Query().Get("W") != lineupWeekParam(week) {
					return false
				}
				return accept(src, body, at)
			})
		},
	})
	var value leaguefeed.Lineups
	if err == nil {
		value, err = livescoring.ToLineups(rows[0])
	}
	a.seasonMu.Lock()
	defer a.seasonMu.Unlock()
	// A kept copy must not masquerade as the newly requested week's lineups either.
	if a.lineupsWeek != week {
		a.seasonLineups = heldSeasonFeed[leaguefeed.Lineups]{}
	}
	a.lineupsWeek = week
	holdSeasonFeed(&a.seasonLineups, value, fresh, err)
}

func lineupWeekParam(week int) string {
	if week == 0 {
		return ""
	}
	return strconv.Itoa(week)
}

func (a *App) refreshPendingTrades(ctx context.Context) {
	// Read the keyring first: with no key, not even the unkeyed host discovery goes out.
	a.keyMu.Lock()
	generation := a.keyGeneration
	var err error
	if a.keyStore == nil {
		err = mflkey.ErrUnavailable
	} else {
		_, err = a.keyStore.Get(ctx)
	}
	a.keyMu.Unlock()
	var value []leaguefeed.PendingTrade
	if err == nil {
		raw, ferr := pendingtrades.Fetch(ctx, a.mflClient, strconv.Itoa(a.season), ingestion.LeagueID)
		err = ferr
		if err == nil {
			value, err = pendingtrades.ToPendingTrades(raw)
		}
	}
	a.keyMu.Lock()
	defer a.keyMu.Unlock()
	if generation != a.keyGeneration {
		return
	}
	a.seasonMu.Lock()
	defer a.seasonMu.Unlock()
	switch {
	case errors.Is(err, mflkey.ErrNoKey):
		a.clearPendingTrades()
	case errors.Is(err, mflkey.ErrUnavailable), errors.Is(err, mflkey.ErrTimeout):
		a.seasonPendingTrades.refreshError = "MFL keyring is locked or unavailable"
	default:
		holdSeasonFeed(&a.seasonPendingTrades, value, liveFreshness(time.Now()), err)
	}
}

// The caller holds seasonMu; keyGeneration prevents an older fetch from republishing after deletion.
func (a *App) clearPendingTrades() {
	a.seasonPendingTrades = heldSeasonFeed[[]leaguefeed.PendingTrade]{refreshError: "MFL not connected"}
}

// refreshPendingTradesInBackground follows a key change. Shutdown cancels workCtx, so it never
// waits out a slow fetch; tests that skip startup run it under the app context.
func (a *App) refreshPendingTradesInBackground() {
	parent := a.workCtx
	if parent == nil {
		parent = a.fallbackParent()
	}
	a.weekWorkers.Add(1)
	go func() {
		defer a.weekWorkers.Done()
		ctx, cancel := context.WithTimeout(parent, refreshTimeout)
		defer cancel()
		a.refreshMu.Lock()
		defer a.refreshMu.Unlock()
		a.refreshPendingTrades(ctx)
		a.seasonChanged(ctx)
	}()
}
