package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/nativecheck"
)

func ledgerHas(t *testing.T, f *bidFixture, principal, entry string) bool {
	t.Helper()
	f.app.commissionerSession = testCommissionerSession{principal}
	ledger, err := f.app.TargetCommissionerLedger(legalityLeague)
	if err != nil {
		t.Fatal(err)
	}
	return slices.ContainsFunc(ledger.Entries, func(e commissioner.Entry) bool { return e.ID == entry })
}

func playerRow(reading commissioner.NativeBids, player string) *commissioner.NativeBidPlayer {
	for i := range reading.Players {
		if reading.Players[i].Player == player {
			return &reading.Players[i]
		}
	}
	return nil
}

func TestNativeBidPlaceRaiseAndWithdrawThroughReopen(t *testing.T) {
	t.Parallel()
	f := bidApp(t)
	first := f.place(t, "owner.one", "franchise.01", "5001", 5*million, 1)
	f.place(t, "owner.two", "franchise.02", "5001", 6*million, 1)
	for range 2 {
		row := playerRow(f.bids(t, "owner.one", "franchise.01"), "5001")
		if row == nil || row.Leader == nil || row.Leader.Franchise != "franchise.02" || row.YouLead ||
			row.Leader.Contract != "1 year at $6,000,000 a year" || row.Leader.Points != "6.000" ||
			row.YourBid == nil || row.YourBid.Points != "5.000" || row.Bids != 2 {
			t.Fatalf("open bid reading: %+v", row)
		}
		f.restart(t)
	}
	raised, err := f.raise(t, "owner.one", "franchise.01", first, 7*million, 2)
	if err != nil {
		t.Fatal(err)
	}
	for range 2 {
		reading := f.bids(t, "owner.one", "franchise.01")
		row := playerRow(reading, "5001")
		if row == nil || !row.YouLead || row.YourBid.ID != raised.Entry || row.YourBid.Points != "8.400" ||
			row.Bids != 2 || reading.ActiveBids != 1 {
			t.Fatalf("raised bid reading: %+v", reading)
		}
		f.restart(t)
	}
	if _, err := f.raise(t, "owner.one", "franchise.01", first, 9*million, 2); err == nil {
		t.Fatal("a bid that was already raised was raised again")
	}
	if err := f.withdraw(t, "owner.one", "franchise.01", raised); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		reading := f.bids(t, "owner.one", "franchise.01")
		row := playerRow(reading, "5001")
		if row == nil || row.YourBid != nil || row.Leader.Franchise != "franchise.02" || reading.ActiveBids != 0 {
			t.Fatalf("withdrawn bid still standing: %+v", reading)
		}
		f.restart(t)
	}
	if err := f.withdraw(t, "owner.one", "franchise.01", raised); err == nil {
		t.Fatal("a withdrawn bid was withdrawn twice")
	}
}

func TestNativeBlindBidIsHiddenFromOtherOwners(t *testing.T) {
	t.Parallel()
	f := bidApp(t)
	f.set(t, "bids.ufa", `"blind"`, "now")
	bid := f.place(t, "owner.one", "franchise.01", "5001", 5*million, 1)
	other := f.place(t, "owner.two", "franchise.02", "5001", 7*million, 1)
	for range 2 {
		for _, viewer := range []struct{ principal, franchise string }{{"owner.three", "franchise.03"},
			{"owner.two", "franchise.02"}} {
			reading := f.bids(t, viewer.principal, viewer.franchise)
			body, err := json.Marshal(reading)
			if err != nil {
				t.Fatal(err)
			}
			mine := viewer.franchise == "franchise.02"
			if strings.Contains(string(body), bid.Entry) || strings.Contains(string(body), "5.000") ||
				strings.Contains(string(body), "franchise.01") {
				t.Fatalf("a blind bid reached %s: %s", viewer.franchise, body)
			}
			row := playerRow(reading, "5001")
			if mine != (row != nil) || (mine && (row.Leader != nil || row.YourBid.ID != other.Entry || row.Mode != "blind")) {
				t.Fatalf("blind reading for %s: %+v", viewer.franchise, row)
			}
		}
		for _, principal := range []string{"owner.three", "creator", "owner.two"} {
			if ledgerHas(t, f, principal, bid.Entry) {
				t.Fatalf("a blind bid reached the ledger of %s", principal)
			}
		}
		if !ledgerHas(t, f, "owner.one", bid.Entry) {
			t.Fatal("the bidder cannot see their own bid in the ledger")
		}
		f.restart(t)
	}
}

