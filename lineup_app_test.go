package main

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/league"
	"github.com/secureprospective/TheWarRoom/internal/normalize"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
)

type lineupRulesSource struct{ cfg league.RawConfig }

func (s lineupRulesSource) Fetch(context.Context) (league.RawConfig, error) { return s.cfg, nil }

func lineupTestRules(t *testing.T, a *App, bad bool) {
	t.Helper()
	body, err := os.ReadFile("internal/lineup/testdata/league.json")
	if err != nil {
		t.Fatal(err)
	}
	var raw struct {
		League struct {
			Starters struct {
				Count     string                         `json:"count"`
				IOP       string                         `json:"iop_starters"`
				IDP       string                         `json:"idp_starters"`
				Positions []struct{ Name, Limit string } `json:"position"`
			} `json:"starters"`
		} `json:"league"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		t.Fatal(err)
	}
	s := raw.League.Starters
	cfg := league.RawConfig{Source: "test:real league", Starters: league.Starters{
		Count: s.Count, IOPStarters: s.IOP, IDPStarters: s.IDP,
	}}
	for _, p := range s.Positions {
		cfg.Starters.Positions = append(cfg.Starters.Positions, league.PositionLimit{Name: p.Name, Limit: p.Limit})
	}
	if bad {
		cfg.Starters.Count = "unreadable"
	}
	if err := a.rulebook.Initialize(a.ctx, lineupRulesSource{cfg: cfg}); err != nil {
		t.Fatal(err)
	}
}

func lineupTestSnapshot(t *testing.T, a *App) []string {
	t.Helper()
	// Real roster export joined to the archived 2026 directory subset.
	body, err := os.ReadFile("internal/lineup/testdata/directory-0025.json")
	if err != nil {
		t.Fatal(err)
	}
	var players []struct{ ID, Name, Position string }
	if err := json.Unmarshal(body, &players); err != nil {
		t.Fatal(err)
	}
	snap := snapshot.Snapshot{}
	snap.Franchises.Value = []snapshot.Franchise{{ID: "0025"}}
	roster := snapshot.Roster{FranchiseID: "0025"}
	for _, p := range players {
		pid, err := playerid.New(p.ID)
		if err != nil {
			t.Fatal(err)
		}
		pos, _ := normalize.PositionFromMFL(p.Position)
		snap.Players.Value = append(snap.Players.Value, snapshot.Player{ID: pid, Position: pos, Name: p.Name})
	}
	body, err = os.ReadFile("internal/lineup/testdata/roster-0025.json")
	if err != nil {
		t.Fatal(err)
	}
	var raw struct {
		Players []struct{ ID, Status string } `json:"player"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		t.Fatal(err)
	}
	for _, p := range raw.Players {
		pid, err := playerid.New(p.ID)
		if err != nil {
			t.Fatal(err)
		}
		roster.Players = append(roster.Players, snapshot.RosterPlayer{
			ID: pid, RosterStatus: domain.RosterStatus(p.Status),
		})
	}

	ids := []string{
		"16150", "13133", "15754", "16195", "15761", "16617", "15798", "16641", "16846",
		"16694", "16264", "16230", "16303", "16734", "13813", "14892", "15836", "16267",
		"16460", "13322", "15850",
	}

	snap.Rosters.Value = []snapshot.Roster{roster}
	a.targetSnapshot, a.hasTargetSnapshot = snap, true
	return ids
}

