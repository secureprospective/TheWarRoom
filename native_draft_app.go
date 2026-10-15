package main

import (
	"context"
	"errors"
	"fmt"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

const (
	draftPickName  = "draft.pick"
	draftPickOpens = leaguerules.Action("draft_pick")
)

// draftPickCall carries one owner's pick to the registered action: the action itself takes only the
// gate's permit, so the details travel in the context the gate hands it.
type draftPickCall struct {
	app     *App
	input   commissioner.DraftPickInput
	receipt commissioner.Receipt
}

type draftPickKey struct{}

// runDraftPick is the registered native owner action for a rookie pick. The gate has already said yes for
// this franchise; the store records the pick only for that franchise's seated owner.
func runDraftPick(ctx context.Context, permit commissioner.Permit) error {
	call, ok := ctx.Value(draftPickKey{}).(*draftPickCall)
	if !ok {
		return fmt.Errorf("rookie pick: %w", commissioner.Refused(errors.New("choose a rookie for your pick")))
	}
	principal, err := call.app.commissionerPrincipal()
	if err != nil {
		return err
	}
	receipt, err := call.app.history.RecordDraftPick(ctx, permit, call.input, principal,
		call.app.nativeFacts(), call.app.ownerZone())
	if err != nil {
		return fmt.Errorf("record rookie pick: %w", err)
	}
	call.receipt = receipt
	return nil
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
	ctx, cancel := context.WithTimeout(a.ctx, refreshTimeout)
	defer cancel()
	call := &draftPickCall{app: a, input: input}
	err := a.runNativeActionIn(context.WithValue(ctx, draftPickKey{}, call), nativeOwnerActions(),
		input.League, input.Franchise, draftPickName)
	var refused *commissioner.Refusal
	if errors.As(err, &refused) {
		return commissioner.Receipt{}, errors.New(refused.Reason())
	}
	if err != nil {
		return commissioner.Receipt{}, fmt.Errorf("rookie pick: %w", err)
	}
	return call.receipt, nil
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
