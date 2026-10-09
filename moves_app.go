package main

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/envelope"
	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
)

// TargetHandOff marks a Ready move handed off, saves it, then opens its MFL page; the owner
// submits on MFL and the watcher looks for the change. Nothing is saved or opened on an error.
func (a *App) TargetHandOff(correlationID string) (envelope.Receipt, error) {
	if err := a.ready(); err != nil {
		return envelope.Receipt{}, fmt.Errorf("target hand-off: startup: %w", err)
	}
	ctx, cancel := context.WithTimeout(a.ctx, m2Timeout)
	defer cancel()
	a.movesMu.Lock()
	defer a.movesMu.Unlock()
	e, err := a.moves.Get(ctx, correlationID)
	if err != nil {
		return envelope.Receipt{}, fmt.Errorf("target hand-off: load: %w", err)
	}
	next, err := e.HandOff(a.movesNow())
	if err != nil {
		return envelope.Receipt{}, fmt.Errorf("target hand-off: %w", err)
	}
	target := next.Receipt().Spec.Target.URL
	if err := validateMoveURL(target); err != nil {
		return envelope.Receipt{}, fmt.Errorf("target hand-off from %s: %w", e.State(), err)
	}
	if err := a.moves.Save(ctx, next); err != nil {
		return envelope.Receipt{}, fmt.Errorf("target hand-off: save: %w", err)
	}
	a.openURL(a.ctx, target)
	a.logMove(next)
	a.movesChanged(a.ctx)
	a.wakeMoves()
	return next.Receipt(), nil
}

func validateMoveURL(target string) error {
	u, err := url.Parse(target)
	if err != nil {
		return fmt.Errorf("move URL: %w", err)
	}
	host := strings.ToLower(u.Hostname())
	if u.Scheme != "https" || u.User != nil ||
		(host != "myfantasyleague.com" && !strings.HasSuffix(host, ".myfantasyleague.com")) {
		return fmt.Errorf("move URL: HTTPS MFL host without user info required")
	}
	return nil
}

// TargetCheckMoves asks the watcher for a pass now (rate limited to one a minute); it returns at
// once.
func (a *App) TargetCheckMoves() error {
	if err := a.ready(); err != nil {
		return fmt.Errorf("target check moves: startup: %w", err)
	}
	a.wakeMoves()
	return nil
}

func (a *App) wakeMoves() {
	select {
	case a.movesWake <- struct{}{}:
	default:
	}
}

func (a *App) logMove(e envelope.Envelope) {
	r := e.Receipt()
	entry := r.Audit[len(r.Audit)-1]
	a.movesLog(fmt.Sprintf("move %s %s → %s: %s", e.ID(), entry.From, entry.To, entry.Note))
}

func movePredicate(intent string) envelope.Predicate {
	registry := map[string]envelope.Predicate{
		"roster.ir":    envelope.IRPredicate{},
		"roster.taxi":  envelope.TaxiPredicate{},
		"lineup.set":   envelope.LineupPredicate{},
		"trade.accept": envelope.TradePredicate{},
	}
	return registry[intent]
}

func (a *App) moveSources(entries []envelope.Envelope) map[envelope.Source]bool {
	sources := make(map[envelope.Source]bool)
	for _, e := range entries {
		intent := e.Receipt().Spec.Intent
		predicate := movePredicate(intent)
		if predicate == nil {
			if !a.movesUnknown[intent] {
				a.movesUnknown[intent] = true
				a.movesLog(fmt.Sprintf("move intent %s: no landing predicate", intent))
			}
			continue
		}
		for _, source := range predicate.Sources() {
			sources[source] = true
		}
	}
	return sources
}

func (a *App) checkMoves(ctx context.Context) error {
	entries, err := a.moves.Awaiting(ctx)
	if err != nil {
		return fmt.Errorf("watch moves: awaiting: %w", err)
	}
	if len(entries) == 0 {
		return nil
	}
	sources := a.moveSources(entries)
	if len(sources) == 0 {
		return nil
	}
	obs, err := a.refreshedObservation(ctx, sources)
	if err != nil {
		return err
	}
	// Bindings take movesMu too, so it covers the local observe-and-save only, never MFL.
	a.movesMu.Lock()
	defer a.movesMu.Unlock()
	changed := false
	defer func() {
		if changed {
			a.movesChanged(ctx)
		}
	}()
	for _, listed := range entries {
		predicate := movePredicate(listed.Receipt().Spec.Intent)
		if predicate == nil {
			continue
		}
		// Re-read: a binding may have moved it while MFL was being fetched.
		e, err := a.moves.Get(ctx, listed.ID())
		if err != nil {
			return fmt.Errorf("watch moves: reload %s: %w", listed.ID(), err)
		}
		if e.State() != listed.State() {
			continue
		}
		next, err := e.Observe(a.movesNow(), obs, predicate)
		if errors.Is(err, envelope.ErrStaleObservation) {
			continue
		}
		if err != nil {
			return fmt.Errorf("watch moves: observe %s: %w", e.ID(), err)
		}
		if next.State() == e.State() {
			continue
		}
		if err := a.moves.Save(ctx, next); err != nil {
			return fmt.Errorf("watch moves: save %s: %w", e.ID(), err)
		}
		changed = true
		a.logMove(next)
	}
	return nil
}

