package envelope

import (
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
)

func TestRestoreReplaysAndRejectsCorruption(t *testing.T) {
	original, at, snap := testEnvelope(t)
	ready, err := original.Check(at.Add(2*time.Second), snap, IRCheck{})
	if err != nil {
		t.Fatal(err)
	}
	handed, err := ready.HandOff(at.Add(3 * time.Second))
	if err != nil {
		t.Fatal(err)
	}
	waiting, err := handed.Observe(at.Add(time.Minute), observeAt(snap, at.Add(time.Minute)), IRPredicate{})
	if err != nil {
		t.Fatal(err)
	}
	snap.Rosters.Value[0].Players[0].RosterStatus = domain.RosterIR
	landed, err := waiting.Observe(at.Add(2*time.Minute), observeAt(snap, at.Add(2*time.Minute)), IRPredicate{})
	if err != nil {
		t.Fatal(err)
	}
	r := landed.Receipt()
	restored, err := Restore(landed.ID(), landed.Created(), r.Spec, r.Audit)
	if err != nil || !reflect.DeepEqual(restored, landed) {
		t.Fatalf("restore: %+v, %v", restored, err)
	}
	for name, corrupt := range map[string]func([]AuditEntry){
		"wrong From":    func(a []AuditEntry) { a[1].From = Draft },
		"wrong To":      func(a []AuditEntry) { a[1].To = Landed },
		"backwards":     func(a []AuditEntry) { a[1].At = at.Add(time.Second) },
		"unknown event": func(a []AuditEntry) { a[1].Event = "bogus" },
	} {
		t.Run(name, func(t *testing.T) {
			audit := append([]AuditEntry{}, r.Audit...)
			corrupt(audit)
			if _, err := Restore(landed.ID(), at, r.Spec, audit); err == nil ||
				!strings.Contains(err.Error(), "entry 1") {
				t.Fatalf("corruption: %v", err)
			}
		})
	}
	early := append([]AuditEntry{}, r.Audit...)
	early[0].At = at.Add(-time.Second)
	if _, err := Restore(landed.ID(), at, r.Spec, early); err == nil ||
		!strings.Contains(err.Error(), "entry 0") {
		t.Fatalf("before created: %v", err)
	}
	bad := r.Spec
	bad.Intent = ""
	if _, err := Restore(landed.ID(), at, bad, r.Audit); err == nil {
		t.Fatal("invalid spec restored")
	}
	r.Audit[0].Note = "mutated"
	if reflect.DeepEqual(r.Audit, restored.Receipt().Audit) {
		t.Fatal("restored audit aliases input")
	}
}

type feedPredicate struct{ sources []Source }

func (p feedPredicate) Sources() []Source                { return p.sources }
func (feedPredicate) Evaluate(Spec, Observation) Verdict { return Verdict{Event: Match} }

func TestObserveDeclaredSourcesOnly(t *testing.T) {
	e, at, snap := testEnvelope(t)
	e, err := e.Check(at, snap, IRCheck{})
	if err != nil {
		t.Fatal(err)
	}
	e, err = e.HandOff(at)
	if err != nil {
		t.Fatal(err)
	}
	now := at.Add(time.Minute)
	fresh := domain.Freshness{State: domain.FreshLive, FetchedAt: now.Format(time.RFC3339)}
	obs := Observation{LeagueID: "1", Rosters: snap.Rosters}
	obs.Rosters.Provenance.Freshness.State = domain.FreshFail
	obs.Lineups.Provenance.Freshness = fresh
	predicate := feedPredicate{sources: []Source{Lineups}}
	got, err := e.Observe(now, obs, predicate)
	if err != nil || got.State() != Landed {
		t.Fatalf("undeclared rosters consulted: %+v, %v", got, err)
	}
	for _, source := range []Source{Rosters, Transactions, Lineups, PendingTrades} {
		for name, stamp := range map[string]domain.Freshness{
			"fail":      {State: domain.FreshFail, FetchedAt: fresh.FetchedAt},
			"old":       {State: domain.FreshLive, FetchedAt: at.Add(-time.Second).Format(time.RFC3339)},
			"equal":     {State: domain.FreshLive, FetchedAt: at.Format(time.RFC3339)},
			"future":    {State: domain.FreshLive, FetchedAt: now.Add(time.Second).Format(time.RFC3339)},
			"malformed": {State: domain.FreshLive, FetchedAt: "bad"},
		} {
			t.Run(string(source)+"/"+name, func(t *testing.T) {
				observation := Observation{LeagueID: "1"}
				observation.Rosters.Provenance.Freshness = fresh
				observation.Transactions.Provenance.Freshness = fresh
				observation.Lineups.Provenance.Freshness = fresh
				observation.PendingTrades.Provenance.Freshness = fresh
				switch source {
				case Rosters:
					observation.Rosters.Provenance.Freshness = stamp
				case Transactions:
					observation.Transactions.Provenance.Freshness = stamp
				case Lineups:
					observation.Lineups.Provenance.Freshness = stamp
				case PendingTrades:
					observation.PendingTrades.Provenance.Freshness = stamp
				}
				before := e.Receipt()
				_, err := e.Observe(now, observation, feedPredicate{
					sources: []Source{Rosters, Transactions, Lineups, PendingTrades},
				})
				if !errors.Is(err, ErrStaleObservation) || !reflect.DeepEqual(before, e.Receipt()) {
					t.Fatalf("declared bad feed: %v", err)
				}
			})
		}
	}
	for _, sources := range [][]Source{nil, {"unknown"}} {
		if _, err := e.Observe(now, obs, feedPredicate{sources: sources}); !errors.Is(err, ErrStaleObservation) {
			t.Fatalf("unverifiable predicate: %v", err)
		}
	}
}
