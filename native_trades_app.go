package main

import (
	"fmt"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

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
	records, err := a.leagueRecords(a.ctx, league)
	if err != nil {
		return commissioner.NativeTrades{}, err
	}
	reading, err := commissioner.NativeTradesReading(league, viewer, principal, records, a.timeNow(), a.ownerZone())
	if err != nil {
		return commissioner.NativeTrades{}, fmt.Errorf("read trades: %w", err)
	}
	return reading, nil
}
