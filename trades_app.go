package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"slices"
	"strings"
	"time"

	contractsapp "github.com/secureprospective/TheWarRoom/internal/app/contracts"
	"github.com/secureprospective/TheWarRoom/internal/app/core"
	tradesapp "github.com/secureprospective/TheWarRoom/internal/app/trades"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/envelope"
	"github.com/secureprospective/TheWarRoom/internal/leagueclock"
	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
	"github.com/secureprospective/TheWarRoom/internal/transactions"
)

// tradesHost is the trades API's Host. It is never bound: only trades.API exposes methods to
// Wails. The plan lifecycle stays here because it spans the what-if, refresh and moves locks and
// calls the what-if, asset, clock and moves features' root code.
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
	now := a.core.MovesNow()
	req := a.core.TradeRequest(franchiseID, offer)
	check := envelope.TradeCheck{PendingTrades: held, OfferingName: tradesapp.FranchiseName(snap, offer.Offering)}
	if req.Target.Kind == envelope.Unmapped {
		check.Blocks = append(check.Blocks, "MFL host not yet known")
	}
	e, err := envelope.DraftTrade(now, req, snap, check, func() string { return id })
	if err != nil {
		return envelope.Receipt{}, fmt.Errorf("target draft trade accept: %w", err)
	}
	ctx, cancel := context.WithTimeout(a.core.Ctx, m2Timeout)
	defer cancel()
	if err := contractsapp.SupersedeMoves(ctx, a.core.Moves, e, now); err != nil {
		return envelope.Receipt{}, err //nolint:wrapcheck // contracts errors already name the step; wrapping would change the refusal
	}
	if err := a.core.Moves.Save(ctx, e); err != nil {
		return envelope.Receipt{}, fmt.Errorf("target draft trade accept: save: %w", err)
	}
	return e.Receipt(), nil
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
	snap, held, err := a.tradeState(r.Mine)
	if err != nil {
		return out, err
	}
	out.Teams = a.tradeBuilderTeams(ctx, snap)
	if r.Action == "preview" || r.Action == "save" {
		preview, err := a.tradeProjection(ctx, r, snap)
		if err != nil {
			return out, err
		}
		out.Preview = &preview
		if r.Action == "save" {
			receipt, err := a.keepTradePlan(ctx, r, preview.Reasons)
			if err != nil {
				return out, err
			}
			out.Preview.ID, out.Preview.Receipt = receipt.CorrelationID, &receipt
		}
	}
	if r.Action == "reject" || r.Action == "revoke" {
		receipt, err := a.draftTradeExit(ctx, r, held)
		if err != nil {
			return out, err
		}
		out.Preview = &tradesapp.TradeBuilderPlan{ID: receipt.CorrelationID, Reasons: []string{}, Teams: []tradesapp.TradeProjection{}, Receipt: &receipt}
	}
	if err := a.tradeBuilderPlanAction(ctx, r, snap); err != nil {
		return out, err
	}
	out.Plans, err = a.savedTradePlans(ctx, r.Mine, snap)
	return out, err
}

