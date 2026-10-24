package governance

import (
	"fmt"
	"slices"
	"sort"
	"time"
)

type ElectionPeriodsSet struct{ Election, Runoff time.Duration }

func (ElectionPeriodsSet) Private() bool { return false }
func (e ElectionPeriodsSet) apply(s *State) error {
	if err := s.requireCreated(); err != nil {
		return err
	}
	if e.Election <= 0 || e.Runoff <= 0 {
		return fmt.Errorf("voting periods must be positive")
	}
	s.electionPeriod, s.runoffPeriod = e.Election, e.Runoff
	return nil
}
func ElectionThreshold(franchises int) int { return franchises/2 + 1 }

func rankedCandidates(v Vote) ([]Principal, map[Principal]int) {
	counts := make(map[Principal]int)
	for _, p := range v.Candidates {
		counts[p] = 0
	}
	for _, p := range v.Ballots {
		counts[p]++
	}
	ps := make([]Principal, 0, len(counts))
	for p := range counts {
		ps = append(ps, p)
	}
	sort.Slice(ps, func(i, j int) bool {
		if counts[ps[i]] == counts[ps[j]] {
			return ps[i] < ps[j]
		}
		return counts[ps[i]] > counts[ps[j]]
	})
	return ps, counts
}
func topCandidates(ps []Principal, counts map[Principal]int, places int) []Principal {
	if len(ps) == 0 {
		return nil
	}
	cutoff := counts[ps[min(places, len(ps))-1]]
	end := 0
	for end < len(ps) && counts[ps[end]] >= cutoff {
		end++
	}
	return slices.Clone(ps[:end])
}
func (s State) electionResult(v Vote) Principal {
	ps, counts := rankedCandidates(v)
	if len(ps) == 0 {
		return ""
	}
	if !v.Runoff {
		if counts[ps[0]] >= ElectionThreshold(len(s.franchises)) {
			return ps[0]
		}
		return ""
	}
	leaders := topCandidates(ps, counts, 1)
	if len(leaders) == 1 {
		return leaders[0]
	}
	return s.tieWinner(v, leaders)
}
func (s *State) closeElection(v Vote, e VoteClosed) error {
	if e.Now.IsZero() || e.Now.Before(v.EndsAt) {
		return fmt.Errorf("the voting period has not ended")
	}
	winner := s.electionResult(v)
	if e.Winner != "" && e.Winner != winner {
		return fmt.Errorf("the named winner does not match the ballots")
	}
	if winner == "" {
		if v.Runoff {
			return fmt.Errorf("%s: the runoff is tied and no one is eligible to break it", "Needs a ruling")
		}
		ps, counts := rankedCandidates(v)
		v.Candidates = topCandidates(ps, counts, 2)
		if len(v.Candidates) == 0 {
			return fmt.Errorf("the election needs ballots before a runoff can open")
		}
		v.FirstBallots, v.Ballots = v.Ballots, make(map[FranchiseID]Principal)
		v.Runoff = true
		v.OpenedAt, v.EndsAt = e.Now, e.Now.Add(s.runoffPeriod)
		v.TieBallots = make(map[Principal]Principal)
	} else {
		s.transfer(v.Target, winner)
		v.Closed, v.Passed, v.Winner = true, true, winner
	}
	s.votes[e.ID] = v
	return nil
}
func (s State) ElectionStatus(id string, now time.Time) string {
	v, ok := s.votes[id]
	if !ok || !v.Election {
		return "No commissioner election"
	}
	if v.Closed {
		return fmt.Sprintf("Commissioner elected: %s", v.Winner)
	}
	if now.Before(v.EndsAt) {
		if v.Runoff {
			return "Runoff voting is open"
		}
		return "Commissioner voting is open"
	}
	if winner := s.electionResult(v); winner != "" {
		return fmt.Sprintf("Ready to close: %s wins", winner)
	}
	if v.Runoff {
		return "Needs a ruling: the runoff is tied and no one is eligible to break it"
	}
	return "Ready to close: a runoff is needed"
}

type ElectionTieBreakCast struct {
	ID                string
	Principal, Choice Principal
	Now               time.Time
}

func (ElectionTieBreakCast) Private() bool { return false }
func (e ElectionTieBreakCast) apply(s *State) error {
	v, ok := s.votes[e.ID]
	if !ok || !v.Runoff || v.Closed || e.Now.IsZero() || e.Now.Before(v.EndsAt) {
		return fmt.Errorf("a tie ruling needs an open runoff whose voting period has ended")
	}
	ps, counts := rankedCandidates(v)
	leaders := topCandidates(ps, counts, 1)
	if len(leaders) < 2 || !slices.Contains(leaders, e.Choice) {
		return fmt.Errorf("choose one of the candidates tied for the most votes")
	}
	cc, dot := s.tieJudges(v)
	if e.Principal == "" || (e.Principal != cc && !slices.Contains(dot, e.Principal)) {
		return fmt.Errorf("you are not eligible to decide this runoff tie")
	}
	v.TieBallots[e.Principal] = e.Choice
	s.votes[e.ID] = v
	return nil
}
func (s State) tieJudges(v Vote) (Principal, []Principal) {
	if p := s.firstDeputy(v.Candidates); p != "" {
		return p, nil
	}
	eligible := []Principal{}
	for _, p := range s.members(DOT) {
		if !slices.Contains(v.Candidates, p) {
			eligible = append(eligible, p)
		}
	}
	return "", eligible
}
func (s State) tieWinner(v Vote, leaders []Principal) Principal {
	cc, dot := s.tieJudges(v)
	if cc != "" {
		if choice := v.TieBallots[cc]; slices.Contains(leaders, choice) {
			return choice
		}
		return ""
	}
	counts := make(map[Principal]int)
	for _, p := range dot {
		counts[v.TieBallots[p]]++
	}
	for _, p := range leaders {
		if counts[p] >= len(dot)/2+1 {
			return p
		}
	}
	return ""
}
