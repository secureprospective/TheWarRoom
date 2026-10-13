package commissioner

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/secureprospective/TheWarRoom/internal/governance"
)

type RawPeople struct {
	Franchise    string          `json:"franchise,omitempty"`
	Principal    string          `json:"principal,omitempty"`
	Name         string          `json:"name,omitempty"`
	Abbreviation string          `json:"abbreviation,omitempty"`
	ID           string          `json:"id,omitempty"`
	Role         governance.Role `json:"role,omitempty"`
	Ability      string          `json:"ability,omitempty"`
	On           bool            `json:"on,omitempty"`
	All          bool            `json:"all,omitempty"`
	Report       string          `json:"report,omitempty"`
	Audience     string          `json:"audience,omitempty"`
}

type peopleDefinition struct {
	command, control, gravity, label string
}

func peopleDefinitions() []peopleDefinition {
	return []peopleDefinition{
		{"commish.franchise.edit", "CM-018", "G2", "Changed a franchise's name and abbreviation"},
		{"commish.seat.invite", "CM-019", "G2", "Issued an owner invitation"},
		{"commish.seat.invite.revoke", "CM-019", "G2", "Revoked an owner invitation"},
		{"commish.seat.invite.redeem", "CM-019", "G2", "Accepted an owner invitation"},
		{"commish.seat.remove", "CM-019", "G3", "Removed an owner from a franchise"},
		{"commish.seat.replace", "CM-019", "G3", "Replaced a franchise's owner"},
		{"commish.abilities.set", "CM-020", "G2", "Changed an owner ability"},
		{"commish.visibility.set", "CM-020", "G2", "Changed who may read a report"},
		{"commish.role.grant", "CM-021", "G3", "Granted a league role"},
		{"commish.role.revoke", "CM-021", "G3", "Revoked a league role"},
	}
}

func peopleDefinitionFor(command string) (peopleDefinition, bool) {
	for _, d := range peopleDefinitions() {
		if d.command == command {
			return d, true
		}
	}
	return peopleDefinition{}, false
}

func (r RawPeople) Validate(command string) error {
	selected, err := r.selected(command)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(selected, r) {
		return fmt.Errorf("send only the details this people change needs")
	}
	for _, text := range []string{r.Franchise, r.Principal, r.Name, r.Abbreviation, r.ID,
		r.Ability, r.Report, r.Audience, string(r.Role)} {
		if text != strings.TrimSpace(text) {
			return fmt.Errorf("remove surrounding spaces from the people change")
		}
	}
	if command == "commish.abilities.set" {
		return r.validateAbility()
	}
	return nil
}

func (r RawPeople) validateAbility() error {
	if r.All == (r.Franchise != "") {
		return fmt.Errorf("choose one franchise or all franchises, not both")
	}
	for _, name := range []string{"lineups", "add/drop", "drop-only", "trades", "pick trades", "IR", "taxi",
		"articles", "board", "chat", "polls", "customize franchise", "home page"} {
		if r.Ability == name {
			return nil
		}
	}
	return fmt.Errorf("choose a supported owner ability")
}

func (r RawPeople) selected(command string) (RawPeople, error) {
	switch command {
	case "commish.franchise.edit":
		return RawPeople{Franchise: r.Franchise, Name: r.Name, Abbreviation: r.Abbreviation}, nil
	case "commish.seat.invite":
		return RawPeople{ID: r.ID, Franchise: r.Franchise, Principal: r.Principal}, nil
	case "commish.seat.invite.revoke":
		return RawPeople{ID: r.ID}, nil
	case "commish.seat.invite.redeem":
		return RawPeople{ID: r.ID, Principal: r.Principal}, nil
	case "commish.seat.remove":
		return RawPeople{Franchise: r.Franchise}, nil
	case "commish.seat.replace":
		return RawPeople{Franchise: r.Franchise, Principal: r.Principal}, nil
	case "commish.abilities.set":
		return RawPeople{Franchise: r.Franchise, All: r.All, Ability: r.Ability, On: r.On}, nil
	case "commish.visibility.set":
		return RawPeople{Report: r.Report, Audience: r.Audience}, nil
	case "commish.role.grant", "commish.role.revoke":
		return RawPeople{Principal: r.Principal, Role: r.Role}, nil
	default:
		return RawPeople{}, fmt.Errorf("choose an available people change")
	}
}

func (r RawPeople) label(command string) string {
	switch command {
	case "commish.franchise.edit":
		return fmt.Sprintf("Named franchise %s “%s” (%s)", r.Franchise, r.Name, r.Abbreviation)
	case "commish.role.grant":
		return fmt.Sprintf("Granted %s to %s", roleLabel(r.Role), r.Principal)
	case "commish.role.revoke":
		return fmt.Sprintf("Revoked %s from %s", roleLabel(r.Role), r.Principal)
	case "commish.seat.replace":
		return fmt.Sprintf("Assigned %s to franchise %s", r.Principal, r.Franchise)
	case "commish.seat.remove":
		return fmt.Sprintf("Removed the owner of franchise %s; the franchise remains", r.Franchise)
	case "commish.abilities.set":
		who, status := "franchise "+r.Franchise, "off"
		if r.All {
			who = "all franchises"
		}
		if r.On {
			status = "on"
		}
		return fmt.Sprintf("Set %s %s for %s", r.Ability, status, who)
	case "commish.visibility.set":
		who := "all owners"
		if r.Audience == "commish" {
			who = "the commissioner and co-commissioner"
		}
		return fmt.Sprintf("Made report %s visible to %s", r.Report, who)
	default:
		d, _ := peopleDefinitionFor(command)
		return d.label + " “" + r.ID + "”"
	}
}

func (r RawPeople) event(command string) (governance.Event, error) {
	if err := r.Validate(command); err != nil {
		return nil, err
	}
	f, p := governance.FranchiseID(r.Franchise), governance.Principal(r.Principal)
	switch command {
	case "commish.franchise.edit":
		return governance.FranchiseIdentitySet{Franchise: f, Name: r.Name, Abbreviation: r.Abbreviation}, nil
	case "commish.seat.invite":
		return governance.InviteIssued{ID: r.ID, Franchise: f, Redeemer: p}, nil
	case "commish.seat.invite.revoke":
		return governance.InviteRevoked{ID: r.ID}, nil
	case "commish.seat.invite.redeem":
		return governance.InviteRedeemed{ID: r.ID, Principal: p}, nil
	case "commish.seat.remove":
		return governance.SeatRemoved{Franchise: f}, nil
	case "commish.seat.replace":
		return governance.SeatReplaced{Franchise: f, Principal: p}, nil
	case "commish.abilities.set":
		return governance.AbilitySet{Franchise: f, All: r.All, Ability: r.Ability, On: r.On}, nil
	case "commish.visibility.set":
		return governance.VisibilitySet{Report: r.Report, Audience: r.Audience}, nil
	case "commish.role.grant":
		return governance.RoleGranted{Principal: p, Role: r.Role}, nil
	case "commish.role.revoke":
		return governance.RoleRevoked{Principal: p, Role: r.Role}, nil
	default:
		return nil, fmt.Errorf("choose an available people change")
	}
}
