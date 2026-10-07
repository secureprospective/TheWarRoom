package envelope

import (
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
)

func testEnvelope(t *testing.T) (Envelope, time.Time, snapshot.Snapshot) {
	t.Helper()
	at := time.Date(2026, 10, 6, 12, 0, 0, 0, time.UTC)
	id, err := playerid.New("0531")
	if err != nil {
		t.Fatal(err)
	}
	e, err := New(at, Spec{
		Intent: "roster.ir", LeagueID: "1", FranchiseID: "0001",
		Subject:  Subject{Players: []playerid.PlayerID{id}},
		Expected: ExpectedChange{Player: id, RosterStatus: domain.RosterIR},
		Gravity:  G2, Undo: Reversible, Target: Target{Kind: Mapped, URL: "https://fixture.invalid/options?O=18"},
	}, func() string { return "id" })
	if err != nil {
		t.Fatal(err)
	}
	roster := snapshot.Roster{FranchiseID: "0001",
		Players: []snapshot.RosterPlayer{{ID: id, RosterStatus: domain.RosterActive}}}
	fresh := domain.Freshness{State: domain.FreshLive, FetchedAt: at.Format(time.RFC3339)}
	snap := snapshot.Snapshot{Rosters: snapshot.Sourced[[]snapshot.Roster]{
		Value: []snapshot.Roster{roster}, Provenance: snapshot.Provenance{Freshness: fresh},
	}}
	return e, at, snap
}

// observeAt stamps the snapshot's roster fetch time, which is the observation time.
func observeAt(snap snapshot.Snapshot, at time.Time) Observation {
	snap.Rosters.Provenance.Freshness.FetchedAt = at.Format(time.RFC3339)
	return Observation{LeagueID: "1", Rosters: snap.Rosters}
}

// Written out by hand, independently of the table, so the test is not a copy of the code.
func wantTransitions() map[State]map[Event]State {
	return map[State]map[Event]State{
		Draft:         {ChecksPass: Ready, ChecksBlock: Blocked},
		Blocked:       {ChecksPass: Ready, ChecksBlock: Blocked},
		Ready:         {HandOff: HandedOff, Invalidate: Stale},
		Stale:         {Rebase: Draft},
		Failed:        {Rebase: Draft},
		Landed:        {},
		HandedOff:     observed(NotYetDone, true),
		NotYetDone:    observed(NotYetDone, true),
		NotVerified:   observed(NotYetDone, true),
		DOTReview:     observed(DOTReview, false),
		BidPending:    observed(BidPending, false),
		WaiverPending: observed(WaiverPending, false),
	}
}

func observed(noChange State, canAwait bool) map[Event]State {
	m := map[Event]State{NoChange: noChange, Match: Landed, Partial: NotVerified,
		Contradiction: Failed, Invalidate: Stale}
	if canAwait {
		m[AwaitDOT], m[AwaitBid], m[AwaitWaiver] = DOTReview, BidPending, WaiverPending
	}
	return m
}

func TestExhaustiveTransitionTable(t *testing.T) {
	events := []Event{ChecksPass, ChecksBlock, HandOff, NoChange, Match, Partial, Contradiction,
		Invalidate, Rebase, AwaitDOT, AwaitBid, AwaitWaiver}
	want := wantTransitions()
	listed := 0
	for _, to := range want {
		listed += len(to)
	}
	if table := transitions(); listed != len(table) {
		t.Fatalf("table lists %d transitions, test expects %d", len(table), listed)
	}
	e, at, _ := testEnvelope(t)
	pairs := 0
	for s, legal := range want {
		for _, event := range events {
			pairs++
			t.Run(string(s)+"/"+string(event), func(t *testing.T) {
				original := e
				original.state = s
				got, err := original.move(at, event, "test")
				to, ok := legal[event]
				if !ok {
					var illegal ErrIllegalTransition
					if !errors.As(err, &illegal) || illegal.From != s || illegal.Event != event {
						t.Fatalf("expected typed error: %v", err)
					}
					return
				}
				entry := AuditEntry{At: at, From: s, Event: event, To: to, Note: "test"}
				if err != nil || got.State() != to || len(got.audit) != 1 || got.audit[0] != entry {
					t.Fatalf("transition: %+v %v", got, err)
				}
				if original.state != s || len(original.audit) != 0 {
					t.Fatal("mutated original")
				}
			})
		}
	}
	t.Logf("exhaustive (state,event) pairs: %d; listed transitions: %d", pairs, listed)
}

