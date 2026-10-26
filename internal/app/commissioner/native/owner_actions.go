package native

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"slices"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

type ExecuteOwner func(context.Context, commissioner.RawCommand) (commissioner.Receipt, error)

// OwnerFamily is one kind of native owner command. seat says which franchise a command acts for and which
// calendar action opens it (ok is false when the command carries none of the family's details); choose and
// label are the words that ask for the command and prefix its errors.
type OwnerFamily struct {
	seat   func(raw commissioner.RawCommand) (franchise string, action leaguerules.Action, ok bool)
	choose string
	label  string
}

// RecordOwnerAction is the Run of a registered owner action: it records a command the gate has cleared. The
// store asks the gate again inside its own transaction, where it also judges the action itself.
func RecordOwnerAction(ctx context.Context, family OwnerFamily, execute ExecuteOwner, permit commissioner.Permit, input json.RawMessage) (commissioner.Receipt, error) {
	var raw commissioner.RawCommand
	if err := json.Unmarshal(input, &raw); err != nil {
		return commissioner.Receipt{}, fmt.Errorf("choose %s", family.choose)
	}
	franchise, action, ok := family.seat(raw)
	if !ok {
		return commissioner.Receipt{}, fmt.Errorf("choose %s", family.choose)
	}
	if permit.League() != raw.League || permit.Franchise() != franchise || permit.Action() != action {
		return commissioner.Receipt{}, fmt.Errorf("ask the league calendar before acting")
	}
	receipt, err := execute(ctx, raw)
	if err != nil {
		return commissioner.Receipt{}, fmt.Errorf("record %s: %w", family.label, err)
	}
	return receipt, nil
}

func TradeFamily() OwnerFamily {
	return OwnerFamily{choose: "to offer, accept, decline or withdraw a trade", label: "trade action",
		seat: func(raw commissioner.RawCommand) (string, leaguerules.Action, bool) {
			if raw.Args.Trade == nil {
				return "", "", false
			}
			return raw.Args.Trade.Franchise, commissioner.OwnerTradeAction(raw.Command), true
		}}
}

func WaiverFamily() OwnerFamily {
	return OwnerFamily{choose: "a native cut or waiver claim", label: "native waiver action",
		seat: func(raw commissioner.RawCommand) (string, leaguerules.Action, bool) {
			if raw.Args.Waiver == nil {
				return "", "", false
			}
			return raw.Args.Waiver.Franchise, commissioner.OwnerWaiverAction(raw.Command), true
		}}
}

func BidFamily() OwnerFamily {
	return OwnerFamily{choose: "to place, raise or withdraw a bid", label: "native bid action",
		seat: func(raw commissioner.RawCommand) (string, leaguerules.Action, bool) {
			if raw.Args.Bid == nil {
				return "", "", false
			}
			return raw.Args.Bid.Franchise, commissioner.OwnerBidAction(raw.Command), true
		}}
}

func RecordLineup(ctx context.Context, permit commissioner.Permit, input json.RawMessage,
	readPrincipal func() (string, error), execute func(context.Context, commissioner.RawCommand, string) (commissioner.Receipt, error)) (commissioner.Receipt, error) {
	in, err := DecodeLineupSet(input)
	if err != nil {
		return commissioner.Receipt{}, err
	}
	if permit.League() != in.League || permit.Franchise() != in.Franchise || permit.Action() != leaguerules.Action("lineup") {
		return commissioner.Receipt{}, fmt.Errorf("this lineup was not cleared by the league calendar")
	}
	principal, err := readPrincipal()
	if err != nil {
		return commissioner.Receipt{}, err
	}
	raw := commissioner.RawCommand{League: in.League, Command: commissioner.LineupSet, Expected: in.Expected,
		Request: in.Request, Effective: "now", Visibility: "private",
		Args: commissioner.RawArgs{Lineup: &commissioner.RawLineupChange{Season: in.Season, Franchise: in.Franchise,
			Week: in.Week, Starters: in.Starters}}}
	receipt, err := execute(ctx, raw, principal)
	if err != nil {
		return commissioner.Receipt{}, fmt.Errorf("record the lineup: %w", err)
	}
	return receipt, nil
}

