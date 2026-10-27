package commissioner

import (
	"fmt"
	"slices"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

// standingAt restarts the auction clock on every new bid, even if that bid is later withdrawn.
func (a *bidAuction) standingAt() time.Time {
	var latest time.Time
	for _, b := range a.Bids {
		if b.PlacedAt.After(latest) {
			latest = b.PlacedAt
		}
	}
	return latest
}

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

type waiverWindow struct {
	Results    []WaiverClaimResult
	PriorOrder []string
	ID         string
	Player     NativePlayer
	CutAt      time.Time
	Season     int
	Dead       domain.Money
	Claims     []waiverClaim
	Processed  bool
	Winner     string
	ClosedAt   time.Time
}

type waiverClaim struct{ ID, Franchise, Principal string }

type waiverState struct {
	Windows     []waiverWindow
	BaseOrder   []string
	Order       []string
	OrderSeason int
	Source      string
	// Bids rides with the waiver state because both fold the same ledger into the same rosters.
	Bids   bidBook
	Trades []TradeState
}

func (s *waiverState) window(id string) (*waiverWindow, error) {
	for i := range s.Windows {
		if s.Windows[i].ID == id {
			return &s.Windows[i], nil
		}
	}
	return nil, fmt.Errorf("choose a recorded waiver window")
}

func (s *waiverState) claim(id string) (*waiverWindow, waiverClaim, error) {
	for i := range s.Windows {
		for _, c := range s.Windows[i].Claims {
			if c.ID == id {
				return &s.Windows[i], c, nil
			}
		}
	}
	return nil, waiverClaim{}, fmt.Errorf("choose a recorded waiver claim")
}