func TestIRChecksAndUnmapped(t *testing.T) {
	for _, tc := range []struct {
		name              string
		status            domain.RosterStatus
		missing, unmapped bool
		want              State
		note              string
	}{
		{"ready but eligibility unknown", domain.RosterActive, false, false, Ready,
			"not verified (league rules not captured)"},
		{"already IR", domain.RosterIR, false, false, Blocked, "player already on IR"},
		{"not owned", domain.RosterActive, true, false, Blocked, "player is not on this franchise's roster"},
		{"unmapped keeps the check's note", domain.RosterIR, false, true, Blocked,
			"player already on IR; MFL target not verified"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e, at, snap := testEnvelope(t)
			snap.Rosters.Value[0].Players[0].RosterStatus = tc.status
			if tc.missing {
				snap.Rosters.Value[0].Players = nil
			}
			if tc.unmapped {
				e.spec.Target = Target{Kind: Unmapped}
			}
			got, err := e.Check(at, snap, IRCheck{})
			if err != nil || got.State() != tc.want || got.audit[0].Note != tc.note {
				t.Fatalf("%+v %v", got, err)
			}
		})
	}
}

func TestUnmappedNeverLeavesDraftOrBlocked(t *testing.T) {
	e, at, _ := testEnvelope(t)
	e.spec.Target = Target{Kind: Unmapped}
	for _, s := range []State{Draft, Blocked} {
		e.state = s
		if _, err := e.move(at, ChecksPass, ""); err == nil {
			t.Fatal("unmapped escaped", s)
		}
		if got, err := e.move(at, ChecksBlock, ""); err != nil || got.State() != Blocked {
			t.Fatal("unmapped could not block", s, err)
		}
	}
}

func TestIRObservations(t *testing.T) {
	for _, tc := range []struct {
		name                          string
		status                        domain.RosterStatus
		missing, unavailable, foreign bool
		want                          State
	}{
		{name: "landed", status: domain.RosterIR, want: Landed},
		{name: "unchanged", status: domain.RosterActive, want: NotYetDone},
		{name: "missing status", want: NotVerified},
		{name: "gone", missing: true, want: Failed},
		{name: "unavailable", unavailable: true, want: NotVerified},
		{name: "wrong league", status: domain.RosterIR, foreign: true, want: NotVerified},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e, at, snap := testEnvelope(t)
			e.state = Ready
			e, err := e.HandOff(at)
			if err != nil {
				t.Fatal(err)
			}
			snap.Rosters.Value[0].Players[0].RosterStatus = tc.status
			if tc.missing {
				snap.Rosters.Value[0].Players = nil
			}
			if tc.unavailable {
				snap.Rosters.Provenance.Freshness.State = domain.FreshFail
			}
			obs := observeAt(snap, at.Add(time.Minute))
			if tc.foreign {
				obs.LeagueID = "2"
			}
			got, err := e.Observe(at.Add(time.Minute), obs, IRPredicate{})
			if tc.unavailable {
				if !errors.Is(err, ErrStaleObservation) || e.State() != HandedOff {
					t.Fatalf("failed source changed state: %+v %v", got, err)
				}
				return
			}
			if err != nil || got.State() != tc.want {
				t.Fatalf("%+v %v", got, err)
			}
		})
	}
}

func TestOldEvidenceChangesNothing(t *testing.T) {
	e, at, snap := testEnvelope(t)
	e.state = Ready
	e, err := e.HandOff(at)
	if err != nil {
		t.Fatal(err)
	}
	snap.Rosters.Value[0].Players[0].RosterStatus = domain.RosterIR
	for name, observedAt := range map[string]time.Time{"at hand-off": at, "from the future": at.Add(time.Hour)} {
		if _, err := e.Observe(at.Add(time.Minute), observeAt(snap, observedAt), IRPredicate{}); !errors.Is(
			err, ErrStaleObservation) {
			t.Fatal(name, err)
		}
	}
	if e.State() != HandedOff {
		t.Fatal(e.State())
	}
}

