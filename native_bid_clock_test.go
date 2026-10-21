package main

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
)

func (f *bidFixture) tickBids(t *testing.T) {
	t.Helper()
	if err := f.app.core.History.ProcessDueBids(context.Background(), legalityLeague, f.app.nativeFacts()); err != nil {
		t.Fatal(err)
	}
}

func (f *bidFixture) closeAt(t *testing.T, player string, want time.Time) {
	t.Helper()
	row := playerRow(f.bids(t, "owner.one", "franchise.01"), player)
	if row == nil || row.ClosesAt == nil || !row.ClosesAt.Equal(want) || !strings.Contains(row.Closes, "PDT") {
		t.Fatalf("closing time: %+v; want %s in the viewer's zone", row, want)
	}
}

func TestNativeBidStandForGovernsClockRevertAndReopen(t *testing.T) {
	t.Parallel()
	f := bidApp(t)
	placed := f.now
	bid := f.place(t, "owner.one", "franchise.01", "5004", 5*million, 1)
	salary := int64(5 * million)
	for cycle := range 2 {
		if cycle > 0 {
			reopened := f.set(t, "bids.standFor", `259200000000000`, "now")
			salary += million
			var err error
			bid, err = f.raise(t, "owner.one", "franchise.01", bid, salary, 1)
			if err != nil {
				t.Fatal(err)
			}
			placed = f.now
			f.revert(t, reopened)
		}
		f.closeAt(t, "5004", placed.Add(24*time.Hour))
		rule := f.set(t, "bids.standFor", `172800000000000`, "now")
		f.closeAt(t, "5004", placed.Add(48*time.Hour))
		f.restart(t)
		f.closeAt(t, "5004", placed.Add(48*time.Hour))
		f.now = placed.Add(48*time.Hour - time.Microsecond)
		f.tickBids(t)
		if _, ok := f.roster(t, "5004"); ok {
			t.Fatal("awarded before closing")
		}
		f.advance(time.Microsecond)
		f.tickBids(t)
		p, ok := f.roster(t, "5004")
		if !ok || p.Franchise != "franchise.01" || p.Salary != moneyOf(salary) {
			t.Fatalf("automatic award: %+v", p)
		}
		records, err := f.app.leagueRecords(context.Background(), legalityLeague)
		if err != nil {
			t.Fatal(err)
		}
		decision := records[len(records)-1]
		if decision.Principal != "native.bid.clock" || decision.Command.Command != commissioner.BidsProcess {
			t.Fatal("not the shared processing command")
		}
		f.restart(t)
		if _, ok := f.roster(t, "5004"); !ok {
			t.Fatal("award lost on replay")
		}
		f.tickBids(t)
		f.revert(t, decision.Receipt)
		f.revert(t, rule)
		f.closeAt(t, "5004", placed.Add(24*time.Hour))
		f.restart(t)
		f.closeAt(t, "5004", placed.Add(24*time.Hour))
		// Restoring the old rule makes the reopened bid due immediately.
		f.tickBids(t)
		records, err = f.app.leagueRecords(context.Background(), legalityLeague)
		if err != nil {
			t.Fatal(err)
		}
		if _, ok := f.roster(t, "5004"); !ok {
			t.Fatal("restored rule did not govern automatic award")
		}
		f.revert(t, records[len(records)-1].Receipt)
	}
}

func TestNativeBidNewBidRestartsTheClock(t *testing.T) {
	t.Parallel()
	f := bidApp(t)
	f.place(t, "owner.one", "franchise.01", "5004", 5*million, 1)
	f.advance(12 * time.Hour)
	raised, err := f.raise(t, "owner.one", "franchise.01", commissioner.Receipt{Entry: playerRow(f.bids(t, "owner.one", "franchise.01"), "5004").YourBid.ID}, 6*million, 1)
	if err != nil {
		t.Fatal(err)
	}
	f.closeAt(t, "5004", raised.RecordedAt.Add(24*time.Hour))
	f.advance(12 * time.Hour)
	f.tickBids(t)
	if _, ok := f.roster(t, "5004"); ok {
		t.Fatal("new bid did not restart the clock")
	}
	f.advance(12 * time.Hour)
	f.app.processNativeBidsDue(context.Background())
	if p, ok := f.roster(t, "5004"); !ok || p.Salary != moneyOf(6*million) {
		t.Fatalf("restarted clock never awarded: %+v", p)
	}
}

