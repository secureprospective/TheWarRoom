package composition

import (
	"context"
	"fmt"
	"time"

	platformapp "github.com/secureprospective/TheWarRoom/internal/app/platform"

	contractsapp "github.com/secureprospective/TheWarRoom/internal/app/contracts"
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
	return (platformapp.MoveObservation{ //nolint:wrapcheck // Preserve observation context.
		Snapshot: func(ctx context.Context) (snapshot.Snapshot, error) {
			return snapshot.Build(ctx, snapshot.NewSource(a.core.League, a.core.Rulebook), snapshot.Directory{}, snapshot.Injuries{})
		},
		Season: func() (envelope.Observation, error) {
			season, err := a.controlAPI().TargetSeason()
			return envelope.Observation{Transactions: season.Transactions, Lineups: season.Lineups, PendingTrades: season.PendingTrades}, err //nolint:wrapcheck // Observation service adds the original held-season context.
		}, Checked: func() time.Time { return a.core.RostersCheckedAt }, League: ingestion.LeagueID,
		Contracts: func() []envelope.ContractEvidence {
			mirror, _, _ := a.core.League.ContractsView()
			evidence := []envelope.ContractEvidence{}
			for _, p := range mirror.Players {
				evidence = append(evidence, contractsapp.Evidence(mirror, p.FranchiseID, p.MFLID))
			}
			return evidence
		},
	}).Read(ctx)
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
	(platformapp.RosterRevalidation{Release: a.revalidateReleaseAfterRefresh, ContractsAvailable: a.core.Moves != nil && a.core.Whatif != nil,
		Contracts: func(ctx context.Context) error {
			a.core.MovesMu.Lock()
			defer a.core.MovesMu.Unlock()
			return contractsapp.RevalidateContractPlans(ctx, a.core)
		}, Ready: a.ready, List: func(ctx context.Context) ([]envelope.Receipt, error) {
			return a.core.Moves.List(ctx, ingestion.LeagueID, "")
		},
		Snapshot: func(ctx context.Context) (snapshot.Snapshot, error) {
			dir := a.targetDirectory(ctx)
			return snapshot.Build(ctx, snapshot.NewSource(a.core.League, a.core.Rulebook), dir, a.heldInjuries())
		}, Lock: a.core.MovesMu.Lock, Unlock: a.core.MovesMu.Unlock, Decision: a.planDecision, Return: a.returnPlanToDraft, Changed: a.core.MovesChanged, Log: a.core.MovesLog}).Run(ctx)
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
	platformapp.RevalidateRelease(ctx, a.core.Moves != nil && a.core.Whatif != nil, func(ctx context.Context) error {
		a.core.MovesMu.Lock()
		defer a.core.MovesMu.Unlock()
		return contractsapp.RevalidateReleasePlans(ctx, a.core)
	}, a.core.MovesLog)
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
	return platformapp.MoveActions{Store: a.core.Moves, Own: a.requireOwnFranchise, Now: a.core.MovesNow,
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

func (a *App) requireOwnFranchise(franchise string) error {
	return platformapp.AuthorizeTeam(franchise, a.ready, a.core.WithTeamIdentity) //nolint:wrapcheck // Preserve shared team authorization refusals.
}
