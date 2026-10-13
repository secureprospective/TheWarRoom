package governance

import (
	"fmt"
	"time"
)

type ActivityObserved struct {
	Principal Principal
	At        time.Time
}

func (ActivityObserved) Private() bool { return false }
func (e ActivityObserved) apply(s *State) error {
	if err := s.requireCreated(); err != nil {
		return err
	}
	if e.Principal == "" || e.At.IsZero() {
		return fmt.Errorf("activity needs a person and a recorded time")
	}
	if e.At.After(s.activity[e.Principal]) {
		s.activity[e.Principal] = e.At
	}
	return nil
}

type InactivitySet struct{ InSeason, Offseason time.Duration }

func (InactivitySet) Private() bool { return false }
func (e InactivitySet) apply(s *State) error {
	if err := s.requireCreated(); err != nil {
		return err
	}
	if e.InSeason <= 0 || e.Offseason <= 0 {
		return fmt.Errorf("inactivity periods must be positive")
	}
	s.inSeason = e.InSeason
	s.offseason = e.Offseason
	return nil
}
func InactivityEligible(s State, commissioner Principal, now time.Time, inSeason bool) (bool, error) {
	if !s.HasRole(commissioner, Commissioner) {
		return false, fmt.Errorf("choose a current commissioner")
	}
	at, ok := s.activity[commissioner]
	if !ok {
		return false, fmt.Errorf("no recorded commissioner activity; inactivity cannot be established")
	}
	period := s.offseason
	if inSeason {
		period = s.inSeason
	}
	return !now.Before(at.Add(period)), nil
}

type StepDown struct{ Principal, Successor Principal }

func (StepDown) Private() bool { return false }
func (e StepDown) apply(s *State) error {
	if !s.HasRole(e.Principal, Commissioner) {
		return fmt.Errorf("only a commissioner may step down")
	}
	if s.outgoing != "" {
		return fmt.Errorf("a succession is already in progress")
	}
	if e.Successor == e.Principal {
		return fmt.Errorf("choose a different successor")
	}
	if e.Successor != "" {
		s.transfer(e.Principal, e.Successor)
		return nil
	}
	s.startSuccession(e.Principal)
	return nil
}
func (s *State) transfer(from, to Principal) {
	s.grant(to, Commissioner)
	delete(s.roles[from], Commissioner)
	s.pending = ""
	s.outgoing = ""
}
func (s *State) startSuccession(from Principal) {
	s.outgoing = from
	delete(s.roles[from], Commissioner)
	for _, p := range s.members(CoCommissioner) {
		if p != from {
			s.pending = p
			s.transfer(from, p)
			s.pending = p
			s.outgoing = from
			return
		}
	}
}

type SuccessorDeclined struct{ Principal Principal }

func (SuccessorDeclined) Private() bool { return false }
func (e SuccessorDeclined) apply(s *State) error {
	if s.pending == "" || s.pending != e.Principal {
		return fmt.Errorf("you are not the pending successor")
	}
	delete(s.roles[e.Principal], Commissioner)
	s.pending = ""
	return nil
}

type SuccessorAccepted struct{ Principal Principal }

func (SuccessorAccepted) Private() bool { return false }
func (e SuccessorAccepted) apply(s *State) error {
	if s.pending == "" || s.pending != e.Principal {
		return fmt.Errorf("you are not the pending successor")
	}
	s.pending = ""
	s.outgoing = ""
	return nil
}

type InactivityTakeover struct {
	Principal, Commissioner Principal
	Now                     time.Time
	InSeason                bool
}

func (InactivityTakeover) Private() bool { return false }
func (e InactivityTakeover) apply(s *State) error {
	if !s.HasRole(e.Principal, CoCommissioner) || e.Principal == e.Commissioner {
		return fmt.Errorf("only a co-commissioner may take over after inactivity")
	}
	ok, err := InactivityEligible(*s, e.Commissioner, e.Now, e.InSeason)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("the commissioner has not reached the inactivity period")
	}
	s.transfer(e.Commissioner, e.Principal)
	return nil
}
func (s State) ElectionNeeded() bool { return s.outgoing != "" && s.pending == "" }
