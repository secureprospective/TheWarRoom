// Package governance is an identity-agnostic, pure league policy fold.
// Desktop callers must refuse remote ballots and invite redemptions: "Remote
// owners cannot vote or join from this desktop until a league server exists."
// Principals and activity evidence must come from a trusted adapter, not a selector.
package governance

import (
	"fmt"
	"maps"
	"sort"
	"time"
)

type Principal string
type FranchiseID string
type Role string

const (
	Commissioner   Role = "commissioner"
	CoCommissioner Role = "co-commissioner"
	DOT            Role = "DOT"
	Owner          Role = "owner"
)

type Franchise struct {
	ID                 FranchiseID
	Name, Abbreviation string
	Seat               Principal
	Abilities          map[string]bool
}
type Invite struct {
	ID                string
	Franchise         FranchiseID
	Redeemer          Principal
	Revoked, Redeemed bool
}
type State struct {
	created                      bool
	franchises                   map[FranchiseID]Franchise
	roles                        map[Principal]map[Role]bool
	invites                      map[string]Invite
	visibility                   map[string]string
	activity                     map[Principal]time.Time
	inSeason, offseason          time.Duration
	votes                        map[string]Vote
	pending                      Principal
	outgoing                     Principal
	coCommissioner               Principal
	electionPeriod, runoffPeriod time.Duration
}
type Event interface {
	apply(*State) error
	Private() bool
}

func emptyState() State {
	return State{franchises: make(map[FranchiseID]Franchise), roles: make(map[Principal]map[Role]bool),
		invites: make(map[string]Invite), visibility: make(map[string]string), activity: make(map[Principal]time.Time),
		inSeason: 14 * 24 * time.Hour, offseason: 30 * 24 * time.Hour, electionPeriod: 7 * 24 * time.Hour,
		runoffPeriod: 7 * 24 * time.Hour, votes: make(map[string]Vote)}
}
func (s State) clone() State {
	n := s
	n.franchises = maps.Clone(s.franchises)
	for id, f := range n.franchises {
		f.Abilities = maps.Clone(f.Abilities)
		n.franchises[id] = f
	}
	n.roles = maps.Clone(s.roles)
	for p, roles := range n.roles {
		n.roles[p] = maps.Clone(roles)
	}
	n.invites = maps.Clone(s.invites)
	n.visibility = maps.Clone(s.visibility)
	n.activity = maps.Clone(s.activity)
	n.votes = maps.Clone(s.votes)
	for id, v := range n.votes {
		v = v.clone()
		n.votes[id] = v
	}
	return n
}
func Apply(s State, e Event) (State, error) {
	if e == nil {
		return s, fmt.Errorf("no league change was supplied")
	}
	n := s.clone()
	if !n.created {
		n = emptyState()
	}
	if err := e.apply(&n); err != nil {
		return s, fmt.Errorf("league change refused: %w", err)
	}
	return n, nil
}
func Replay(events []Event) (State, error) {
	s := emptyState()
	for i, e := range events {
		n, err := Apply(s, e)
		if err != nil {
			return State{}, fmt.Errorf("change %d: %w", i+1, err)
		}
		s = n
	}
	return s, nil
}
func (s State) HasRole(p Principal, r Role) bool { return s.roles[p][r] }
func (s State) Franchise(id FranchiseID) (Franchise, bool) {
	f, ok := s.franchises[id]
	f.Abilities = maps.Clone(f.Abilities)
	return f, ok
}
func (s State) Invite(id string) (Invite, bool) { i, ok := s.invites[id]; return i, ok }
func (s State) Audience(report string) string   { return s.visibility[report] }
func (s State) members(r Role) []Principal {
	ps := []Principal{}
	for p, roles := range s.roles {
		if roles[r] {
			ps = append(ps, p)
		}
	}
	sort.Slice(ps, func(i, j int) bool { return ps[i] < ps[j] })
	return ps
}
func (s State) seat(p Principal) FranchiseID {
	for id, f := range s.franchises {
		if p != "" && f.Seat == p {
			return id
		}
	}
	return ""
}
func (s *State) grant(p Principal, r Role) {
	if s.roles[p] == nil {
		s.roles[p] = make(map[Role]bool)
	}
	s.roles[p][r] = true
}
func validRole(r Role) bool { return r == Commissioner || r == CoCommissioner || r == DOT }
func (s State) requireCreated() error {
	if !s.created {
		return fmt.Errorf("create the league first")
	}
	return nil
}

// SuccessionPeriods returns the inactivity and voting periods currently in force.
func (s State) SuccessionPeriods() (time.Duration, time.Duration, time.Duration, time.Duration) {
	return s.inSeason, s.offseason, s.electionPeriod, s.runoffPeriod
}

// LastActivity is the recorded activity evidence, never a wall-clock guess.
func (s State) LastActivity(p Principal) time.Time { return s.activity[p] }
