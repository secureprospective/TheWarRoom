package market

import (
	"fmt"
	"slices"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

// openBidsFit holds a franchise's standing bids to its cap together (R3, 10-30): every open bid counts
// against cap room until it is outbid or wins, so a new bid is refused when the bids would pass the cap
// all at once. Roster size is not affected: only one bid's player is ever counted on the roster.
func (x *bidExecution) openBidsFit(candidate *commissioner.RawNativeSnapshot, rules leaguerules.RuleSet,
	player, franchise string, exceptions []nativecheck.DecisionException) error {
	all := *candidate
	all.Players = slices.Clone(candidate.Players)
	others, side := 0, candidate.SideOf(franchise)
	for i := range x.state.Bids.Auctions {
		a := &x.state.Bids.Auctions[i]
		if a.Decided || a.Side == side && a.Player == player {
			continue
		}
		bid, ok := a.ActiveFor(franchise)
		if !ok {
			continue
		}
		others++
		all.Players = append(all.Players, commissioner.NativePlayer{ID: a.Player, Franchise: franchise,
			Status: domain.RosterActive, Salary: bid.Salary, ExpirationYear: rules.Season + bid.Years - 1,
			ContractStatus: domain.CStatusUFA})
	}
	if others == 0 {
		return nil
	}
	players, charges := franchiseCapInputs(&all, franchise, rules.Season)
	capState, err := nativecheck.CheckCap(rules.Cap, players, charges, x.zone, exceptions...)
	if err != nil {
		return fmt.Errorf("check open bids against the cap: %w", err)
	}
	if allows, _ := nativecheck.CapAllows(capState, 0, x.zone); !allows {
		return fmt.Errorf("this bid and your %d other open bids would pass the salary cap together; "+
			"an open bid counts against cap room until it is outbid or wins", others)
	}
	return nil
}
