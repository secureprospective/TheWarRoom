package composition

import (
	"context"
	"fmt"
	"time"

	platformapp "github.com/secureprospective/TheWarRoom/internal/app/platform"

	contractsapp "github.com/secureprospective/TheWarRoom/internal/app/contracts"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/engine/eligibility"
	"github.com/secureprospective/TheWarRoom/internal/envelope"
	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
)

func (a *App) handOffMove(correlationID string) (envelope.Receipt, error) {
	if err := a.ready(); err != nil {
		return envelope.Receipt{}, fmt.Errorf("target hand-off: startup: %w", err)
	}
	ctx, cancel := context.WithTimeout(a.core.Ctx, m2Timeout)
	defer cancel()
	a.core.MovesMu.Lock()
	defer a.core.MovesMu.Unlock()
	return a.moveActions().HandOff(ctx, correlationID) //nolint:wrapcheck // Preserve move transition and authorization errors.
}

func (a *App) requestMovesPass() error { return platformapp.RequestMovesPass(a.ready, a.wakeMoves) } //nolint:wrapcheck // Preserve startup errors.

func (a *App) wakeMoves() { a.core.WakeMoves() }

func (a *App) logMove(e envelope.Envelope) { a.core.MovesLog(platformapp.MoveLog(e)) }

// withBidFacts gives a bid plan's predicate the contract it looks for, which lives in bid_facts.

func (a *App) checkMoves(ctx context.Context) error { return a.moveWatch().Check(ctx) } //nolint:wrapcheck // Preserve watcher errors.

func (a *App) refreshedObservation(
	ctx context.Context, sources map[envelope.Source]bool,
) (envelope.Observation, error) {
	a.core.RefreshMu.Lock()
	defer a.core.RefreshMu.Unlock()
	return platformapp.ReadMoveObservation(ctx, sources, a.refreshMoveSources, a.moveObservation) //nolint:wrapcheck // Preserve observation errors.
}

func (a *App) refreshMoveSources(ctx context.Context, sources map[envelope.Source]bool) error {
	return (platformapp.MoveRefresh{Pause: a.seasonPauseFunc, Awaiting: a.core.Moves.Awaiting, Recent: a.recentMoveSource, Refresh: a.refreshMoveSource}).Run(ctx, sources) //nolint:wrapcheck // Preserve watcher refresh errors.
}
func (a *App) refreshMoveSource(ctx context.Context, source envelope.Source) error {
	return (platformapp.MoveSourceRefresh{ //nolint:wrapcheck // Preserve source refresh errors.
		Override: a.core.MovesRefresh, Rosters: func(ctx context.Context) error {
			_, err := a.refreshLeagueLocked(ctx, a.core.Rulebook, a.core.League)
			return err
		}, Transactions: a.refreshSeasonTransactions, Lineups: a.refreshSeasonLineups, Trades: a.refreshPendingTrades, Log: a.core.MovesLog}).Run(ctx, source)
}

func (a *App) moveObservation(ctx context.Context) (envelope.Observation, error) {
	snap, err := snapshot.Build(ctx, snapshot.NewSource(a.core.League, a.core.Rulebook), snapshot.Directory{}, snapshot.Injuries{})
	if err != nil {
		return envelope.Observation{}, fmt.Errorf("watch moves: roster snapshot: %w", err)
	}
	season, err := a.controlAPI().TargetSeason()
	if err != nil {
		return envelope.Observation{}, fmt.Errorf("watch moves: held season: %w", err)
	}
	// The mirror's time is its last content change. A successful refresh since then confirmed the
	// rosters unchanged, so they count as observed then: without this a move never reaches Not yet
	// done and its deadline never fires. The caller holds refreshMu.
	rosters := snap.Rosters
	fresh := &rosters.Provenance.Freshness
	if changed, err := time.Parse(time.RFC3339, fresh.FetchedAt); err == nil &&
		fresh.State != domain.FreshFail && a.core.RostersCheckedAt.After(changed) {
		fresh.FetchedAt = a.core.RostersCheckedAt.UTC().Format(time.RFC3339)
	}
	m, _, _ := a.core.League.ContractsView()
	contracts := []envelope.ContractEvidence{}
	for _, p := range m.Players {
		contracts = append(contracts, contractsapp.Evidence(m, p.FranchiseID, p.MFLID))
	}
	return envelope.Observation{
		Contracts: snapshot.Sourced[[]envelope.ContractEvidence]{Value: contracts, Provenance: rosters.Provenance},
		LeagueID:  ingestion.LeagueID, Rosters: rosters,
		Transactions: season.Transactions, Lineups: season.Lineups, PendingTrades: season.PendingTrades,
	}, nil
}

// recentMoveSource: a season feed fetched successfully within the last minute and after the
// latest hand-off is already the evidence a pass needs (the hourly season refresh wakes the watcher).
func (a *App) recentMoveSource(source envelope.Source, handed time.Time) bool {
	a.core.SeasonMu.Lock()
	defer a.core.SeasonMu.Unlock()
	return (platformapp.MoveSourceFeeds{Transactions: a.core.SeasonTransactions.FeedState(), Lineups: a.core.SeasonLineups.FeedState(), Pending: a.core.SeasonPendingTrades.FeedState()}).Recent(source, handed, a.core.MovesNow)
}