func TestNativeBlindBidIsNotHeldToAnotherOwnersHiddenBid(t *testing.T) {
	t.Parallel()
	f := bidApp(t)
	f.set(t, "bids.ufa", `"blind"`, "now")
	f.place(t, "owner.one", "franchise.01", "5001", 5*million, 1)
	low := f.place(t, "owner.three", "franchise.03", "5001", 3*million, 1)
	// A refusal that named another owner's bid would give it away, so a bid is held only to the owner's own.
	if _, err := f.raise(t, "owner.three", "franchise.03", low, 309*million/100, 1); err == nil ||
		!strings.Contains(err.Error(), "at least 3.100") || strings.Contains(err.Error(), "5.0") {
		t.Fatalf("a blind raise was not held to the owner's own bid: %v", err)
	}
	if _, err := f.raise(t, "owner.three", "franchise.03", low, 4*million, 1); err != nil {
		t.Fatalf("a blind raise of a full million was refused: %v", err)
	}
}

func TestNativeOpenBidShowsLeaderAndSnipeCountdown(t *testing.T) {
	t.Parallel()
	f := bidApp(t)
	placed := f.now
	f.place(t, "owner.two", "franchise.02", "5001", 5*million, 1)
	for range 2 {
		row := playerRow(f.bids(t, "owner.three", "franchise.03"), "5001")
		pdt := time.FixedZone("PDT", -7*60*60)
		if row == nil || row.Leader == nil || row.Leader.Franchise != "franchise.02" ||
			!row.LateAt.Equal(placed.Add(20*time.Hour)) || row.LateAt.In(pdt).Format("3:04 PM") != "1:00 AM" ||
			row.LateNow || !strings.Contains(row.LateWords, "Oct 14, 2026 at 1:00 AM PDT") ||
			!strings.Contains(row.LateWords, "by 1.000 points") {
			t.Fatalf("snipe countdown: %+v", row)
		}
		f.restart(t)
	}
	f.advance(20*time.Hour + time.Second)
	if row := playerRow(f.bids(t, "owner.three", "franchise.03"), "5001"); !row.LateNow ||
		!strings.Contains(row.LateWords, "Late bids now") {
		t.Fatalf("the countdown did not reach the mark: %+v", row)
	}
}

func TestNativeSnipeBidNeedsAFullPoint(t *testing.T) {
	t.Parallel()
	f := bidApp(t)
	f.place(t, "owner.two", "franchise.02", "5001", 5*million, 1)
	f.advance(20*time.Hour + time.Minute)
	err := f.tryPlace(t, "owner.one", "franchise.01", "5001", 51*million/10, 1)
	if err == nil || !strings.Contains(err.Error(), "late bid must beat") || !strings.Contains(err.Error(), "at least 6.000") {
		t.Fatalf("a late bid cleared at +0.1: %v", err)
	}
	if err := f.tryPlace(t, "owner.one", "franchise.01", "5001", 6*million, 1); err != nil {
		t.Fatalf("a late bid a full point higher was refused: %v", err)
	}
	row := playerRow(f.bids(t, "owner.three", "franchise.03"), "5001")
	if row.Leader.Franchise != "franchise.01" {
		t.Fatalf("the late bid did not take the lead: %+v", row)
	}
}

func TestNativeBidStepWithinTheDayIsATenthOfAPoint(t *testing.T) {
	t.Parallel()
	f := bidApp(t)
	f.place(t, "owner.two", "franchise.02", "5001", 5*million, 1)
	if err := f.tryPlace(t, "owner.one", "franchise.01", "5001", 509*million/100, 1); err == nil ||
		!strings.Contains(err.Error(), "at least 5.100") {
		t.Fatalf("a bid under the step was accepted: %v", err)
	}
	if err := f.tryPlace(t, "owner.one", "franchise.01", "5001", 51*million/10, 1); err != nil {
		t.Fatalf("a bid a tenth higher was refused: %v", err)
	}
}

