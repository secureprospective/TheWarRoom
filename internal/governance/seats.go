package governance

import "fmt"

type InviteIssued struct {
	ID        string
	Franchise FranchiseID
	Redeemer  Principal
}

func (InviteIssued) Private() bool { return true }
func (e InviteIssued) apply(s *State) error {
	f, ok := s.franchises[e.Franchise]
	if !ok || f.Seat != "" || e.ID == "" || e.Redeemer == "" {
		return fmt.Errorf("invite a person to an existing vacant franchise")
	}
	if _, exists := s.invites[e.ID]; exists {
		return fmt.Errorf("that invite identity has already been used")
	}
	s.invites[e.ID] = Invite{ID: e.ID, Franchise: e.Franchise, Redeemer: e.Redeemer}
	return nil
}

type InviteRevoked struct{ ID string }

func (InviteRevoked) Private() bool { return true }
func (e InviteRevoked) apply(s *State) error {
	i, ok := s.invites[e.ID]
	if !ok || i.Redeemed || i.Revoked {
		return fmt.Errorf("that invite is no longer available")
	}
	i.Revoked = true
	s.invites[e.ID] = i
	return nil
}

type InviteRedeemed struct {
	ID        string
	Principal Principal
}

func (InviteRedeemed) Private() bool { return true }
func (e InviteRedeemed) apply(s *State) error {
	i, ok := s.invites[e.ID]
	if !ok || i.Revoked || i.Redeemed || i.Redeemer != e.Principal {
		return fmt.Errorf("that invite is not available to you")
	}
	if err := s.assign(i.Franchise, e.Principal, false); err != nil {
		return err
	}
	i.Redeemed = true
	s.invites[e.ID] = i
	return nil
}

type SeatRemoved struct{ Franchise FranchiseID }

func (SeatRemoved) Private() bool { return false }
func (e SeatRemoved) apply(s *State) error {
	f, ok := s.franchises[e.Franchise]
	if !ok || f.Seat == "" {
		return fmt.Errorf("that franchise has no owner to remove")
	}
	if s.HasRole(f.Seat, Commissioner) && len(s.members(Commissioner)) == 1 {
		return fmt.Errorf("the last commissioner cannot be removed; choose a successor first")
	}
	f.Seat = ""
	s.franchises[e.Franchise] = f
	return nil
}

type SeatReplaced struct {
	Franchise FranchiseID
	Principal Principal
}

func (SeatReplaced) Private() bool          { return false }
func (e SeatReplaced) apply(s *State) error { return s.assign(e.Franchise, e.Principal, true) }
func (s *State) assign(id FranchiseID, p Principal, replace bool) error {
	f, ok := s.franchises[id]
	if !ok || p == "" {
		return fmt.Errorf("choose an existing franchise and a person")
	}
	if occupied := s.seat(p); occupied != "" && occupied != id {
		return fmt.Errorf("a person may hold only one franchise seat")
	}
	if f.Seat != "" && !replace {
		return fmt.Errorf("that franchise already has an owner")
	}
	if f.Seat != p && s.HasRole(f.Seat, Commissioner) && len(s.members(Commissioner)) == 1 {
		return fmt.Errorf("the last commissioner cannot be replaced; choose a successor first")
	}
	f.Seat = p
	s.franchises[id] = f
	return nil
}
