package composition

import (
	"context"
	"encoding/json"

	nativeapp "github.com/secureprospective/TheWarRoom/internal/app/commissioner/native"
	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/leagueservice"
)

func recordOwnerAction(family nativeapp.OwnerFamily) func(context.Context, *App, commissioner.Permit, json.RawMessage) (commissioner.Receipt, error) {
	return func(ctx context.Context, a *App, permit commissioner.Permit, input json.RawMessage) (commissioner.Receipt, error) {
		return nativeapp.RecordOwnerAction(family)(ctx, leagueservice.New(commissionerHost{State: a.core, app: a}).Execute, permit, input)
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
