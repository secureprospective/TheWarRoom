package envelope

import (
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/engine/bid"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
)

func bidAt() time.Time { return time.Date(2026, 5, 4, 12, 0, 0, 0, time.UTC) }

func bidFacts(t *testing.T) BidFacts {
	t.Helper()
	return BidFacts{Player: mustTestID(t), Salary: 500_000_000, Years: 3, Points: 7000}
}

func passingBid() bid.Input {
	return bid.Input{Salary: 500_000_000, Years: 3, Minimum: bid.MinimumSalary(3),
		Experience: bid.Experience{Years: 3, Known: true}, WeekKnown: true}
}

func draftedBid(t *testing.T, input bid.Input) Envelope {
	t.Helper()
	e, err := DraftBid(bidAt(), BidRequest{LeagueID: "1", FranchiseID: "0001", Facts: bidFacts(t)},
		BidCheck{Input: input}, func() string { return "bid-1" })
	if err != nil {
		t.Fatal(err)
	}
	return e
}

func TestDraftBidReadyAndBlocked(t *testing.T) {
	e := draftedBid(t, passingBid())
	r := e.Receipt()
	if r.State != Ready || r.Spec.Intent != BidIntent || r.Spec.Gravity != G3 || r.Spec.Undo != Irreversible ||
		r.Spec.Target.Kind != Manual || r.Spec.Target.URL != "" ||
		r.Spec.Expected.Player != bidFacts(t).Player || r.Spec.Expected.RosterStatus != domain.RosterActive {
		t.Fatalf("ready bid: %+v", r)
	}
	if !strings.Contains(r.Audit[0].Note, "ready to hand off") {
		t.Errorf("note = %q", r.Audit[0].Note)
	}
	over := passingBid()
	over.Years = 5
	blocked := draftedBid(t, over)
	if blocked.State() != Blocked || !strings.Contains(blocked.Receipt().Audit[0].Note, "A bid runs 1 to 4 years, not 5") {
		t.Errorf("blocked bid: %+v", blocked.Receipt())
	}
}

func TestBidRunsTheLifecycleThroughPostedToLanded(t *testing.T) {
	e := draftedBid(t, passingBid())
	handed, err := e.HandOffManual(bidAt().Add(time.Minute), "post text copied; nothing posted")
	if err != nil || handed.State() != HandedOff {
		t.Fatal(handed.State(), err)
	}
	pending, err := handed.Await(bidAt().Add(2*time.Minute), AwaitBid)
	if err != nil || pending.State() != BidPending ||
		!strings.Contains(pending.Receipt().Audit[2].Note, "owner reports the bid is posted") {
		t.Fatal(pending.Receipt(), err)
	}
	obs := bidObservation(t, "0001", 500_000_000, 3, bidAt().Add(time.Hour))
	landed, err := pending.Observe(bidAt().Add(2*time.Hour), obs, BidPredicate{Facts: bidFacts(t)})
	if err != nil || landed.State() != Landed {
		t.Fatal(landed.Receipt(), err)
	}
}

func TestBidCannotLeaveBlockedOrUseAnMFLHandOff(t *testing.T) {
	over := passingBid()
	over.Years = 5
	if _, err := draftedBid(t, over).HandOffManual(bidAt().Add(time.Second), "x"); err == nil {
		t.Error("a blocked bid was handed off")
	}
	ir, at, _ := testEnvelope(t)
	if _, err := ir.HandOffManual(at, "x"); err == nil {
		t.Error("a mapped MFL plan took a manual hand-off")
	}
	if _, err := draftedBid(t, passingBid()).HandOff(bidAt().Add(time.Second)); err != nil {
		t.Errorf("a manual plan could not take the shared hand-off: %v", err)
	}
}

// bidObservation is the rosters export after the hand-off with the player held by holder.
func bidObservation(t *testing.T, holder string, salary domain.Money, years int, at time.Time) Observation {
	t.Helper()
	id := mustTestID(t)
	mine := snapshot.Roster{FranchiseID: "0001"}
	other := snapshot.Roster{FranchiseID: "0002"}
	if holder != "" {
		rp := snapshot.RosterPlayer{ID: id, Salary: salary, YearsRemaining: &years, RosterStatus: domain.RosterActive}
		if holder == "0001" {
			mine.Players = append(mine.Players, rp)
		} else {
			other.Players = append(other.Players, rp)
		}
	}
	fresh := domain.Freshness{State: domain.FreshLive, FetchedAt: at.Format(time.RFC3339)}
	return Observation{LeagueID: "1", Rosters: snapshot.Sourced[[]snapshot.Roster]{
		Value: []snapshot.Roster{mine, other}, Provenance: snapshot.Provenance{Freshness: fresh},
	}}
}

func TestBidPredicateReadsTheRosters(t *testing.T) {
	later := bidAt().Add(time.Hour)
	for _, tc := range []struct {
		name   string
		obs    Observation
		closes string
		want   Event
		note   string
	}{
		{"matching contract on my roster", bidObservation(t, "0001", 500_000_000, 3, later), "", Match, "Landed: on your roster at $5,000,000 for 3 years"},
		{"different salary", bidObservation(t, "0001", 450_000_000, 3, later), "", Partial, "not the planned $5,000,000 for 3 years"},
		{"different years", bidObservation(t, "0001", 500_000_000, 2, later), "", Partial, "on your roster at $5,000,000 for 2 years"},
		{"on another roster", bidObservation(t, "0002", 500_000_000, 3, later), "", AwardedElsewhere, "he is on another franchise's roster"},
		{"still a free agent", bidObservation(t, "", 0, 0, later), "", NoChange, "No matching MFL change"},
		{"window closed, not mine", bidObservation(t, "", 0, 0, later), bidAt().Format(time.RFC3339), AwardedElsewhere, "the bidding window closed"},
		{"window not yet closed", bidObservation(t, "", 0, 0, later), later.Add(time.Hour).Format(time.RFC3339), NoChange, ""},
		{"window closed but he landed", bidObservation(t, "0001", 500_000_000, 3, later), bidAt().Format(time.RFC3339), Match, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			facts := bidFacts(t)
			facts.Closes = tc.closes
			e := draftedBid(t, passingBid())
			v := BidPredicate{Facts: facts}.Evaluate(e.Receipt().Spec, tc.obs)
			if v.Event != tc.want || !strings.Contains(v.Note, tc.note) {
				t.Errorf("verdict = %+v, want %s %q", v, tc.want, tc.note)
			}
		})
	}
}

