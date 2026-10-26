package composition

import (
	"context"
	"encoding/json"
	"time"

	nativeapp "github.com/secureprospective/TheWarRoom/internal/app/commissioner/native"
	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
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

// errActionClosed marks the gate's no, so a caller can tell a shut window from an action that failed.
var errActionClosed = nativeapp.ErrActionClosed

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
