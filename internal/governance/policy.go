package governance

import (
	"fmt"
	"slices"
)

type CommandRef struct {
	ID          string
	MinimumRole Role
	COnly       bool
	Ability     string
	Franchise   FranchiseID
}

func Allow(s State, p Principal, c CommandRef) (Role, error) {
	if err := validateCommand(p, c); err != nil {
		return "", err
	}
	if s.HasRole(p, Commissioner) {
		return Commissioner, nil
	}
	if s.HasRole(p, CoCommissioner) && !c.COnly {
		return CoCommissioner, nil
	}
	if s.HasRole(p, DOT) && c.MinimumRole == DOT && !c.COnly &&
		dotCommand(c.ID) {
		return DOT, nil
	}
	return allowOwner(s, p, c)
}
func validateCommand(p Principal, c CommandRef) error {
	if p == "" || c.ID == "" {
		return fmt.Errorf("identify yourself and choose an action")
	}
	if row, ok := CensusControl(c.ID); ok {
		if row.MinimumRole == "" {
			return fmt.Errorf("choose a specific succession action")
		}
		if c.MinimumRole != row.MinimumRole || c.COnly != row.COnly {
			return fmt.Errorf("this action's permissions do not match the league controls")
		}
	}
	if c.MinimumRole != Owner && !validRole(c.MinimumRole) {
		return fmt.Errorf("this action has no supported permission")
	}
	return nil
}
func allowOwner(s State, p Principal, c CommandRef) (Role, error) {
	if c.MinimumRole == Owner && !c.COnly && s.seat(p) != "" {
		id := s.seat(p)
		if c.Franchise != "" && id != c.Franchise {
			return "", fmt.Errorf("you may act only for your own franchise")
		}
		if c.ID == "CM-056" || c.ID == "vote.cast" ||
			(c.Ability != "" && s.franchises[id].Abilities[c.Ability]) {
			return Owner, nil
		}
	}
	return "", fmt.Errorf("your league role does not permit this action")
}
func dotCommand(id string) bool {
	return id == "CM-037" || id == "trade.review" || id == "trade.recuse" ||
		id == "succession.voteout.open" || id == "succession.election.open"
}
func TradeReviewReady(s State) error {
	n := len(s.members(DOT))
	if n < 4 {
		return fmt.Errorf("%s: DOT has %d of 4 seats", "Needs a ruling", n)
	}
	return nil
}
func Reviewers(s State, parties []Principal) ([]Principal, error) {
	if err := TradeReviewReady(s); err != nil {
		return nil, err
	}
	eligible := []Principal{}
	for _, p := range s.members(DOT) {
		if !slices.Contains(parties, p) {
			eligible = append(eligible, p)
		}
	}
	if len(eligible) == 0 {
		return nil, fmt.Errorf("this trade has no eligible DOT reviewer; it must wait")
	}
	return eligible, nil
}
func AppealJudge(s State, parties []Principal) (Principal, error) {
	for _, p := range s.members(Commissioner) {
		if !slices.Contains(parties, p) {
			return p, nil
		}
	}
	for _, p := range s.members(CoCommissioner) {
		if !slices.Contains(parties, p) {
			return p, nil
		}
	}
	return "", fmt.Errorf("this appeal has no eligible commissioner or co-commissioner; it must wait")
}
