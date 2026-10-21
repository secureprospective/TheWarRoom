package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

// ownerFamily is one kind of native owner command. seat says which franchise a command acts for and which
// calendar action opens it (ok is false when the command carries none of the family's details); choose and
// label are the words that ask for the command and prefix its errors.
type ownerFamily struct {
	seat   func(raw commissioner.RawCommand) (franchise string, action leaguerules.Action, ok bool)
	choose string
	label  string
}

// recordOwnerAction is the Run of a registered owner action: it records a command the gate has cleared. The
// store asks the gate again inside its own transaction, where it also judges the action itself.
func recordOwnerAction(family ownerFamily) func(context.Context, *App, commissioner.Permit,
	json.RawMessage) (commissioner.Receipt, error) {
	return func(ctx context.Context, a *App, permit commissioner.Permit, input json.RawMessage) (commissioner.Receipt, error) {
		var raw commissioner.RawCommand
		if err := json.Unmarshal(input, &raw); err != nil {
			return commissioner.Receipt{}, fmt.Errorf("choose %s", family.choose)
		}
		franchise, action, ok := family.seat(raw)
		if !ok {
			return commissioner.Receipt{}, fmt.Errorf("choose %s", family.choose)
		}
		if permit.League() != raw.League || permit.Franchise() != franchise || permit.Action() != action {
			return commissioner.Receipt{}, fmt.Errorf("ask the league calendar before acting")
		}
		principal, err := a.commissionerPrincipal()
		if err != nil {
			return commissioner.Receipt{}, err
		}
		ctx, cancel := context.WithTimeout(ctx, refreshTimeout)
		defer cancel()
		receipt, err := a.core.History.ExecuteCommissionerWithFacts(ctx, raw, principal, nil, a.nativeFacts(), a.ownerZone())
		if err != nil {
			return commissioner.Receipt{}, fmt.Errorf("record %s: %w", family.label, err)
		}
		return receipt, nil
	}
}

// submitOwnerAction is the owner door shared by every native owner action family: identity comes from the
// local session, a retried request returns its recorded receipt, and the calendar's gate is asked before the
// action runs.
func (a *App) submitOwnerAction(raw commissioner.RawCommand, family ownerFamily) (commissioner.Receipt, error) {
	franchise, _, ok := family.seat(raw)
	if !ok {
		return commissioner.Receipt{}, fmt.Errorf("choose %s", family.choose)
	}
	if err := raw.Validate(); err != nil {
		return commissioner.Receipt{}, fmt.Errorf("%s input: %w", family.label, err)
	}
	if err := a.ready(); err != nil {
		return commissioner.Receipt{}, fmt.Errorf("%s startup: %w", family.label, err)
	}
	principal, err := a.commissionerPrincipal()
	if err != nil {
		return commissioner.Receipt{}, err
	}
	if receipt, retried, err := a.nativeActionRetry(raw, principal); retried || err != nil {
		return receipt, err
	}
	body, err := json.Marshal(raw)
	if err != nil {
		return commissioner.Receipt{}, fmt.Errorf("%s input: %w", family.label, err)
	}
	receipt, err := a.runNativeAction(nativeOwnerActions(), raw.League, franchise, raw.Command, body)
	if err != nil {
		return commissioner.Receipt{}, fmt.Errorf("%s: %w", family.label, err)
	}
	return receipt, nil
}

func tradeFamily() ownerFamily {
	return ownerFamily{choose: "to offer, accept, decline or withdraw a trade", label: "trade action",
		seat: func(raw commissioner.RawCommand) (string, leaguerules.Action, bool) {
			if raw.Args.Trade == nil {
				return "", "", false
			}
			return raw.Args.Trade.Franchise, commissioner.OwnerTradeAction(raw.Command), true
		}}
}

func tradeAction(command string) NativeOwnerAction {
	return NativeOwnerAction{Name: command, Opens: commissioner.OwnerTradeAction(command), Run: recordOwnerAction(tradeFamily())}
}

func (a *App) TargetNativeTradeAction(raw commissioner.RawCommand) (commissioner.Receipt, error) {
	if commissioner.OwnerTradeAction(raw.Command) == "" || raw.Args.TradeOutcome != nil {
		return commissioner.Receipt{}, fmt.Errorf("choose to offer, accept, decline or withdraw a trade")
	}
	return a.submitOwnerAction(raw, tradeFamily())
}

// TargetNativeTrades reads the league's native trade desk; viewer is the seated owner's franchise.
func (a *App) TargetNativeTrades(league, viewer string) (commissioner.NativeTrades, error) {
	if err := a.ready(); err != nil {
		return commissioner.NativeTrades{}, fmt.Errorf("trade startup: %w", err)
	}
	principal, err := a.commissionerPrincipal()
	if err != nil {
		return commissioner.NativeTrades{}, err
	}
	records, err := a.leagueRecords(a.core.Ctx, league)
	if err != nil {
		return commissioner.NativeTrades{}, err
	}
	reading, err := commissioner.NativeTradesReading(league, viewer, principal, records, a.timeNow(), a.ownerZone())
	if err != nil {
		return commissioner.NativeTrades{}, fmt.Errorf("read trades: %w", err)
	}
	return reading, nil
}

