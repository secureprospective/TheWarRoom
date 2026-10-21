package composition

import (
	"context"
	"encoding/json"
	"fmt"

	nativeapp "github.com/secureprospective/TheWarRoom/internal/app/commissioner/native"
	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/leagueweek"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

// lineupWindow is the calendar action that lets an owner set a lineup. Like every action the gate asks
// about, it is closed unless a window or an exception opens it.
const lineupWindow leaguerules.Action = "lineup"

// RawLineupSet is an owner's request to set the starters of their own franchise for a week. It is not a
// Wails model (the binding takes the body as a string), so it aliases the service type.
type RawLineupSet = nativeapp.LineupSet

// lineupSetAction is the registered owner action; native_gate_test.go holds it to the gate with the rest.
func lineupSetAction() NativeOwnerAction {
	return NativeOwnerAction{Name: "lineup.set", Opens: lineupWindow, Run: runLineupSet}
}

// runLineupSet records the lineup. It is reached only through runNativeAction, and it checks the permit
// it was handed is for this very lineup, so a permit for anything else cannot start it.
func runLineupSet(ctx context.Context, a *App, permit commissioner.Permit,
	input json.RawMessage) (commissioner.Receipt, error) {
	in, err := nativeapp.DecodeLineupSet(input)
	if err != nil {
		//nolint:wrapcheck // the lineup's own words, unchanged from before the move
		return commissioner.Receipt{}, err
	}
	if permit.League() != in.League || permit.Franchise() != in.Franchise || permit.Action() != lineupWindow {
		return commissioner.Receipt{}, fmt.Errorf("this lineup was not cleared by the league calendar")
	}
	principal, err := a.commissionerPrincipal()
	if err != nil {
		return commissioner.Receipt{}, err
	}
	raw := commissioner.RawCommand{League: in.League, Command: commissioner.LineupSet, Expected: in.Expected,
		Request: in.Request, Effective: "now", Visibility: "private",
		Args: commissioner.RawArgs{Lineup: &commissioner.RawLineupChange{Season: in.Season, Franchise: in.Franchise,
			Week: in.Week, Starters: in.Starters}}}
	ctx, cancel := context.WithTimeout(ctx, refreshTimeout)
	defer cancel()
	receipt, err := a.core.History.ExecuteCommissioner(ctx, raw, principal, a.core.CommissionerAuthorizer)
	if err != nil {
		return commissioner.Receipt{}, fmt.Errorf("record the lineup: %w", err)
	}
	return receipt, nil
}

// appLineupEnv gives lineups the football facts the app already holds: players from the held directory and
// kickoff times from the held NFL schedule, with no fetch.
type appLineupEnv struct{ app *App }

func (e appLineupEnv) PlayerFacts() nativecheck.Facts { return e.app.nativeFacts() }

func (e appLineupEnv) LineupFacts(ctx context.Context, season, week int) (commissioner.LineupFacts, error) {
	weeks, err := e.app.nativeScheduleDates(ctx, season, e.app.timeNow())
	if err != nil {
		return commissioner.LineupFacts{}, err
	}
	out := commissioner.LineupFacts{Facts: e.app.nativeFacts(), Zone: e.app.ownerZone()}
	for _, w := range weeks {
		if w.Number == week {
			out.Locks, out.ScheduleHeld = leagueweek.Locks(w), true
		}
	}
	return out, nil
}
