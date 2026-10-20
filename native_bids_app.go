package main

import (
	"context"
	"fmt"
	"log"
	"time"

	nativeapp "github.com/secureprospective/TheWarRoom/internal/app/native"
	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

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
	reading, err := a.history.NativeBids(a.ctx, league, franchise, principal, a.timeNow(), a.ownerZone(), a.nativeFacts())
	if err != nil {
		return commissioner.NativeBids{}, fmt.Errorf("native bids: %w", err)
	}
	return reading, nil
}

// appBidEnv reads the season and playoff field from held native data, never from an imported league.
type appBidEnv struct{ app *App }

func (e appBidEnv) BidEnvironment(ctx context.Context, league string, at time.Time) (commissioner.BidEnvironment, error) {
	records, err := e.app.leagueRecords(ctx, league)
	if err != nil {
		return commissioner.BidEnvironment{}, err
	}
	season, err := commissioner.SeasonAt(records, commissioner.ReadingInstant(records, at))
	if err != nil {
		return commissioner.BidEnvironment{}, fmt.Errorf("read the league season: %w", err)
	}
	weeks, err := e.app.nativeScheduleDates(ctx, season, at)
	if err != nil {
		return commissioner.BidEnvironment{}, err
	}
	phase := nativeapp.WeekAt(weeks, at)
	if !phase.WeekKnown && !phase.Offseason {
		phase.Offseason, err = commissioner.BidOffseasonAt(records, at, season)
		if err != nil {
			return commissioner.BidEnvironment{}, fmt.Errorf("read bidding offseason: %w", err)
		}
	}
	field := map[string]nativecheck.PlayoffStanding{}
	if phase.WeekKnown {
		bracket, err := e.app.history.NativeBracket(ctx, league, season)
		if err != nil {
			return commissioner.BidEnvironment{}, fmt.Errorf("read bidding playoff field: %w", err)
		}
		directory, err := commissioner.NativeDirectoryReading(league, records, at)
		if err != nil {
			return commissioner.BidEnvironment{}, fmt.Errorf("read bidding franchises: %w", err)
		}
		field = nativeapp.PlayoffField(bracket, directory)
	}
	return commissioner.BidEnvironment{Season: phase, Playoff: field}, nil
}

func (a *App) processNativeBidsDue(ctx context.Context) {
	leagues, err := a.history.NativeLeagues(ctx)
	if err != nil {
		log.Printf("the war room: read native bid leagues: %v", err)
		return
	}
	facts := a.nativeFacts()
	for _, league := range leagues {
		if err := a.history.ProcessDueBids(ctx, league.ID, facts); err != nil {
			log.Printf("the war room: award native bids for %s: %v", league.ID, err)
		}
	}
}
