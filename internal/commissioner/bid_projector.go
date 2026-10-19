package commissioner

import (
	"fmt"
	"slices"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

type bidRecord struct {
	ID, Franchise, Principal string
	Salary                   domain.Money
	Years                    int
	Points                   leaguerules.Decimal
	PlacedAt                 time.Time
	Replaced, Withdrawn      bool
}

func (b bidRecord) live() bool { return !b.Replaced && !b.Withdrawn }

// bidAuction is every bid on one player until the app decides them. A decided auction with no winner stays
// recorded; a later bid on the same player starts a new one.
type bidAuction struct {
	Player     string
	Bids       []bidRecord
	Decided    bool
	Winner     string
	WinningBid string
	Tied       []string
	Results    []BidResult
	Season     int
	DecidedAt  time.Time
}

type bidBook struct{ Auctions []bidAuction }

// open is the player's auction still waiting for a decision.
func (k *bidBook) open(player string) *bidAuction {
	for i := len(k.Auctions) - 1; i >= 0; i-- {
		if k.Auctions[i].Player == player {
			if k.Auctions[i].Decided {
				return nil
			}
			return &k.Auctions[i]
		}
	}
	return nil
}

// latest is the player's newest auction, decided or not.
func (k *bidBook) latest(player string) *bidAuction {
	for i := len(k.Auctions) - 1; i >= 0; i-- {
		if k.Auctions[i].Player == player {
			return &k.Auctions[i]
		}
	}
	return nil
}

func (k *bidBook) find(id string) (*bidAuction, int, error) {
	for i := range k.Auctions {
		for j, b := range k.Auctions[i].Bids {
			if b.ID == id {
				return &k.Auctions[i], j, nil
			}
		}
	}
	return nil, 0, fmt.Errorf("choose a recorded bid")
}

// activeFor is the franchise's live bid on this auction, if any.
func (a *bidAuction) activeFor(franchise string) (bidRecord, bool) {
	for _, b := range a.Bids {
		if b.Franchise == franchise && b.live() {
			return b, true
		}
	}
	return bidRecord{}, false
}

func (a *bidAuction) liveBids() []bidRecord {
	return slices.DeleteFunc(slices.Clone(a.Bids), func(b bidRecord) bool { return !b.live() })
}

// leader is the live bid with the most points; among equals the one placed first.
func leaderOf(bids []bidRecord) (bidRecord, bool) {
	var lead bidRecord
	found := false
	for _, b := range bids {
		if !found || b.Points > lead.Points || (b.Points == lead.Points && b.PlacedAt.Before(lead.PlacedAt)) {
			lead, found = b, true
		}
	}
	return lead, found
}

// activeCount is how many open auctions hold a live bid from the franchise, apart from one player's.
func (k *bidBook) activeCount(franchise, except string) int {
	n := 0
	for i := range k.Auctions {
		a := &k.Auctions[i]
		if a.Decided || a.Player == except {
			continue
		}
		if _, ok := a.activeFor(franchise); ok {
			n++
		}
	}
	return n
}

func projectBid(seed *RawNativeSnapshot, state *waiverState, r Record) error {
	args, outcome := r.Command.Args.Bid, r.Command.Args.BidOutcome
	if args == nil || outcome == nil {
		return fmt.Errorf("a bid change has no recorded details")
	}
	book := &state.Bids
	switch r.Command.Command {
	case BidPlace:
		return projectPlacedBid(seed, book, r)
	case BidRaise:
		return projectRaisedBid(book, r)
	case BidWithdraw:
		a, j, err := book.find(args.Bid)
		if err != nil {
			return err
		}
		if a.Decided || !a.Bids[j].live() {
			return fmt.Errorf("reverse the later bid changes before reversing this one")
		}
		a.Bids[j].Withdrawn = true
	case BidsProcess, BidsOverride:
		return projectBidAward(seed, book, r)
	}
	return nil
}

func projectPlacedBid(seed *RawNativeSnapshot, book *bidBook, r Record) error {
	args, outcome := r.Command.Args.Bid, r.Command.Args.BidOutcome
	if slices.ContainsFunc(seed.Players, func(p NativePlayer) bool { return p.ID == args.Player }) {
		return fmt.Errorf("this player already belongs to a roster")
	}
	bid := bidRecord{ID: r.Receipt.Entry, Franchise: args.Franchise, Principal: r.Principal, Salary: args.Salary,
		Years: args.Years, Points: outcome.Points, PlacedAt: r.Receipt.RecordedAt}
	if a := book.open(args.Player); a != nil {
		if _, held := a.activeFor(args.Franchise); held {
			return fmt.Errorf("this franchise already has a bid on this player")
		}
		a.Bids = append(a.Bids, bid)
		return nil
	}
	book.Auctions = append(book.Auctions, bidAuction{Player: args.Player, Bids: []bidRecord{bid}})
	return nil
}

func projectRaisedBid(book *bidBook, r Record) error {
	args, outcome := r.Command.Args.Bid, r.Command.Args.BidOutcome
	a, j, err := book.find(args.Bid)
	if err != nil {
		return err
	}
	if a.Decided || !a.Bids[j].live() || a.Bids[j].Franchise != args.Franchise {
		return fmt.Errorf("reverse the later bid changes before reversing this one")
	}
	a.Bids[j].Replaced = true
	a.Bids = append(a.Bids, bidRecord{ID: r.Receipt.Entry, Franchise: args.Franchise, Principal: r.Principal,
		Salary: args.Salary, Years: args.Years, Points: outcome.Points,
		PlacedAt: r.Receipt.RecordedAt})
	return nil
}

// projectBidAward decides an auction, or corrects a decision: the previous winner leaves the roster and the
// new one signs the contract his live bid named.
func projectBidAward(seed *RawNativeSnapshot, book *bidBook, r Record) error {
	args, outcome := r.Command.Args.Bid, r.Command.Args.BidOutcome
	var a *bidAuction
	if r.Command.Command == BidsProcess {
		if a = book.open(args.Player); a == nil {
			return fmt.Errorf("this player has no bids waiting for a decision")
		}
	} else {
		found, _, err := book.find(args.Bid)
		if err != nil {
			return err
		}
		if a = found; !a.Decided || book.latest(a.Player) != a {
			return fmt.Errorf("correct only the newest decision on a player")
		}
	}
	if err := replaceBidWinner(seed, a, outcome); err != nil {
		return err
	}
	a.Decided, a.Winner, a.WinningBid, a.Tied = true, outcome.Winner, outcome.WinningBid, slices.Clone(outcome.Tied)
	a.Results, a.Season, a.DecidedAt = slices.Clone(outcome.Results), outcome.Season, r.Receipt.RecordedAt
	return nil
}

func replaceBidWinner(seed *RawNativeSnapshot, a *bidAuction, outcome *BidOutcome) error {
	if a.Winner != "" {
		index := slices.IndexFunc(seed.Players, func(p NativePlayer) bool {
			return p.ID == a.Player && p.Franchise == a.Winner
		})
		if index < 0 {
			return fmt.Errorf("a later roster change depends on this award; reverse it first")
		}
		seed.Players = slices.Delete(seed.Players, index, index+1)
	}
	if outcome.Winner == "" {
		return nil
	}
	if slices.ContainsFunc(seed.Players, func(p NativePlayer) bool { return p.ID == a.Player }) {
		return fmt.Errorf("this player already belongs to a roster")
	}
	won := slices.IndexFunc(a.Bids, func(b bidRecord) bool { return b.ID == outcome.WinningBid })
	if won < 0 || a.Bids[won].Franchise != outcome.Winner {
		return fmt.Errorf("the winning bid is not a recorded bid of the winner")
	}
	bid := a.Bids[won]
	seed.Players = append(seed.Players, NativePlayer{ID: a.Player, Franchise: bid.Franchise, Status: domain.RosterActive,
		Salary: bid.Salary, ExpirationYear: outcome.Season + bid.Years - 1, ContractStatus: domain.CStatusUFA,
		ContractInfo: "Won in free-agent bidding: " + nativecheck.ContractWords(bid.Salary, bid.Years)})
	return nil
}

// bidPlayerOf follows a bid change back to the player it concerns.
func bidPlayerOf(r Record, records []Record) string {
	for range len(records) {
		args := r.Command.Args.Bid
		if args == nil {
			return ""
		}
		if args.Player != "" {
			return args.Player
		}
		index := slices.IndexFunc(records, func(v Record) bool { return v.Receipt.Entry == args.Bid })
		if index < 0 {
			return ""
		}
		r = records[index]
	}
	return ""
}

// privateBidRecord hides an owner's bid, and the reversal of one, from everyone else.
func privateBidRecord(r Record, records []Record) bool {
	for range len(records) {
		if r.Command.Command != Revert {
			break
		}
		index := slices.IndexFunc(records, func(v Record) bool { return v.Receipt.Entry == r.Command.Args.Entry })
		if index < 0 {
			return false
		}
		r = records[index]
	}
	return OwnerBidAction(r.Command.Command) != ""
}

// decidedBidRecord says whether the player's bids have been decided, which is when administrators may see
// them. A decision that is itself reversed does not count.
func decidedBidRecord(r Record, records []Record) bool {
	player := bidPlayerOf(r, records)
	if player == "" {
		return false
	}
	active := Replay(records).Active
	return slices.ContainsFunc(records, func(v Record) bool {
		return active[v.Receipt.Entry] && v.Command.Command == BidsProcess && v.Command.Args.Bid.Player == player
	})
}
