package contracts

import (
	"context"
	"encoding/json"
	"os"
	"slices"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/league"
	mflwhatif "github.com/secureprospective/TheWarRoom/internal/mflref/whatif"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
)

// The hosted checks (internal/leagueservice mfl_checks_test.go) read these same files.

func TestDesktopLineupCheckMatchesTheHostedGolden(t *testing.T) {
	var g struct {
		Starters  league.Starters   `json:"starters"`
		Positions map[string]string `json:"positions"`
		Cases     []struct {
			Name  string   `json:"name"`
			IDs   []string `json:"ids"`
			Full  bool     `json:"full"`
			Legal bool     `json:"legal"`
			Kinds []string `json:"kinds"`
		} `json:"cases"`
	}
	body, err := os.ReadFile("../../../mflref/lineup/testdata/hosted_golden.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(body, &g); err != nil {
		t.Fatal(err)
	}
	api, host := planHost(t)
	if _, err := host.Rulebook.Sync(context.Background(), league.RawConfig{Starters: g.Starters}); err != nil {
		t.Logf("rulebook sync: %v", err)
	}
	snap := snapshot.Snapshot{}
	snap.Franchises.Value = []snapshot.Franchise{{ID: "0001"}}
	for id, pos := range g.Positions {
		pid, _ := playerid.New(id)
		snap.Players.Value = append(snap.Players.Value, snapshot.Player{ID: pid, Position: domain.Position(pos)})
	}
	host.snap, host.loaded = snap, true
	for _, c := range g.Cases {
		got, err := api.TargetCheckLineup("0001", c.IDs)
		if err != nil {
			t.Fatalf("%s: %v", c.Name, err)
		}
		kinds := []string{}
		for _, p := range got.Problems {
			kinds = append(kinds, p.Kind)
		}
		if got.Full != c.Full || got.Legal != c.Legal || (len(kinds)+len(c.Kinds) != 0 && !slices.Equal(kinds, c.Kinds)) {
			t.Fatalf("%s: desktop says full=%v legal=%v kinds=%v, golden says full=%v legal=%v kinds=%v",
				c.Name, got.Full, got.Legal, kinds, c.Full, c.Legal, c.Kinds)
		}
	}
}

func TestDesktopAcquisitionLimitsMatchTheHostedGolden(t *testing.T) {
	var g struct {
		Cap   int64 `json:"cap"`
		Cases []struct {
			Name    string   `json:"name"`
			After   int64    `json:"after"`
			Active  int      `json:"active"`
			Reasons []string `json:"reasons"`
		} `json:"cases"`
	}
	body, err := os.ReadFile("../../../nativecheck/testdata/acquire_golden.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(body, &g); err != nil {
		t.Fatal(err)
	}
	salaryCap := domain.Money(g.Cap)
	for _, c := range g.Cases {
		out := &AcquisitionView{Reasons: []string{}}
		acquisitionLimits(out, mflwhatif.WhatIfSession{Cap: &salaryCap}, 0, domain.Money(c.After), c.Active)
		if !slices.Equal(out.Reasons, c.Reasons) {
			t.Fatalf("%s: desktop reasons %v, golden %v", c.Name, out.Reasons, c.Reasons)
		}
	}
}
