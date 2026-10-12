package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"slices"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/envelope"
	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

type RawTradeBuilder struct {
	Mine      string   `json:"mine"`
	Other     string   `json:"other"`
	Action    string   `json:"action"`
	ID        string   `json:"id"`
	Give      []string `json:"give"`
	Get       []string `json:"get"`
	Rationale string   `json:"rationale"`
}

func (r RawTradeBuilder) Validate() error {
	if err := (snapshot.RawContractScope{Mine: r.Mine, Sort: "name"}).Validate(); err != nil {
		return fmt.Errorf("trade builder scope: %w", err)
	}
	if r.Mine == "" || !slices.Contains([]string{"read", "preview", "save", "reject", "revoke", "handoff"}, r.Action) {
		return fmt.Errorf("trade builder: franchise and known action required")
	}
	if slices.Contains([]string{"reject", "revoke", "handoff"}, r.Action) && r.ID == "" {
		return fmt.Errorf("trade builder: offer or plan required")
	}
	if r.Action != "preview" && r.Action != "save" {
		return nil
	}
	if err := (snapshot.RawContractScope{Mine: r.Other, Sort: "name"}).Validate(); err != nil {
		return fmt.Errorf("trade builder other franchise: %w", err)
	}
	if r.Other == "" || r.Other == r.Mine || len(r.Give) == 0 || len(r.Get) == 0 || len(r.Give)+len(r.Get) > 256 {
		return fmt.Errorf("trade builder: distinct franchises and assets on both sides required")
	}
	return validateTradeAssets(r.Give, r.Get)
}

type TradeBuilderTeam struct {
	ID     string       `json:"id"`
	Name   string       `json:"name"`
	Assets []TradeAsset `json:"assets"`
}
type TradeProjection struct {
	Team    string         `json:"team"`
	Figures []WhatIfFigure `json:"figures"`
	Roster  []TradeAsset   `json:"roster"`
}
type TradeBuilderPlan struct {
	ID      string            `json:"id"`
	Post    string            `json:"post"`
	Reasons []string          `json:"reasons"`
	Resets  []TradeReset      `json:"resets,omitempty"`
	Teams   []TradeProjection `json:"teams"`
	Receipt *envelope.Receipt `json:"receipt,omitempty"`
}

// TradeReset offers the Reset for a what-if MFL has moved past, so the deal is checked afresh.
type TradeReset struct {
	Franchise string `json:"franchise"`
	Label     string `json:"label"`
}

type TradeBuilderView struct {
	Teams   []TradeBuilderTeam `json:"teams"`
	Plans   []TradeBuilderPlan `json:"plans"`
	Preview *TradeBuilderPlan  `json:"preview,omitempty"`
}

