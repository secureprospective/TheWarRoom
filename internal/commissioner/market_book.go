package commissioner

import (
	"fmt"
	"slices"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

// standingAt restarts the auction clock on every new bid, even if that bid is later withdrawn.
func (a *BidAuction) StandingAt() time.Time {
	var latest time.Time
	for _, b := range a.Bids {
		if b.PlacedAt.After(latest) {
			latest = b.PlacedAt
		}
	}
	return latest
}

type BidRecord struct {
	ID, Franchise, Principal string
	Salary                   domain.Money
	Years                    int
	Points                   leaguerules.Decimal
	PlacedAt                 time.Time
	Replaced, Withdrawn      bool
}

func (b BidRecord) Live() bool { return !b.Replaced && !b.Withdrawn }

// bidAuction is every bid on one player until the app decides them. A decided auction with no winner stays
// recorded; a later bid on the same player starts a new one.
type BidAuction struct {
	Player     string
	Bids       []BidRecord
	Decided    bool
	Winner     string
	WinningBid string
	Tied       []string
	Results    []BidResult
	Season     int
	DecidedAt  time.Time
}

type BidBook struct{ Auctions []BidAuction }

// open is the player's auction still waiting for a decision.
func (k *BidBook) Open(player string) *BidAuction {
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
func (k *BidBook) Latest(player string) *BidAuction {
	for i := len(k.Auctions) - 1; i >= 0; i-- {
		if k.Auctions[i].Player == player {
			return &k.Auctions[i]
		}
	}
	return nil
}

func nestedIndex[P, C any](parents []P, children func(*P) []C, match func(C) bool) (int, int) {
	for i := range parents {
		if j := slices.IndexFunc(children(&parents[i]), match); j >= 0 {
			return i, j
		}
	}
	return -1, -1
}

func (k *BidBook) Find(id string) (*BidAuction, int, error) {
	i, j := nestedIndex(k.Auctions, func(a *BidAuction) []BidRecord { return a.Bids }, func(b BidRecord) bool { return b.ID == id })
	if i >= 0 {
		return &k.Auctions[i], j, nil
	}
	return nil, 0, fmt.Errorf("choose a recorded bid")
}

// activeFor is the franchise's live bid on this auction, if any.
func (a *BidAuction) ActiveFor(franchise string) (BidRecord, bool) {
	for _, b := range a.Bids {
		if b.Franchise == franchise && b.Live() {
			return b, true
		}
	}
	return BidRecord{}, false
}

func (a *BidAuction) LiveBids() []BidRecord {
	return slices.DeleteFunc(slices.Clone(a.Bids), func(b BidRecord) bool { return !b.Live() })
}

// leader is the live bid with the most points; among equals the one placed first.
func LeaderOf(bids []BidRecord) (BidRecord, bool) {
	var lead BidRecord
	found := false
	for _, b := range bids {
		if !found || b.Points > lead.Points || (b.Points == lead.Points && b.PlacedAt.Before(lead.PlacedAt)) {
			lead, found = b, true
		}
	}
	return lead, found
}

// activeCount is how many open auctions hold a live bid from the franchise, apart from one player's.
func (k *BidBook) ActiveCount(franchise, except string) int {
	n := 0
	for i := range k.Auctions {
		a := &k.Auctions[i]
		if a.Decided || a.Player == except {
			continue
		}
		if _, ok := a.ActiveFor(franchise); ok {
			n++
		}
	}
	return n
}

type WaiverWindow struct {
	Results    []WaiverClaimResult
	PriorOrder []string
	ID         string
	Player     NativePlayer
	CutAt      time.Time
	Season     int
	Dead       domain.Money
	Claims     []WaiverBookClaim
	Processed  bool
	Winner     string
	ClosedAt   time.Time
}

type WaiverBookClaim struct{ ID, Franchise, Principal string }

type MarketBook struct {
	Windows     []WaiverWindow
	BaseOrder   []string
	Order       []string
	OrderSeason int
	Source      string
	// Bids rides with the waiver state because both fold the same ledger into the same rosters.
	Bids   BidBook
	Trades []TradeState
}

func (s *MarketBook) Window(id string) (*WaiverWindow, error) {
	for i := range s.Windows {
		if s.Windows[i].ID == id {
			return &s.Windows[i], nil
		}
	}
	return nil, fmt.Errorf("choose a recorded waiver window")
}

func (s *MarketBook) Claim(id string) (*WaiverWindow, WaiverBookClaim, error) {
	i, j := nestedIndex(s.Windows, func(w *WaiverWindow) []WaiverBookClaim { return w.Claims }, func(c WaiverBookClaim) bool { return c.ID == id })
	if i >= 0 {
		return &s.Windows[i], s.Windows[i].Claims[j], nil
	}
	return nil, WaiverBookClaim{}, fmt.Errorf("choose a recorded waiver claim")
}

// lastBidOf is the franchise's final bid on the auction, standing or not.
func (a *BidAuction) LastBidOf(franchise string) (BidRecord, bool) {
	for i := len(a.Bids) - 1; i >= 0; i-- {
		if a.Bids[i].Franchise == franchise && !a.Bids[i].Withdrawn {
			return a.Bids[i], true
		}
	}
	return BidRecord{}, false
}
func NearerFutureDeadline(next, deadline, at time.Time) time.Time {
	if deadline.After(at) && (next.IsZero() || deadline.Before(next)) {
		return deadline
	}
	return next
}
func CompleteWaiverOrder(order []string, seed *RawNativeSnapshot) error {
	if len(order) != len(seed.Franchises) {
		return fmt.Errorf("waiver order must list every franchise exactly once")
	}
	seen := map[string]bool{}
	for _, id := range order {
		found := false
		for _, f := range seed.Franchises {
			if f.ID == id {
				found = true
			}
		}
		if !found || seen[id] {
			return fmt.Errorf("waiver order must list every franchise exactly once")
		}
		seen[id] = true
	}
	return nil
}