// revalidateReadyRosterMoves returns Ready IR and taxi plans whose player's held eligibility
// turned ineligible to Draft, through the envelope's own route (Ready may reach Draft only via
// Invalidate to Stale, then Rebase). Unknown eligibility never yanks a plan: MFL decides, as
// it does at hand-off. The caller holds refreshMu.
func (a *App) revalidateReadyRosterMoves(ctx context.Context) {
	a.revalidateReleaseAfterRefresh(ctx)
	if a.core.Moves != nil && a.core.Whatif != nil {
		a.core.MovesMu.Lock()
		err := contractsapp.RevalidateContractPlans(ctx, a.core)
		a.core.MovesMu.Unlock()
		if err != nil {
			a.core.MovesLog(fmt.Sprintf("contract validation: %v", err))
		}
	}
	if err := a.ready(); err != nil {
		return
	}
	receipts, err := a.core.Moves.List(ctx, ingestion.LeagueID, "")
	if err != nil {
		a.core.MovesLog(fmt.Sprintf("revalidate moves: list: %v", err))
		return
	}
	candidates := []string{}
	for _, r := range receipts {
		if r.State == envelope.Ready &&
			(r.Spec.Intent == "roster.ir" || r.Spec.Intent == "roster.taxi") {
			candidates = append(candidates, r.CorrelationID)
		}
	}
	if len(candidates) == 0 {
		return
	}
	dir := a.targetDirectory(ctx)
	snap, err := snapshot.Build(ctx, snapshot.NewSource(a.core.League, a.core.Rulebook), dir, a.heldInjuries())
	if err != nil {
		a.core.MovesLog(fmt.Sprintf("revalidate moves: build: %v", err))
		return
	}
	a.core.MovesMu.Lock()
	defer a.core.MovesMu.Unlock()
	changed := false
	for _, id := range candidates {
		decision, found := a.planDecision(ctx, snap, id)
		if a.returnPlanToDraft(ctx, id, decision, found) {
			changed = true
		}
	}
	if changed {
		a.core.MovesChanged(ctx)
	}
}

// planDecision reads the held verdict for one plan's move slot: the IR plan watches the IR
// slot, the taxi plan the taxi slot. The caller holds refreshMu.
func (a *App) planDecision(ctx context.Context, snap snapshot.Snapshot, id string) (eligibility.Decision, bool) {
	return a.rosterPlanReview().Decision(ctx, snap, id)
}

// returnPlanToDraft walks one Ready plan through the envelope's only route back to Draft
// (Invalidate, then Rebase), both entries carrying the reason. The caller holds movesMu.
func (a *App) returnPlanToDraft(ctx context.Context, id string, decision eligibility.Decision, found bool) bool {
	return a.rosterPlanReview().ReturnToDraft(ctx, id, decision, found)
}

func (a *App) revalidateReleaseAfterRefresh(ctx context.Context) {
	if a.core.Moves != nil && a.core.Whatif != nil {
		a.core.MovesMu.Lock()
		err := contractsapp.RevalidateReleasePlans(ctx, a.core)
		a.core.MovesMu.Unlock()
		if err != nil {
			a.core.MovesLog(fmt.Sprintf("release validation: %v", err))
		}
	}
}

func (a *App) withdrawRosterMove(correlationID string) (envelope.Receipt, error) {
	if err := a.ready(); err != nil {
		return envelope.Receipt{}, fmt.Errorf("withdraw roster: startup: %w", err)
	}
	ctx, cancel := context.WithTimeout(a.core.Ctx, m2Timeout)
	defer cancel()
	a.core.MovesMu.Lock()
	defer a.core.MovesMu.Unlock()
	return a.moveActions().Withdraw(ctx, correlationID) //nolint:wrapcheck // Preserve move transition and authorization errors.
}

func (a *App) moveActions() platformapp.MoveActions {
	return platformapp.MoveActions{Store: a.core.Moves, Own: a.core.RequireOwnFranchise, Now: a.core.MovesNow,
		Open: func(target string) { a.core.OpenURL(a.core.Ctx, target) }, Audit: a.logMove,
		Changed: func() { a.core.MovesChanged(a.core.Ctx) }, Wake: a.wakeMoves}
}
func (a *App) moveWatch() platformapp.MoveWatch {
	return platformapp.MoveWatch{Store: a.core.Moves, Unknown: a.core.MovesUnknown, Log: a.core.MovesLog,
		Audit: a.logMove, Now: a.core.MovesNow, Observation: a.refreshedObservation,
		Lock: a.core.MovesMu.Lock, Unlock: a.core.MovesMu.Unlock, Changed: a.core.MovesChanged}
}
func (a *App) rosterPlanReview() platformapp.RosterPlanReview {
	return platformapp.RosterPlanReview{Store: a.core.Moves, Now: a.core.MovesNow, Log: a.core.MovesLog, Audit: a.logMove}
}

func newMovesTimer(delay time.Duration) (<-chan time.Time, func()) {
	timer := time.NewTimer(delay)
	return timer.C, func() { timer.Stop() }
}

func (a *App) watchMoves(parent context.Context) { a.movePolling().Run(parent) }
func (a *App) movePolling() platformapp.MovePolling {
	return platformapp.MovePolling{Now: a.core.MovesNow, Timer: a.core.MovesTimer, Wake: a.core.MovesWake, Timeout: refreshTimeout, Check: a.checkMoves, Log: a.core.MovesLog, Lock: a.core.MovesMu.Lock, Unlock: a.core.MovesMu.Unlock, Awaiting: a.core.Moves.Awaiting}
}
