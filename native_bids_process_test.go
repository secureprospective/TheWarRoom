package main

import (
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/governance"
)

func committed(t *testing.T, f *bidFixture, franchise string) int64 {
	t.Helper()
	row := f.capOf(t, franchise)
	if row.CommittedCents == nil {
		t.Fatalf("no committed salary for %s", franchise)
	}
	return int64(*row.CommittedCents)
}

func awardOf(reading commissioner.NativeBids, player string) *commissioner.NativeBidAward {
	for i := range reading.Awards {
		if reading.Awards[i].Player == player {
			return &reading.Awards[i]
		}
	}
	return nil
}

func TestNativeBidProcessingAwardsMostPointsAndSignsTheBidContract(t *testing.T) {
	f := bidApp(t)
	f.place(t, "owner.one", "franchise.01", "5004", 5*million, 1)
	three := f.place(t, "owner.three", "franchise.03", "5004", 4*million, 3)
	one, three0 := committed(t, f, "franchise.01"), committed(t, f, "franchise.03")
	processed := f.decide(t, "5004")
	for range 2 {
		p, ok := f.roster(t, "5004")
		if !ok || p.Franchise != "franchise.03" || p.Salary != moneyOf(4*million) || p.ExpirationYear != 2028 ||
			p.ContractStatus != domain.CStatusUFA || !strings.Contains(p.ContractInfo, "3 years at $4,000,000 a year") {
			t.Fatalf("the winner did not sign the bid's contract: %+v %v", p, ok)
		}
		if committed(t, f, "franchise.03") != three0+4*million || committed(t, f, "franchise.01") != one {
			t.Fatal("the cap did not follow the award")
		}
		winner, loser := f.bids(t, "owner.three", "franchise.03"), f.bids(t, "owner.one", "franchise.01")
		wa, la := awardOf(winner, "5004"), awardOf(loser, "5004")
		if wa == nil || wa.Winner != "franchise.03" || wa.Contract != "3 years at $4,000,000 a year" ||
			wa.Points != "5.600" || wa.Yours != "" || la == nil || la.Yours != "Your bid was not awarded" ||
			la.Winner != "franchise.03" || len(winner.Players) != 0 || winner.ActiveBids != 0 {
			t.Fatalf("award readings: %+v / %+v", wa, la)
		}
		if other := awardOf(f.bids(t, "owner.two", "franchise.02"), "5004"); other == nil || other.Yours != "" {
			t.Fatalf("a bystander's award reading: %+v", other)
		}
		f.restart(t)
	}
	if err := f.tryPlace(t, "owner.two", "franchise.02", "5004", 9*million, 1); err == nil ||
		!strings.Contains(err.Error(), "on a roster") {
		t.Fatalf("a decided player was bid on again: %v", err)
	}
	if _, err := f.raise(t, "owner.three", "franchise.03", three, 9*million, 1); err == nil {
		t.Fatal("a decided bid was raised")
	}
	f.revert(t, processed)
	if _, ok := f.roster(t, "5004"); ok || committed(t, f, "franchise.03") != three0 {
		t.Fatal("reversing the decision left the player signed")
	}
	if row := playerRow(f.bids(t, "owner.one", "franchise.01"), "5004"); row == nil || row.Bids != 2 {
		t.Fatalf("the reversed decision did not reopen the bids: %+v", row)
	}
	f.decide(t, "5004")
}

func TestNativeBidProcessingJudgesEachBidAgainAtTheDecision(t *testing.T) {
	f := bidApp(t)
	f.place(t, "owner.three", "franchise.03", "5004", 4*million, 1)
	f.place(t, "owner.one", "franchise.01", "5004", 9*million, 1)
	f.set(t, "roster.limits", `{"min":0,"max":40}`, "now")
	f.decide(t, "5004")
	if p, ok := f.roster(t, "5004"); !ok || p.Franchise != "franchise.03" {
		t.Fatalf("a bid that no longer fits the roster won: %+v", p)
	}
	award := awardOf(f.bids(t, "owner.one", "franchise.01"), "5004")
	if award == nil || !strings.Contains(award.Yours, "could not be awarded: roster would not be legal") {
		t.Fatalf("the refusal was not explained: %+v", award)
	}
}

func TestNativeBidProcessingNeverBreaksATie(t *testing.T) {
	f := bidApp(t)
	f.set(t, "bids.ufa", `"blind"`, "now")
	f.place(t, "owner.one", "franchise.01", "5004", 5*million, 1)
	three := f.place(t, "owner.three", "franchise.03", "5004", 5*million, 1)
	f.decide(t, "5004")
	for range 2 {
		if _, ok := f.roster(t, "5004"); ok {
			t.Fatal("a tie was settled by arrival order")
		}
		award := awardOf(f.bids(t, "owner.one", "franchise.01"), "5004")
		if award == nil || award.Winner != "" || !strings.Contains(award.Words, "tied") {
			t.Fatalf("tie reading: %+v", award)
		}
		f.restart(t)
	}
	f.override(t, three, "franchise.03")
	if p, ok := f.roster(t, "5004"); !ok || p.Franchise != "franchise.03" {
		t.Fatalf("the commissioner could not settle the tie: %+v", p)
	}
}

