package composition

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/leagueservice"
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
		receipt, err := leagueservice.New(commissionerHost{State: a.core, app: a}).Execute(ctx, raw)
		if err != nil {
			return commissioner.Receipt{}, fmt.Errorf("record %s: %w", family.label, err)
		}
		return receipt, nil
	}
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
