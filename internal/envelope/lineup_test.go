package envelope

import (
	"encoding/json"
	"os"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/league"
	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
	"github.com/secureprospective/TheWarRoom/internal/lineup"
	"github.com/secureprospective/TheWarRoom/internal/normalize"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
)

func lineupFixture(t *testing.T) (LineupRequest, snapshot.Snapshot, LineupCheck) {
	t.Helper()
	// Decode only the archived facts these domain tests consume.
	body, err := os.ReadFile("../lineup/testdata/league.json")
	if err != nil {
		t.Fatal(err)
	}
	var raw struct {
		League struct {
			Starters struct {
				Count     string
				IOP       string                 `json:"iop_starters"`
				IDP       string                 `json:"idp_starters"`
				Positions []league.PositionLimit `json:"position"`
			}
		}
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		t.Fatal(err)
	}
	rs := raw.League.Starters
	rules, err := lineup.ParseRules(league.Starters{
		Count: rs.Count, IOPStarters: rs.IOP, IDPStarters: rs.IDP, Positions: rs.Positions,
	})
	if err != nil {
		t.Fatal(err)
	}
	body, err = os.ReadFile("../lineup/testdata/players-0025.json")
	if err != nil {
		t.Fatal(err)
	}
	var players []struct {
		ID       playerid.PlayerID
		Position string
	}
	if err := json.Unmarshal(body, &players); err != nil {
		t.Fatal(err)
	}
	req := LineupRequest{LeagueID: "14432", FranchiseID: "0025", Week: 5,
		Target: Target{Kind: Mapped, URL: "https://fixture.invalid/lineup"}}
	snap := snapshot.Snapshot{}
	roster := snapshot.Roster{FranchiseID: req.FranchiseID}
	for _, p := range players {
		pos, _ := normalize.PositionFromMFL(p.Position)
		req.Baseline = append(req.Baseline, p.ID)
		snap.Players.Value = append(snap.Players.Value, snapshot.Player{ID: p.ID, Position: pos})
		roster.Players = append(roster.Players, snapshot.RosterPlayer{ID: p.ID, RosterStatus: domain.RosterActive})
	}
	bench, err := playerid.New("16387")
	if err != nil {
		t.Fatal(err)
	}
	snap.Players.Value = append(snap.Players.Value, snapshot.Player{ID: bench, Position: domain.PosWR})
	roster.Players = append(roster.Players, snapshot.RosterPlayer{ID: bench, RosterStatus: domain.RosterActive})
	snap.Rosters.Value = []snapshot.Roster{roster}
	req.Starters = slices.Clone(req.Baseline)
	for i, p := range players {
		if p.Position == "WR" {
			req.Starters[i] = bench
			break
		}
	}
	return req, snap, LineupCheck{Rules: rules, BaselineKnown: true}
}

func TestLineupSpecValidationAndLegacyIR(t *testing.T) {
	req, snap, check := lineupFixture(t)
	at := time.Date(2026, 10, 7, 0, 0, 0, 0, time.UTC)
	e, err := DraftLineup(at, req, snap, check, func() string { return "lineup-test" })
	if err != nil {
		t.Fatal(err)
	}
	valid := e.Receipt().Spec
	ir, _, _ := testEnvelope(t)
	for _, tc := range []struct {
		name   string
		base   Spec
		change func(*Spec)
		valid  bool
	}{
		{"lineup", valid, func(*Spec) {}, true},
		{"IR", ir.spec, func(*Spec) {}, true},
		{"IR no status", ir.spec, func(s *Spec) { s.Expected.RosterStatus = "" }, false},
		{"IR no subject", ir.spec, func(s *Spec) { s.Subject.Players = nil }, false},
		{"lineup missing", valid, func(s *Spec) { s.Expected.Lineup = nil }, false},
		{"week", valid, func(s *Spec) { s.Expected.Lineup.Week = 0 }, false},
		{"empty", valid, func(s *Spec) { s.Expected.Lineup.Starters = nil }, false},
		{"duplicate", valid, func(s *Spec) {
			s.Expected.Lineup.Starters = append(s.Expected.Lineup.Starters, s.Expected.Lineup.Starters[0])
		}, false},
		{"subjects", valid, func(s *Spec) { s.Subject.Players = s.Subject.Players[:1] }, false},
		{"roster mixed", valid, func(s *Spec) { s.Expected.RosterStatus = domain.RosterIR }, false},
		{"unknown empty", valid, func(s *Spec) {
			s.Intent = "other"
			s.Expected = ExpectedChange{}
		}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			spec := cloneSpec(tc.base)
			tc.change(&spec)
			if err := validateSpec(spec); (err == nil) != tc.valid {
				t.Fatalf("validation: %v", err)
			}
		})
	}
	// Pre-phase IR JSON has no lineup field and must replay unchanged.
	legacy := `{"intent":"roster.ir","leagueId":"1","franchiseId":"0001",` +
		`"subject":{"players":["0531"],"picks":[]},"expected":{"player":"0531","rosterStatus":"IR"},` +
		`"gravity":"G2","undo":"reversible","target":{"kind":"unmapped"}}`
	var spec Spec
	if err := json.Unmarshal([]byte(legacy), &spec); err != nil {
		t.Fatal(err)
	}
	old, err := Restore("old-ir", at, spec, []AuditEntry{
		{At: at, From: Draft, Event: ChecksBlock, To: Blocked, Note: "original note"},
	})
	if err != nil || old.Receipt().Audit[0].Note != "original note" ||
		old.Receipt().Spec.Expected.Lineup != nil {
		t.Fatalf("legacy IR: %+v %v", old, err)
	}
}