// tradeState is the held snapshot and pending trades for a franchise; acquisitions reads it too.
func (a *App) tradeState(franchiseID string) (
	snapshot.Snapshot, snapshot.Sourced[[]leaguefeed.PendingTrade], error,
) {
	// Reuse the held-snapshot franchise validation, with no player selection to parse.
	snap, _, err := contractsapp.LineupSelection(a.core, franchiseID, nil)
	if err != nil {
		return snapshot.Snapshot{}, snapshot.Sourced[[]leaguefeed.PendingTrade]{},
			fmt.Errorf("trade state: %w", err)
	}
	return snap, a.core.PendingTrades(), nil
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
	values := map[string]tradesapp.Value{}
	if err == nil {
		for _, v := range board.Values {
			values[v.ID] = tradesapp.Value{ID: v.ID, Now: tradesapp.Cell(v.Now), Dynasty: tradesapp.Cell(v.Dynasty)}
		}
	}
	return tradesapp.WireTeams(tradesapp.Teams(snap, m, picks, values, func(year, round int, original string) *tradesapp.Cell {
		cell := tradesapp.Cell(*a.builderPickValue(year, round, original, time.Now()))
		return &cell
	}))
}
func tradeID() (string, error) {
	var bytes [16]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return "", fmt.Errorf("trade ID: %w", err)
	}
	return "trade-" + hex.EncodeToString(bytes[:]), nil
}
func (a *App) keepTradePlan(ctx context.Context, r tradesapp.RawTradeBuilder, blocks []string) (envelope.Receipt, error) {
	a.core.MovesMu.Lock()
	defer a.core.MovesMu.Unlock()
	id, err := tradeID()
	if err != nil {
		return envelope.Receipt{}, fmt.Errorf("trade draft: %w", err)
	}
	req := a.core.TradeRequest(r.Mine, leaguefeed.PendingTrade{Offering: r.Mine, OfferedTo: r.Other})
	req.Trade = envelope.ExpectedTrade{TradeID: id, Offering: r.Mine, Accepting: r.Other, OfferingGives: r.Give, AcceptingGives: r.Get}
	if req.Target.Kind == envelope.Unmapped {
		blocks = append(blocks, "MFL host not yet known")
	}
	e, err := envelope.DraftTradeAct(a.core.MovesNow(), "trade.propose", req, blocks, func() string { return id })
	if err != nil {
		return envelope.Receipt{}, fmt.Errorf("trade draft: %w", err)
	}
	if err := state.NewTradePlans(a.core.WhatifPools).Save(ctx, state.TradePlan{ID: id, Mine: r.Mine, Other: r.Other, Give: r.Give, Get: r.Get, Rationale: r.Rationale}); err != nil {
		return envelope.Receipt{}, fmt.Errorf("trade draft: %w", err)
	}
	if err := a.core.Moves.Save(ctx, e); err != nil {
		return envelope.Receipt{}, fmt.Errorf("trade plan receipt save: %w", err)
	}
	a.core.MovesChanged(ctx)
	return e.Receipt(), nil
}
func (a *App) draftTradeExit(ctx context.Context, r tradesapp.RawTradeBuilder,
	held snapshot.Sourced[[]leaguefeed.PendingTrade]) (envelope.Receipt, error) {
	index := slices.IndexFunc(held.Value, func(p leaguefeed.PendingTrade) bool { return p.ID == r.ID })
	if index < 0 {
		return envelope.Receipt{}, fmt.Errorf("offer no longer pending on MFL")
	}
	p := held.Value[index]
	if (r.Action == "reject" && p.OfferedTo != r.Mine) || (r.Action == "revoke" && p.Offering != r.Mine) {
		return envelope.Receipt{}, fmt.Errorf("trade action belongs to the other owner")
	}
	blocks := []string{}
	if reason := envelope.TradeFeedReason(held.Provenance); reason != "" {
		blocks = append(blocks, "pending trades unavailable: "+reason)
	}
	req := a.core.TradeRequest(r.Mine, p)
	if req.Target.Kind == envelope.Unmapped {
		blocks = append(blocks, "MFL host not yet known")
	}
	id, err := tradeID()
	if err != nil {
		return envelope.Receipt{}, fmt.Errorf("trade draft: %w", err)
	}
	now := a.core.MovesNow()
	e, err := envelope.DraftTradeAct(now, "trade."+r.Action, req, blocks, func() string { return id })
	if err != nil {
		return envelope.Receipt{}, fmt.Errorf("trade draft: %w", err)
	}
	a.core.MovesMu.Lock()
	defer a.core.MovesMu.Unlock()
	// One live exit plan per offer, as with accept plans: a new one supersedes the old.
	if err := contractsapp.SupersedeMoves(ctx, a.core.Moves, e, now); err != nil {
		return envelope.Receipt{}, err //nolint:wrapcheck // contracts errors already name the step; wrapping would change the refusal
	}
	if err := a.core.Moves.Save(ctx, e); err != nil {
		return envelope.Receipt{}, fmt.Errorf("trade exit save: %w", err)
	}
	a.core.MovesChanged(ctx)
	return e.Receipt(), nil
}

