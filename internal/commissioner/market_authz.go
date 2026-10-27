package commissioner

import (
	"fmt"
	"slices"

	"github.com/secureprospective/TheWarRoom/internal/governance"
)

func marketOwnerFranchise(raw RawCommand) string {
	if OwnerWaiverAction(raw.Command) != "" {
		return raw.Args.Waiver.Franchise
	}
	if OwnerBidAction(raw.Command) != "" {
		return raw.Args.Bid.Franchise
	}
	return ""
}

func authorizeTrade(raw RawCommand, principal string, records []Record) (string, error) {
	state, err := GovernanceState(records)
	if err != nil {
		return "", err
	}
	if OwnerTradeAction(raw.Command) != "" || raw.Command == TradeAppeal {
		id := raw.Args.Trade.Franchise
		f, ok := state.Franchise(governance.FranchiseID(id))
		if !ok || f.Seat != governance.Principal(principal) || principal == "" || !f.Abilities["trades"] {
			return "", fmt.Errorf("only the seated owner with trading enabled may act for this franchise")
		}
		if slices.ContainsFunc(raw.Args.Trade.Assets, func(a TradeAsset) bool { return a.Player == "" }) && !f.Abilities["pick trades"] {
			return "", fmt.Errorf("pick trading is turned off for this franchise")
		}
		return "owner", nil
	}
	if raw.Command == TradeOverride {
		if !state.HasRole(governance.Principal(principal), governance.Commissioner) {
			return "", fmt.Errorf("only the commissioner may override a trade")
		}
		return "commissioner", nil
	}
	// Eligibility and the review route are judged again under the append lock.
	if state.HasRole(governance.Principal(principal), governance.DOT) {
		return "DOT", nil
	}
	if state.HasRole(governance.Principal(principal), governance.Commissioner) {
		return "commissioner", nil
	}
	if state.HasRole(governance.Principal(principal), governance.CoCommissioner) {
		return "cocommissioner", nil
	}
	return "", fmt.Errorf("only an eligible trade reviewer may decide this trade")
}

func AuthorizeFranchiseOwner(records []Record, principal, franchise string) error {
	state, err := GovernanceState(records)
	if err != nil {
		return err
	}
	seat, ok := state.Franchise(governance.FranchiseID(franchise))
	if !ok || seat.Seat == "" || seat.Seat != governance.Principal(principal) {
		return fmt.Errorf("you may act only for your own franchise")
	}
	return nil
}
