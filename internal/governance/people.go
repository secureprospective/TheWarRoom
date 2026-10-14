package governance

import "fmt"

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
	if e.Role == CoCommissioner && s.coCommissioner != "" {
		return fmt.Errorf("%s: revoke them first", "This league already has a co-commissioner")
	}
	if e.Role == DOT && !s.HasRole(e.Principal, DOT) && len(s.members(DOT)) >= 10 {
		return fmt.Errorf("dOT already has its maximum of 10 seats")
	}
	s.grant(e.Principal, e.Role)
	if e.Role == CoCommissioner {
		s.coCommissioner = e.Principal
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
		s.coCommissioner = ""
	}
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