func TestDeadlineStaleAndValueIsolation(t *testing.T) {
	e, at, snap := testEnvelope(t)
	deadline := at
	e.spec.Deadline = &deadline
	e.state = Ready
	e, err := e.HandOff(at)
	if err != nil {
		t.Fatal(err)
	}
	for i, want := range []State{NotYetDone, Stale} {
		obsAt := at.Add(time.Duration(i+1) * time.Minute)
		e, err = e.Observe(obsAt, observeAt(snap, obsAt), IRPredicate{})
		if err != nil || e.State() != want {
			t.Fatalf("%+v %v", e, err)
		}
	}
	if _, err := e.HandOff(at.Add(time.Hour)); err == nil {
		t.Fatal("stale handed off")
	}
	rebased, err := e.Rebase(at.Add(time.Hour))
	if err != nil || rebased.State() != Draft {
		t.Fatal(rebased, err)
	}
	log := NewMemoryLog()
	if err := log.Append(e); err != nil {
		t.Fatal(err)
	}
	receipt := e.Receipt()
	receipt.Spec.Subject.Players[0] = playerid.PlayerID{}
	receipt.Audit[0].Note = "mutated"
	*receipt.Spec.Deadline = time.Time{}
	if reflect.DeepEqual(receipt, e.Receipt()) {
		t.Fatal("receipt aliases envelope")
	}
	listed := log.List(Filter{LeagueID: "1"})
	listed[0].Audit[0].Note = "mutated"
	if log.List(Filter{})[0].Audit[0].Note == "mutated" || len(log.List(Filter{LeagueID: "2"})) != 0 {
		t.Fatal("audit log isolation/filter")
	}
}

func TestConstructorAndZeroValueRejectInvalid(t *testing.T) {
	e, at, _ := testEnvelope(t)
	for name, mutate := range map[string]func(*Spec){
		"no intent":       func(s *Spec) { s.Intent = "" },
		"bad gravity":     func(s *Spec) { s.Gravity = "G9" },
		"bad undo":        func(s *Spec) { s.Undo = "maybe" },
		"no subject":      func(s *Spec) { s.Subject.Players = nil },
		"no status":       func(s *Spec) { s.Expected.RosterStatus = "" },
		"http target":     func(s *Spec) { s.Target.URL = "http://fixture.invalid" },
		"unmapped w/ URL": func(s *Spec) { s.Target = Target{Kind: Unmapped, URL: "https://fixture.invalid"} },
	} {
		s := e.Receipt().Spec
		mutate(&s)
		if _, err := New(at, s, func() string { return "id" }); err == nil {
			t.Fatal("invalid spec accepted:", name)
		}
	}
	if _, err := New(at, e.spec, func() string { return "" }); err == nil {
		t.Fatal("empty ID")
	}
	if _, err := (Envelope{}).HandOff(at); err == nil {
		t.Fatal("zero envelope")
	}
	if _, err := e.move(at.Add(-time.Second), ChecksPass, ""); err == nil {
		t.Fatal("time reversed")
	}
}

func TestPendingStagesFollowHandOff(t *testing.T) {
	e, at, snap := testEnvelope(t)
	deadline := at
	e.spec.Deadline = &deadline
	got, err := e.Check(at.Add(time.Second), snap, IRCheck{})
	if err != nil || got.State() != Blocked {
		t.Fatal(got, err)
	}
	e.state = Ready
	if _, err := e.HandOff(at.Add(time.Second)); err == nil {
		t.Fatal("handed off after deadline")
	}
	e.state = HandedOff
	for _, tc := range []struct {
		intent string
		event  Event
		want   State
	}{
		{"trade.propose", AwaitDOT, DOTReview},
		{"bid.submit", AwaitBid, BidPending},
		{"waiver.claim", AwaitWaiver, WaiverPending},
	} {
		e.spec.Intent = tc.intent
		pending, err := e.Await(at, tc.event)
		if err != nil || pending.State() != tc.want {
			t.Fatal(pending, err)
		}
		unchanged, err := pending.move(at, NoChange, "no resolution")
		if err != nil || unchanged.State() != tc.want {
			t.Fatal("pending resolution lost", unchanged, err)
		}
	}
	e.spec.Intent = "roster.ir"
	if _, err := e.Await(at, AwaitDOT); err == nil {
		t.Fatal("IR entered DOT")
	}
	e.state, e.spec.Intent = Landed, "trade.propose"
	if _, err := e.Await(at, AwaitDOT); err == nil {
		t.Fatal("landed is terminal")
	}
}
