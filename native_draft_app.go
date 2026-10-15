package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

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
		return commissioner.Receipt{}, fmt.Errorf("rookie pick: %w", commissioner.Refused(errors.New("choose a rookie for your pick")))
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
	receipt, err := a.history.RecordDraftPick(ctx, permit, input, principal, a.nativeFacts(), a.ownerZone())
	if err != nil {
		return commissioner.Receipt{}, fmt.Errorf("record rookie pick: %w", err)
	}
	return receipt, nil
}

// TargetNativeDraftPick makes the owner's pick for the franchise on the clock. It asks the calendar first:
// a closed draft window refuses the pick in the calendar's own words.
func (a *App) TargetNativeDraftPick(input commissioner.DraftPickInput) (commissioner.Receipt, error) {
	if err := input.Validate(); err != nil {
		return commissioner.Receipt{}, fmt.Errorf("rookie pick input: %w", err)
	}
	if err := a.ready(); err != nil {
		return commissioner.Receipt{}, fmt.Errorf("rookie pick startup: %w", err)
	}
	body, err := json.Marshal(input)
	if err != nil {
		return commissioner.Receipt{}, fmt.Errorf("rookie pick input: %w", err)
	}
	receipt, err := a.runNativeAction(nativeOwnerActions(), input.League, input.Franchise, draftPickName, body)
	var refused *commissioner.Refusal
	if errors.As(err, &refused) {
		return commissioner.Receipt{}, errors.New(refused.Reason())
	}
	if err != nil {
		return commissioner.Receipt{}, fmt.Errorf("rookie pick: %w", err)
	}
	return receipt, nil
}

// TargetNativeDraft reads the rookie draft for a franchise: the order and its owners, the picks made, who is
// on the clock until when in the viewer's time, and the franchise's own next pick.
func (a *App) TargetNativeDraft(scope commissioner.RawNativeDraftScope) (commissioner.NativeDraft, error) {
	if scope.At.IsZero() {
		scope.At = a.timeNow()
	}
	if err := scope.Validate(); err != nil {
		return commissioner.NativeDraft{}, fmt.Errorf("rookie draft input: %w", err)
	}
	if err := a.ready(); err != nil {
		return commissioner.NativeDraft{}, fmt.Errorf("rookie draft startup: %w", err)
	}
	ctx, cancel := context.WithTimeout(a.ctx, refreshTimeout)
	defer cancel()
	reading, err := a.history.NativeDraft(ctx, scope, a.ownerZone())
	if err != nil {
		return commissioner.NativeDraft{}, fmt.Errorf("rookie draft: %w", err)
	}
	return reading, nil
}