func RecordDraftPick(ctx context.Context, permit commissioner.Permit, body json.RawMessage,
	readPrincipal func() (string, error), record func(context.Context, commissioner.Permit, commissioner.DraftPickInput, string) (commissioner.Receipt, error)) (commissioner.Receipt, error) {
	var input commissioner.DraftPickInput
	if err := json.Unmarshal(body, &input); err != nil {
		return commissioner.Receipt{}, fmt.Errorf("rookie pick: %w",
			commissioner.Refused(errors.New("choose a rookie for your pick")))
	}
	if err := input.Validate(); err != nil {
		return commissioner.Receipt{}, fmt.Errorf("rookie pick input: %w", commissioner.Refused(err))
	}
	principal, err := readPrincipal()
	if err != nil {
		return commissioner.Receipt{}, err
	}
	receipt, err := record(ctx, permit, input, principal)
	if err != nil {
		return commissioner.Receipt{}, fmt.Errorf("record rookie pick: %w", err)
	}
	return receipt, nil
}

var ErrActionClosed = errors.New("league action closed")

type OwnerAction struct {
	Name  string
	Opens leaguerules.Action
	Run   func(context.Context, commissioner.Permit, json.RawMessage) (commissioner.Receipt, error)
}

type ContractOwnerAction struct {
	Name   string
	Opens  leaguerules.Action
	Action nativecheck.ContractAction
}

func ContractOwnerActions() []ContractOwnerAction {
	var out []ContractOwnerAction
	for _, action := range nativecheck.ContractActions() {
		window, gated := commissioner.OwnerContractWindow(action)
		if !gated {
			continue
		}
		out = append(out, ContractOwnerAction{Name: "contract." + string(action), Opens: window, Action: action})
	}
	return out
}

func StartContractAction(ctx context.Context, permit commissioner.Permit, action nativecheck.ContractAction,
	recordsFor func(context.Context, string) ([]commissioner.Record, error), now func() time.Time) error {
	records, err := recordsFor(ctx, permit.League())
	if err != nil {
		return err
	}
	if err := commissioner.ContractActionOn(records, now(), action); err != nil {
		return fmt.Errorf("start %s: %w", action.Name(), err)
	}
	return nil
}

type OwnerGate struct {
	Ready   func() error
	Records func(context.Context, string) ([]commissioner.Record, error)
	Now     func() time.Time
	Zone    func() *time.Location
}

func RunOwnerAction(ctx context.Context, registry []OwnerAction, league, franchise, name string, input json.RawMessage, source OwnerGate) (commissioner.Receipt, error) {
	index := slices.IndexFunc(registry, func(r OwnerAction) bool { return r.Name == name })
	if index < 0 {
		return commissioner.Receipt{}, fmt.Errorf("this league action is not available")
	}
	action := registry[index]
	if err := source.Ready(); err != nil {
		return commissioner.Receipt{}, fmt.Errorf("league action startup: %w", err)
	}
	records, err := source.Records(ctx, league)
	if err != nil {
		return commissioner.Receipt{}, err
	}
	gate, err := commissioner.NativeActionGate(league, records, franchise, action.Opens, source.Now(), source.Zone())
	if err != nil {
		return commissioner.Receipt{}, fmt.Errorf("league action gate: %w", err)
	}
	permit, ok := gate.Permit()
	if !ok {
		return commissioner.Receipt{}, fmt.Errorf("%w: %w", ErrActionClosed, commissioner.Refused(errors.New(gate.Sentence)))
	}
	return action.Run(ctx, permit, input)
}

// CloseFinishedDrafts is best-effort after the successful pick; a close failure never undoes it.
func CloseFinishedDrafts(ctx context.Context, league string, closeDrafts func(context.Context, string) error, logFailure func(string)) {
	if err := closeDrafts(ctx, league); err != nil {
		logFailure(fmt.Sprintf("the war room: close finished rookie drafts for %s: %v", league, err))
	}
}