func TestNativeBidLimitsOnLengthAndSalary(t *testing.T) {
	t.Parallel()
	f := bidApp(t)
	for _, c := range []struct {
		salary int64
		years  int
		refuse string
	}{
		{12 * million, 1, ""}, {12*million + 1_000_000, 1, "no more than $12,000,000"},
		{13 * million, 2, ""}, {24 * million, 2, ""}, {25 * million, 2, "needs at least 3 years"},
		{25 * million, 3, ""}, {5*million + 1, 1, "multiple of $10,000"}, {200_000_00, 1, "minimum salary"},
	} {
		player := "5002"
		err := f.tryPlace(t, "owner.one", "franchise.01", player, c.salary, c.years)
		if (c.refuse == "") != (err == nil) || (err != nil && !strings.Contains(err.Error(), c.refuse)) {
			t.Fatalf("%d for %d years: %v, want %q", c.salary, c.years, err, c.refuse)
		}
		if err == nil {
			row := playerRow(f.bids(t, "owner.one", "franchise.01"), player)
			if err := f.withdraw(t, "owner.one", "franchise.01", commissioner.Receipt{Entry: row.YourBid.ID}); err != nil {
				t.Fatal(err)
			}
		}
	}
}

func TestNativeBidActiveLimitCountsPlayersNotRaises(t *testing.T) {
	t.Parallel()
	f := bidApp(t)
	f.set(t, "bids.maximumActive", `2`, "now")
	first := f.place(t, "owner.one", "franchise.01", "5001", 5*million, 1)
	f.place(t, "owner.one", "franchise.01", "5002", 5*million, 1)
	if err := f.tryPlace(t, "owner.one", "franchise.01", "5003", 5*million, 1); err == nil ||
		!strings.Contains(err.Error(), "already have 2 active bids") {
		t.Fatalf("a bid over the active limit was accepted: %v", err)
	}
	if _, err := f.raise(t, "owner.one", "franchise.01", first, 6*million, 1); err != nil {
		t.Fatalf("a raise counted as a new bid: %v", err)
	}
	if err := f.tryPlace(t, "owner.one", "franchise.01", "5001", 9*million, 1); err == nil ||
		!strings.Contains(err.Error(), "raise it instead") {
		t.Fatalf("a second bid on one player was accepted: %v", err)
	}
}

func TestNativeBidPhaseRules(t *testing.T) {
	t.Parallel()
	f := bidApp(t)
	out := map[string]nativecheck.PlayoffStanding{"franchise.01": nativecheck.PlayoffIn, "franchise.02": nativecheck.PlayoffOut}
	f.season.playoff = out
	try := func(week int, franchise, principal string, salary int64, years int) error {
		f.season.week = week
		err := f.tryPlace(t, principal, franchise, "5001", salary, years)
		if err == nil {
			row := playerRow(f.bids(t, principal, franchise), "5001")
			if werr := f.withdraw(t, principal, franchise, commissioner.Receipt{Entry: row.YourBid.ID}); werr != nil {
				t.Fatal(werr)
			}
		}
		return err
	}
	for _, c := range []struct {
		name      string
		week      int
		franchise string
		principal string
		salary    int64
		years     int
		refuse    string
	}{
		{"week 12 anyone may bid long", 12, "franchise.02", "owner.two", 5 * million, 3, ""},
		{"multi-year bids stop after week 12 for a playoff team", 13, "franchise.01", "owner.one", 5 * million, 2,
			"a playoff team may bid for 1 year at most"},
		{"a playoff team bids one year in week 13", 13, "franchise.01", "owner.one", 5 * million, 1, ""},
		{"a team that missed the playoffs may not bid", 13, "franchise.02", "owner.two", 5 * million, 1,
			"only playoff teams may bid, and yours is not one"},
		{"a team the app cannot place may not bid", 13, "franchise.03", "owner.three", 5 * million, 1,
			"cannot tell whether yours is one"},
		{"the playoff cap lifts in the playoffs", 14, "franchise.01", "owner.one", 30 * million, 1, ""},
		{"the cap holds before the playoffs", 13, "franchise.01", "owner.one", 30 * million, 1, "no more than $12,000,000"},
		{"multi-year bids closed", 14, "franchise.01", "owner.one", 5 * million, 3, "closed at the end of week 13"},
	} {
		err := try(c.week, c.franchise, c.principal, c.salary, c.years)
		if (c.refuse == "") != (err == nil) || (err != nil && !strings.Contains(err.Error(), c.refuse)) {
			t.Errorf("%s: %v, want %q", c.name, err, c.refuse)
		}
	}
	f.season.known = false
	if err := f.tryPlace(t, "owner.one", "franchise.01", "5001", 5*million, 1); err == nil ||
		!strings.Contains(err.Error(), "holds no NFL week") {
		t.Fatalf("a bid was judged without a week: %v", err)
	}
}

