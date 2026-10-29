// Package trades shapes held trade offers and local projections without owning state or permission.
package trades

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
	"github.com/secureprospective/TheWarRoom/internal/mflref/envelope"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
)

type Asset struct {
	Team     string          `json:"team,omitempty"`
	Contract string          `json:"contract,omitempty"`
	Now      *Cell           `json:"now,omitempty"`
	Dynasty  *Cell           `json:"dynasty,omitempty"`
	Token    string          `json:"token"`
	Name     string          `json:"name"`
	Position domain.Position `json:"position,omitempty"`
}

type Offer struct {
	TradeID   string    `json:"tradeId"`
	Direction string    `json:"direction"`
	OtherID   string    `json:"otherId"`
	OtherName string    `json:"otherName"`
	Give      []Asset   `json:"give"`
	Get       []Asset   `json:"get"`
	Expires   time.Time `json:"expires"`
	Comments  string    `json:"comments"`
}

type Reading struct {
	Offers     []Offer             `json:"offers"`
	Provenance snapshot.Provenance `json:"provenance"`
}

type Input struct {
	Mine      string   `json:"mine"`
	Other     string   `json:"other"`
	Action    string   `json:"action"`
	ID        string   `json:"id"`
	Give      []string `json:"give"`
	Get       []string `json:"get"`
	Rationale string   `json:"rationale"`
}

type Team struct {
	ID     string  `json:"id"`
	Name   string  `json:"name"`
	Assets []Asset `json:"assets"`
}
type Projection struct {
	Team    string   `json:"team"`
	Figures []Figure `json:"figures"`
	Roster  []Asset  `json:"roster"`
}
type Cell struct {
	Text   string `json:"text"`
	Sub    string `json:"sub"`
	Detail string `json:"detail"`
	Held   bool   `json:"held"`
}
type Figure struct {
	Label  string `json:"label"`
	Before string `json:"before"`
	After  string `json:"after"`
	Change string `json:"change"`
}
type Value struct {
	ID           string
	Now, Dynasty Cell
}

func FranchiseName(snap snapshot.Snapshot, id string) string {
	for _, f := range snap.Franchises.Value {
		if f.ID == id && f.Name != "" {
			return f.Name
		}
	}
	return id
}

func tradeAssets(assets []leaguefeed.Asset, players map[string]snapshot.Player, snap snapshot.Snapshot) []Asset {
	out := make([]Asset, 0, len(assets))
	for _, asset := range assets {
		token := leaguefeed.AssetToken(asset)
		item := Asset{Token: token, Name: heldTradeAssetName(asset, snap)}
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

func heldTradeAssetName(asset leaguefeed.Asset, snap snapshot.Snapshot) string {
	switch {
	case asset.CurrentPick != nil:
		return fmt.Sprintf("Round %d · pick %d", asset.CurrentPick.Round, asset.CurrentPick.Pick)
	case asset.FuturePick != nil:
		pick := asset.FuturePick
		return FuturePickName(pick.Year, pick.Round, pick.Franchise, snap)
	case asset.BlindBidCents != nil:
		return fmt.Sprintf("Blind-bid cash: %d cents", *asset.BlindBidCents)
	default:
		return "Unnamed asset"
	}
}

func FuturePickName(year, round int, original string, snap snapshot.Snapshot) string {
	name := FranchiseName(snap, original)
	if name == original {
		name = "Franchise name unavailable"
	}
	return fmt.Sprintf("%d round %d · %s", year, round, name)
}

func LabelReceipts(receipts []envelope.Receipt, snap snapshot.Snapshot) {
	for i, receipt := range receipts {
		trade := receipt.Spec.Expected.Trade
		if trade == nil {
			continue
		}
		tokens := append(slices.Clone(trade.OfferingGives), trade.AcceptingGives...)
		for _, token := range tokens {
			assets, err := leaguefeed.ParseAssets(token)
			name := "Unnamed asset"
			if err == nil && len(assets) == 1 && assets[0].Player == nil {
				name = heldTradeAssetName(assets[0], snap)
			}
			receipts[i].TradeAssets = append(receipts[i].TradeAssets, envelope.AssetLabel{Token: token, Name: name})
		}
	}
}

func Read(franchiseID string, snap snapshot.Snapshot, held snapshot.Sourced[[]leaguefeed.PendingTrade]) Reading {
	reading := Reading{Offers: []Offer{}, Provenance: held.Provenance}
	if envelope.TradeFeedReason(held.Provenance) != "" {
		return reading
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
		reading.Offers = append(reading.Offers, Offer{
			TradeID: offer.ID, Direction: direction, OtherID: other,
			OtherName: FranchiseName(snap, other),
			Give:      tradeAssets(give, players, snap), Get: tradeAssets(get, players, snap),
			Expires: offer.Expires, Comments: offer.Comments,
		})
	}
	slices.SortFunc(reading.Offers, func(a, b Offer) int {
		if order := a.Expires.Compare(b.Expires); order != 0 {
			return order
		}
		return strings.Compare(a.TradeID, b.TradeID)
	})
	return reading
}
