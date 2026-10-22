package trades

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"slices"

	"github.com/secureprospective/TheWarRoom/internal/app/catalog"
	"github.com/secureprospective/TheWarRoom/internal/app/report"
	"github.com/secureprospective/TheWarRoom/internal/envelope"
	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
)

// Host supplies detached held trade state and runs the root-owned plan lifecycle (simulation,
// supersession, saves, hand-off and notification) under its original locks. No lock enters the API.
type Host interface {
	RequireOwnFranchise(string) error
	Ready() error
	TradeState(franchiseID string) (snapshot.Snapshot, snapshot.Sourced[[]leaguefeed.PendingTrade], error)
	// DraftTradeAccept saves the accept plan for offer under the moves lock, superseding the last.
	DraftTradeAccept(franchiseID string, offer leaguefeed.PendingTrade, snap snapshot.Snapshot,
		held snapshot.Sourced[[]leaguefeed.PendingTrade], id string) (envelope.Receipt, error)
	// TradeBuilder runs a validated request under the what-if and refresh locks, filling out; on
	// failure it returns what it had filled so far.
	TradeBuilder(r RawTradeBuilder, out TradeBuilderView) (TradeBuilderView, error)
	TradesBuildStamp() string
}

type API struct{ host Host }

func New(host Host) *API { return &API{host: host} }

func (a *API) TargetTrades(franchiseID string) (TradeReading, error) {
	snap, held, err := a.host.TradeState(franchiseID)
	if err != nil {
		return TradeReading{}, fmt.Errorf("target trades: %w", err)
	}
	return WireReading(Read(franchiseID, snap, held)), nil
}

func (a *API) TargetDraftTradeAccept(franchiseID, tradeID string) (envelope.Receipt, error) {
	if err := a.host.RequireOwnFranchise(franchiseID); err != nil {
		return envelope.Receipt{}, err //nolint:wrapcheck // Preserve the shared authorization refusal.
	}
	snap, held, err := a.host.TradeState(franchiseID)
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
	//nolint:wrapcheck // The host's errors already start "target draft trade accept" or are the supersede refusal.
	return a.host.DraftTradeAccept(franchiseID, offer, snap, held, "trade-"+hex.EncodeToString(random[:]))
}

func (a *API) TargetTradeBuilder(r RawTradeBuilder) TradeBuilderReading {
	out := TradeBuilderView{Teams: []TradeBuilderTeam{}, Plans: []TradeBuilderPlan{}}
	if err := r.Validate(); err != nil {
		return a.builderReading(out, err)
	}
	if err := a.host.Ready(); err != nil {
		return a.builderReading(out, err)
	}
	if err := a.host.RequireOwnFranchise(r.Mine); err != nil {
		return a.builderReading(out, err)
	}
	out, err := a.host.TradeBuilder(r, out)
	return a.builderReading(out, err)
}

func (a *API) builderReading(v TradeBuilderView, err error) TradeBuilderReading {
	return TradeBuilderReading(report.MirrorReading(v, "", "whatif.db; held MFL rosters and futureDraftPicks", "Your pinned what-if and the counterparty’s held MFL roster are projected through the trade coordinator. Current cap includes MFL adjustments and modelled cuts; cents retained. Next-season gross commitments use flat-filled held salaries and this season's cap, not a known future cap. IR may be traded; taxi status carries with the player. The trade deadline is the league clock's: Week 9's first kickoff unless the commissioner moved it. Compensatory picks are not identified by this export: commissioner verification required. Three DOT approvals, then commissioner approval on MFL. Nothing submitted by the app.", len(v.Plans), a.host.TradesBuildStamp(), err))
}

// Commands records existing bindings without inventing publication metadata. The builder can open
// MFL's trade desk on hand-off, so it is desktop-only.
func Commands() []catalog.Command {
	return []catalog.Command{
		{ID: "trade.accept", Method: "TargetDraftTradeAccept", Version: catalog.Unset, Role: "gm,commish", Gravity: "G2", Availability: "both"},
		{ID: catalog.Unset, Method: "TargetTradeBuilder", Version: catalog.Unset, Role: "gm,commish", Gravity: "G2", Availability: "desktop"},
		{ID: catalog.Unset, Method: "TargetTrades", Version: catalog.Unset, Role: "gm,commish,admin", Gravity: "G0", Availability: "both"},
	}
}