func (a *App) savedTradePlans(ctx context.Context, mine string, snap snapshot.Snapshot) ([]tradesapp.TradeBuilderPlan, error) {
	out := []tradesapp.TradeBuilderPlan{}

	plans, err := state.NewTradePlans(a.core.WhatifPools).List(ctx, mine)
	if err != nil {
		return nil, fmt.Errorf("saved trade plans: %w", err)
	}
	for _, plan := range plans {
		e, err := a.core.Moves.Get(ctx, plan.ID)
		if err != nil {
			return nil, fmt.Errorf("saved trade receipt: %w", err)
		}
		if e.State() == envelope.Withdrawn {
			continue
		}
		p, err := a.tradeProjection(ctx, tradesapp.RawTradeBuilder{Mine: plan.Mine, Other: plan.Other, Give: plan.Give,
			Get: plan.Get, Rationale: plan.Rationale}, snap)
		if err != nil {
			return nil, err
		}
		receipt := e.Receipt()
		p.ID, p.Receipt = plan.ID, &receipt
		out = append(out, p)
	}
	return out, nil
}

func (a *App) builderPickValue(year, round int, original string, at time.Time) *AssetCell {
	model := a.marketHeld.SnapshotModel()
	if model == nil {
		return &AssetCell{Text: "Not valued", Sub: "Lab model not compiled"}
	}
	cell, _, _ := dynastyValueCells(model.Pick(year, round, original, at))
	return &cell
}

func (a *App) removeTradePlan(ctx context.Context, r tradesapp.RawTradeBuilder) error {
	a.core.MovesMu.Lock()
	defer a.core.MovesMu.Unlock()
	e, err := a.core.Moves.Get(ctx, r.ID)
	if err != nil {
		return fmt.Errorf("remove trade plan: %w", err)
	}
	if e.Receipt().Spec.FranchiseID != r.Mine {
		return fmt.Errorf("remove trade plan: belongs to another franchise")
	}
	next, err := e.WithdrawTrade(a.core.MovesNow())
	if err != nil {
		return fmt.Errorf("remove trade plan: %w", err)
	}
	if err := a.core.Moves.Save(ctx, next); err != nil {
		return fmt.Errorf("remove trade plan: save: %w", err)
	}
	a.logMove(next)
	a.core.MovesChanged(ctx)
	return nil
}

func (a *App) tradeBuilderPlanAction(ctx context.Context, r tradesapp.RawTradeBuilder, snap snapshot.Snapshot) error {
	switch r.Action {
	case "remove":
		return a.removeTradePlan(ctx, r)
	case "handoff":
		return a.tradeBuilderHandOff(ctx, r, snap)
	default:
		return nil
	}
}

type tradeSimulation struct {
	Sessions []state.WhatIfSession
	Store    *state.Store
	Before   map[string]domain.Money
}

