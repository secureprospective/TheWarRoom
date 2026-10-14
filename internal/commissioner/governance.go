package commissioner

import (
	"fmt"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/governance"
)

func recordEvents(r Record) ([]governance.Event, error) {
	if r.Command.Command == Create {
		return creationEvents(r), nil
	}
	if r.Command.Args.People != nil {
		people := r.Command.Args.People
		var event governance.Event
		var err error
		if _, ok := successionDefinitionFor(r.Command.Command); ok {
			event, err = people.successionEvent(r.Command.Command, r.Receipt.RecordedAt)
		} else {
			event, err = people.event(r.Command.Command)
		}
		if err != nil {
			return nil, err
		}
		return []governance.Event{event}, nil
	}
	return nil, nil
}

func GovernanceState(records []Record) (governance.State, error) {
	projection := Replay(records)
	events := []governance.Event{}
	for _, r := range records {
		// Creation remains bootstrap metadata, just as in the ledger's identity projection.
		if !projection.Active[r.Receipt.Entry] && r.Command.Command != Create {
			// A reversed change was still the person's activity when it was recorded.
			events = withActivity(events, r)
			continue
		}
		recorded, err := recordEvents(r)
		if err != nil {
			return governance.State{}, err
		}
		events = append(events, recorded...)
		events = withActivity(events, r)
	}
	state, err := governance.Replay(events)
	if err != nil {
		return governance.State{}, fmt.Errorf("read league people: %w", err)
	}
	return state, nil
}

type GovernanceAuthorizer struct{ State governance.State }

func (a GovernanceAuthorizer) Allow(_ string, principal, command string) (string, error) {
	if d, ok := successionDefinitionFor(command); ok {
		return successionPermission(a.State, principal, d)
	}
	control := ""
	switch command {
	case ContractEdit, CapAdjust, CapRelief:
		control = "CM-034"
	case ReserveException:
		control = "CM-038"
	case Create, NativeSeed:
		control = "CM-001"
	case Revert:
		control = "CM-039"
	case "commish.ledger.read":
		p := governance.Principal(principal)
		if a.State.HasRole(p, governance.Commissioner) {
			return "commissioner", nil
		}
		if a.State.HasRole(p, governance.CoCommissioner) {
			return "cocommissioner", nil
		}
		return "owner", nil
	default:
		if d, ok := peopleDefinitionFor(command); ok {
			control = d.control
		}
	}
	row, ok := governance.CensusControl(control)
	if !ok {
		return "", fmt.Errorf("this league action is not available")
	}
	role, err := governance.Allow(a.State, governance.Principal(principal), governance.CommandRef{
		ID: row.ID, MinimumRole: row.MinimumRole, COnly: row.COnly})
	if err != nil {
		return "", fmt.Errorf("league permission: %w", err)
	}
	if role == governance.CoCommissioner {
		return "cocommissioner", nil
	}
	return string(role), nil
}

func AuthorizeRecords(raw RawCommand, principal string, records []Record) (string, error) {
	if len(records) == 0 && raw.Command == Create {
		return "commissioner", nil
	}
	state, err := GovernanceState(records)
	if err != nil {
		return "", err
	}
	if raw.Command == "commish.seat.invite.redeem" {
		return authorizeRedemption(state, raw, principal)
	}
	if _, ok := successionDefinitionFor(raw.Command); ok {
		return authorizeSuccession(state, raw, principal)
	}
	auth := GovernanceAuthorizer{State: state}
	if raw.Command == RulesSet {
		return authorizeRule(auth, raw, principal)
	}
	role, err := auth.Allow(raw.League, principal, raw.Command)
	if err != nil {
		return "", err
	}
	if raw.Command == Revert {
		if err := authorizeReversal(auth, raw.Args.Entry, principal, records); err != nil {
			return "", err
		}
	}
	return role, nil
}

func authorizeRedemption(state governance.State, raw RawCommand, principal string) (string, error) {
	if raw.Args.People.Principal != principal {
		return "", fmt.Errorf("%s", "Other owners act from their own app; this desktop can only act as you")
	}
	invite, ok := state.Invite(raw.Args.People.ID)
	if !ok || invite.Redeemer != governance.Principal(principal) {
		return "", fmt.Errorf("that invite is not available to you")
	}
	return "owner", nil
}

func authorizeReversal(auth GovernanceAuthorizer, entry, principal string, records []Record) error {
	for i := len(records) - 1; i >= 0; i-- {
		r := records[i]
		if r.Receipt.Entry != entry {
			continue
		}
		if r.Command.Command == Revert {
			entry = r.Command.Args.Entry
			continue
		}
		var err error
		if r.Command.Command == RulesSet {
			_, err = authorizeRule(auth, r.Command, principal)
		} else {
			_, err = auth.Allow(r.Command.League, principal, r.Command.Command)
		}
		if err != nil {
			return fmt.Errorf("reversing this change needs its original permission: %w", err)
		}
		return nil
	}
	return fmt.Errorf("that recorded change is not in this league")
}

func CheckGovernanceAppend(raw RawCommand, principal string, records []Record, at time.Time) error {
	trial := make([]Record, len(records), len(records)+1)
	copy(trial, records)
	trial = append(trial, Record{Command: raw, Principal: principal,
		Receipt: Receipt{Entry: "proposed." + raw.Request, RecordedAt: at}})
	if _, err := GovernanceState(trial); err != nil {
		return Refused(fmt.Errorf("people change refused: %w", err))
	}
	if _, ok := successionDefinitionFor(raw.Command); !ok {
		return nil
	}
	before, err := GovernanceState(records)
	if err != nil {
		return err
	}
	if err := takeoverWaitedLongest(before, raw, at); err != nil {
		return Refused(fmt.Errorf("people change refused: %w", err))
	}
	return nil
}