func TestNativeBidProcessingRefusals(t *testing.T) {
	f := bidApp(t)
	if err := f.tryDecide(t, "creator", "5004"); err == nil || !strings.Contains(err.Error(), "no bids waiting") {
		t.Fatalf("a player with no bids was decided: %v", err)
	}
	bid := f.place(t, "owner.one", "franchise.01", "5004", 5*million, 1)
	if err := f.tryDecide(t, "owner.one", "5004"); err == nil {
		t.Fatal("an owner decided bids")
	}
	f.command(t, "commish.role.grant", commissioner.RawArgs{People: &commissioner.RawPeople{
		Principal: "owner.three", Role: governance.CoCommissioner}}, "private")
	if err := f.withdraw(t, "owner.one", "franchise.01", bid); err != nil {
		t.Fatal(err)
	}
	if err := f.tryDecide(t, "owner.three", "5004"); err == nil || !strings.Contains(err.Error(), "has been withdrawn") {
		t.Fatalf("withdrawn bids were decided: %v", err)
	}
	f.place(t, "owner.one", "franchise.01", "5004", 5*million, 1)
	if err := f.tryDecide(t, "owner.three", "5004"); err != nil {
		t.Fatalf("a co-commissioner could not decide bids: %v", err)
	}
	if err := f.tryDecide(t, "creator", "5004"); err == nil {
		t.Fatal("a decided player was decided twice")
	}
	if p, ok := f.roster(t, "5004"); !ok || p.Franchise != "franchise.01" {
		t.Fatalf("the lone bidder did not win: %+v", p)
	}
}

func TestNativeBidsWaitForWaiversToClear(t *testing.T) {
	f := bidApp(t)
	cut := f.cut(t, "1000")
	f.advance(24 * time.Hour)
	cleared := f.process(t, cut.Entry)
	f.place(t, "owner.one", "franchise.01", "1000", 5*million, 1)
	// Undoing the waiver decision puts the player back on waivers with a bid already standing.
	f.revert(t, cleared)
	if err := f.tryDecide(t, "creator", "1000"); err == nil ||
		!strings.Contains(err.Error(), "still on waivers, and bids are decided once he clears them") {
		t.Fatalf("a player back on waivers had his bids decided: %v", err)
	}
	if _, ok := f.roster(t, "1000"); ok {
		t.Fatal("a player on waivers was signed")
	}
	f.process(t, cut.Entry)
	f.decide(t, "1000")
	if p, ok := f.roster(t, "1000"); !ok || p.Franchise != "franchise.01" || p.Salary != moneyOf(5*million) {
		t.Fatalf("a cleared player was not signed on the bid's terms: %+v", p)
	}
}

func TestNativeBidAwardOverrideGovernsCompensationAndReplay(t *testing.T) {
	f := bidApp(t)
	one := f.place(t, "owner.one", "franchise.01", "5004", 5*million, 1)
	f.place(t, "owner.three", "franchise.03", "5004", 6*million, 2)
	f.decide(t, "5004")
	base1, base3 := committed(t, f, "franchise.01"), committed(t, f, "franchise.03")
	assertWinner := func(franchise string, salary int64, expiry int) {
		t.Helper()
		p, ok := f.roster(t, "5004")
		if !ok || p.Franchise != franchise || p.Salary != moneyOf(salary) || p.ExpirationYear != expiry {
			t.Fatalf("award is %+v, want %s at %d through %d", p, franchise, salary, expiry)
		}
		award := awardOf(f.bids(t, "owner.two", "franchise.02"), "5004")
		if award == nil || award.Winner != franchise {
			t.Fatalf("award reading is %+v, want %s", award, franchise)
		}
	}
	assertWinner("franchise.03", 6*million, 2027)
	for range 2 {
		correction := f.override(t, one, "franchise.01")
		for range 2 {
			assertWinner("franchise.01", 5*million, 2026)
			if committed(t, f, "franchise.01") != base1+5*million || committed(t, f, "franchise.03") != base3-6*million {
				t.Fatal("the cap did not follow the corrected award")
			}
			f.restart(t)
		}
		f.revert(t, correction)
		for range 2 {
			assertWinner("franchise.03", 6*million, 2027)
			if committed(t, f, "franchise.03") != base3 || committed(t, f, "franchise.01") != base1 {
				t.Fatal("reversing the correction did not restore the cap")
			}
			f.restart(t)
		}
	}
}

