package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/url"
	"slices"
	"time"

	tradesapp "github.com/secureprospective/TheWarRoom/internal/app/trades"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/envelope"
	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
)

type TradeAsset struct {
	Team     string          `json:"team,omitempty"`
	Contract string          `json:"contract,omitempty"`
	Now      *AssetCell      `json:"now,omitempty"`
	Dynasty  *AssetCell      `json:"dynasty,omitempty"`
	Token    string          `json:"token"`
	Name     string          `json:"name"`
	Position domain.Position `json:"position,omitempty"`
}

type TradeOffer struct {
	TradeID   string       `json:"tradeId"`
	Direction string       `json:"direction"`
	OtherID   string       `json:"otherId"`
	OtherName string       `json:"otherName"`
	Give      []TradeAsset `json:"give"`
	Get       []TradeAsset `json:"get"`
	Expires   time.Time    `json:"expires"`
	Comments  string       `json:"comments"`
}

type TradeReading struct {
	Offers     []TradeOffer        `json:"offers"`
	Provenance snapshot.Provenance `json:"provenance"`
}

func (a *App) tradeState(franchiseID string) (
	snapshot.Snapshot, snapshot.Sourced[[]leaguefeed.PendingTrade], error,
) {
	// Reuse the held-snapshot franchise validation, with no player selection to parse.
	snap, _, err := a.lineupSelection(franchiseID, nil)
	if err != nil {
		return snapshot.Snapshot{}, snapshot.Sourced[[]leaguefeed.PendingTrade]{},
			fmt.Errorf("trade state: %w", err)
	}
	a.seasonMu.Lock()
	defer a.seasonMu.Unlock()
	held := a.seasonPendingTrades
	return snap, snapshot.Sourced[[]leaguefeed.PendingTrade]{
		Value:      append([]leaguefeed.PendingTrade{}, held.value...),
		Provenance: held.provenance("pendingTrades", "pending trades"),
	}, nil
}

func (a *App) tradeRequest(franchiseID string, offer leaguefeed.PendingTrade) envelope.TradeRequest {
	req := envelope.TradeRequest{
		LeagueID: ingestion.LeagueID, FranchiseID: franchiseID,
		Trade: envelope.ExpectedTrade{
			TradeID: offer.ID, Offering: offer.Offering, Accepting: offer.OfferedTo,
			OfferingGives: leaguefeed.AssetTokens(offer.Gives), AcceptingGives: leaguefeed.AssetTokens(offer.Gets),
		},
		Target: envelope.Target{Kind: envelope.Unmapped},
	}
	if !offer.Expires.IsZero() {
		req.Deadline = &offer.Expires
	}
	if a.mflClient != nil && a.mflClient.LeagueHost() != "" {
		req.Target = envelope.Target{Kind: envelope.Mapped, URL: fmt.Sprintf(
			"https://%s/%d/options?L=%s&O=05", a.mflClient.LeagueHost(), a.season,
			url.QueryEscape(req.LeagueID))}
	}
	return req
}

func (a *App) TargetDraftTradeAccept(franchiseID, tradeID string) (envelope.Receipt, error) {
	snap, held, err := a.tradeState(franchiseID)
	if err != nil {
		return envelope.Receipt{}, fmt.Errorf("target draft trade accept: %w", err)
	}
	index := slices.IndexFunc(held.Value, func(t leaguefeed.PendingTrade) bool { return t.ID == tradeID })
	if index < 0 {
		if reason := envelope.TradeFeedReason(held.Provenance); reason != "" {
			return envelope.Receipt{}, fmt.Errorf("target draft trade accept: pending trades unavailable: %s", reason)
		}
		return envelope.Receipt{}, fmt.Errorf("target draft trade accept: offer %s is no longer pending", tradeID)
	}
	offer := held.Value[index]
	if offer.Offering == franchiseID {
		return envelope.Receipt{}, fmt.Errorf("you made this offer; accept is the other owner's")
	}
	var random [16]byte
	if _, err := rand.Read(random[:]); err != nil {
		return envelope.Receipt{}, fmt.Errorf("target draft trade accept: correlation ID: %w", err)
	}
	a.movesMu.Lock()
	defer a.movesMu.Unlock()
	now := a.movesNow()
	req := a.tradeRequest(franchiseID, offer)
	check := envelope.TradeCheck{PendingTrades: held, OfferingName: tradeFranchiseName(snap, offer.Offering)}
	if req.Target.Kind == envelope.Unmapped {
		check.Blocks = append(check.Blocks, "MFL host not yet known")
	}
	id := "trade-" + hex.EncodeToString(random[:])
	e, err := envelope.DraftTrade(now, req, snap, check, func() string { return id })
	if err != nil {
		return envelope.Receipt{}, fmt.Errorf("target draft trade accept: %w", err)
	}
	ctx, cancel := context.WithTimeout(a.ctx, m2Timeout)
	defer cancel()
	if err := a.supersedeMoves(ctx, e, now); err != nil {
		return envelope.Receipt{}, err
	}
	if err := a.moves.Save(ctx, e); err != nil {
		return envelope.Receipt{}, fmt.Errorf("target draft trade accept: save: %w", err)
	}
	return e.Receipt(), nil
}