func TestNativeBidClockLeavesTiesUndecided(t *testing.T) {
	t.Parallel()
	f := bidApp(t)
	f.set(t, "bids.ufa", `"blind"`, "now")
	f.place(t, "owner.one", "franchise.01", "5004", 5*million, 1)
	f.place(t, "owner.three", "franchise.03", "5004", 5*million, 1)
	f.advance(24 * time.Hour)
	for range 2 {
		f.tickBids(t)
		if r := f.bids(t, "owner.one", "franchise.01"); len(r.Players) != 1 || len(r.Awards) != 0 {
			t.Fatalf("clock decided a tie: %+v", r)
		}
		f.restart(t)
	}
	f.decide(t, "5004")
	if a := awardOf(f.bids(t, "owner.one", "franchise.01"), "5004"); a == nil || !strings.Contains(a.Words, "tied") {
		t.Fatal("commissioner cannot decide tie")
	}
}

func TestNativeBidClockWaitsForWaivers(t *testing.T) {
	t.Parallel()
	f := bidApp(t)
	cut := f.cut(t, "1000")
	f.advance(24 * time.Hour)
	cleared := f.process(t, cut.Entry)
	f.place(t, "owner.one", "franchise.01", "1000", 5*million, 1)
	f.revert(t, cleared)
	f.advance(24 * time.Hour)
	f.tickBids(t)
	if r := f.bids(t, "owner.one", "franchise.01"); len(r.Players) != 1 || len(r.Awards) != 0 {
		t.Fatalf("clock decided a player on waivers: %+v", r)
	}
	f.process(t, cut.Entry)
	f.tickBids(t)
	if _, ok := f.roster(t, "1000"); !ok {
		t.Fatal("cleared bid was not awarded")
	}
}

func TestNativeBidJobsShowBuiltQueues(t *testing.T) {
	t.Parallel()
	f := bidApp(t)
	f.place(t, "owner.one", "franchise.01", "5004", 5*million, 1)
	f.cut(t, "1000")
	jobs, err := f.app.TargetCommissionerJobs(legalityLeague)
	if err != nil {
		t.Fatal(err)
	}
	found := 0
	for _, job := range jobs.Jobs {
		if job.ID != "bids" && job.ID != "waivers" {
			continue
		}
		found++
		if job.State != commissioner.JobLive || !strings.Contains(job.Words, ": 1.") || job.Closes == "" {
			t.Fatalf("queue state: %+v", job)
		}
	}
	if found != 2 {
		t.Fatal("built jobs missing")
	}
}

// clockTestAuthority exercises the transaction's own due check without the worker's preflight scan.
type clockTestAuthority struct{}

func (clockTestAuthority) Allow(string, string, string) (string, error) { return "automatic", nil }

func TestNativeBidClockRechecksDueInsideTransaction(t *testing.T) {
	t.Parallel()
	f := bidApp(t)
	bid := f.place(t, "owner.one", "franchise.01", "5004", 5*million, 1)
	try := func(want string) {
		t.Helper()
		raw := f.raw(t, commissioner.BidsProcess, commissioner.RawArgs{Bid: &commissioner.RawBidArgs{Player: "5004"}}, "league")
		_, err := f.app.core.History.ExecuteCommissionerWithFacts(t.Context(), raw, "native.bid.clock", clockTestAuthority{}, f.app.nativeFacts(), time.UTC)
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Fatalf("transaction bypass: %v", err)
		}
	}
	try("stood its full time")
	f.advance(12 * time.Hour)
	if _, err := f.raise(t, "owner.one", "franchise.01", bid, 6*million, 1); err != nil {
		t.Fatal(err)
	}
	f.advance(12 * time.Hour)
	try("stood its full time")
	f.set(t, "bids.standFor", `172800000000000`, "now")
	f.advance(12 * time.Hour)
	try("stood its full time")
	f.advance(24 * time.Hour)
	f.tickBids(t)
	if _, ok := f.roster(t, "5004"); !ok {
		t.Fatal("fully stood bid not awarded")
	}
}

