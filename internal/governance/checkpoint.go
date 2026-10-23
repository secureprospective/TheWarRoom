package governance

import (
	"fmt"
	"time"
)

// RawCheckpoint is derived state, not an alternate source of league history.
type RawCheckpoint struct {
	Created                           bool
	Franchises                        map[FranchiseID]Franchise
	Roles                             map[Principal]map[Role]bool
	Invites                           map[string]Invite
	Visibility                        map[string]string
	Activity                          map[Principal]time.Time
	InSeason, Offseason               time.Duration
	Votes                             map[string]Vote
	Pending, Outgoing, CoCommissioner Principal
	ElectionPeriod, RunoffPeriod      time.Duration
}

func (r RawCheckpoint) Validate() error {
	if !r.Created || len(r.Franchises) == 0 || len(r.Roles) == 0 || r.Invites == nil || r.Visibility == nil || r.Activity == nil || r.Votes == nil ||
		r.InSeason <= 0 || r.Offseason <= 0 || r.ElectionPeriod <= 0 || r.RunoffPeriod <= 0 {
		return fmt.Errorf("invalid league people checkpoint")
	}
	if err := r.validateMembers(); err != nil {
		return err
	}
	return r.validateEvidence()
}

func (r RawCheckpoint) validateMembers() error {
	for id, f := range r.Franchises {
		if id == "" || f.ID != id || f.Abilities == nil {
			return fmt.Errorf("invalid checkpoint franchise")
		}
	}
	for p, roles := range r.Roles {
		if p == "" || roles == nil {
			return fmt.Errorf("invalid checkpoint principal")
		}
		for role := range roles {
			if !validRole(role) {
				return fmt.Errorf("invalid checkpoint role")
			}
		}
	}
	return nil
}

func (r RawCheckpoint) validateEvidence() error {
	for id, invite := range r.Invites {
		if id == "" || invite.ID != id {
			return fmt.Errorf("invalid checkpoint invitation")
		}
	}
	for id, vote := range r.Votes {
		if id == "" || vote.ID != id || vote.Ballots == nil {
			return fmt.Errorf("invalid checkpoint vote")
		}
	}
	for p, at := range r.Activity {
		if p == "" || at.IsZero() {
			return fmt.Errorf("invalid checkpoint activity")
		}
	}
	return nil
}

func (s State) Checkpoint() RawCheckpoint {
	n := s.clone()
	return RawCheckpoint{Created: n.created, Franchises: n.franchises, Roles: n.roles, Invites: n.invites,
		Visibility: n.visibility, Activity: n.activity, InSeason: n.inSeason, Offseason: n.offseason, Votes: n.votes,
		Pending: n.pending, Outgoing: n.outgoing, CoCommissioner: n.coCommissioner, ElectionPeriod: n.electionPeriod, RunoffPeriod: n.runoffPeriod}
}

// Resume owns its copy, so a failed suffix cannot mutate the checkpoint.
func Resume(raw RawCheckpoint, events []Event, offset int) (State, error) {
	if err := raw.Validate(); err != nil {
		return State{}, err
	}
	s := State{created: raw.Created, franchises: raw.Franchises, roles: raw.Roles, invites: raw.Invites,
		visibility: raw.Visibility, activity: raw.Activity, inSeason: raw.InSeason, offseason: raw.Offseason, votes: raw.Votes,
		pending: raw.Pending, outgoing: raw.Outgoing, coCommissioner: raw.CoCommissioner, electionPeriod: raw.ElectionPeriod, runoffPeriod: raw.RunoffPeriod}
	s = s.clone()
	for i, e := range events {
		n, err := Apply(s, e)
		if err != nil {
			return State{}, fmt.Errorf("change %d: %w", offset+i+1, err)
		}
		s = n
	}
	return s, nil
}