func TestTargetLineupHeldReading(t *testing.T) {
	a, transport, _, _ := seasonTestApp(t)
	lineupTestRules(t, a, false)
	ids := lineupTestSnapshot(t, a)
	r, err := a.TargetLineup("0025")
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Starters) != 0 || len(r.Bench) != 38 || !r.Check.Legal {
		t.Fatalf("before refresh: %+v", r)
	}
	if r.Provenance.Freshness.State != FreshFail {
		t.Fatal(r.Provenance)
	}
	if len(transport.requests) != 0 {
		t.Fatal("binding requested network")
	}
	a.week.Number = 5
	a.refreshSeasonLineups(a.ctx)
	transport.requests = nil
	// A held refresh lock must not block this read.
	a.refreshMu.Lock()
	r, err = a.TargetLineup("0025")
	a.refreshMu.Unlock()
	if err != nil {
		t.Fatal(err)
	}
	got := []string{}
	for _, p := range r.Starters {
		got = append(got, p.ID)
	}
	if !reflect.DeepEqual(got, ids) || r.Week != 5 || r.StarterCount != 21 || !r.Check.Full || !r.Check.Legal {
		t.Fatalf("%+v", r)
	}
	want := []string{
		"15252", "17041", "16387", "17685", "17274", "14315", "16428", "15941",
		"15357", "17150", "16923", "16439", "14168", "16793", "16255", "15904", "16880",
	}
	bench := []string{}
	for _, p := range r.Bench {
		bench = append(bench, p.ID)
	}
	if !reflect.DeepEqual(bench, want) {
		t.Fatalf("bench %+v", r.Bench)
	}

	if len(transport.requests) != 0 {
		t.Fatal("binding requested network")
	}
	body, err := json.Marshal(r)
	if err != nil || strings.Contains(string(body), ":null") {
		t.Fatalf("%s %v", body, err)
	}
	if _, err := a.TargetLineup("nope"); err == nil {
		t.Fatal("unknown franchise accepted")
	}
	a.hasTargetSnapshot = false
	if _, err := a.TargetLineup("0025"); err == nil {
		t.Fatal("unloaded snapshot accepted")
	}
}

func TestTargetLineupBadRulesAndAbsentFeed(t *testing.T) {
	a, transport, _, _ := seasonTestApp(t)
	lineupTestRules(t, a, true)
	lineupTestSnapshot(t, a)
	r, err := a.TargetLineup("0025")
	if err != nil {
		t.Fatal(err)
	}
	if r.Check.Legal || len(r.Check.Problems) != 1 || r.StarterCount != 0 ||
		r.RulesSource.Freshness.State != FreshFail {
		t.Fatalf("bad rules: %+v", r)
	}
	a.week.Number = 5
	a.refreshSeasonLineups(a.ctx)
	a.seasonLineups.value.Franchises = nil
	transport.requests = nil
	r, err = a.TargetLineup("0025")
	if err != nil || len(r.Starters) != 0 || len(r.Bench) != 38 || r.Provenance.Freshness.State != FreshFail {
		t.Fatalf("absent: %+v %v", r, err)
	}
	if note := r.Provenance.Freshness.Note; note != "no saved lineup for franchise 0025" {
		t.Fatalf("absent note %q", note)
	}
	if len(transport.requests) != 0 {
		t.Fatal("binding requested network")
	}
}

func TestTargetLineupStaleAndUnknownPosition(t *testing.T) {
	a, transport, _, _ := seasonTestApp(t)
	lineupTestRules(t, a, false)
	lineupTestSnapshot(t, a)
	a.week.Number = 5
	a.refreshSeasonLineups(a.ctx)
	transport.failure = "liveScoring"
	a.refreshSeasonLineups(a.ctx)
	transport.requests = nil
	// The saved starter remains visible even when the directory lacks its facts.
	players := a.targetSnapshot.Players.Value
	a.targetSnapshot.Players.Value = slices.DeleteFunc(players, func(p snapshot.Player) bool {
		return p.ID.String() == "16150"
	})
	r, err := a.TargetLineup("0025")
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Starters) != 21 || r.Provenance.Freshness.State != FreshStale {
		t.Fatalf("lost stale starters: %+v", r)
	}
	unknown := false
	for _, p := range r.Check.Problems {
		if p.Kind == "unknown" {
			unknown = true
		}
	}
	if !unknown || r.Check.Legal {
		t.Fatalf("unknown position ignored: %+v", r.Check)
	}
	if len(transport.requests) != 0 {
		t.Fatal("binding requested network")
	}
}

func TestTargetLineupNotReady(t *testing.T) {
	a := NewApp()
	a.startupErr = errors.New("synthetic startup failure")
	close(a.started)
	if _, err := a.TargetLineup("0025"); err == nil {
		t.Fatal("unready app accepted")
	}
}