func TestNativeBidEveryNewBidResetsEvenWhenNotLeadingOrWithdrawn(t *testing.T) {
	t.Parallel()
	f := bidApp(t)
	f.set(t, "bids.ufa", `"blind"`, "now")
	f.place(t, "owner.one", "franchise.01", "5004", 5*million, 1)
	f.advance(12 * time.Hour)
	lower := f.place(t, "owner.three", "franchise.03", "5004", 4*million, 1)
	if err := f.withdraw(t, "owner.three", "franchise.03", lower); err != nil {
		t.Fatal(err)
	}
	f.set(t, "bids.ufa", `"open"`, "now")
	f.closeAt(t, "5004", lower.RecordedAt.Add(24*time.Hour))
	f.advance(12 * time.Hour)
	f.tickBids(t)
	if _, ok := f.roster(t, "5004"); ok {
		t.Fatal("withdrawing a lower bid erased its clock reset")
	}
	f.advance(12 * time.Hour)
	f.tickBids(t)
	if p, ok := f.roster(t, "5004"); !ok || p.Franchise != "franchise.01" {
		t.Fatalf("standing leader not awarded: %+v", p)
	}
}

func TestNativeBidJobsDoNotRevealBlindAuctionCountsOrTimes(t *testing.T) {
	t.Parallel()
	f := bidApp(t)
	f.set(t, "bids.ufa", `"blind"`, "now")
	read := func() commissioner.Job {
		t.Helper()
		jobs, err := f.app.TargetCommissionerJobs(legalityLeague)
		if err != nil {
			t.Fatal(err)
		}
		for _, job := range jobs.Jobs {
			if job.ID == "bids" {
				return job
			}
		}
		t.Fatal("bid job missing")
		return commissioner.Job{}
	}
	before := read()
	f.place(t, "owner.one", "franchise.01", "5004", 5*million, 1)
	after := read()
	if before.Words != after.Words || after.Closes != "" || after.State != commissioner.JobLive {
		t.Fatalf("public jobs leaked blind bids: %+v / %+v", before, after)
	}
}

func TestNativeBidClockRechecksLegalityAndClosesUnawardableBids(t *testing.T) {
	t.Parallel()
	f := bidApp(t)
	f.place(t, "owner.three", "franchise.03", "5004", 4*million, 1)
	f.place(t, "owner.one", "franchise.01", "5004", 9*million, 1)
	f.place(t, "owner.one", "franchise.01", "5005", 5*million, 1)
	f.set(t, "roster.limits", `{"min":0,"max":40}`, "now")
	f.advance(24 * time.Hour)
	f.tickBids(t)
	if p, ok := f.roster(t, "5004"); !ok || p.Franchise != "franchise.03" {
		t.Fatalf("illegal leader awarded: %+v", p)
	}
	reading := f.bids(t, "owner.one", "franchise.01")
	award := awardOf(reading, "5005")
	if award == nil || award.Winner != "" || !strings.Contains(award.Yours, "roster would not be legal") {
		t.Fatalf("unawardable auction not closed with its reason: %+v", award)
	}
	if _, ok := f.roster(t, "5005"); ok {
		t.Fatal("an illegal bid signed a contract")
	}
}

func TestNativeBidActionsRefusedAtAuctionDeadline(t *testing.T) {
	t.Parallel()
	for _, action := range []string{"place", "raise", "withdraw"} {
		t.Run(action, func(t *testing.T) {
			f := bidApp(t)
			bid := f.place(t, "owner.one", "franchise.01", "5004", 5*million, 1)
			f.now = bid.RecordedAt.Add(24 * time.Hour)
			var err error
			switch action {
			case "place":
				err = f.tryPlace(t, "owner.two", "franchise.02", "5004", 7*million, 1)
			case "raise":
				_, err = f.raise(t, "owner.one", "franchise.01", bid, 6*million, 1)
			case "withdraw":
				err = f.withdraw(t, "owner.one", "franchise.01", bid)
			}
			if err == nil || !strings.Contains(err.Error(), "closed") {
				t.Fatalf("%s at deadline: %v", action, err)
			}
		})
	}
}
