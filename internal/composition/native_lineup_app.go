package composition

import (
	"context"
	"encoding/json"

	nativeapp "github.com/secureprospective/TheWarRoom/internal/app/commissioner/native"
	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
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
	//nolint:wrapcheck // Preserve the lineup refusal.
	return nativeapp.RecordLineup(ctx, permit, input, a.commissionerPrincipal,
		func(ctx context.Context, raw commissioner.RawCommand, principal string) (commissioner.Receipt, error) {
			ctx, cancel := context.WithTimeout(ctx, refreshTimeout)
			defer cancel()
			return a.core.History.ExecuteCommissioner(ctx, raw, principal, a.core.CommissionerAuthorizer)
		})

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
	return nativeapp.HeldLineupFacts(weeks, week, e.app.nativeFacts(), e.app.ownerZone()), nil
}
