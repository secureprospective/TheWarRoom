package commissioner

import (
	"fmt"
	"reflect"
	"slices"

	"github.com/secureprospective/TheWarRoom/internal/domain"
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

// RawBidArgs names a recorded player or bid and the contract an owner offers. It never carries a decision,
// a price the app works out or an owner identity.
type RawBidArgs struct {
	Player    string       `json:"player,omitempty"`
	Franchise string       `json:"franchise,omitempty"`
	Bid       string       `json:"bid,omitempty"`
	Salary    domain.Money `json:"salaryCents,omitempty"`
	Years     int          `json:"years,omitempty"`
}

// BidResult is how one bid fared when its player's bids were decided.
type BidResult struct {
	ID      string `json:"id"`
	Refusal string `json:"refusal,omitempty"`
}

// BidOutcome is worked out by the app under the append lock; a request cannot supply it.
type BidOutcome struct {
	Player     string              `json:"player,omitempty"`
	Season     int                 `json:"season"`
	Points     leaguerules.Decimal `json:"points,omitempty"`
	Late       bool                `json:"late,omitempty"`
	Winner     string              `json:"winner,omitempty"`
	WinningBid string              `json:"winningBid,omitempty"`
	Tied       []string            `json:"tied,omitempty"`
	Results    []BidResult         `json:"results,omitempty"`
}

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

func (a RawArgs) validateBid(command string) error {
	other := a
	other.Bid, other.BidOutcome = nil, nil
	if a.Bid == nil || !reflect.DeepEqual(other, RawArgs{}) {
		return fmt.Errorf("send only the details of this bid action")
	}
	return a.Bid.Validate(command)
}

// bidDetails says which details a bid command takes, and the sentence that asks for them.
type bidDetails struct {
	player, franchise, bid, contract bool
	ask                              string
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
		return bidDetails{player: true, ask: "choose a player whose bids are to be decided"}
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
