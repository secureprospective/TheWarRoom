package leagueservice

import (
	"context"
	"errors"
	"fmt"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

// Execute dispatches generic and owner market commands to the same atomic append.
// The store owns retry lookup, current ownership, calendar gates and held-fact legality.
func (a *Service) Execute(ctx context.Context, raw commissioner.RawCommand) (commissioner.Receipt, error) {
	if raw.Args.HasMarketDecision() || raw.Args.DraftOutcome != nil {
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
	case commissioner.Cutover:
		return commissioner.Receipt{}, fmt.Errorf("switch a league over from MFL from its review, which shows what differs")
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
	if err := commissioner.AdmitCredential(a.viewContext(ctx), commissioner.Access{Principal: principal,
		League: raw.League, Operation: raw.Command, Mode: "execute", Raw: raw}); err != nil {
		return commissioner.Receipt{}, fmt.Errorf("admit league change: %w", err)
	}
	ctx, cancel := context.WithTimeout(a.viewContext(ctx), refreshTimeout)
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

// MigrationLedger is the league store as the move from MFL uses it: the same replay, checks and atomic
// appends every other league change goes through.
type MigrationLedger struct {
	Store *history.Store
	// Auth is the commissioner authorization in force; nil reads permissions from the league's own ledger.
	Auth func() commissioner.Authorizer
}

func (l MigrationLedger) authorizer() commissioner.Authorizer {
	if l.Auth == nil {
		return nil
	}
	return l.Auth()
}

// Records replays the league's ledger.
func (l MigrationLedger) Records(ctx context.Context, league string) ([]commissioner.Record, error) {
	return Records(ctx, l.Store, league, "move")
}

// Batch records every change or none.
func (l MigrationLedger) Batch(ctx context.Context, raws []commissioner.RawCommand,
	principal string) ([]commissioner.Receipt, error) {
	receipts, err := l.Store.ExecuteCommissionerBatch(ctx, raws, principal, l.authorizer())
	return receipts, ChangeError(err)
}

// Execute records one change.
func (l MigrationLedger) Execute(ctx context.Context, raw commissioner.RawCommand,
	principal string) (commissioner.Receipt, error) {
	receipt, err := l.Store.ExecuteCommissioner(ctx, raw, principal, l.authorizer())
	return receipt, ChangeError(err)
}
