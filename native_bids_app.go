package main

import (
	"context"
	"fmt"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/leagueweek"
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

// appBidEnv tells the store which NFL week a bid is made in, from the NFL week dates the app holds. The app
// reads no playoff field, so every franchise's playoff standing is unknown and a limit that depends on it is
// refused in words rather than guessed.
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
	return commissioner.BidEnvironment{Season: nflWeekAt(weeks, at), Playoff: map[string]nativecheck.PlayoffStanding{}}, nil
}

// nflWeekAt is the week in play: the first held week not yet finished, or the last held week once every one
// is. With no week held the app cannot say.
func nflWeekAt(weeks []leagueweek.Week, at time.Time) nativecheck.BidSeason {
	played := make([]leagueweek.Week, 0, len(weeks))
	for _, w := range weeks {
		if len(w.Games) != 0 {
			played = append(played, w)
		}
	}
	if len(played) == 0 {
		return nativecheck.BidSeason{}
	}
	if current, found := leagueweek.LineupWeek(at, played...); found {
		return nativecheck.BidSeason{Week: current.Number, WeekKnown: true}
	}
	last := 0
	for _, w := range played {
		last = max(last, w.Number)
	}
	return nativecheck.BidSeason{Week: last, WeekKnown: true}
}