func TestNativeBidRefusesWhatTheRostersAndWaiversForbid(t *testing.T) {
	t.Parallel()
	f := bidApp(t)
	if err := f.tryPlace(t, "owner.one", "franchise.01", "1000", 5*million, 1); err == nil ||
		!strings.Contains(err.Error(), "on a roster") {
		t.Fatalf("a rostered player was bid on: %v", err)
	}
	if err := f.tryPlace(t, "owner.one", "franchise.01", "9999", 5*million, 1); err == nil ||
		!strings.Contains(err.Error(), "player the app holds") {
		t.Fatalf("an unknown player was bid on: %v", err)
	}
	cut := f.cut(t, "1000")
	if err := f.tryPlace(t, "owner.one", "franchise.01", "1000", 5*million, 1); err == nil ||
		!strings.Contains(err.Error(), "on waivers, and bids open once he clears them") {
		t.Fatalf("a player on waivers was bid on: %v", err)
	}
	f.advance(24 * time.Hour)
	f.process(t, cut.Entry)
	if err := f.tryPlace(t, "owner.two", "franchise.02", "1000", 5*million, 1); err == nil ||
		!strings.Contains(err.Error(), "you released") {
		t.Fatalf("a franchise re-signed the player it released this season: %v", err)
	}
	f.set(t, "contracts.releaseReSignSameSeason", `true`, "now")
	f.place(t, "owner.two", "franchise.02", "1000", 5*million, 1)
	f.place(t, "owner.one", "franchise.01", "1000", 6*million, 1)
}

func TestNativeBidRefusesWhatTheRosterAndCapForbid(t *testing.T) {
	t.Parallel()
	f := bidApp(t)
	f.set(t, "roster.limits", `{"min":0,"max":40}`, "now")
	if err := f.tryPlace(t, "owner.one", "franchise.01", "5001", 5*million, 1); err == nil ||
		!strings.Contains(err.Error(), "roster would not be legal") {
		t.Fatalf("a bid that breaks the roster limit was accepted: %v", err)
	}
	f.set(t, "roster.limits", `{"min":0,"max":100}`, "now")
	f.set(t, "cap.amount", `100000000`, "now")
	if err := f.tryPlace(t, "owner.one", "franchise.01", "5001", 5*million, 1); err == nil ||
		!strings.Contains(strings.ToLower(err.Error()), "cap") {
		t.Fatalf("a bid that breaks the cap was accepted: %v", err)
	}
	if reading := f.bids(t, "owner.one", "franchise.01"); len(reading.Players) != 0 {
		t.Fatal("a refused bid was recorded")
	}
}

func TestNativeBidActionsCannotSkipTheGateOrTheOwnerSeat(t *testing.T) {
	t.Parallel()
	f := bidApp(t)
	standing := f.place(t, "owner.one", "franchise.01", "5001", 5*million, 1)
	f.set(t, "calendar.windows", `{}`, "now")
	args := commissioner.RawBidArgs{Player: "5002", Franchise: "franchise.01", Salary: moneyOf(5 * million), Years: 1}
	raw := f.raw(t, commissioner.BidPlace, commissioner.RawArgs{Bid: &args}, "private")
	f.app.commissionerSession = testCommissionerSession{"owner.one"}
	if _, err := f.app.TargetNativeBidAction(raw); err == nil {
		t.Fatal("a bid skipped the native gate")
	}
	if _, err := f.app.history.ExecuteCommissionerWithFacts(t.Context(), raw, "owner.one", nil,
		f.app.nativeFacts(), f.app.ownerZone()); err == nil {
		t.Fatal("a bid skipped the gate inside the transaction")
	}
	body, err := jsonOf(raw)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.app.TargetCommissionerExecute(body); err == nil {
		t.Fatal("an owner bid used the commissioner door")
	}
	f.set(t, "calendar.windows", `{"native":{"start":{"month":1,"day":1},"opens":["cut","waiver_claim","ufa_bid"]}}`, "now")
	if err := f.tryPlace(t, "owner.two", "franchise.01", "5002", 5*million, 1); err == nil {
		t.Fatal("an owner bid for another franchise")
	}
	if err := f.tryPlace(t, "creator", "franchise.01", "5002", 5*million, 1); err == nil {
		t.Fatal("a commissioner bid as an owner")
	}
	if err := f.withdraw(t, "owner.two", "franchise.02", standing); err == nil {
		t.Fatal("an owner withdrew another owner's bid")
	}
	if _, err := f.raise(t, "owner.two", "franchise.02", standing, 9*million, 1); err == nil {
		t.Fatal("an owner raised another owner's bid")
	}
	f.app.commissionerSession = testCommissionerSession{"owner.one"}
	if _, err := f.app.TargetNativeBids(legalityLeague, "franchise.01"); err != nil {
		t.Fatal(err)
	}
	f.app.commissionerSession = testCommissionerSession{"owner.two"}
	if _, err := f.app.TargetNativeBids(legalityLeague, "franchise.01"); err == nil {
		t.Fatal("an owner read another franchise's bids")
	}
}

