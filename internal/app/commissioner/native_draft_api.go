package commissioner

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
)

// TargetNativeDraftPick makes the owner's pick for the franchise on the clock. It asks the calendar first:
// a closed draft window refuses the pick in the calendar's own words.
func (a *API) TargetNativeDraftPick(input commissioner.DraftPickInput) (commissioner.Receipt, error) {
	if err := input.Validate(); err != nil {
		return commissioner.Receipt{}, fmt.Errorf("rookie pick input: %w", err)
	}
	if err := a.host.Ready(); err != nil {
		return commissioner.Receipt{}, fmt.Errorf("rookie pick startup: %w", err)
	}
	body, err := json.Marshal(input)
	if err != nil {
		return commissioner.Receipt{}, fmt.Errorf("rookie pick input: %w", err)
	}
	receipt, err := a.host.RunNativeOwnerAction(input.League, input.Franchise, draftPickName, body)
	var refused *commissioner.Refusal
	if errors.As(err, &refused) {
		return commissioner.Receipt{}, errors.New(refused.Reason())
	}
	if err != nil {
		return commissioner.Receipt{}, fmt.Errorf("rookie pick: %w", err)
	}
	a.host.CloseFinishedDrafts(input.League)
	return receipt, nil
}

// TargetNativeDraft reads the rookie draft for a franchise: the order and its owners, the picks made, who is
// on the clock until when in the viewer's time, and the franchise's own next pick.
func (a *API) TargetNativeDraft(scope commissioner.RawNativeDraftScope) (commissioner.NativeDraft, error) {
	if scope.At.IsZero() {
		scope.At = a.host.CommissionerNow()
	}
	if err := scope.Validate(); err != nil {
		return commissioner.NativeDraft{}, fmt.Errorf("rookie draft input: %w", err)
	}
	if err := a.host.Ready(); err != nil {
		return commissioner.NativeDraft{}, fmt.Errorf("rookie draft startup: %w", err)
	}
	ctx, cancel := context.WithTimeout(a.host.CommissionerContext(), refreshTimeout)
	defer cancel()
	reading, err := a.host.NativeDraft(ctx, scope, a.host.CommissionerZone(), a.host.NativeFacts())
	if err != nil {
		return commissioner.NativeDraft{}, fmt.Errorf("rookie draft: %w", err)
	}
	return reading, nil
}