func (a *App) TargetTrades(franchiseID string) (TradeReading, error) {
	snap, held, err := a.tradeState(franchiseID)
	if err != nil {
		return TradeReading{}, fmt.Errorf("target trades: %w", err)
	}
	return wireTrades(tradesapp.Read(franchiseID, snap, held)), nil
}
func tradeFranchiseName(snap snapshot.Snapshot, id string) string {
	return tradesapp.FranchiseName(snap, id)
}
func futureTradePickName(year, round int, original string, snap snapshot.Snapshot) string {
	return tradesapp.FuturePickName(year, round, original, snap)
}
func labelTradeReceipts(receipts []envelope.Receipt, snap snapshot.Snapshot) {
	tradesapp.LabelReceipts(receipts, snap)
}

func wireTradeAssets(in []tradesapp.Asset) []TradeAsset {
	out := make([]TradeAsset, len(in))
	for i, v := range in {
		out[i] = TradeAsset{Team: v.Team, Contract: v.Contract, Token: v.Token, Name: v.Name, Position: v.Position}
		if v.Now != nil {
			cell := AssetCell(*v.Now)
			out[i].Now = &cell
		}
		if v.Dynasty != nil {
			cell := AssetCell(*v.Dynasty)
			out[i].Dynasty = &cell
		}
	}
	return wireSlice(in, out)
}
func serviceTradeAssets(in []TradeAsset) []tradesapp.Asset {
	out := make([]tradesapp.Asset, len(in))
	for i, v := range in {
		out[i] = tradesapp.Asset{Team: v.Team, Contract: v.Contract, Token: v.Token, Name: v.Name, Position: v.Position}
		if v.Now != nil {
			cell := tradesapp.Cell(*v.Now)
			out[i].Now = &cell
		}
		if v.Dynasty != nil {
			cell := tradesapp.Cell(*v.Dynasty)
			out[i].Dynasty = &cell
		}
	}
	return wireSlice(in, out)
}
func wireTradeTeams(in []tradesapp.Team) []TradeBuilderTeam {
	out := make([]TradeBuilderTeam, len(in))
	for i, v := range in {
		out[i] = TradeBuilderTeam{ID: v.ID, Name: v.Name, Assets: wireTradeAssets(v.Assets)}
	}
	return wireSlice(in, out)
}
func serviceTradeTeams(in []TradeBuilderTeam) []tradesapp.Team {
	out := make([]tradesapp.Team, len(in))
	for i, v := range in {
		out[i] = tradesapp.Team{ID: v.ID, Name: v.Name, Assets: serviceTradeAssets(v.Assets)}
	}
	return wireSlice(in, out)
}
func wireTradeProjection(in tradesapp.Projection) TradeProjection {
	figures := make([]WhatIfFigure, len(in.Figures))
	for i, f := range in.Figures {
		figures[i] = WhatIfFigure(f)
	}
	return TradeProjection{Team: in.Team, Figures: wireSlice(in.Figures, figures), Roster: wireTradeAssets(in.Roster)}
}
func tradeFigure(label string, before, after domain.Money) tradesapp.Figure {
	return tradesapp.Figure(whatIfFigure(label, before, after))
}
func wireTrades(in tradesapp.Reading) TradeReading {
	offers := make([]TradeOffer, len(in.Offers))
	for i, v := range in.Offers {
		offers[i] = TradeOffer{TradeID: v.TradeID, Direction: v.Direction, OtherID: v.OtherID, OtherName: v.OtherName,
			Give: wireTradeAssets(v.Give), Get: wireTradeAssets(v.Get), Expires: v.Expires, Comments: v.Comments}
	}
	return TradeReading{Offers: wireSlice(in.Offers, offers), Provenance: in.Provenance}
}
