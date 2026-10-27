package commissioner

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner/market"

	nativeapp "github.com/secureprospective/TheWarRoom/internal/app/commissioner/native"
	"github.com/secureprospective/TheWarRoom/internal/commissioner"
)

func (a *API) TargetNativeTradeAction(raw commissioner.RawCommand) (commissioner.Receipt, error) {
	if commissioner.OwnerTradeAction(raw.Command) == "" || raw.Args.Trade == nil || raw.Args.TradeOutcome != nil {
		return commissioner.Receipt{}, fmt.Errorf("choose to offer, accept, decline or withdraw a trade")
	}
	//nolint:wrapcheck // Preserve the existing refusal text and error identity.
	return a.reads.Execute(a.host.CommissionerContext(), raw)
}

// TargetNativeTrades reads the league's native trade desk; viewer is the seated owner's franchise.
func (a *API) TargetNativeTrades(league, viewer string) (market.NativeTrades, error) {
	//nolint:wrapcheck // The shared service supplies the existing user-facing context.
	return a.reads.TargetNativeTrades(league, viewer)
}

func (a *API) TargetNativeWaiverAction(raw commissioner.RawCommand) (commissioner.Receipt, error) {
	if commissioner.OwnerWaiverAction(raw.Command) == "" || raw.Args.Waiver == nil || raw.Args.WaiverOutcome != nil {
		return commissioner.Receipt{}, fmt.Errorf("choose a native cut or waiver claim")
	}
	//nolint:wrapcheck // Preserve the existing refusal text and error identity.
	return a.reads.Execute(a.host.CommissionerContext(), raw)
}

func (a *API) TargetNativeWaivers(league, franchise string, at time.Time) (market.NativeWaivers,
	error) {
	//nolint:wrapcheck // The shared service supplies the existing user-facing context.
	return a.reads.TargetNativeWaivers(league, franchise, at)
}

func (a *API) TargetNativeBidAction(raw commissioner.RawCommand) (commissioner.Receipt, error) {
	if commissioner.OwnerBidAction(raw.Command) == "" || raw.Args.Bid == nil || raw.Args.BidOutcome != nil {
		return commissioner.Receipt{}, fmt.Errorf("choose to place, raise or withdraw a bid")
	}
	//nolint:wrapcheck // Preserve the existing refusal text and error identity.
	return a.reads.Execute(a.host.CommissionerContext(), raw)
}

func (a *API) TargetNativeBids(league, franchise string) (market.NativeBids, error) {
	//nolint:wrapcheck // The shared service supplies the existing user-facing context.
	return a.reads.TargetNativeBids(league, franchise)
}

// TargetNativeDraftPick makes the owner's pick for the franchise on the clock. It asks the calendar first:
// a closed draft window refuses the pick in the calendar's own words.
func (a *API) TargetNativeDraftPick(input commissioner.DraftPickInput) (commissioner.Receipt, error) {
	if err := input.Validate(); err != nil {
		return commissioner.Receipt{}, fmt.Errorf("rookie pick input: %w", err)
	}
	if err := a.host.Ready(); err != nil {
		return commissioner.Receipt{}, fmt.Errorf("rookie pick startup: %w", err)
	}
	body, err := json.Marshal(input)
	if err != nil {
		return commissioner.Receipt{}, fmt.Errorf("rookie pick input: %w", err)
	}
	receipt, err := a.host.RunNativeOwnerAction(input.League, input.Franchise, draftPickName, body)
	var refused *commissioner.Refusal
	if errors.As(err, &refused) {
		return commissioner.Receipt{}, errors.New(refused.Reason())
	}
	if err != nil {
		return commissioner.Receipt{}, fmt.Errorf("rookie pick: %w", err)
	}
	a.host.CloseFinishedDrafts(input.League)
	return receipt, nil
}

// TargetNativeDraft reads the rookie draft for a franchise: the order and its owners, the picks made, who is
// on the clock until when in the viewer's time, and the franchise's own next pick.
func (a *API) TargetNativeDraft(scope commissioner.RawNativeDraftScope) (commissioner.NativeDraft, error) {
	//nolint:wrapcheck // The shared service supplies the existing user-facing context.
	return a.reads.TargetNativeDraft(scope)
}

// TargetNativeLineupSet sets an owner's lineup through the native action gate and the league ledger. The
// lineup must be legal under the league's lineup check, and no player whose game has started can move.
func (a *API) TargetNativeLineupSet(body string) (commissioner.Receipt, error) {
	in, err := nativeapp.DecodeLineupSet([]byte(body))
	if err != nil {
		return commissioner.Receipt{}, fmt.Errorf("lineup input: %w", err)
	}
	receipt, err := a.host.RunNativeOwnerAction(in.League, in.Franchise, "lineup.set", json.RawMessage(body))
	var refused *commissioner.Refusal
	if errors.As(err, &refused) {
		return commissioner.Receipt{}, errors.New(refused.Reason())
	}
	if err != nil {
		return commissioner.Receipt{}, fmt.Errorf("set lineup: %w", err)
	}
	return receipt, nil
}

// TargetNativeLineup reads a franchise's lineup for a week: starters and bench, who is locked and when it
// locks in the owner's time, and the league's verdict. at is an RFC 3339 instant, or empty for now.
func (a *API) TargetNativeLineup(league, franchise string, week int,
	at string) (commissioner.NativeLineup, error) {
	//nolint:wrapcheck // The shared service supplies the existing user-facing context.
	return a.reads.TargetNativeLineup(league, franchise, week, at)
}
