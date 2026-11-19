package commissioner

import (
	"encoding/json"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/engine/competition"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

func TestStandingsReadCarriesTheLatestDecidedStreak(t *testing.T) {
	directory := NativeDirectory{Franchises: []NativeDirectorySeat{
		{PeopleSeat: PeopleSeat{Franchise: "team.a", Name: "A"}},
		{PeopleSeat: PeopleSeat{Franchise: "team.b", Name: "B"}},
		{PeopleSeat: PeopleSeat{Franchise: "team.c", Name: "C"}},
	}}
	counted := seasonGames{decided: 5, games: []competition.RankGame{
		{Home: "team.a", Away: "team.b", HomePoints: 3, AwayPoints: 1},
		{Home: "team.a", Away: "team.c", HomePoints: 3, AwayPoints: 2},
		{Home: "team.c", Away: "team.b", HomePoints: 3, AwayPoints: 2},
		{Home: "team.c", Away: "team.a", HomePoints: 2, AwayPoints: 1},
		{Home: "team.c", Away: "team.b", HomePoints: 3, AwayPoints: 1},
	}}
	reading, err := (NativeStandings{}).rank(leaguerules.LegacyNFL(), directory, counted,
		StandingsAdjustments{}, conferenceLayout{})
	if err != nil {
		t.Fatal(err)
	}
	body, err := json.Marshal(reading)
	if err != nil {
		t.Fatal(err)
	}
	var response struct {
		Computed []struct {
			Franchise string `json:"franchise"`
			Streak    string `json:"streak"`
		} `json:"computed"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		t.Fatal(err)
	}
	got := map[string]string{}
	for _, row := range response.Computed {
		got[row.Franchise] = row.Streak
	}
	if got["team.c"] != "W3" || got["team.b"] != "L3" || got["team.a"] != "L1" {
		t.Fatalf("standings streaks = %v; want team.c W3, team.b L3, team.a L1", got)
	}

	withoutGames, err := (NativeStandings{}).rank(leaguerules.LegacyNFL(), directory, seasonGames{},
		StandingsAdjustments{}, conferenceLayout{})
	if err != nil {
		t.Fatal(err)
	}
	body, err = json.Marshal(withoutGames)
	if err != nil {
		t.Fatal(err)
	}
	if containsStreakField(body) {
		t.Fatalf("standings with no decided games invented a streak: %s", body)
	}
}

func TestNativeAssetsReadReportsRuleBasedOpenBenchSlots(t *testing.T) {
	b := seededLedger(t)
	assets, err := NativeAssetsReading("native.test", b.records)
	if err != nil {
		t.Fatal(err)
	}
	if assets.OpenBenchSlots["franchise.02"] != 79 || assets.OpenBenchSlots["franchise.01"] != 80 {
		t.Fatalf("open bench slots = %v; want 79 for the one-player active roster, 80 for the empty roster",
			assets.OpenBenchSlots)
	}
	body, err := json.Marshal(assets)
	if err != nil {
		t.Fatal(err)
	}
	var response struct {
		OpenBenchSlots map[string]int `json:"openBenchSlots"`
	}
	if err := json.Unmarshal(body, &response); err != nil || response.OpenBenchSlots["franchise.02"] != 79 {
		t.Fatalf("native.assets.read openBenchSlots = %v, %v", response.OpenBenchSlots, err)
	}

	unruled := nativeLedger(t, 2, nil)
	assets, err = NativeAssetsReading("native.test", unruled.records)
	if err != nil {
		t.Fatal(err)
	}
	if len(assets.OpenBenchSlots) != 0 {
		t.Fatalf("unknown roster ceiling invented open slots: %v", assets.OpenBenchSlots)
	}
}

func containsStreakField(body []byte) bool {
	var response map[string]json.RawMessage
	if err := json.Unmarshal(body, &response); err != nil {
		return true
	}
	var rows []map[string]json.RawMessage
	if err := json.Unmarshal(response["computed"], &rows); err != nil {
		return true
	}
	for _, row := range rows {
		if _, ok := row["streak"]; ok {
			return true
		}
	}
	return false
}
