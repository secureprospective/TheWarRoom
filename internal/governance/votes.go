package governance

import (
	"fmt"
	"maps"
	"slices"
	"time"
)

type Vote struct {
	ID               string
	Target           Principal
	Election         bool
	Ballots          map[FranchiseID]Principal
	Closed, Passed   bool
	Runoff           bool
	Candidates       []Principal
	FirstBallots     map[FranchiseID]Principal
	TieBallots       map[Principal]Principal
	OpenedAt, EndsAt time.Time
	Winner           Principal
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
	Now       time.Time
	ID        string
	Principal Principal
}

func (ElectionOpened) Private() bool { return false }
func (e ElectionOpened) apply(s *State) error {
	if !s.ElectionNeeded() {
		return fmt.Errorf("an election needs a commissioner succession first")
	}
	if e.Now.IsZero() {
		return fmt.Errorf("record when voting opens")
	}
	if err := s.openVote(e.ID, e.Principal, s.outgoing, true); err != nil {
		return err
	}
	v := s.votes[e.ID]
	v.OpenedAt, v.EndsAt = e.Now, e.Now.Add(s.electionPeriod)
	s.votes[e.ID] = v
	return nil
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
	Now       time.Time
}

func (BallotCast) Private() bool { return false }
func (e BallotCast) apply(s *State) error {
	v, ok := s.votes[e.ID]
	id := s.seat(e.Principal)
	if !ok || v.Closed || id == "" {
		return fmt.Errorf("only a seated owner may vote in an open ballot")
	}
	if v.Election {
		if e.Choice == "" {
			return fmt.Errorf("choose a successor")
		}
		if e.Now.IsZero() || e.Now.Before(v.OpenedAt) || !e.Now.Before(v.EndsAt) {
			return fmt.Errorf("cast your ballot during the voting period")
		}
		if len(v.TieBallots) != 0 {
			return fmt.Errorf("the runoff ballots are sealed for a tie ruling")
		}
		if v.Runoff && !slices.Contains(v.Candidates, e.Choice) {
			return fmt.Errorf("choose a runoff candidate")
		}
	}
	if !v.Election && e.Choice != "yes" && e.Choice != "no" {
		return fmt.Errorf("choose yes or no")
	}
	v.Ballots[id] = e.Choice
	s.votes[e.ID] = v
	return nil
}

type VoteClosed struct {
	Now    time.Time
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
		return s.closeElection(v, e)
	}
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
	v.Closed = true
	s.votes[e.ID] = v
	return nil
}
func VoteOutThreshold(franchises int) int { return franchises - franchises/3 }
func (s State) Vote(id string) (Vote, bool) {
	v, ok := s.votes[id]
	return v.clone(), ok
}

func (v Vote) clone() Vote {
	v.Ballots = maps.Clone(v.Ballots)
	v.FirstBallots = maps.Clone(v.FirstBallots)
	v.TieBallots = maps.Clone(v.TieBallots)
	v.Candidates = slices.Clone(v.Candidates)
	return v
}