func (a *App) tradeSimulation(ctx context.Context, mine, other string) (tradeSimulation, error) {
	out := tradeSimulation{Before: map[string]domain.Money{}}
	merged := state.MirrorSnapshot{}
	for _, fid := range []string{mine, other} {
		s, store, err := contractsapp.OpenWhatIf(ctx, contractsapp.StoresOf(a.core), contractsapp.RawWhatIfRequest{Mine: fid, Action: "read"})
		if err != nil {
			return out, err //nolint:wrapcheck // contracts errors already name the step; wrapping would change the refusal
		}
		out.Sessions = append(out.Sessions, s)
		used, _ := store.CapUsed(fid)
		out.Before[fid] = s.Used + used - s.SeedUsed
		merged.Season = s.Snapshot.Season
		for _, p := range s.Snapshot.Players {
			if p.FranchiseID != fid {
				continue
			}
			if live, ok := store.Player(p.MFLID); !ok || live.FranchiseID != fid {
				continue
			}
			merged.Players = append(merged.Players, p)
		}
		for _, adjust := range s.Snapshot.Adjustments {
			if adjust.FranchiseID == fid {
				merged.Adjustments = append(merged.Adjustments, adjust)
			}
		}
	}
	seed := out.Sessions[0]
	seed.Generation = "trade-preview:" + fmt.Sprint(time.Now().UnixNano())
	seed.Snapshot = merged
	store, err := contractsapp.LoadWhatIf(ctx, a.core.WhatifPools, seed)
	if err != nil {
		return out, err //nolint:wrapcheck // contracts errors already name the step; wrapping would change the refusal
	}
	out.Store = store
	return out, nil
}
func (a *App) tradeProjection(ctx context.Context, r tradesapp.RawTradeBuilder, snap snapshot.Snapshot) (tradesapp.TradeBuilderPlan, error) {
	out := tradesapp.TradeBuilderPlan{Reasons: []string{}, Teams: []tradesapp.TradeProjection{}}
	sim, err := a.tradeSimulation(ctx, r.Mine, r.Other)
	if err != nil {
		return out, err
	}
	stamps, err := a.core.League.Stamps(ctx)
	if err != nil {
		return out, fmt.Errorf("trade baseline stamps: %w", err)
	}
	for _, s := range sim.Sessions {
		if stamps.NewerThan(s.Stamps) {
			name := tradesapp.FranchiseName(snap, s.Mine)
			out.Reasons = append(out.Reasons, "MFL changed since "+possessive(name)+" what-if started; reset it to "+
				"check this deal against MFL as it stands.")
			out.Resets = append(out.Resets, tradesapp.TradeReset{Franchise: s.Mine, Label: "Reset " + possessive(name) + " what-if"})
		}
	}
	out.Reasons = append(out.Reasons, a.tradeClockReasons(ctx)...)
	if reason := envelope.TradeFeedReason(snap.Rosters.Provenance); reason != "" {
		out.Reasons = append(out.Reasons, "Roster unavailable: "+reason)
	}
	if reason := a.tradePickReason(r); reason != "" {
		out.Reasons = append(out.Reasons, reason)
	}
	catalog := a.tradeBuilderTeams(ctx, snap)
	moves, blocks := tradesapp.Legs(tradesapp.Input(r), tradesapp.ServiceTeams(catalog), sim.Store.Player)
	out.Reasons = append(out.Reasons, blocks...)
	before := map[string]domain.Money{}
	for _, s := range sim.Sessions {
		before[s.Mine], _ = sim.Store.CapUsed(s.Mine)
	}
	if len(blocks) == 0 && len(moves) > 0 {
		coord, err := transactions.New(sim.Store.Writer(), nil, nil)
		if err != nil {
			return out, fmt.Errorf("trade preview coordinator: %w", err)
		}
		if _, err := coord.Execute(ctx, transactions.Trade{Moves: moves, Rationale: "Local what-if trade projection"}); err != nil {
			return out, fmt.Errorf("trade preview: %w", err)
		}
	}
	players := map[string]snapshot.Player{}
	for _, p := range snap.Players.Value {
		players[p.ID.String()] = p
	}
	for _, s := range sim.Sessions {
		projected, blocks := tradesapp.ProjectedTeam(tradeProjectionInput(sim, s, before[s.Mine], players, tradesapp.FranchiseName(snap, s.Mine)), tradeFigure)
		projected.Roster = tradesapp.RichRoster(projected.Roster, tradesapp.ServiceTeams(catalog))
		team := tradesapp.WireProjection(projected)
		out.Teams, out.Reasons = append(out.Teams, team), append(out.Reasons, blocks...)
	}
	out.Post = tradesapp.DealText(tradesapp.Input(r), tradesapp.ServiceTeams(catalog), sim.Sessions[0].Snapshot)
	return out, nil
}
func (a *App) tradeBuilderHandOff(ctx context.Context, r tradesapp.RawTradeBuilder, snap snapshot.Snapshot) error {
	plans, err := state.NewTradePlans(a.core.WhatifPools).List(ctx, r.Mine)
	if err != nil {
		return fmt.Errorf("trade hand-off plans: %w", err)
	}
	blocks := []string{}
	for _, p := range plans {
		if p.ID != r.ID {
			continue
		}
		preview, err := a.tradeProjection(ctx, tradesapp.RawTradeBuilder{Mine: p.Mine, Other: p.Other, Give: p.Give, Get: p.Get, Rationale: p.Rationale}, snap)
		if err != nil {
			return err
		}
		blocks = preview.Reasons
	}
	if len(blocks) > 0 {
		return fmt.Errorf("trade hand-off blocked: %s", strings.Join(blocks, "; "))
	}
	a.core.MovesMu.Lock()
	defer a.core.MovesMu.Unlock()
	e, err := a.core.Moves.Get(ctx, r.ID)
	if err != nil {
		return fmt.Errorf("trade hand-off load: %w", err)
	}
	spec := e.Receipt().Spec
	if spec.FranchiseID != r.Mine || !envelope.IsTrade(spec.Intent) || spec.Intent == "trade.accept" {
		return fmt.Errorf("trade hand-off outside scope")
	}
	if err := a.checkTradeExitPending(r.Mine, spec); err != nil {
		return err
	}
	if err := validateMoveURL(spec.Target.URL); err != nil {
		return err
	}
	next, err := e.HandOff(a.core.MovesNow())
	if err != nil {
		return fmt.Errorf("trade hand-off: %w", err)
	}
	if err := a.core.Moves.Save(ctx, next); err != nil {
		return fmt.Errorf("trade hand-off save: %w", err)
	}
	a.core.OpenURL(ctx, spec.Target.URL)
	a.core.MovesChanged(ctx)
	a.wakeMoves()
	return nil
}

