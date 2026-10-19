package main

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

const million = int64(100_000_000)

// heldSeason is the NFL week and playoff field a test says the app holds.
type heldSeason struct {
	week    int
	known   bool
	playoff map[string]nativecheck.PlayoffStanding
}

func (h *heldSeason) BidEnvironment(context.Context, string, time.Time) (commissioner.BidEnvironment, error) {
	return commissioner.BidEnvironment{Season: nativecheck.BidSeason{Week: h.week, WeekKnown: h.known},
		Playoff: h.playoff}, nil
}

// bidFixture is the waiver league with the free-agent window open, three owners and a fixed NFL week.
// Players 5001..5006 are free agents the app holds; nothing else in the league is.
type bidFixture struct {
	*waiverFixture
	season *heldSeason
}

func bidApp(t *testing.T) *bidFixture {
	t.Helper()
	f := &bidFixture{waiverFixture: waiverApp(t), season: &heldSeason{week: 5, known: true,
		playoff: map[string]nativecheck.PlayoffStanding{}}}
	f.set(t, "calendar.windows", `{"native":{"start":{"month":1,"day":1},"opens":["cut","waiver_claim","ufa_bid"]}}`, "now")
	f.app.history.SetBidEnvironment(f.season)
	return f
}

func (f *bidFixture) restart(t *testing.T) {
	t.Helper()
	f.waiverFixture.restart(t)
	f.app.history.SetBidEnvironment(f.season)
}

func (f *bidFixture) ownerBid(t *testing.T, principal, command string, args commissioner.RawBidArgs) (commissioner.Receipt, error) {
	t.Helper()
	f.app.commissionerSession = testCommissionerSession{principal}
	return f.app.TargetNativeBidAction(f.raw(t, command, commissioner.RawArgs{Bid: &args}, "private"))
}

func (f *bidFixture) place(t *testing.T, principal, franchise, player string, salary int64, years int) commissioner.Receipt {
	t.Helper()
	receipt, err := f.ownerBid(t, principal, commissioner.BidPlace, commissioner.RawBidArgs{Player: player,
		Franchise: franchise, Salary: moneyOf(salary), Years: years})
	if err != nil {
		t.Fatal(err)
	}
	return receipt
}

func (f *bidFixture) tryPlace(t *testing.T, principal, franchise, player string, salary int64, years int) error {
	t.Helper()
	_, err := f.ownerBid(t, principal, commissioner.BidPlace, commissioner.RawBidArgs{Player: player,
		Franchise: franchise, Salary: moneyOf(salary), Years: years})
	return err
}

func (f *bidFixture) raise(t *testing.T, principal, franchise string, bid commissioner.Receipt, salary int64, years int) (commissioner.Receipt, error) {
	t.Helper()
	return f.ownerBid(t, principal, commissioner.BidRaise, commissioner.RawBidArgs{Bid: bid.Entry, Franchise: franchise,
		Salary: moneyOf(salary), Years: years})
}

func (f *bidFixture) withdraw(t *testing.T, principal, franchise string, bid commissioner.Receipt) error {
	t.Helper()
	_, err := f.ownerBid(t, principal, commissioner.BidWithdraw, commissioner.RawBidArgs{Bid: bid.Entry, Franchise: franchise})
	return err
}

func (f *bidFixture) bids(t *testing.T, principal, franchise string) commissioner.NativeBids {
	t.Helper()
	f.app.commissionerSession = testCommissionerSession{principal}
	reading, err := f.app.TargetNativeBids(legalityLeague, franchise)
	if err != nil {
		t.Fatal(err)
	}
	return reading
}

func (f *bidFixture) decide(t *testing.T, player string) commissioner.Receipt {
	t.Helper()
	return f.command(t, commissioner.BidsProcess, commissioner.RawArgs{Bid: &commissioner.RawBidArgs{Player: player}}, "league")
}

func (f *bidFixture) tryDecide(t *testing.T, principal, player string) error {
	t.Helper()
	f.app.commissionerSession = testCommissionerSession{principal}
	body, err := jsonOf(f.raw(t, commissioner.BidsProcess, commissioner.RawArgs{Bid: &commissioner.RawBidArgs{Player: player}}, "league"))
	if err != nil {
		t.Fatal(err)
	}
	_, err = f.app.TargetCommissionerExecute(body)
	return err
}

func (f *bidFixture) override(t *testing.T, bid commissioner.Receipt, winner string) commissioner.Receipt {
	t.Helper()
	return f.command(t, commissioner.BidsOverride, commissioner.RawArgs{Bid: &commissioner.RawBidArgs{Bid: bid.Entry,
		Franchise: winner}}, "league")
}

func (f *bidFixture) roster(t *testing.T, player string) (commissioner.NativePlayer, bool) {
	t.Helper()
	assets, err := f.app.TargetNativeAssets(legalityLeague)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range assets.Snapshot.Players {
		if p.ID == player {
			return p, true
		}
	}
	return commissioner.NativePlayer{}, false
}

func (f *bidFixture) capOf(t *testing.T, franchise string) commissioner.NativeCapFranchise {
	t.Helper()
	check, err := f.app.TargetNativeCapCheck(legalityLeague, "America/New_York")
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range check.Franchises {
		if row.Franchise == franchise {
			return row
		}
	}
	t.Fatalf("no cap row for %s", franchise)
	return commissioner.NativeCapFranchise{}
}

func (f *bidFixture) advance(d time.Duration) { f.now = f.now.Add(d) }

func moneyOf(cents int64) domain.Money { return domain.Money(cents) }

func jsonOf(v any) (string, error) {
	body, err := json.Marshal(v)
	return string(body), err
}