func TestBidPredicateUnverifiableCases(t *testing.T) {
	facts := bidFacts(t)
	spec := draftedBid(t, passingBid()).Receipt().Spec
	later := bidAt().Add(time.Hour)
	failed := bidObservation(t, "0001", 500_000_000, 3, later)
	failed.Rosters.Provenance.Freshness.State = domain.FreshFail
	wrongLeague := bidObservation(t, "0001", 500_000_000, 3, later)
	wrongLeague.LeagueID = "other"
	noRoster := bidObservation(t, "", 0, 0, later)
	noRoster.Rosters.Value = noRoster.Rosters.Value[1:]
	noTerm := bidObservation(t, "0001", 500_000_000, 3, later)
	noTerm.Rosters.Value[0].Players[0].YearsRemaining = nil
	for name, obs := range map[string]Observation{"failed feed": failed, "wrong league": wrongLeague,
		"my roster missing": noRoster, "no contract length": noTerm} {
		if v := (BidPredicate{Facts: facts}).Evaluate(spec, obs); v.Event != Partial {
			t.Errorf("%s: %+v, want Partial", name, v)
		}
	}
	if v := (BidPredicate{Facts: facts}).Evaluate(Spec{Intent: "roster.ir"}, failed); v.Event != Partial {
		t.Errorf("another intent: %+v", v)
	}
}

func TestBidAwardedOffAppIsTerminalAndNotAFailure(t *testing.T) {
	e := draftedBid(t, passingBid())
	handed, err := e.HandOffManual(bidAt().Add(time.Minute), "copied")
	if err != nil {
		t.Fatal(err)
	}
	obs := bidObservation(t, "0002", 500_000_000, 3, bidAt().Add(time.Hour))
	gone, err := handed.Observe(bidAt().Add(2*time.Hour), obs, BidPredicate{Facts: bidFacts(t)})
	if err != nil || gone.State() != AwardedOffApp {
		t.Fatal(gone.Receipt(), err)
	}
	for _, event := range []Event{Match, Partial, NoChange, Contradiction, Invalidate, Rebase, HandOff, AwaitBid} {
		if _, err := gone.move(bidAt().Add(3*time.Hour), event, "x"); err == nil {
			t.Errorf("%s left the terminal state", event)
		}
	}
	if gone.State() == Failed {
		t.Error("awarded off-app is not a failure")
	}
}

func TestBidMismatchCanStillLandWhenTheContractMatches(t *testing.T) {
	handed, err := draftedBid(t, passingBid()).HandOffManual(bidAt().Add(time.Minute), "copied")
	if err != nil {
		t.Fatal(err)
	}
	predicate := BidPredicate{Facts: bidFacts(t)}
	off, err := handed.Observe(bidAt().Add(2*time.Hour), bidObservation(t, "0001", 450_000_000, 3, bidAt().Add(time.Hour)), predicate)
	if err != nil || off.State() != NotVerified {
		t.Fatal(off.Receipt(), err)
	}
	fixed, err := off.Observe(bidAt().Add(4*time.Hour), bidObservation(t, "0001", 500_000_000, 3, bidAt().Add(3*time.Hour)), predicate)
	if err != nil || fixed.State() != Landed {
		t.Fatal(fixed.Receipt(), err)
	}
}

func TestWithdrawBidEndsAnUnpostedPlanAndRefusesAPostedOne(t *testing.T) {
	ready := draftedBid(t, passingBid())
	gone, err := ready.WithdrawBid(bidAt().Add(time.Minute), "not wanted")
	if err != nil || gone.State() != Withdrawn || gone.Receipt().Audit[1].Event != Withdraw {
		t.Fatal(gone.Receipt(), err)
	}
	handed, err := ready.HandOffManual(bidAt().Add(time.Minute), "copied")
	if err != nil {
		t.Fatal(err)
	}
	copied, err := handed.WithdrawBid(bidAt().Add(2*time.Minute), "did not post")
	if err != nil || copied.State() != Withdrawn {
		t.Fatal(copied.Receipt(), err)
	}
	for _, event := range []Event{Match, Partial, NoChange, Contradiction, Invalidate, Rebase, HandOff, AwaitBid,
		AwardedElsewhere, Withdraw} {
		if _, err := copied.move(bidAt().Add(3*time.Minute), event, "x"); err == nil {
			t.Errorf("%s left Withdrawn", event)
		}
	}
	posted, err := handed.Await(bidAt().Add(2*time.Minute), AwaitBid)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := posted.WithdrawBid(bidAt().Add(3*time.Minute), "x"); err == nil {
		t.Error("a posted bid was withdrawn")
	}
	ir := Envelope{id: "ir", spec: Spec{Intent: "roster.ir"}, state: Ready, created: bidAt()}
	if _, err := ir.WithdrawBid(bidAt().Add(time.Minute), "x"); err == nil {
		t.Error("a roster move was withdrawn as a bid")
	}
}