func (a *App) TargetTradeBuilder(r RawTradeBuilder) Reading[TradeBuilderView] {
	out := TradeBuilderView{Teams: []TradeBuilderTeam{}, Plans: []TradeBuilderPlan{}}
	if err := r.Validate(); err != nil {
		return tradeBuilderReading(out, err)
	}
	if err := a.ready(); err != nil {
		return tradeBuilderReading(out, err)
	}
	a.whatifMu.Lock()
	defer a.whatifMu.Unlock()
	a.refreshMu.Lock()
	defer a.refreshMu.Unlock()
	ctx, cancel := context.WithTimeout(a.ctx, m2Timeout)
	defer cancel()
	snap, held, err := a.tradeState(r.Mine)
	if err != nil {
		return tradeBuilderReading(out, err)
	}
	out.Teams = a.tradeBuilderTeams(snap)
	if r.Action == "preview" || r.Action == "save" {
		preview, err := a.tradeProjection(ctx, r, snap)
		if err != nil {
			return tradeBuilderReading(out, err)
		}
		out.Preview = &preview
		if r.Action == "save" {
			receipt, err := a.keepTradePlan(ctx, r, preview.Reasons)
			if err != nil {
				return tradeBuilderReading(out, err)
			}
			out.Preview.ID, out.Preview.Receipt = receipt.CorrelationID, &receipt
		}
	}
	if r.Action == "reject" || r.Action == "revoke" {
		receipt, err := a.draftTradeExit(ctx, r, held)
		if err != nil {
			return tradeBuilderReading(out, err)
		}
		out.Preview = &TradeBuilderPlan{ID: receipt.CorrelationID, Reasons: []string{}, Teams: []TradeProjection{}, Receipt: &receipt}
	}
	if r.Action == "handoff" {
		if err := a.tradeBuilderHandOff(ctx, r, snap); err != nil {
			return tradeBuilderReading(out, err)
		}
	}
	out.Plans, err = a.savedTradePlans(ctx, r.Mine, snap)
	if err != nil {
		return tradeBuilderReading(out, err)
	}
	return tradeBuilderReading(out, nil)
}
func tradeBuilderReading(v TradeBuilderView, err error) Reading[TradeBuilderView] {
	return contractReading(v, "", "whatif.db; held MFL rosters and futureDraftPicks", "Both pinned what-if rosters are projected through the trade coordinator. Current cap includes MFL adjustments and modelled cuts; cents retained. Next-season gross commitments use flat-filled held salaries and this season's cap, not a known future cap. IR may be traded; taxi status carries with the player. Week 9 is the trade deadline; its exact instant is not held. Compensatory picks are not identified by this export: commissioner verification required. Three DOT approvals, then commissioner approval on MFL. Nothing submitted by the app.", len(v.Plans), err)
}
func (a *App) tradeBuilderTeams(snap snapshot.Snapshot) []TradeBuilderTeam {
	players := map[string]snapshot.Player{}
	for _, p := range snap.Players.Value {
		players[p.ID.String()] = p
	}
	m, _, _ := a.league.ContractsView()
	a.seasonMu.Lock()
	picks := slices.Clone(a.seasonFuturePicks.value)
	a.seasonMu.Unlock()
	out := []TradeBuilderTeam{}
	for _, f := range snap.Franchises.Value {
		team := TradeBuilderTeam{ID: f.ID, Name: f.Name, Assets: []TradeAsset{}}
		for _, p := range m.Players {
			if p.FranchiseID == f.ID {
				name := players[p.MFLID].Name
				if name == "" {
					name = "Name unavailable"
				}
				team.Assets = append(team.Assets, TradeAsset{Token: p.MFLID, Name: name, Position: players[p.MFLID].Position})
			}
		}
		for _, pick := range picks {
			if pick.Owner != f.ID || pick.Year > m.Season+2 {
				continue
			}
			team.Assets = append(team.Assets, TradeAsset{Token: fmt.Sprintf("FP_%s_%d_%d", pick.Original, pick.Year, pick.Round),
				Name: fmt.Sprintf("%d round %d · %s", pick.Year, pick.Round, tradeFranchiseName(snap, pick.Original))})
		}
		slices.SortFunc(team.Assets, func(a, b TradeAsset) int { return strings.Compare(a.Name, b.Name) })
		out = append(out, team)
	}
	return out
}
func tradeID() (string, error) {
	var bytes [16]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return "", fmt.Errorf("trade ID: %w", err)
	}
	return "trade-" + hex.EncodeToString(bytes[:]), nil
}
func (a *App) keepTradePlan(ctx context.Context, r RawTradeBuilder, blocks []string) (envelope.Receipt, error) {
	a.movesMu.Lock()
	defer a.movesMu.Unlock()
	id, err := tradeID()
	if err != nil {
		return envelope.Receipt{}, fmt.Errorf("trade draft: %w", err)
	}
	req := a.tradeRequest(r.Mine, leaguefeed.PendingTrade{Offering: r.Mine, OfferedTo: r.Other})
	req.Trade = envelope.ExpectedTrade{TradeID: id, Offering: r.Mine, Accepting: r.Other, OfferingGives: r.Give, AcceptingGives: r.Get}
	if req.Target.Kind == envelope.Unmapped {
		blocks = append(blocks, "MFL host not yet known")
	}
	e, err := envelope.DraftTradeAct(a.movesNow(), "trade.propose", req, blocks, func() string { return id })
	if err != nil {
		return envelope.Receipt{}, fmt.Errorf("trade draft: %w", err)
	}
	if err := state.NewTradePlans(a.whatifPools).Save(ctx, state.TradePlan{ID: id, Mine: r.Mine, Other: r.Other, Give: r.Give, Get: r.Get, Rationale: r.Rationale}); err != nil {
		return envelope.Receipt{}, fmt.Errorf("trade draft: %w", err)
	}
	if err := a.moves.Save(ctx, e); err != nil {
		return envelope.Receipt{}, fmt.Errorf("trade plan receipt save: %w", err)
	}
	a.movesChanged(ctx)
	return e.Receipt(), nil
}
func (a *App) draftTradeExit(ctx context.Context, r RawTradeBuilder,
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
	req := a.tradeRequest(r.Mine, p)
	if req.Target.Kind == envelope.Unmapped {
		blocks = append(blocks, "MFL host not yet known")
	}
	id, err := tradeID()
	if err != nil {
		return envelope.Receipt{}, fmt.Errorf("trade draft: %w", err)
	}
	e, err := envelope.DraftTradeAct(a.movesNow(), "trade."+r.Action, req, blocks, func() string { return id })
	if err != nil {
		return envelope.Receipt{}, fmt.Errorf("trade draft: %w", err)
	}
	a.movesMu.Lock()
	defer a.movesMu.Unlock()
	if err := a.moves.Save(ctx, e); err != nil {
		return envelope.Receipt{}, fmt.Errorf("trade exit save: %w", err)
	}
	a.movesChanged(ctx)
	return e.Receipt(), nil
}

func validateTradeAssets(give, get []string) error {

	seen := map[string]bool{}
	for _, token := range append(slices.Clone(give), get...) {
		assets, err := leaguefeed.ParseAssets(token)
		if err != nil {
			return fmt.Errorf("trade builder asset: %w", err)
		}
		if len(assets) != 1 || assets[0].BlindBidCents != nil || assets[0].CurrentPick != nil {
			return fmt.Errorf("trade builder: player or held future pick required")
		}
		key := leaguefeed.AssetToken(assets[0])
		if seen[key] {
			return fmt.Errorf("trade builder: duplicate asset")
		}
		seen[key] = true
	}
	return nil
}

func (a *App) savedTradePlans(ctx context.Context, mine string, snap snapshot.Snapshot) ([]TradeBuilderPlan, error) {
	out := []TradeBuilderPlan{}

	plans, err := state.NewTradePlans(a.whatifPools).List(ctx, mine)
	if err != nil {
		return nil, fmt.Errorf("saved trade plans: %w", err)
	}
	for _, plan := range plans {
		p, err := a.tradeProjection(ctx, RawTradeBuilder{Mine: plan.Mine, Other: plan.Other, Give: plan.Give,
			Get: plan.Get, Rationale: plan.Rationale}, snap)
		if err != nil {
			return nil, err
		}
		e, err := a.moves.Get(ctx, plan.ID)
		if err != nil {
			return nil, fmt.Errorf("saved trade receipt: %w", err)
		}
		receipt := e.Receipt()
		p.ID, p.Receipt = plan.ID, &receipt
		out = append(out, p)
	}
	return out, nil
}
