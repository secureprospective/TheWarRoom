package trades

import (
	"reflect"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/leagueclock"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

func TestDeadlineBoundary(t *testing.T) {
	t.Parallel()
	at := time.Date(2026, 11, 1, 12, 0, 0, 0, time.UTC)
	windows := []leagueclock.Window{{Kind: leagueclock.TradeDeadline, Status: leagueclock.WindowDated, At: &at, Reason: "commissioner override"}}
	for _, tc := range []struct {
		offset time.Duration
		want   []string
	}{
		{-time.Nanosecond, []string{}},
		{0, []string{"Trade deadline passed Nov 1, 2026, 12:00 PM · commissioner override"}},
		{time.Nanosecond, []string{"Trade deadline passed Nov 1, 2026, 12:00 PM · commissioner override"}},
	} {
		if got := DeadlineReasons(at.Add(tc.offset), windows, time.UTC); !reflect.DeepEqual(got, tc.want) {
			t.Fatalf("offset %s: %v", tc.offset, got)
		}
	}
	if got := DeadlineReasons(at, nil, time.UTC); !reflect.DeepEqual(got, []string{"Trade deadline cannot be checked: league window unavailable"}) {
		t.Fatal(got)
	}
}

func TestLegsPreserveBothSidesAndHeldOwnership(t *testing.T) {
	t.Parallel()
	request := Input{Mine: "0001", Other: "0002", Give: []string{"0099", "FP_0001_2027_1"}, Get: []string{"12345"}}
	teams := []Team{{ID: "0001", Assets: []Asset{{Token: "0099"}, {Token: "FP_0001_2027_1"}}}, {ID: "0002", Assets: []Asset{{Token: "12345"}}}} //nolint:gosec // fixture MFL asset tokens, not credentials
	owners := map[string]state.PlayerState{"0099": {FranchiseID: "0001"}, "12345": {FranchiseID: "0002"}}
	player := func(id string) (state.PlayerState, bool) { p, ok := owners[id]; return p, ok }
	moves, blocks := Legs(request, teams, player)
	if len(blocks) != 0 || len(moves) != 2 || moves[0].MFLID != "0099" || moves[0].ToFranchiseID != "0002" || moves[1].ToFranchiseID != "0001" {
		t.Fatal(moves, blocks)
	}
	delete(owners, "12345")
	moves, blocks = Legs(request, teams, player)
	if len(moves) != 1 || !reflect.DeepEqual(blocks, []string{"Selected player was removed in what-if; Undo first"}) {
		t.Fatal(moves, blocks)
	}
	request.Give = []string{"12345"}
	_, blocks = Legs(request, teams, player)
	if len(blocks) != 2 || blocks[0] != "Selected asset is no longer held by that franchise" {
		t.Fatal(blocks)
	}
}

func TestCentsCorrectionFollowsOwnership(t *testing.T) {
	t.Parallel()
	p := state.MirrorPlayer{MFLID: "0099", FranchiseID: "0001", Salary: 123_000_037, RosterStatus: domain.RosterActive}
	for _, tc := range []struct {
		mine, after string
		want        domain.Money
	}{
		{"0001", "0001", 0}, {"0001", "0002", -37}, {"0002", "0002", 37}, {"0002", "0001", 0},
	} {
		if got := CentsCorrection(p, tc.after, state.WhatIfSession{Mine: tc.mine}); got != tc.want {
			t.Fatalf("%s -> %s: %d", tc.mine, tc.after, got)
		}
	}
}
