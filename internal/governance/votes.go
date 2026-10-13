package governance

import (
	"fmt"
	"maps"
)

type Vote struct {
	ID             string
	Target         Principal
	Election       bool
	Ballots        map[FranchiseID]Principal
	Closed, Passed bool
}
type VoteOutOpened struct {
	ID                      string
	Principal, Commissioner Principal
}

func (VoteOutOpened) Private() bool { return false }
func (e VoteOutOpened) apply(s *State) error {
	if !s.HasRole(e.Commissioner, Commissioner) {
		return fmt.Errorf("choose a current commissioner")
	}
	return s.openVote(e.ID, e.Principal, e.Commissioner, false)
}

type ElectionOpened struct {
	ID        string
	Principal Principal
}

func (ElectionOpened) Private() bool { return false }
func (e ElectionOpened) apply(s *State) error {
	if !s.ElectionNeeded() {
		return fmt.Errorf("an election needs a commissioner succession first")
	}
	return s.openVote(e.ID, e.Principal, s.outgoing, true)
}
func (s *State) openVote(id string, p, target Principal, election bool) error {
	if !s.HasRole(p, CoCommissioner) && !s.HasRole(p, DOT) {
		return fmt.Errorf("only a co-commissioner or DOT member may open this vote")
	}
	if id == "" {
		return fmt.Errorf("give the vote an identity")
	}
	if _, exists := s.votes[id]; exists {
		return fmt.Errorf("that vote identity already exists")
	}
	for _, v := range s.votes {
		if !v.Closed {
			return fmt.Errorf("close the current vote first")
		}
	}
	s.votes[id] = Vote{ID: id, Target: target, Election: election, Ballots: make(map[FranchiseID]Principal)}
	return nil
}

type BallotCast struct {
	ID        string
	Principal Principal
	Choice    Principal
}

func (BallotCast) Private() bool { return false }
func (e BallotCast) apply(s *State) error {
	v, ok := s.votes[e.ID]
	id := s.seat(e.Principal)
	if !ok || v.Closed || id == "" {
		return fmt.Errorf("only a seated owner may vote in an open ballot")
	}
	if v.Election && e.Choice == "" {
		return fmt.Errorf("choose a successor")
	}
	if !v.Election && e.Choice != "yes" && e.Choice != "no" {
		return fmt.Errorf("choose yes or no")
	}
	v.Ballots[id] = e.Choice
	s.votes[e.ID] = v
	return nil
}

type VoteClosed struct {
	ID     string
	Winner Principal
}

func (VoteClosed) Private() bool { return false }
func (e VoteClosed) apply(s *State) error {
	v, ok := s.votes[e.ID]
	if !ok || v.Closed {
		return fmt.Errorf("choose an open vote")
	}
	if v.Election {
		// Election selection belongs to the caller: the spec sets no election formula.
		if e.Winner == "" {
			return fmt.Errorf("record the owners' elected successor")
		}
		found := false
		for _, choice := range v.Ballots {
			if choice == e.Winner {
				found = true
			}
		}
		if !found {
			return fmt.Errorf("the successor needs an owner ballot")
		}
		s.transfer(v.Target, e.Winner)
		v.Passed = true
	} else {
		yes := 0
		for _, choice := range v.Ballots {
			if choice == "yes" {
				yes++
			}
		}
		v.Passed = yes >= VoteOutThreshold(len(s.franchises))
		if v.Passed {
			s.startSuccession(v.Target)
		}
	}
	v.Closed = true
	s.votes[e.ID] = v
	return nil
}
func VoteOutThreshold(franchises int) int { return franchises - franchises/3 }
func (s State) Vote(id string) (Vote, bool) {
	v, ok := s.votes[id]
	v.Ballots = maps.Clone(v.Ballots)
	return v, ok
}
