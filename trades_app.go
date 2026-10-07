package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/envelope"
	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
)

type TradeAsset struct {
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
		Provenance: held.provenance("pendingTrades", "pendingTrades"),
	}, nil
}

func tradeFranchiseName(snap snapshot.Snapshot, id string) string {
	for _, f := range snap.Franchises.Value {
		if f.ID == id && f.Name != "" {
			return f.Name
		}
	}
	return id
}

func tradeAssets(assets []leaguefeed.Asset, players map[string]snapshot.Player) []TradeAsset {
	out := make([]TradeAsset, 0, len(assets))
	for _, asset := range assets {
		token := leaguefeed.AssetToken(asset)
		item := TradeAsset{Token: token, Name: token}
		if asset.Player != nil {
			if player, found := players[asset.Player.String()]; found {
				if player.Name != "" {
					item.Name = player.Name
				}
				item.Position = player.Position
			}
		}
		out = append(out, item)
	}
	return out
}

func (a *App) TargetTrades(franchiseID string) (TradeReading, error) {
	snap, held, err := a.tradeState(franchiseID)
	if err != nil {
		return TradeReading{}, fmt.Errorf("target trades: %w", err)
	}
	reading := TradeReading{Offers: []TradeOffer{}, Provenance: held.Provenance}
	if envelope.TradeFeedReason(held.Provenance) != "" {
		return reading, nil
	}
	players := make(map[string]snapshot.Player, len(snap.Players.Value))
	for _, player := range snap.Players.Value {
		players[player.ID.String()] = player
	}
	for _, offer := range held.Value {
		direction, other, give, get := "to_you", offer.Offering, offer.Gets, offer.Gives
		switch franchiseID {
		case offer.Offering:
			direction, other, give, get = "by_you", offer.OfferedTo, offer.Gives, offer.Gets
		case offer.OfferedTo:
		default:
			continue
		}
		reading.Offers = append(reading.Offers, TradeOffer{
			TradeID: offer.ID, Direction: direction, OtherID: other,
			OtherName: tradeFranchiseName(snap, other),
			Give:      tradeAssets(give, players), Get: tradeAssets(get, players),
			Expires: offer.Expires, Comments: offer.Comments,
		})
	}
	slices.SortFunc(reading.Offers, func(a, b TradeOffer) int {
		if order := a.Expires.Compare(b.Expires); order != 0 {
			return order
		}
		return strings.Compare(a.TradeID, b.TradeID)
	})
	return reading, nil
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
