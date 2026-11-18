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

func authorizeTrade(raw RawCommand, principal string, records []Record, deputyPermitted bool) (string, error) {
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
		return authorizeTradeOverride(raw, principal, records, state, deputyPermitted)
	}
	return authorizeTradeReviewer(raw, principal, records, state, deputyPermitted)
}

func authorizeTradeOverride(raw RawCommand, principal string, records []Record, state governance.State, deputyPermitted bool) (string, error) {
	person := governance.Principal(principal)
	switch {
	case state.HasRole(person, governance.Commissioner):
		return "commissioner", nil
	case deputyPermitted && state.HasRole(person, governance.Deputy):
		if err := authorizeDeputyTradeRuling(raw, principal, records, state); err != nil {
			return "", err
		}
		return "deputy", nil
	default:
		return "", fmt.Errorf("only the commissioner may override a trade")
	}
}

func authorizeTradeReviewer(raw RawCommand, principal string, records []Record, state governance.State, deputyPermitted bool) (string, error) {
	person := governance.Principal(principal)
	// Eligibility and the review route are judged again under the append lock.
	if deputyPermitted && state.HasRole(person, governance.Deputy) {
		if err := authorizeDeputyTradeRuling(raw, principal, records, state); err != nil {
			return "", err
		}
		return "deputy", nil
	}
	if state.HasRole(person, governance.DOT) {
		return "DOT", nil
	}
	if state.HasRole(person, governance.Commissioner) {
		return "commissioner", nil
	}
	if state.HasRole(person, governance.CoCommissioner) {
		return "cocommissioner", nil
	}
	return "", fmt.Errorf("only an eligible trade reviewer may decide this trade")
}

func authorizeDeputyTradeRuling(raw RawCommand, principal string, records []Record, state governance.State) error {
	_, book, err := nativeAssetsState(raw.League, records)
	if err != nil {
		return fmt.Errorf("check deputy trade eligibility: %w", err)
	}
	trade, err := TradeByID(book.Trades, raw.Args.Trade.ID)
	if err != nil {
		return fmt.Errorf("check deputy trade eligibility: %w", err)
	}
	for _, party := range trade.Parties {
		franchise, ok := state.Franchise(governance.FranchiseID(party))
		if ok && franchise.Seat == governance.Principal(principal) {
			return fmt.Errorf("a deputy may not rule on a trade involving their own franchise")
		}
	}
	return nil
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