func TestLineupChecksAndDraft(t *testing.T) {
	for _, name := range []string{"ready", "baseline", "unknown baseline", "IR", "taxi", "unowned",
		"failed roster", "unknown position", "over", "partial"} {
		t.Run(name, func(t *testing.T) {
			req, snap, check := lineupFixture(t)
			want, note := Ready, "legal"
			switch name {
			case "baseline":
				req.Starters = slices.Clone(req.Baseline)
				want, note = Blocked, "already your saved week 5"
			case "unknown baseline":
				check.BaselineKnown = false
				want, note = Blocked, "saved lineup is unknown"
			case "IR", "taxi":
				status := domain.RosterIR
				if name == "taxi" {
					status = domain.RosterTaxi
				}
				for i, p := range snap.Rosters.Value[0].Players {
					if p.ID == req.Starters[0] {
						snap.Rosters.Value[0].Players[i].RosterStatus = status
					}
				}
				want, note = Blocked, "ROSTER-status"
			case "unowned":
				snap.Rosters.Value = nil
				want, note = Blocked, "ROSTER-status"
			case "failed roster":
				snap.Rosters.Provenance.Freshness.State = domain.FreshFail
				want, note = Blocked, "roster unavailable"
			case "unknown position":
				snap.Players.Value = nil
				want, note = Blocked, "Position unknown"
			case "over":
				for _, id := range req.Baseline {
					if !slices.Contains(req.Starters, id) {
						req.Starters = append(req.Starters, id)
						break
					}
				}
				want, note = Blocked, "at most"
			case "partial":
				req.Starters = req.Starters[:1]
				note = "legal, partial;"
			}
			at := time.Date(2026, 10, 7, 0, 0, 0, 0, time.UTC)
			e, err := DraftLineup(at, req, snap, check, func() string { return "lineup" })
			if err != nil || e.State() != want || !strings.Contains(e.Receipt().Audit[0].Note, note) {
				t.Fatalf("draft: %+v %v", e.Receipt(), err)
			}
			r := e.Receipt()
			if name == "unknown baseline" && len(r.Spec.Expected.Lineup.Baseline) != 0 {
				t.Fatal("unknown baseline retained")
			}
			if name == "partial" && !strings.Contains(r.Audit[0].Note, "needs at least") {
				t.Fatal("partial omitted short problems")
			}
			r.Spec.Expected.Lineup.Starters[0] = playerid.PlayerID{}
			if e.Receipt().Spec.Expected.Lineup.Starters[0].IsZero() {
				t.Fatal("aliased starters")
			}
		})
	}
}

func TestLineupPredicateVerdicts(t *testing.T) {
	req, snap, check := lineupFixture(t)
	at := time.Date(2026, 10, 7, 0, 0, 0, 0, time.UTC)
	e, err := DraftLineup(at, req, snap, check, func() string { return "lineup" })
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"match", "baseline", "third", "feed", "league", "week", "absent"} {
		t.Run(name, func(t *testing.T) {
			obs := Observation{LeagueID: req.LeagueID}
			obs.Lineups.Value = leaguefeed.Lineups{Week: 5, Franchises: []leaguefeed.Lineup{{
				Franchise: req.FranchiseID, Starters: slices.Clone(req.Starters),
			}}}
			slices.Reverse(obs.Lineups.Value.Franchises[0].Starters)
			want, note := Match, "drafted starters"
			switch name {
			case "baseline":
				obs.Lineups.Value.Franchises[0].Starters = req.Baseline
				want, note = NoChange, "No matching"
			case "third":
				obs.Lineups.Value.Franchises[0].Starters = nil
				want, note = Partial, "different lineup (0 of 21 drafted starters)"
			case "feed":
				obs.Lineups.Provenance.Freshness.State = domain.FreshFail
				want, note = Partial, "feed failed"
			case "league":
				obs.LeagueID = "other"
				want, note = Partial, "wrong league"
			case "week":
				obs.Lineups.Value.Week = 4
				want, note = Partial, "week"
			case "absent":
				obs.Lineups.Value.Franchises = nil
				want, note = Partial, "franchise absent"
			}
			got := (LineupPredicate{}).Evaluate(e.Receipt().Spec, obs)
			if got.Event != want || !strings.Contains(got.Note, note) {
				t.Fatalf("verdict: %+v", got)
			}
			if want == Partial && !strings.HasPrefix(got.Note, "Not verified: ") {
				t.Fatal(got)
			}
		})
	}
	if !reflect.DeepEqual((LineupPredicate{}).Sources(), []Source{Lineups}) {
		t.Fatal("wrong sources")
	}
}
