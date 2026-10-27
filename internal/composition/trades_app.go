package composition

import (
	"context"
	"time"

	marketapp "github.com/secureprospective/TheWarRoom/internal/app/market"
	platformapp "github.com/secureprospective/TheWarRoom/internal/app/platform"

	contractsapp "github.com/secureprospective/TheWarRoom/internal/app/contracts"
	"github.com/secureprospective/TheWarRoom/internal/app/core"
	tradesapp "github.com/secureprospective/TheWarRoom/internal/app/trades"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/envelope"
	"github.com/secureprospective/TheWarRoom/internal/leagueclock"
	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

// tradesHost is the trades API's Host. It is never bound: only trades.API exposes methods to
// Wails. Composition retains lock ownership and cross-feature resource adapters.
type tradesHost struct {
	*core.State
	app *App
}

func (a *App) tradesAPI() *tradesapp.API { return tradesapp.New(tradesHost{State: a.core, app: a}) }

func (h tradesHost) TradeState(franchiseID string) (
	snapshot.Snapshot, snapshot.Sourced[[]leaguefeed.PendingTrade], error,
) {
	return h.app.tradeState(franchiseID)
}

func (h tradesHost) TradesBuildStamp() string { return buildLabel() + " " + buildDate }

func (h tradesHost) DraftTradeAccept(franchiseID string, offer leaguefeed.PendingTrade, snap snapshot.Snapshot,
	held snapshot.Sourced[[]leaguefeed.PendingTrade], id string) (envelope.Receipt, error) {
	a := h.app
	a.core.MovesMu.Lock()
	defer a.core.MovesMu.Unlock()
	return a.tradePlans().Accept(a.core.Ctx, m2Timeout, franchiseID, offer, snap, held, id) //nolint:wrapcheck // Preserve accept draft and supersession errors.
}

// TradeBuilder runs after the API's validation and readiness checks, in their original order.
func (h tradesHost) TradeBuilder(r tradesapp.RawTradeBuilder, out tradesapp.TradeBuilderView) (tradesapp.TradeBuilderView, error) {
	a := h.app
	a.core.WhatifMu.Lock()
	defer a.core.WhatifMu.Unlock()
	a.core.RefreshMu.Lock()
	defer a.core.RefreshMu.Unlock()
	ctx, cancel := context.WithTimeout(a.core.Ctx, m2Timeout)
	defer cancel()
	return (tradesapp.BuilderWorkflow{RequireOwner: a.requireTradePlanFranchise, State: a.tradeState, Teams: a.tradeBuilderTeams, Project: a.tradeProjection, Keep: a.keepTradePlan, Exit: a.draftTradeExit, Action: a.tradeBuilderPlanAction, Saved: a.savedTradePlans}).Run(ctx, r, out) //nolint:wrapcheck // Preserve builder partial output and errors.
}

// tradeState is the held snapshot and pending trades for a franchise; acquisitions reads it too.
func (a *App) tradeState(franchiseID string) (
	snapshot.Snapshot, snapshot.Sourced[[]leaguefeed.PendingTrade], error,
) {
	//nolint:wrapcheck // Preserve existing feature refusal text.
	return tradesapp.ReadState(franchiseID, func(id string) (snapshot.Snapshot, error) {
		snap, _, err := contractsapp.LineupSelection(contractsHost{State: a.core, app: a}, id, nil)
		return snap, err //nolint:wrapcheck // The shared service supplies trade-state context.
	}, a.core.PendingTrades)

}

func labelTradeReceipts(receipts []envelope.Receipt, snap snapshot.Snapshot) {
	tradesapp.LabelReceipts(receipts, snap)
}
func tradeFigure(label string, before, after domain.Money) tradesapp.Figure {
	return tradesapp.Figure(contractsapp.Figure(label, before, after))
}
func (a *App) tradeBuilderTeams(ctx context.Context, snap snapshot.Snapshot) []tradesapp.TradeBuilderTeam {
	m, _, _ := a.core.League.ContractsView()
	picks := a.core.TradeFuturePicks()
	board, err := a.assetBoard(ctx)
	values := make([]tradesapp.Value, len(board.Values))
	for i, v := range board.Values {
		values[i] = tradesapp.Value{ID: v.ID, Now: tradesapp.Cell(v.Now), Dynasty: tradesapp.Cell(v.Dynasty)}
	}
	return tradesapp.BuilderTeams(snap, m, picks, values, err,
		func(year, round int, original string) *tradesapp.Cell {
			cell := tradesapp.Cell(*a.builderPickValue(year, round, original, time.Now()))
			return &cell
		})

}

func (a *App) keepTradePlan(ctx context.Context, r tradesapp.RawTradeBuilder, blocks []string) (envelope.Receipt, error) {
	//nolint:wrapcheck // Preserve trade plan workflow errors.
	return a.tradePlans().Keep(ctx, r, blocks)
}
func (a *App) draftTradeExit(ctx context.Context, r tradesapp.RawTradeBuilder,
	held snapshot.Sourced[[]leaguefeed.PendingTrade]) (envelope.Receipt, error) {
	//nolint:wrapcheck // Preserve trade plan workflow errors.
	return a.tradePlans().Exit(ctx, r, held)
}

func (a *App) savedTradePlans(ctx context.Context, mine string, snap snapshot.Snapshot) ([]tradesapp.TradeBuilderPlan, error) {
	//nolint:wrapcheck // Preserve trade plan workflow errors.
	return a.tradePlans().Saved(ctx, mine, snap)
}

func (a *App) builderPickValue(year, round int, original string, at time.Time) *AssetCell {
	return marketapp.PickValue(a.marketHeld.SnapshotModel(), year, round, original, at)

}

func (a *App) removeTradePlan(ctx context.Context, r tradesapp.RawTradeBuilder) error {
	//nolint:wrapcheck // Preserve trade plan workflow errors.
	return a.tradePlans().Remove(ctx, r)
}

func (a *App) tradeBuilderPlanAction(ctx context.Context, r tradesapp.RawTradeBuilder, snap snapshot.Snapshot) error {
	//nolint:wrapcheck // Preserve trade plan workflow errors.
	return a.tradePlans().Action(ctx, r, snap)
}

func (a *App) tradeSimulation(ctx context.Context, mine, other string) (tradesapp.Simulation, error) {
	//nolint:wrapcheck // Preserve what-if service errors.
	return (tradesapp.SimulationWorkflow{
		Open: func(ctx context.Context, fid string) (state.WhatIfSession, *state.Store, error) {
			return contractsapp.OpenWhatIf(ctx, contractsapp.StoresOf(a.core), contractsapp.RawWhatIfRequest{Mine: fid, Action: "read"})
		},
		Mirror: func(ctx context.Context, fid string) (state.WhatIfSession, error) {
			return contractsapp.MirrorWhatIf(ctx, contractsapp.StoresOf(a.core), fid)
		},
		Load: func(ctx context.Context, session state.WhatIfSession) (*state.Store, error) {
			return contractsapp.LoadWhatIf(ctx, a.core.WhatifPools, session)
		},
		Now: time.Now,
	}).Run(ctx, mine, other)
}

func (a *App) tradeProjection(ctx context.Context, r tradesapp.RawTradeBuilder, snap snapshot.Snapshot) (tradesapp.TradeBuilderPlan, error) {
	//nolint:wrapcheck // Preserve preview errors and partial output.
	return (tradesapp.ProjectionWorkflow{Simulate: a.tradeSimulation, Stamps: a.core.League.Stamps, Possessive: possessive, ClockReasons: a.tradeClockReasons, PickReason: a.tradePickReason, Teams: a.tradeBuilderTeams, Figure: tradeFigure}).Run(ctx, r, snap)
}

func (a *App) tradeBuilderHandOff(ctx context.Context, r tradesapp.RawTradeBuilder, snap snapshot.Snapshot) error {
	//nolint:wrapcheck // Preserve handoff refusal text.
	return a.tradePlans().Handoff(ctx, r, snap)
}

func (a *App) tradePickReason(r tradesapp.RawTradeBuilder) string {
	return tradesapp.PickOwnershipReason(r.Give, r.Get, a.core.TradeFuturePicksFeed)
}

func (a *App) tradeClockReasons(ctx context.Context) []string {
	return tradesapp.ClockReasons(func() (snapshot.Sourced[leagueclock.Reading], error) { return a.targetClock(ctx) }, time.Now, time.Local)
}

func tradeCentsCorrection(p state.MirrorPlayer, afterOwner string, s state.WhatIfSession) domain.Money {
	return tradesapp.CentsCorrection(p, afterOwner, s)
}
func tradeDeadlineReasons(now time.Time, windows []leagueclock.Window) []string {
	return tradesapp.DeadlineReasons(now, windows, time.Local)
}

func (a *App) requireTradePlanFranchise(ctx context.Context, r tradesapp.RawTradeBuilder) error {
	//nolint:wrapcheck // Preserve trade plan workflow errors.
	return a.tradePlans().RequireOwner(ctx, r)
}

func (a *App) tradePlans() tradesapp.PlanWorkflow {
	return tradesapp.PlanWorkflow{Receipts: a.core.Moves, Request: (tradesHost{State: a.core, app: a}).TradeRequest, Now: a.core.MovesNow, Lock: a.core.MovesMu.Lock, Unlock: a.core.MovesMu.Unlock,
		SavePlan: func(ctx context.Context, plan state.TradePlan) error {
			return state.NewTradePlans(a.core.WhatifPools).Save(ctx, plan)
		},
		Plans: func(ctx context.Context, mine string) ([]state.TradePlan, error) {
			return state.NewTradePlans(a.core.WhatifPools).List(ctx, mine)
		},
		Own: a.requireOwnFranchise, Project: a.tradeProjection, Supersede: func(ctx context.Context, e envelope.Envelope, now time.Time) error {
			return contractsapp.SupersedeMoves(ctx, a.core.Moves, e, now)
		}, Audit: a.logMove, Changed: a.core.MovesChanged, HandOff: a.tradeBuilderHandOff, State: a.tradeState, ValidateURL: platformapp.ValidateMoveURL, OpenURL: a.core.OpenURL, Wake: a.wakeMoves}
}

func (h tradesHost) TradeRequest(franchise string, offer leaguefeed.PendingTrade) envelope.TradeRequest {
	return tradesapp.Request(franchise, offer, func() string {
		host, _ := h.ContractsLeagueHost()
		return host
	}, h.Season)
}

func (h tradesHost) RequireOwnFranchise(franchise string) error {
	return h.app.requireOwnFranchise(franchise)
}
