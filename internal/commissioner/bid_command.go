package commissioner

import (
	"fmt"
	"reflect"
	"slices"

	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
)

const (
	BidPlace     = "bid.place"
	BidRaise     = "bid.raise"
	BidWithdraw  = "bid.withdraw"
	BidsProcess  = "bids.process"
	BidsOverride = "bids.award.override"
)

func BidCommand(command string) bool {
	return slices.Contains([]string{BidPlace, BidRaise, BidWithdraw, BidsProcess, BidsOverride}, command)
}

// OwnerBidAction is the calendar action an owner's bid asks the gate for; it is empty for the commissioner's.
func OwnerBidAction(command string) leaguerules.Action {
	switch command {
	case BidPlace, BidRaise, BidWithdraw:
		return "ufa_bid"
	}
	return ""
}

func (a RawArgs) onlyMarketDetails(clearFields func(*RawArgs), missing bool, sentence string) error {
	clearFields(&a)
	if missing || !reflect.DeepEqual(a, RawArgs{}) {
		return fmt.Errorf("%s", sentence)
	}
	return nil
}

func (a RawArgs) validateBid(command string) error {
	if err := a.onlyMarketDetails(func(v *RawArgs) { v.Bid, v.BidOutcome = nil, nil }, a.Bid == nil, "send only the details of this bid action"); err != nil {
		return err
	}
	return a.Bid.Validate(command)
}

// bidDetails says which details a bid command takes, and the sentence that asks for them.
type bidDetails struct {
	player, franchise, bid, contract, side bool
	ask                                    string
}

func bidDetailsOf(command string) bidDetails {
	switch command {
	case BidPlace:
		return bidDetails{player: true, franchise: true, contract: true,
			ask: "choose a free agent, your franchise, a yearly salary and a number of years"}
	case BidRaise:
		return bidDetails{bid: true, franchise: true, contract: true,
			ask: "choose your bid, a new yearly salary and a number of years"}
	case BidWithdraw:
		return bidDetails{bid: true, franchise: true, ask: "choose your bid to withdraw"}
	case BidsProcess:
		return bidDetails{player: true, side: true, ask: "choose a player whose bids are to be decided"}
	}
	return bidDetails{bid: true, franchise: true, ask: "choose a recorded bid and the franchise that wins"}
}

// keep is r with only the details this command takes, so anything else sent can be seen as foreign.
func (d bidDetails) keep(r RawBidArgs) RawBidArgs {
	var kept RawBidArgs
	if d.player {
		kept.Player = r.Player
	}
	if d.franchise {
		kept.Franchise = r.Franchise
	}
	if d.bid {
		kept.Bid = r.Bid
	}
	if d.contract {
		kept.Salary, kept.Years = r.Salary, r.Years
	}
	if d.side {
		kept.Side = r.Side
	}
	return kept
}

// incomplete says whether a detail the command takes is missing or unusable.
func (d bidDetails) incomplete(r RawBidArgs) bool {
	if _, err := canonicalPlayer(r.Player); d.player && err != nil {
		return true
	}
	return (d.franchise && blank(r.Franchise)) || (d.bid && blank(r.Bid)) ||
		(d.contract && (r.Salary <= 0 || r.Years < 1 || r.Years > 99))
}

func (r RawBidArgs) Validate(command string) error {
	d := bidDetailsOf(command)
	if !reflect.DeepEqual(r, d.keep(r)) {
		return fmt.Errorf("send only the details of this bid action")
	}
	if d.incomplete(r) {
		return fmt.Errorf("%s", d.ask)
	}
	return nil
}

func canonicalPlayer(raw string) (string, error) {
	id, err := playerid.New(raw)
	if err != nil || id.String() != raw {
		return "", fmt.Errorf("choose a player")
	}
	return id.String(), nil
}

func (r RawCommand) validateBidPlacement() error {
	visibility := "league"
	if OwnerBidAction(r.Command) != "" {
		visibility = "private"
	}
	if r.Effective != "now" || r.Visibility != visibility {
		return fmt.Errorf("bid actions take effect now; an owner's bids stay private and decisions are public")
	}
	return nil
}

func bidLabel(r Record) string {
	switch r.Command.Command {
	case BidPlace:
		return "Placed a private free-agent bid"
	case BidRaise:
		return "Raised a private free-agent bid"
	case BidWithdraw:
		return "Withdrew a private free-agent bid"
	case BidsProcess:
		return "Decided a player's free-agent bids"
	case BidsOverride:
		return "Corrected a free-agent bid award"
	}
	return "Recorded a bid change"
}
