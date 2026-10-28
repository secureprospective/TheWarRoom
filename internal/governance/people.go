package governance

import (
	"fmt"
	"slices"
)

type LeagueCreated struct {
	Creator    Principal
	Franchises []FranchiseID
}

func (LeagueCreated) Private() bool { return false }
func (e LeagueCreated) apply(s *State) error {
	if s.created || e.Creator == "" || len(e.Franchises) == 0 {
		return fmt.Errorf("a new league needs a creator and franchises")
	}
	for _, id := range e.Franchises {
		if id == "" {
			return fmt.Errorf("a franchise needs an identity")
		}
		if _, ok := s.franchises[id]; ok {
			return fmt.Errorf("a franchise was listed twice")
		}
		s.franchises[id] = Franchise{ID: id, Abilities: defaultAbilities()}
	}
	s.created = true
	s.grant(e.Creator, Commissioner)
	return nil
}

// FranchisesResized sets the league's franchises to the listed ones, in order. A new franchise starts
// vacant under its listed name; a franchise left off must be vacant with no open invitation.
type FranchisesResized struct {
	Franchises []FranchiseIdentitySet
}

func (FranchisesResized) Private() bool { return false }
func (e FranchisesResized) apply(s *State) error {
	if err := s.requireCreated(); err != nil {
		return err
	}
	if len(e.Franchises) == 0 {
		return fmt.Errorf("a league needs franchises")
	}
	listed := make(map[FranchiseID]bool, len(e.Franchises))
	for _, f := range e.Franchises {
		listed[f.Franchise] = true
	}
	for id := range s.franchises {
		if listed[id] {
			continue
		}
		if err := s.removable(id); err != nil {
			return err
		}
	}
	for id := range s.franchises {
		if !listed[id] {
			delete(s.franchises, id)
		}
	}
	return s.addListed(e.Franchises)
}

// removable refuses to remove a franchise someone holds or has been invited to.
func (s *State) removable(id FranchiseID) error {
	f := s.franchises[id]
	if f.Seat != "" || f.Caretaker != "" {
		return fmt.Errorf("%s still has an owner; free the seat before removing the team", f.Name)
	}
	for _, invite := range s.invites {
		if invite.Franchise == id && !invite.Revoked && !invite.Redeemed {
			return fmt.Errorf("%s has an open invitation; revoke it before removing the team", f.Name)
		}
	}
	return nil
}

// addListed seats each listed franchise the league does not hold yet, vacant under its listed name.
func (s *State) addListed(franchises []FranchiseIdentitySet) error {
	for _, f := range franchises {
		if _, ok := s.franchises[f.Franchise]; ok {
			continue
		}
		if f.Franchise == "" || f.Name == "" || f.Abbreviation == "" {
			return fmt.Errorf("a franchise needs an identity")
		}
		s.franchises[f.Franchise] = Franchise{ID: f.Franchise, Name: f.Name, Abbreviation: f.Abbreviation,
			Abilities: defaultAbilities()}
	}
	return nil
}

type FranchiseIdentitySet struct {
	Franchise          FranchiseID
	Name, Abbreviation string
}

func (FranchiseIdentitySet) Private() bool { return false }
func (e FranchiseIdentitySet) apply(s *State) error {
	f, ok := s.franchises[e.Franchise]
	if !ok || e.Name == "" || e.Abbreviation == "" {
		return fmt.Errorf("choose a franchise, name and abbreviation")
	}
	f.Name = e.Name
	f.Abbreviation = e.Abbreviation
	s.franchises[e.Franchise] = f
	return nil
}

type RoleGranted struct {
	Principal Principal
	Role      Role
}

func (RoleGranted) Private() bool { return true }
func (e RoleGranted) apply(s *State) error {
	if err := s.requireCreated(); err != nil {
		return err
	}
	if e.Principal == "" || !validRole(e.Role) {
		return fmt.Errorf("choose a person and a league role")
	}
	if e.Role == CoCommissioner && s.HasRole(e.Principal, CoCommissioner) {
		return fmt.Errorf("that person is already a co-commissioner")
	}
	s.grant(e.Principal, e.Role)
	if e.Role == CoCommissioner {
		s.coCommissioners = append(s.coCommissioners, e.Principal)
	}
	return nil
}

type RoleRevoked struct {
	Principal Principal
	Role      Role
}

func (RoleRevoked) Private() bool { return true }
func (e RoleRevoked) apply(s *State) error {
	if !validRole(e.Role) || !s.HasRole(e.Principal, e.Role) {
		return fmt.Errorf("that person does not hold that role")
	}
	if e.Role == Commissioner && len(s.members(Commissioner)) == 1 {
		return fmt.Errorf("the last commissioner cannot be removed; choose a successor first")
	}
	delete(s.roles[e.Principal], e.Role)
	if e.Role == CoCommissioner {
		if s.pending == e.Principal {
			delete(s.roles[e.Principal], Commissioner)
			s.advanceSuccession(e.Principal)
		}
		s.coCommissioners = slices.DeleteFunc(s.coCommissioners, func(p Principal) bool { return p == e.Principal })
	}
	return nil
}

// DeputyOrderSet changes priority without changing anyone's roles.
type DeputyOrderSet struct{ Order []Principal }

func (DeputyOrderSet) Private() bool { return true }
func (e DeputyOrderSet) apply(s *State) error {
	if err := s.requireCreated(); err != nil {
		return err
	}
	if len(e.Order) != len(s.coCommissioners) {
		return fmt.Errorf("list every current co-commissioner exactly once, with no added names")
	}
	seen := make(map[Principal]bool, len(e.Order))
	for _, p := range e.Order {
		if !s.HasRole(p, CoCommissioner) || seen[p] {
			return fmt.Errorf("list every current co-commissioner exactly once, with no added names")
		}
		seen[p] = true
	}
	s.coCommissioners = slices.Clone(e.Order)
	return nil
}

type AbilitySet struct {
	Franchise FranchiseID
	All       bool
	Ability   string
	On        bool
}

func (AbilitySet) Private() bool { return false }
func (e AbilitySet) apply(s *State) error {
	if e.Ability == "" {
		return fmt.Errorf("choose an owner ability")
	}
	if e.All {
		if err := s.requireCreated(); err != nil {
			return err
		}
		for id, f := range s.franchises {
			f.Abilities[e.Ability] = e.On
			s.franchises[id] = f
		}
		return nil
	}
	f, ok := s.franchises[e.Franchise]
	if !ok {
		return fmt.Errorf("choose an existing franchise")
	}
	f.Abilities[e.Ability] = e.On
	s.franchises[e.Franchise] = f
	return nil
}

type VisibilitySet struct{ Report, Audience string }

func (VisibilitySet) Private() bool { return false }
func (e VisibilitySet) apply(s *State) error {
	if err := s.requireCreated(); err != nil {
		return err
	}
	if e.Report == "" || (e.Audience != "all" && e.Audience != "commish") {
		return fmt.Errorf("choose a report and who may read it")
	}
	s.visibility[e.Report] = e.Audience
	return nil
}

func defaultAbilities() map[string]bool {
	return map[string]bool{
		"lineups": true, "add/drop": true, "drop-only": true, "trades": true,
		"pick trades": true, "IR": true, "taxi": true, "articles": true,
		"board": true, "chat": true, "polls": true,
		"customize franchise": false, "home page": false,
	}
}
