package commissioner

import (
	"fmt"
	"sort"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/governance"
)

type PeopleSeat struct {
	Franchise    string `json:"franchise"`
	Name         string `json:"name"`
	Abbreviation string `json:"abbreviation"`
	Owner        string `json:"owner"`
}

type PeopleRole struct {
	Person string `json:"person"`
	Role   string `json:"role"`
}

type PeopleInvite struct {
	ID        string `json:"id"`
	Franchise string `json:"franchise"`
	Person    string `json:"person"`
	Status    string `json:"status"`
}

type People struct {
	League     string         `json:"league"`
	Name       string         `json:"name"`
	Revision   int64          `json:"revision"`
	Seats      []PeopleSeat   `json:"seats"`
	Roles      []PeopleRole   `json:"roles"`
	Invites    []PeopleInvite `json:"invites"`
	DOTCount   int            `json:"dotCount"`
	DOTStatus  string         `json:"dotStatus"`
	Votes      []PeopleVote   `json:"votes"`
	Succession string         `json:"succession"`
	// YourRole and the two flags say what this principal may start, so the forms can stay out of the way.
	YourRole string `json:"yourRole"`
	// YourFranchise is the franchise this principal owns, or empty when they own none.
	YourFranchise  string `json:"yourFranchise"`
	MayChangeRules bool   `json:"mayChangeRules"`
	MayGrantRoles  bool   `json:"mayGrantRoles"`
	MayPostNotices bool   `json:"mayPostNotices"`
}

func PeopleReading(league, principal string, records []Record) (People, error) {
	state, err := GovernanceState(records)
	if err != nil {
		return People{}, err
	}
	out := People{League: league, Name: Replay(records).Name, Revision: int64(len(records)),
		Seats: []PeopleSeat{}, Roles: []PeopleRole{}, Invites: []PeopleInvite{}}
	p := governance.Principal(principal)
	private := state.HasRole(p, governance.Commissioner) || state.HasRole(p, governance.CoCommissioner)
	persons, invites := peopleInventory(records)
	// Seats and names do not depend on the clock, only the alignment beside them does.
	directory, err := nativeDirectory(league, records, state, time.Time{})
	if err != nil {
		return People{}, err
	}
	for _, f := range directory.Franchises {
		out.Seats = append(out.Seats, f.PeopleSeat)
	}
	for _, person := range persons {
		for _, role := range []governance.Role{governance.Commissioner, governance.CoCommissioner, governance.DOT} {
			if !state.HasRole(governance.Principal(person), role) {
				continue
			}
			if role == governance.DOT {
				out.DOTCount++
			}
			if private {
				out.Roles = append(out.Roles, PeopleRole{Person: person, Role: roleLabel(role)})
			}
		}
	}
	out.DOTStatus = fmt.Sprintf("DOT has %d seats", out.DOTCount)
	if out.DOTCount < 4 {
		out.DOTStatus = fmt.Sprintf("Needs a ruling: DOT has %d of 4 seats", out.DOTCount)
	}
	if private {
		out.Invites = readInvites(state, invites)
	}
	out.YourRole = yourRole(state, p, out.Seats)
	out.YourFranchise = yourFranchise(p, out.Seats)
	out.MayChangeRules = private
	_, grantErr := GovernanceAuthorizer{State: state}.Allow(league, principal, "commish.role.grant")
	out.MayGrantRoles = grantErr == nil
	_, postErr := GovernanceAuthorizer{State: state}.Allow(league, principal, NoticePublish)
	out.MayPostNotices = postErr == nil
	out.Votes = readVotes(state, records, len(out.Seats), private)
	out.Succession = successionStatus(state, records)
	return out, nil
}

func yourRole(state governance.State, p governance.Principal, seats []PeopleSeat) string {
	if state.HasRole(p, governance.Commissioner) {
		return roleLabel(governance.Commissioner)
	}
	if state.HasRole(p, governance.CoCommissioner) {
		return roleLabel(governance.CoCommissioner)
	}
	for _, seat := range seats {
		if seat.Owner == string(p) {
			return roleLabel(governance.Owner)
		}
	}
	return "Visitor"
}

func yourFranchise(p governance.Principal, seats []PeopleSeat) string {
	for _, seat := range seats {
		if seat.Owner == string(p) {
			return seat.Franchise
		}
	}
	return ""
}

func peopleInventory(records []Record) ([]string, []string) {
	persons, invites := make(map[string]bool), make(map[string]bool)
	for _, r := range records {
		persons[r.Principal] = true
		if p := r.Command.Args.People; p != nil {
			if p.Principal != "" {
				persons[p.Principal] = true
			}
			if r.Command.Command == "commish.seat.invite" {
				invites[p.ID] = true
			}
		}
	}
	ps, is := []string{}, []string{}
	for p := range persons {
		ps = append(ps, p)
	}
	for id := range invites {
		is = append(is, id)
	}
	sort.Strings(ps)
	sort.Strings(is)
	return ps, is
}

func readSeats(state governance.State, ids []string) []PeopleSeat {
	seats := make([]PeopleSeat, 0, len(ids))
	for _, id := range ids {
		f, ok := state.Franchise(governance.FranchiseID(id))
		if !ok {
			continue
		}
		owner := string(f.Seat)
		if owner == "" {
			owner = "Vacant"
		}
		seats = append(seats, PeopleSeat{Franchise: id, Name: f.Name, Abbreviation: f.Abbreviation, Owner: owner})
	}
	return seats
}

func readInvites(state governance.State, ids []string) []PeopleInvite {
	out := []PeopleInvite{}
	for _, id := range ids {
		i, ok := state.Invite(id)
		if !ok {
			continue
		}
		status := "Available"
		if i.Revoked {
			status = "Revoked"
		} else if i.Redeemed {
			status = "Accepted"
		}
		out = append(out, PeopleInvite{id, string(i.Franchise), string(i.Redeemer), status})
	}
	return out
}

func roleLabel(role governance.Role) string {
	switch role {
	case governance.Commissioner:
		return "Commissioner"
	case governance.CoCommissioner:
		return "Co-commissioner"
	case governance.DOT:
		return "DOT member"
	case governance.Owner:
		return "Owner"
	default:
		return "Unknown role"
	}
}
