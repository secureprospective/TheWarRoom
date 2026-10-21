package main

import (
	"context"
	"encoding/json"
	"fmt"

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
	return func(ctx context.Context, a *App, permit commissioner.Permit,
		input json.RawMessage) (commissioner.Receipt, error) {
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
		receipt, err := a.core.History.ExecuteCommissionerWithFacts(ctx, raw, principal, nil, a.nativeFacts(),
			a.ownerZone())
		if err != nil {
			return commissioner.Receipt{}, fmt.Errorf("record %s: %w", family.label, err)
		}
		return receipt, nil
	}
}

// submitOwnerAction is the owner door shared by every native owner action family: identity comes from the
// local session, a retried request returns its recorded receipt, and the calendar's gate is asked before the
// action runs.
func (a *App) submitOwnerAction(raw commissioner.RawCommand, family ownerFamily) (commissioner.Receipt,
	error) {
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
	return NativeOwnerAction{Name: command, Opens: commissioner.OwnerTradeAction(command),
		Run: recordOwnerAction(tradeFamily())}
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

func waiverAction(command string) NativeOwnerAction {
	return NativeOwnerAction{Name: command, Opens: commissioner.OwnerWaiverAction(command),
		Run: recordOwnerAction(waiverFamily())}
}

func (a *App) nativeActionRetry(raw commissioner.RawCommand, principal string) (commissioner.Receipt,
	bool, error) {
	_, found, err := a.core.History.NativeSeedRetry(a.core.Ctx, raw.League, raw.Request)
	if err != nil {
		return commissioner.Receipt{}, false, fmt.Errorf("read native action retry: %w", err)
	}
	if !found {
		return commissioner.Receipt{}, false, nil
	}
	receipt, err := a.core.History.ExecuteCommissionerWithFacts(a.core.Ctx, raw, principal, nil,
		a.nativeFacts(), a.ownerZone())
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

func bidAction(command string) NativeOwnerAction {
	return NativeOwnerAction{Name: command, Opens: commissioner.OwnerBidAction(command),
		Run: recordOwnerAction(bidFamily())}
}
