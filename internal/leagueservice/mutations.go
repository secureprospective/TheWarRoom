package leagueservice

import (
	"context"
	"errors"
	"fmt"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
)

// Execute dispatches generic and owner market commands to the same atomic append.
// The store owns retry lookup, current ownership, calendar gates and held-fact legality.
func (a *Service) Execute(ctx context.Context, raw commissioner.RawCommand) (commissioner.Receipt, error) {
	if raw.Args.WaiverOutcome != nil || raw.Args.BidOutcome != nil || raw.Args.TradeOutcome != nil || raw.Args.DraftOutcome != nil {
		return commissioner.Receipt{}, fmt.Errorf("command decisions are made by the app")
	}
	raw, err := raw.Prepared()
	if err != nil {
		return commissioner.Receipt{}, fmt.Errorf("prepare league change: %w", err)
	}
	if err := raw.Validate(); err != nil {
		return commissioner.Receipt{}, fmt.Errorf("league change input: %w", err)
	}
	switch raw.Command {
	case commissioner.NativeSeed:
		return commissioner.Receipt{}, fmt.Errorf("record the starting snapshot through the held-data action")
	case commissioner.LineupSet:
		return commissioner.Receipt{}, fmt.Errorf("set a lineup from the lineup screen, which asks the league calendar first")
	case commissioner.DraftPick:
		return commissioner.Receipt{}, fmt.Errorf("a rookie pick is made by the franchise on the clock, in the draft room")
	}
	if err := a.host.Ready(); err != nil {
		return commissioner.Receipt{}, fmt.Errorf("league change startup: %w", err)
	}
	principal, err := a.host.CommissionerPrincipal()
	if err != nil {
		//nolint:wrapcheck // Preserve the session refusal and its identity.
		return commissioner.Receipt{}, err
	}
	ctx, cancel := context.WithTimeout(ctx, refreshTimeout)
	defer cancel()
	receipt, err := a.host.ExecuteCommissionerWithFacts(ctx, raw, principal,
		a.host.CommissionerAuthorization(), a.host.NativeFacts(), a.host.CommissionerZone())
	if err != nil {
		return commissioner.Receipt{}, ChangeError(err)
	}
	return receipt, nil
}

// ChangeError keeps domain refusals in plain words and adds context to storage failures.
func ChangeError(err error) error {
	var refused *commissioner.Refusal
	if errors.As(err, &refused) {
		return errors.New(refused.Reason())
	}
	if err != nil {
		return fmt.Errorf("league change: %w", err)
	}
	return nil
}
