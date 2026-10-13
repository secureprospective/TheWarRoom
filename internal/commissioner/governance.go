package commissioner

import (
	"fmt"
	"strings"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/governance"
)

func validateFranchises(ids []string) error {
	if len(ids) == 0 {
		return fmt.Errorf("list the new league's franchises")
	}
	seen := make(map[string]bool, len(ids))
	for _, id := range ids {
		if id == "" || strings.TrimSpace(id) != id || seen[id] {
			return fmt.Errorf("give each franchise its own nonempty identity")
		}
		seen[id] = true
	}
	return nil
}

func recordEvent(r Record) (governance.Event, error) {
	if r.Command.Command == Create {
		ids := make([]governance.FranchiseID, len(r.Command.Args.Franchises))
		for i, id := range r.Command.Args.Franchises {
			ids[i] = governance.FranchiseID(id)
		}
		return governance.LeagueCreated{Creator: governance.Principal(r.Principal), Franchises: ids}, nil
	}
	if r.Command.Args.People != nil {
		return r.Command.Args.People.event(r.Command.Command)
	}
	return nil, nil
}

func GovernanceState(records []Record) (governance.State, error) {
	projection := Replay(records)
	events := []governance.Event{}
	for _, r := range records {
		// Creation remains bootstrap metadata, just as in the ledger's identity projection.
		if !projection.Active[r.Receipt.Entry] && r.Command.Command != Create {
			continue
		}
		event, err := recordEvent(r)
		if err != nil {
			return governance.State{}, err
		}
		if event != nil {
			events = append(events, event)
		}
	}
	state, err := governance.Replay(events)
	if err != nil {
		return governance.State{}, fmt.Errorf("read league people: %w", err)
	}
	return state, nil
}

type GovernanceAuthorizer struct{ State governance.State }

func (a GovernanceAuthorizer) Allow(_ string, principal, command string) (string, error) {
	control := ""
	switch command {
	case Create:
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
	auth := GovernanceAuthorizer{State: state}
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
		if _, err := auth.Allow(r.Command.League, principal, r.Command.Command); err != nil {
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
		return fmt.Errorf("people change refused: %w", err)
	}
	return nil
}
