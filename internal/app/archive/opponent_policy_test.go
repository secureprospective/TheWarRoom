package archive

import (
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
)

func TestOpponentPrefersCurrentLineupBeforeOptionalFallback(t *testing.T) {
	lineups := leaguefeed.Lineups{Week: 5, Matchups: []leaguefeed.Matchup{{Home: "0001", Away: "0002"}, {Home: "0001", Away: "0003"}}}
	fallback := func(int, string) string { t.Fatal("fallback despite current matchup"); return "" }
	if got := Opponent(5, "0001", lineups, fallback); got != "0002" {
		t.Fatal(got)
	}
	if got := Opponent(5, "0002", lineups, fallback); got != "0001" {
		t.Fatal(got)
	}
	for _, week := range []int{4, 5} {
		got := Opponent(week, "0004", lineups, func(w int, team string) string {
			if w != week || team != "0004" {
				t.Fatal(w, team)
			}
			return "0005"
		})
		if got != "0005" {
			t.Fatal(got)
		}
	}
	if got := Opponent(4, "0001", lineups, nil); got != "" {
		t.Fatal(got)
	}
	if got := Opponent(5, "missing", lineups, nil); got != "" {
		t.Fatal(got)
	}
}
