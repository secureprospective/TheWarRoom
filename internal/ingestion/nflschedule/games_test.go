package nflschedule

import "testing"

func TestSeasonGames(t *testing.T) {
	weeks := []RawWeek{
		{Week: "1", Matchups: []RawMatchup{{Teams: [2]string{"A", "B"}}, {Teams: [2]string{"C", "D"}}}},
		{Week: "2", Matchups: []RawMatchup{{Teams: [2]string{"A", "C"}}}},
		{Week: "3", Matchups: []RawMatchup{{Teams: [2]string{"B", "D"}}}},
	}
	if games, ok := SeasonGames(weeks); !ok || games != 2 {
		t.Fatalf("got %d, %v; want 2, true", games, ok)
	}
	if _, ok := SeasonGames(weeks[:2]); ok {
		t.Fatal("a partial season passed for a complete one")
	}
	uneven := append(append([]RawWeek{}, weeks...), RawWeek{Week: "4", Matchups: []RawMatchup{{Teams: [2]string{"A", "B"}}}})
	if _, ok := SeasonGames(uneven); ok {
		t.Fatal("a season with unequal games passed")
	}
}