func TestNativeBidDecisionsCannotBeSuppliedByTheRequest(t *testing.T) {
	t.Parallel()
	f := bidApp(t)
	args := commissioner.RawBidArgs{Player: "5001", Franchise: "franchise.01", Salary: moneyOf(5 * million), Years: 1}
	raw := f.raw(t, commissioner.BidPlace, commissioner.RawArgs{Bid: &args,
		BidOutcome: &commissioner.BidOutcome{Points: 1}}, "private")
	f.app.commissionerSession = testCommissionerSession{"owner.one"}
	if _, err := f.app.TargetNativeBidAction(raw); err == nil {
		t.Fatal("a request carried its own bid decision")
	}
	body, err := jsonOf(raw)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.app.TargetCommissionerExecute(body); err == nil {
		t.Fatal("the commissioner door took a bid decision")
	}
}

func TestNativeBidRetryReturnsTheRecordedReceipt(t *testing.T) {
	t.Parallel()
	f := bidApp(t)
	args := commissioner.RawBidArgs{Player: "5001", Franchise: "franchise.01", Salary: moneyOf(5 * million), Years: 1}
	raw := f.raw(t, commissioner.BidPlace, commissioner.RawArgs{Bid: &args}, "private")
	f.app.commissionerSession = testCommissionerSession{"owner.one"}
	first, err := f.app.TargetNativeBidAction(raw)
	if err != nil {
		t.Fatal(err)
	}
	again, err := f.app.TargetNativeBidAction(raw)
	if err != nil || again.Entry != first.Entry {
		t.Fatalf("a retried bid recorded twice: %v %v", again, err)
	}
	f.restart(t)
	f.app.commissionerSession = testCommissionerSession{"owner.one"}
	if again, err = f.app.TargetNativeBidAction(raw); err != nil || again.Entry != first.Entry {
		t.Fatalf("a retry after a restart recorded twice: %v %v", again, err)
	}
	if reading := f.bids(t, "owner.one", "franchise.01"); reading.ActiveBids != 1 {
		t.Fatalf("the retry changed the bids: %+v", reading)
	}
	stale := raw
	stale.Request = "stale"
	if _, err := f.app.TargetNativeBidAction(stale); err == nil {
		t.Fatal("a stale league version was accepted")
	}
}

func TestNativeBidsWire(t *testing.T) {
	t.Parallel()
	f := bidApp(t)
	f.place(t, "owner.one", "franchise.01", "5004", 5*million, 1)
	f.place(t, "owner.three", "franchise.03", "5004", 4*million, 3)
	f.decide(t, "5004")
	f.place(t, "owner.two", "franchise.02", "5001", 5*million, 1)
	f.place(t, "owner.one", "franchise.01", "5001", 6*million, 2)
	reading := f.bids(t, "owner.one", "franchise.01")
	body, err := json.MarshalIndent(reading, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	body = append(body, '\n')
	path := filepath.Join("internal", "commissioner", "testdata", "native-bids.wire.json")
	if os.Getenv("TWR_WRITE_WIRE") == "1" {
		if err := os.WriteFile(path, body, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(want) != string(body) {
		t.Fatal("native bids wire differs; regenerate with Go")
	}
}
