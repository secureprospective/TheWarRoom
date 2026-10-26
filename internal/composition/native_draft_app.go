package composition

import (
	"context"
	"encoding/json"
	"log"

	nativeapp "github.com/secureprospective/TheWarRoom/internal/app/commissioner/native"
	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

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
