package composition

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"

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
	var input commissioner.DraftPickInput
	if err := json.Unmarshal(body, &input); err != nil {
		return commissioner.Receipt{}, fmt.Errorf("rookie pick: %w",
			commissioner.Refused(errors.New("choose a rookie for your pick")))
	}
	if err := input.Validate(); err != nil {
		return commissioner.Receipt{}, fmt.Errorf("rookie pick input: %w", commissioner.Refused(err))
	}
	principal, err := a.commissionerPrincipal()
	if err != nil {
		return commissioner.Receipt{}, err
	}
	ctx, cancel := context.WithTimeout(ctx, refreshTimeout)
	defer cancel()
	receipt, err := a.core.History.RecordDraftPick(ctx, permit, input, principal, a.nativeFacts(), a.ownerZone())
	if err != nil {
		return commissioner.Receipt{}, fmt.Errorf("record rookie pick: %w", err)
	}
	return receipt, nil
}

// closeFinishedDrafts closes a draft whose last pick has just been used, so it does not wait for the clock's
// next look. A failure is logged and the pick that led to it stands.
func (a *App) closeFinishedDrafts(league string) {
	ctx, cancel := context.WithTimeout(a.core.Ctx, refreshTimeout)
	defer cancel()
	if err := a.core.History.CloseFinishedDrafts(ctx, league, a.nativeFacts()); err != nil {
		log.Printf("the war room: close finished rookie drafts for %s: %v", league, err)
	}
}

// processNativeDraftsDue closes every league's finished drafts: the last pick can be used up by the pick
// clock running out, which no one records.
func (a *App) processNativeDraftsDue(ctx context.Context) {
	leagues, err := a.core.History.NativeLeagues(ctx)
	if err != nil {
		log.Printf("the war room: read native draft leagues: %v", err)
		return
	}
	facts := a.nativeFacts()
	for _, league := range leagues {
		if err := a.core.History.CloseFinishedDrafts(ctx, league.ID, facts); err != nil {
			log.Printf("the war room: close native drafts for %s: %v", league.ID, err)
		}
	}
}