func (a *App) checkTradeExitPending(mine string, spec envelope.Spec) error {
	if spec.Intent == "trade.propose" {
		return nil
	}
	_, held, err := a.tradeState(mine)
	if err != nil {
		return err
	}
	if envelope.TradeFeedReason(held.Provenance) != "" || !slices.ContainsFunc(held.Value, func(p leaguefeed.PendingTrade) bool {
		return p.ID == spec.Expected.Trade.TradeID && envelope.MatchesPendingTrade(*spec.Expected.Trade, p)
	}) {
		return fmt.Errorf("offer no longer available; refresh the trade desk")
	}
	return nil
}

func (a *App) tradePickReason(r tradesapp.RawTradeBuilder) string {
	if !slices.ContainsFunc(append(slices.Clone(r.Give), r.Get...), func(token string) bool { return strings.HasPrefix(token, "FP_") }) {
		return ""
	}
	fetchedAt, refreshError := a.core.TradeFuturePicksFeed()
	if fetchedAt.IsZero() {
		return "Draft-pick ownership unavailable; refresh before handing off."
	}
	if refreshError != "" {
		return "Draft-pick ownership unavailable: " + refreshError
	}
	return ""
}

func (a *App) tradeClockReasons(ctx context.Context) []string {
	clock, err := a.targetClock(ctx)
	if err != nil {
		return []string{"Trade deadline cannot be checked: " + err.Error()}
	}
	return tradeDeadlineReasons(time.Now(), clock.Value.Windows)
}

func tradeCentsCorrection(p state.MirrorPlayer, afterOwner string, s state.WhatIfSession) domain.Money {
	return tradesapp.CentsCorrection(p, afterOwner, s)
}
func tradeDeadlineReasons(now time.Time, windows []leagueclock.Window) []string {
	return tradesapp.DeadlineReasons(now, windows, time.Local)
}

func tradeProjectionInput(sim tradeSimulation, s state.WhatIfSession, seed domain.Money,
	players map[string]snapshot.Player, name string) tradesapp.ProjectionInput {
	used, _ := sim.Store.CapUsed(s.Mine)
	original := tradesapp.SimulationPlayers(sim.Sessions)
	live := make(map[string]state.PlayerState, len(original))
	for _, p := range original {
		if value, ok := sim.Store.Player(p.MFLID); ok {
			live[p.MFLID] = value
		}
	}
	return tradesapp.ProjectionInput{Session: s, Name: name, Before: sim.Before[s.Mine], Seed: seed,
		Used: used, Players: original, Live: live, Names: players}
}