func waiverFamily() ownerFamily {
	return ownerFamily{choose: "a native cut or waiver claim", label: "native waiver action",
		seat: func(raw commissioner.RawCommand) (string, leaguerules.Action, bool) {
			if raw.Args.Waiver == nil {
				return "", "", false
			}
			return raw.Args.Waiver.Franchise, commissioner.OwnerWaiverAction(raw.Command), true
		}}
}

// waiverAction is the registered owner action for a native cut or waiver claim; native_gate_test.go holds
// both to the gate with the rest.

func waiverAction(command string) NativeOwnerAction {
	return NativeOwnerAction{Name: command, Opens: commissioner.OwnerWaiverAction(command),
		Run: recordOwnerAction(waiverFamily())}
}

// TargetNativeWaiverAction is the owner door; identity comes from the local session, not the request.

func (a *App) TargetNativeWaiverAction(raw commissioner.RawCommand) (commissioner.Receipt, error) {
	if commissioner.OwnerWaiverAction(raw.Command) == "" || raw.Args.WaiverOutcome != nil {
		return commissioner.Receipt{}, fmt.Errorf("choose a native cut or waiver claim")
	}
	return a.submitOwnerAction(raw, waiverFamily())
}

func (a *App) TargetNativeWaivers(league, franchise string, at time.Time) (commissioner.NativeWaivers, error) {
	if at.IsZero() {
		return commissioner.NativeWaivers{}, fmt.Errorf("choose when to read waivers")
	}
	if err := a.ready(); err != nil {
		return commissioner.NativeWaivers{}, fmt.Errorf("waivers startup: %w", err)
	}
	principal, err := a.commissionerPrincipal()
	if err != nil {
		return commissioner.NativeWaivers{}, err
	}
	reading, err := a.core.History.NativeWaivers(a.core.Ctx, league, franchise, principal, at, a.ownerZone(), a.nativeFacts())
	if err != nil {
		return commissioner.NativeWaivers{}, fmt.Errorf("native waivers: %w", err)
	}
	return reading, nil
}

// draftLookEvery is how often the clock looks for a rookie draft that has used its last pick.

func (a *App) nativeActionRetry(raw commissioner.RawCommand, principal string) (commissioner.Receipt, bool, error) {
	_, found, err := a.core.History.NativeSeedRetry(a.core.Ctx, raw.League, raw.Request)
	if err != nil {
		return commissioner.Receipt{}, false, fmt.Errorf("read native action retry: %w", err)
	}
	if !found {
		return commissioner.Receipt{}, false, nil
	}
	receipt, err := a.core.History.ExecuteCommissionerWithFacts(a.core.Ctx, raw, principal, nil, a.nativeFacts(), a.ownerZone())
	if err != nil {
		return commissioner.Receipt{}, true, fmt.Errorf("retry native action: %w", err)
	}
	return receipt, true, nil
}

func bidFamily() ownerFamily {
	return ownerFamily{choose: "to place, raise or withdraw a bid", label: "native bid action",
		seat: func(raw commissioner.RawCommand) (string, leaguerules.Action, bool) {
			if raw.Args.Bid == nil {
				return "", "", false
			}
			return raw.Args.Bid.Franchise, commissioner.OwnerBidAction(raw.Command), true
		}}
}

// bidAction is the registered owner action for placing, raising or withdrawing a free-agent bid;
// native_gate_test.go holds each to the gate with the rest.

func bidAction(command string) NativeOwnerAction {
	return NativeOwnerAction{Name: command, Opens: commissioner.OwnerBidAction(command), Run: recordOwnerAction(bidFamily())}
}

// TargetNativeBidAction is the owner door for placing, raising and withdrawing a bid; identity comes from
// the local session, not the request.

func (a *App) TargetNativeBidAction(raw commissioner.RawCommand) (commissioner.Receipt, error) {
	if commissioner.OwnerBidAction(raw.Command) == "" || raw.Args.BidOutcome != nil {
		return commissioner.Receipt{}, fmt.Errorf("choose to place, raise or withdraw a bid")
	}
	return a.submitOwnerAction(raw, bidFamily())
}

// TargetNativeBids reads the bids one franchise's owner may see now: its own, the leading open bids with
// their late-bid countdown, and the decided awards. Times are in the owner's zone.

func (a *App) TargetNativeBids(league, franchise string) (commissioner.NativeBids, error) {
	if err := a.ready(); err != nil {
		return commissioner.NativeBids{}, fmt.Errorf("bids startup: %w", err)
	}
	principal, err := a.commissionerPrincipal()
	if err != nil {
		return commissioner.NativeBids{}, err
	}
	reading, err := a.core.History.NativeBids(a.core.Ctx, league, franchise, principal, a.timeNow(), a.ownerZone(), a.nativeFacts())
	if err != nil {
		return commissioner.NativeBids{}, fmt.Errorf("native bids: %w", err)
	}
	return reading, nil
}

// appBidEnv reads the season and playoff field from held native data, never from an imported league.