func TestNativeBidOverrideRefusals(t *testing.T) {
	f := bidApp(t)
	one := f.place(t, "owner.one", "franchise.01", "5004", 5*million, 1)
	f.place(t, "owner.three", "franchise.03", "5004", 6*million, 1)
	try := func(bid commissioner.Receipt, winner string) error {
		f.app.commissionerSession = testCommissionerSession{"creator"}
		body, err := jsonOf(f.raw(t, commissioner.BidsOverride, commissioner.RawArgs{Bid: &commissioner.RawBidArgs{
			Bid: bid.Entry, Franchise: winner}}, "league"))
		if err != nil {
			t.Fatal(err)
		}
		_, err = f.app.TargetCommissionerExecute(body)
		return err
	}
	if err := try(one, "franchise.01"); err == nil || !strings.Contains(err.Error(), "newest decision") {
		t.Fatalf("an undecided player's award was corrected: %v", err)
	}
	f.decide(t, "5004")
	if err := try(one, "franchise.02"); err == nil || !strings.Contains(err.Error(), "legal bid on this player") {
		t.Fatalf("a franchise with no bid was made the winner: %v", err)
	}
	if err := try(one, "franchise.03"); err == nil || !strings.Contains(err.Error(), "already holds") {
		t.Fatalf("the winner was made the winner again: %v", err)
	}
	if err := try(commissioner.Receipt{Entry: "act.none"}, "franchise.01"); err == nil {
		t.Fatal("an unknown bid was corrected")
	}
	f.set(t, "roster.limits", `{"min":0,"max":40}`, "now")
	if err := try(one, "franchise.01"); err == nil || !strings.Contains(err.Error(), "legal bid on this player") {
		t.Fatalf("an illegal winner was installed: %v", err)
	}
}

func TestNativeBidProcessGovernsDecisionCompensationAndReplay(t *testing.T) {
	f := bidApp(t)
	f.place(t, "owner.one", "franchise.01", "5004", 5*million, 1)
	f.place(t, "owner.three", "franchise.03", "5004", 6*million, 1)
	base3 := committed(t, f, "franchise.03")
	for range 2 {
		decision := f.decide(t, "5004")
		for range 2 {
			if p, ok := f.roster(t, "5004"); !ok || p.Franchise != "franchise.03" ||
				committed(t, f, "franchise.03") != base3+6*million {
				t.Fatalf("the decision did not sign the winner: %+v", p)
			}
			f.restart(t)
		}
		f.revert(t, decision)
		for range 2 {
			if _, ok := f.roster(t, "5004"); ok || committed(t, f, "franchise.03") != base3 {
				t.Fatal("the compensating revert left the winner signed")
			}
			if row := playerRow(f.bids(t, "owner.one", "franchise.01"), "5004"); row == nil || row.Bids != 2 {
				t.Fatalf("the bids did not come back: %+v", row)
			}
			f.restart(t)
		}
	}
}

func (f *bidFixture) tryRevert(t *testing.T, principal string, receipt commissioner.Receipt, visibility string) error {
	t.Helper()
	f.app.commissionerSession = testCommissionerSession{principal}
	body, err := jsonOf(f.raw(t, commissioner.Revert, commissioner.RawArgs{Entry: receipt.Entry}, visibility))
	if err != nil {
		t.Fatal(err)
	}
	_, err = f.app.TargetCommissionerExecute(body)
	return err
}

func TestNativeBidReversalsCannotStrandALaterChange(t *testing.T) {
	f := bidApp(t)
	first := f.place(t, "owner.one", "franchise.01", "5004", 5*million, 1)
	raised, err := f.raise(t, "owner.one", "franchise.01", first, 6*million, 1)
	if err != nil {
		t.Fatal(err)
	}
	decision := f.decide(t, "5004")
	for _, target := range []commissioner.Receipt{first, raised} {
		if err := f.tryRevert(t, "creator", target, "private"); err == nil {
			t.Fatalf("reversing %s stranded the decision that depends on it", target.Entry)
		}
	}
	f.revert(t, decision)
	if err := f.tryRevert(t, "creator", raised, "private"); err != nil {
		t.Fatalf("a commissioner could not reverse an owner's raise once nothing depended on it: %v", err)
	}
	if row := playerRow(f.bids(t, "owner.one", "franchise.01"), "5004"); row == nil || row.YourBid.ID != first.Entry ||
		row.YourBid.Points != "5.000" {
		t.Fatalf("reversing a raise did not bring back the earlier bid: %+v", row)
	}
	if err := f.tryRevert(t, "owner.one", first, "private"); err == nil {
		t.Fatal("an owner reversed their own bid through the commissioner's correction")
	}
	if err := f.tryRevert(t, "creator", first, "league"); err == nil {
		t.Fatal("a private bid was reversed in public")
	}
}
