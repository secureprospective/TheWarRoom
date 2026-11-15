package envelope

import (
	"reflect"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/engine/eligibility"
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
	rebased, err := e.Rebase(at.Add(time.Hour), "explicit revalidation required")
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

func mustTestID(t *testing.T) playerid.PlayerID {
	t.Helper()
	id, err := playerid.New("0531")
	if err != nil {
		t.Fatal(err)
	}
	return id
}

// eligibilitySnapshot installs held verdicts for one player, the shape TargetSnapshot builds.
func eligibilitySnapshot(snap snapshot.Snapshot, id playerid.PlayerID, v eligibility.PlayerVerdicts) snapshot.Snapshot {
	snap.Eligibility = snapshot.Eligibility{
		Players: map[string]eligibility.PlayerVerdicts{id.String(): v},
		Feeds: map[string]snapshot.Provenance{
			eligibility.SourceRosters:  {Freshness: domain.Freshness{State: domain.FreshLive}},
			eligibility.SourceInjuries: {Freshness: domain.Freshness{State: domain.FreshLive}},
			eligibility.SourcePlayers:  {Freshness: domain.Freshness{State: domain.FreshLive}},
		},
	}
	return snap
}

func TestTaxiCheckConsultsHeldEligibility(t *testing.T) {
	id := mustTestID(t)
	for _, tc := range []struct {
		name    string
		status  domain.RosterStatus
		verdict eligibility.Verdict
		reason  string
		source  string
		want    State
		note    string
	}{
		{"demote blocked by experience", domain.RosterActive, eligibility.Ineligible,
			"3 or more years of NFL experience (drafted 2017)", eligibility.SourcePlayers, Blocked,
			"3 or more years of NFL experience (drafted 2017)"},
		{"demote blocked by capacity", domain.RosterActive, eligibility.Ineligible, "taxi full (8/8)",
			eligibility.SourceRosters, Blocked, "taxi full (8/8)"},
		{"demote eligible", domain.RosterActive, eligibility.Eligible, "", eligibility.SourcePlayers,
			Ready, "eligibility checked (MFL players)"},
		{"promote eligible", domain.RosterTaxi, eligibility.Eligible, "", eligibility.SourceRosters,
			Ready, "eligibility checked (MFL rosters)"},
		{"promote not checked", domain.RosterTaxi, eligibility.Unknown, "no draft year held",
			eligibility.SourcePlayers, Ready, "Not checked: no draft year held; MFL will decide"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e, at, snap := testEnvelope(t)
			e.spec.Intent = "roster.taxi"
			e.spec.Expected.RosterStatus = domain.RosterTaxi
			if tc.status == domain.RosterTaxi {
				e.spec.Expected.RosterStatus = domain.RosterActive
			}
			snap.Rosters.Value[0].Players[0].RosterStatus = tc.status
			snap = eligibilitySnapshot(snap, id, eligibility.PlayerVerdicts{
				Taxi: eligibility.Decision{Direction: eligibility.ToTaxi, Verdict: tc.verdict,
					Reason: tc.reason, Source: tc.source},
			})
			got, err := e.Check(at, snap, TaxiCheck{})
			if err != nil || got.State() != tc.want || got.Receipt().Audit[0].Note != tc.note {
				t.Fatalf("%+v %v", got.Receipt(), err)
			}
		})
	}
}
