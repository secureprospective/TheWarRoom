// Package mflseed translates held MFL inputs into a validated native starting snapshot.
package mflseed

import (
	"reflect"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/app/season"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/draftreports"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/league"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
	"github.com/secureprospective/TheWarRoom/internal/snapshot"
	"github.com/secureprospective/TheWarRoom/internal/store/state"
)

func TestHeldConferencesNameEachDivisionsConference(t *testing.T) {
	t.Parallel()
	config := league.RawConfig{
		Conferences: []league.Conference{{ID: "00", Name: "AFC"}, {ID: "01", Name: " "}},
		Divisions:   []league.Division{{ID: "00", Conference: "00"}, {ID: "01", Conference: "01"}, {ID: "02"}}}
	got := Conferences(config)
	want := map[string]string{"00": "AFC", "01": "01"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("a division carries its conference's name, or its id when unnamed, and none when it has no conference: %v", got)
	}
	if len(Conferences(league.RawConfig{})) != 0 {
		t.Fatal("a league without conferences held none")
	}
}

func mustID(t *testing.T, id string) playerid.PlayerID {
	t.Helper()
	pid, err := playerid.New(id)
	if err != nil {
		t.Fatal(err)
	}
	return pid
}

func TestTheStartingSnapshotNumbersFranchisesAndKeepsHeldAssets(t *testing.T) {
	t.Parallel()
	franchises, mapping := Franchises([]league.Franchise{{ID: "0002", Name: "Two", Abbrev: "TW", Division: "01"},
		{ID: "0001", Name: "One", Abbrev: "ON", Division: "00"}})
	if franchises[0].ID != "franchise.01" || franchises[0].HeldID != "0001" || franchises[1].Abbreviation != "TW" ||
		!reflect.DeepEqual(mapping, map[string]string{"0001": "franchise.01", "0002": "franchise.02"}) {
		t.Fatalf("franchises are numbered in held-id order: %+v %v", franchises, mapping)
	}
	seed := commissioner.RawNativeSnapshot{Season: 2026}
	Players(&seed, state.MirrorSnapshot{
		Players: []state.MirrorPlayer{{MFLID: "0099", FranchiseID: "0002", RosterStatus: domain.RosterActive,
			Salary: 5, ContractYear: 2028, ContractStatus: domain.CStatusUFA, ContractInfo: "x"},
			{MFLID: "0098", FranchiseID: "0001"}},
		Adjustments: []state.Adjustment{{ID: "c1", FranchiseID: "0001", Amount: 7, Description: "fine"}}},
		mapping, []snapshot.Player{{ID: mustID(t, "0099"), Name: "Player, Named"}})
	wantPlayers := []commissioner.NativePlayer{{Name: "Player, Named", ID: "0099", Franchise: "franchise.02",
		Status: domain.RosterActive, Salary: 5, ExpirationYear: 2028, ContractStatus: domain.CStatusUFA, ContractInfo: "x"},
		{ID: "0098", Franchise: "franchise.01"}}
	if !reflect.DeepEqual(seed.Players, wantPlayers) ||
		!reflect.DeepEqual(seed.Adjustments, []commissioner.NativeCharge{{ID: "c1", Franchise: "franchise.01", Amount: 7, Description: "fine"}}) {
		t.Fatalf("players keep their held contracts and directory names: %+v %+v", seed.Players, seed.Adjustments)
	}
	player := mustID(t, "5001")
	Picks(&seed, []draftreports.Pick{{Round: 1, Slot: 2, Franchise: "0002", Player: &player},
		{Round: 1, Slot: 3, Franchise: "0001", Skipped: true}},
		[]draftreports.FuturePick{{Year: 2027, Round: 2, Original: "0001", Owner: "0002"}}, mapping)
	wantPicks := []commissioner.NativePick{{Season: 2026, Round: 1, Slot: 2, Owner: "franchise.02", Player: "5001"},
		{Season: 2026, Round: 1, Slot: 3, Owner: "franchise.01", Skipped: true},
		{Season: 2027, Round: 2, Original: "franchise.01", Owner: "franchise.02"}}
	if !reflect.DeepEqual(seed.Picks, wantPicks) {
		t.Fatalf("made picks keep their slot and rookie, future picks their original owner: %+v", seed.Picks)
	}
}

func TestSeedCaptureStopsBeforeReadingUnusableHeldInputs(t *testing.T) {
	t.Parallel()
	if _, err := CaptureSeed(SeedSource{Available: func() bool { return false }}); err == nil {
		t.Fatal("missing stores accepted")
	}
	for _, tc := range []struct {
		season int
		stamp  string
	}{{2025, "bad"}, {2026, "bad"}} {
		source := SeedSource{Available: func() bool { return true }, Roster: func() (state.MirrorSnapshot, string) { return state.MirrorSnapshot{Season: tc.season}, tc.stamp }, Season: func() int { return 2026 }, Draft: func() (season.Feed[[]draftreports.Pick], season.Feed[[]draftreports.FuturePick]) {
			t.Fatal("unusable roster read draft")
			return season.Feed[[]draftreports.Pick]{}, season.Feed[[]draftreports.FuturePick]{}
		}}
		if _, err := CaptureSeed(source); err == nil {
			t.Fatal("mixed season or bad roster time accepted")
		}
	}
}

func TestHeldCaptureStillRefusesEmptyRosters(t *testing.T) {
	t.Parallel()
	at := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	source := SeedSource{
		Available: func() bool { return true },
		Roster: func() (state.MirrorSnapshot, string) {
			return state.MirrorSnapshot{Season: 2026}, at.Format(time.RFC3339Nano)
		},
		Season: func() int { return 2026 },
		Draft: func() (season.Feed[[]draftreports.Pick], season.Feed[[]draftreports.FuturePick]) {
			return season.Feed[[]draftreports.Pick]{FetchedAt: at}, season.Feed[[]draftreports.FuturePick]{FetchedAt: at}
		},
		Config: func() league.RawConfig {
			return league.RawConfig{Franchises: []league.Franchise{{ID: "0001", Name: "One"}, {ID: "0002", Name: "Two"}}}
		},
		Directory: func() []snapshot.Player { return nil },
	}
	_, err := CaptureSeed(source)
	if err == nil || err.Error() != "held starting snapshot: hold the Legacy NFL rosters, franchises and both draft reports before seeding" {
		t.Fatalf("empty held capture error changed: %v", err)
	}
}
