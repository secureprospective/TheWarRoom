package envelope

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
)

func TestDraftTaxiDirectionsAndBlocks(t *testing.T) {
	for _, tc := range []struct {
		name   string
		status domain.RosterStatus
		want   domain.RosterStatus
		state  State
		note   string
	}{
		{"demote", domain.RosterActive, domain.RosterTaxi, Ready, "Not checked: eligibility not held; MFL will decide"},
		{"promote", domain.RosterTaxi, domain.RosterActive, Ready, "Not checked: eligibility not held; MFL will decide"},
		{"IR", domain.RosterIR, domain.RosterTaxi, Blocked, "player on IR"},
		{"unknown status", "", domain.RosterTaxi, Blocked, "active or taxi roster status required"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			original, at, snap := testEnvelope(t)
			snap.Rosters.Value[0].Players[0].RosterStatus = tc.status
			req := TaxiRequest{
				LeagueID: "1", FranchiseID: "0001", Player: original.Receipt().Spec.Expected.Player,
				Target: Target{Kind: Mapped, URL: "https://www47.myfantasyleague.com/2026/options?L=1&O=98"},
			}
			e, err := DraftTaxi(at, req, snap, func() string { return "taxi" })
			if err != nil {
				t.Fatal(err)
			}
			r := e.Receipt()
			if r.State != tc.state || r.Spec.Expected.RosterStatus != tc.want ||
				!strings.Contains(r.Audit[0].Note, tc.note) || r.Spec.Intent != "roster.taxi" ||
				r.Spec.Gravity != G2 || r.Spec.Undo != Reversible {
				t.Fatalf("taxi: %+v", r)
			}
			if r.Spec.Expected.Player != req.Player || !reflect.DeepEqual(r.Spec.Target, req.Target) {
				t.Fatalf("scope/target: %+v", r.Spec)
			}
		})
	}
}

func TestDraftTaxiUnavailable(t *testing.T) {
	for _, mode := range []string{"absent player", "failed roster", "unknown host"} {
		t.Run(mode, func(t *testing.T) {
			original, at, snap := testEnvelope(t)
			req := TaxiRequest{
				LeagueID: "1", FranchiseID: "0001", Player: original.Receipt().Spec.Expected.Player,
				Target: Target{Kind: Mapped, URL: "https://www47.myfantasyleague.com/2026/options?L=1&O=98"},
			}
			want := "player is not on this franchise's roster"
			switch mode {
			case "absent player":
				snap.Rosters.Value[0].Players = nil
			case "failed roster":
				snap.Rosters.Provenance.Freshness.State = domain.FreshFail
				want = "roster unavailable"
			case "unknown host":
				req.Target = Target{Kind: Unmapped}
				req.Blocks = []string{"MFL host not yet known"}
				want = "MFL host not yet known"
			}
			e, err := DraftTaxi(at, req, snap, func() string { return "blocked" })
			if err != nil || e.State() != Blocked || !strings.Contains(e.Receipt().Audit[0].Note, want) {
				t.Fatalf("blocked taxi: %+v %v", e.Receipt(), err)
			}
		})
	}
}

func TestRosterPredicateDirections(t *testing.T) {
	for _, move := range []struct {
		intent    string
		squad     domain.RosterStatus
		other     domain.RosterStatus
		predicate Predicate
	}{
		{"roster.ir", domain.RosterIR, domain.RosterTaxi, IRPredicate{}},
		{"roster.taxi", domain.RosterTaxi, domain.RosterIR, TaxiPredicate{}},
	} {
		for _, destination := range []domain.RosterStatus{domain.RosterActive, move.squad} {
			for _, tc := range []struct {
				mode string
				want Event
			}{
				{"match", Match}, {"baseline", NoChange}, {"dropped", Contradiction},
				{"other squad", Contradiction}, {"failed feed", Partial}, {"wrong league", Partial},
				{"missing franchise", Partial}, {"missing status", Partial}, {"other franchise", Contradiction},
			} {
				t.Run(move.intent+"/"+string(destination)+"/"+tc.mode, func(t *testing.T) {
					e, at, snap := testEnvelope(t)
					spec := e.Receipt().Spec
					spec.Intent = move.intent
					spec.Expected.RosterStatus = destination
					snap.Rosters.Value[0].Players[0].RosterStatus = destination
					obs := observeAt(snap, at.Add(time.Minute))
					switch tc.mode {
					case "baseline":
						baseline := move.squad
						if destination == move.squad {
							baseline = domain.RosterActive
						}
						obs.Rosters.Value[0].Players[0].RosterStatus = baseline
					case "dropped":
						obs.Rosters.Value[0].Players = nil
					case "other squad":
						obs.Rosters.Value[0].Players[0].RosterStatus = move.other
					case "failed feed":
						obs.Rosters.Provenance.Freshness.State = domain.FreshFail
					case "wrong league":
						obs.LeagueID = "other"
					case "missing franchise":
						obs.Rosters.Value = nil
					case "missing status":
						obs.Rosters.Value[0].Players[0].RosterStatus = ""
					case "other franchise":
						moved := obs.Rosters.Value[0]
						moved.FranchiseID = "0002"
						obs.Rosters.Value[0].Players = nil
						obs.Rosters.Value = append(obs.Rosters.Value, moved)
					}
					v := move.predicate.Evaluate(spec, obs)
					if v.Event != tc.want || v.Note == "" {
						t.Fatalf("verdict: %+v", v)
					}
					if !reflect.DeepEqual(move.predicate.Sources(), []Source{Rosters}) {
						t.Fatal("roster moves must watch rosters")
					}
				})
			}
		}
	}
}

func TestTaxiObservationMustFollowHandOff(t *testing.T) {
	original, at, snap := testEnvelope(t)
	e, err := DraftTaxi(at, TaxiRequest{
		LeagueID: "1", FranchiseID: "0001", Player: original.Receipt().Spec.Expected.Player,
		Target: Target{Kind: Mapped, URL: "https://www47.myfantasyleague.com/2026/options?L=1&O=98"},
	}, snap, func() string { return "fresh-taxi" })
	if err != nil {
		t.Fatal(err)
	}
	e, err = e.HandOff(at.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := e.Observe(at.Add(2*time.Minute), observeAt(snap, at), TaxiPredicate{}); err == nil {
		t.Fatal("pre-hand-off roster accepted")
	}
	next, err := e.Observe(at.Add(2*time.Minute), observeAt(snap, at.Add(2*time.Minute)), TaxiPredicate{})
	if err != nil || next.State() != NotYetDone {
		t.Fatalf("unchanged fresh roster: %+v %v", next.Receipt(), err)
	}
	snap.Rosters.Value[0].Players[0].RosterStatus = domain.RosterTaxi
	next, err = next.Observe(at.Add(3*time.Minute), observeAt(snap, at.Add(3*time.Minute)), TaxiPredicate{})
	if err != nil || next.State() != Landed {
		t.Fatalf("landing: %+v %v", next.Receipt(), err)
	}
}
