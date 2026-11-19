package mflsource

import (
	"reflect"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
)

func TestImportLineupsUsesFinishedAndCurrentStartersWithoutInventingAbsences(t *testing.T) {
	t.Parallel()
	played, err := playerid.New("0099")
	if err != nil {
		t.Fatal(err)
	}
	current, err := playerid.New("1000")
	if err != nil {
		t.Fatal(err)
	}
	results := []leaguefeed.WeekResult{
		{Week: 7, Others: []leaguefeed.Side{{Franchise: "0001", Scored: true,
			Players: []leaguefeed.WeekPlayer{{ID: played, Starter: true}}}}},
		{Week: 8, Others: []leaguefeed.Side{{Franchise: "0001", Scored: true,
			Players: []leaguefeed.WeekPlayer{{ID: played, Starter: true}}}}},
	}
	live := &leaguefeed.Lineups{Week: 8, Franchises: []leaguefeed.Lineup{
		{Franchise: "0001", Starters: []playerid.PlayerID{current}},
		{Franchise: "0002"},
	}}
	got, err := ImportLineups(results, live, 2026,
		map[string]string{"0001": "franchise.01", "0002": "franchise.02"},
		map[string]string{"0099": "0099", "1000": "1000"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].Week != 7 || got[0].Starters[0] != "0099" ||
		got[1].Week != 8 || got[1].Starters[0] != "1000" {
		t.Fatalf("imported lineup rows: %+v", got)
	}
	for _, lineup := range got {
		if lineup.Franchise == "franchise.02" {
			t.Fatalf("an absent MFL lineup became an event: %+v", lineup)
		}
	}
}

func TestPlayedStartingLineups(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name        string
		ids         []string
		scored, bad bool
		want        []string
	}{
		{"starters and empty slot", []string{"0099", "0000"}, true, false, []string{"0099"}},
		{"unknown player", []string{"9999"}, true, true, nil},
		{"not played", []string{"9999"}, false, false, nil},
		{"only empty slots", []string{"0000"}, true, false, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			side := leaguefeed.Side{Franchise: "0001", Scored: tc.scored}
			for _, raw := range tc.ids {
				id, err := playerid.New(raw)
				if err != nil {
					t.Fatal(err)
				}
				side.Players = append(side.Players, leaguefeed.WeekPlayer{ID: id, Starter: true})
			}
			bench, err := playerid.New("1000")
			if err != nil {
				t.Fatal(err)
			}
			side.Players = append(side.Players, leaguefeed.WeekPlayer{ID: bench, Starter: false, ShouldStart: true})
			got, err := Lineups([]leaguefeed.WeekResult{{Week: 1, Others: []leaguefeed.Side{side}}}, 2026, map[string]string{"0001": "franchise.01"}, map[string]string{"0099": "0099"})
			if (err != nil) != tc.bad {
				t.Fatalf("got %+v err %v", got, err)
			}
			if tc.bad {
				return
			}
			if !tc.scored || len(tc.want) == 0 {
				if len(got) != 0 {
					t.Fatal(got)
				}
				return
			}
			if len(got) != 1 || got[0].Season != 2026 || got[0].Franchise != "franchise.01" || !reflect.DeepEqual(got[0].Starters, tc.want) {
				t.Fatal(got)
			}
		})
	}
}