func (a *App) refreshedObservation(
	ctx context.Context, sources map[envelope.Source]bool,
) (envelope.Observation, error) {
	a.refreshMu.Lock()
	defer a.refreshMu.Unlock()
	if err := a.refreshMoveSources(ctx, sources); err != nil {
		return envelope.Observation{}, err
	}
	return a.moveObservation(ctx)
}

func (a *App) refreshMoveSources(ctx context.Context, sources map[envelope.Source]bool) error {
	pause := a.seasonPause
	if pause == nil {
		pause = seasonPause
	}
	entries, err := a.moves.Awaiting(ctx)
	if err != nil {
		return fmt.Errorf("watch moves: latest hand-off: %w", err)
	}
	var latest time.Time
	for _, e := range entries {
		for _, entry := range e.Receipt().Audit {
			if entry.Event == envelope.HandOff && entry.At.After(latest) {
				latest = entry.At
			}
		}
	}
	first := true
	for _, source := range []envelope.Source{
		envelope.Rosters, envelope.Transactions, envelope.Lineups, envelope.PendingTrades,
	} {
		if !sources[source] || a.recentMoveSource(source, latest) {
			continue
		}
		if !first {
			if err := pause(ctx, time.Second); err != nil {
				return fmt.Errorf("watch moves: spacing: %w", err)
			}
		}
		first = false
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("watch moves: refresh: %w", err)
		}
		if err := a.refreshMoveSource(ctx, source); err != nil {
			return err
		}
	}
	return nil
}

func (a *App) refreshMoveSource(ctx context.Context, source envelope.Source) error {
	if a.movesRefresh != nil {
		if err := a.movesRefresh(ctx, source); err != nil {
			return fmt.Errorf("watch moves: refresh %s: %w", source, err)
		}
		return nil
	}
	switch source {
	case envelope.Rosters:
		if _, err := a.refreshLeagueLocked(ctx, a.rulebook, a.league); err != nil {
			a.movesLog(fmt.Sprintf("watch moves: roster refresh: %v", err))
		}
	case envelope.Transactions:
		a.refreshSeasonTransactions(ctx)
	case envelope.Lineups:
		a.refreshSeasonLineups(ctx)
	case envelope.PendingTrades:
		a.refreshPendingTrades(ctx)
	}
	return nil
}

func (a *App) moveObservation(ctx context.Context) (envelope.Observation, error) {
	snap, err := snapshot.Build(ctx, snapshot.NewSource(a.league, a.rulebook), snapshot.Directory{})
	if err != nil {
		return envelope.Observation{}, fmt.Errorf("watch moves: roster snapshot: %w", err)
	}
	season, err := a.TargetSeason()
	if err != nil {
		return envelope.Observation{}, fmt.Errorf("watch moves: held season: %w", err)
	}
	// The mirror's time is its last content change. A successful refresh since then confirmed the
	// rosters unchanged, so they count as observed then: without this a move never reaches Not yet
	// done and its deadline never fires. The caller holds refreshMu.
	rosters := snap.Rosters
	fresh := &rosters.Provenance.Freshness
	if changed, err := time.Parse(time.RFC3339, fresh.FetchedAt); err == nil &&
		fresh.State != domain.FreshFail && a.rostersCheckedAt.After(changed) {
		fresh.FetchedAt = a.rostersCheckedAt.UTC().Format(time.RFC3339)
	}
	return envelope.Observation{
		LeagueID: ingestion.LeagueID, Rosters: rosters,
		Transactions: season.Transactions, Lineups: season.Lineups, PendingTrades: season.PendingTrades,
	}, nil
}

// recentMoveSource: a season feed fetched successfully within the last minute and after the
// latest hand-off is already the evidence a pass needs (the hourly season refresh wakes the watcher).
func (a *App) recentMoveSource(source envelope.Source, handed time.Time) bool {
	a.seasonMu.Lock()
	defer a.seasonMu.Unlock()
	var at time.Time
	var failure string
	switch source {
	case envelope.Transactions:
		at, failure = a.seasonTransactions.fetchedAt, a.seasonTransactions.refreshError
	case envelope.Lineups:
		at, failure = a.seasonLineups.fetchedAt, a.seasonLineups.refreshError
	case envelope.PendingTrades:
		at, failure = a.seasonPendingTrades.fetchedAt, a.seasonPendingTrades.refreshError
	case envelope.Rosters: // not a season feed
		return false
	default:
		return false
	}
	now := a.movesNow()
	return failure == "" && at.After(handed) && !at.After(now) && now.Sub(at) < time.Minute
}
