package composition

import (
	"context"
	"encoding/json"
	"log"
	"time"

	nativeapp "github.com/secureprospective/TheWarRoom/internal/app/commissioner/native"
	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/leagueservice"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

// NativeOwnerAction is one thing an owner can do in an app-owned league. Run takes the gate's Permit, so
// the only way to start it is runNativeAction, which asks the calendar first. Input is the owner's request
// as sent; the action decodes it and records its change on the league ledger, returning the receipt.
type NativeOwnerAction struct {
	Name  string
	Opens leaguerules.Action
	Run   func(ctx context.Context, app *App, permit commissioner.Permit, input json.RawMessage) (commissioner.Receipt, error)
}

// nativeOwnerActions is the registry. Each chunk that adds an owner action to the app-owned league adds
// it here; native_gate_test.go holds every entry to the gate. The contract actions an owner takes under a
// window are registered from the shared list of priced actions, so a new one cannot be priced without being
// gated.
func nativeOwnerActions() []NativeOwnerAction {
	registry := []NativeOwnerAction{lineupSetAction(), draftPickAction(), waiverAction(commissioner.RosterCut),
		waiverAction(commissioner.WaiverClaim), bidAction(commissioner.BidPlace), bidAction(commissioner.BidRaise),
		bidAction(commissioner.BidWithdraw), tradeAction(commissioner.TradeOffer), tradeAction(commissioner.TradeAccept),
		tradeAction(commissioner.TradeDecline), tradeAction(commissioner.TradeWithdraw)}
	for _, contract := range nativeapp.ContractOwnerActions() {
		registry = append(registry, NativeOwnerAction{Name: contract.Name, Opens: contract.Opens,
			Run: func(ctx context.Context, a *App, permit commissioner.Permit, _ json.RawMessage) (commissioner.Receipt, error) {
				return commissioner.Receipt{}, a.startContractAction(ctx, permit, contract.Action)
			}})
	}

	return registry
}

// startContractAction is the first thing an owner's contract action does once the gate has said yes:
// confirm the league has that action on. Choosing the contract and paying its price follow it, through the
// same QuoteContract check the preview reads; until that is built the action records nothing.
func (a *App) startContractAction(ctx context.Context, permit commissioner.Permit,
	action nativecheck.ContractAction) error {
	return nativeapp.StartContractAction(ctx, permit, action, a.leagueRecords, a.timeNow) //nolint:wrapcheck // Preserve the contract refusal.

}

// ownerZone is where the owner reads times: the desktop's own zone unless a test pins another.
func (a *App) ownerZone() *time.Location {
	return nativeapp.OwnerZone(a.core.LocalZone)
}

// runNativeAction is the single door to a native owner action: it names the action in the registry, asks
// the gate at the current time and runs the action only on a yes. A closed gate returns its sentence.
func (a *App) runNativeAction(registry []NativeOwnerAction, league, franchise, name string,
	input json.RawMessage) (commissioner.Receipt, error) {
	actions := make([]nativeapp.OwnerAction, len(registry))
	for i, action := range registry {
		actions[i] = nativeapp.OwnerAction{Name: action.Name, Opens: action.Opens,
			Run: func(ctx context.Context, permit commissioner.Permit, body json.RawMessage) (commissioner.Receipt, error) {
				return action.Run(ctx, a, permit, body)
			}}
	}
	//nolint:wrapcheck // Preserve the gate refusal identity.
	return nativeapp.RunOwnerAction(a.core.Ctx, actions, league, franchise, name, input,
		nativeapp.OwnerGate{Ready: a.ready, Records: a.leagueRecords, Now: a.timeNow, Zone: a.ownerZone})
}

func recordOwnerAction(family nativeapp.OwnerFamily) func(context.Context, *App, commissioner.Permit, json.RawMessage) (commissioner.Receipt, error) {
	return func(ctx context.Context, a *App, permit commissioner.Permit, input json.RawMessage) (commissioner.Receipt, error) {
		return nativeapp.RecordOwnerAction(ctx, family, leagueservice.New(commissionerHost{State: a.core, app: a}).Execute, permit, input)
	}
}
func tradeAction(command string) NativeOwnerAction {
	return NativeOwnerAction{Name: command, Opens: commissioner.OwnerTradeAction(command), Run: recordOwnerAction(nativeapp.TradeFamily())}
}
func waiverAction(command string) NativeOwnerAction {
	return NativeOwnerAction{Name: command, Opens: commissioner.OwnerWaiverAction(command), Run: recordOwnerAction(nativeapp.WaiverFamily())}
}
func bidAction(command string) NativeOwnerAction {
	return NativeOwnerAction{Name: command, Opens: commissioner.OwnerBidAction(command), Run: recordOwnerAction(nativeapp.BidFamily())}
}

const (
	draftPickName  = "draft.pick"
	draftPickOpens = leaguerules.Action("draft_pick")
)

// draftPickAction is the registered owner action for a rookie pick; native_gate_test.go holds it to the
// gate with the rest.
func draftPickAction() NativeOwnerAction {
	return NativeOwnerAction{Name: draftPickName, Opens: draftPickOpens, Run: runDraftPick}
}

// runDraftPick records a rookie pick. The gate has already said yes for this franchise; the store records
// the pick only for that franchise's seated owner, and only under this permit.
func runDraftPick(ctx context.Context, a *App, permit commissioner.Permit,
	body json.RawMessage) (commissioner.Receipt, error) {
	//nolint:wrapcheck // Preserve the rookie pick refusal.
	return nativeapp.RecordDraftPick(ctx, permit, body, a.commissionerPrincipal,
		func(ctx context.Context, permit commissioner.Permit, input commissioner.DraftPickInput, principal string) (commissioner.Receipt, error) {
			ctx, cancel := context.WithTimeout(ctx, refreshTimeout)
			defer cancel()
			return a.core.History.RecordDraftPick(ctx, permit, input, principal, a.nativeFacts(), a.ownerZone())
		})

}

// closeFinishedDrafts closes a draft whose last pick has just been used, so it does not wait for the clock's
// next look. A failure is logged and the pick that led to it stands.
func (a *App) closeFinishedDrafts(league string) {
	ctx, cancel := context.WithTimeout(a.core.Ctx, refreshTimeout)
	defer cancel()
	nativeapp.CloseFinishedDrafts(ctx, league, func(ctx context.Context, league string) error {
		return a.core.History.CloseFinishedDrafts(ctx, league, a.nativeFacts())
	}, func(message string) { log.Print(message) })
}

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
