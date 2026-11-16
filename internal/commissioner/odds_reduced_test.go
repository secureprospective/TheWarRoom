package commissioner

import (
	"strings"
	"testing"
)

// With no value read on the host, each side's expected score is its points per game so far, and every line
// the reading fills says it is reduced and why; a franchise that has not played gives no reduced odds.
func TestReducedOddsComeFromResultsAlone(t *testing.T) {
	standings := NativeStandings{Computed: []StandingRow{
		{Franchise: "franchise.01", Wins: 2, PointsFor: 4000000},
		{Franchise: "franchise.02", Losses: 2, PointsFor: 3000000},
	}}
	expected, words := resultsOnly(standings, "no values here")
	if words != "" || expected["franchise.01"] != 200 || expected["franchise.02"] != 150 {
		t.Fatalf("expected %v, words %q", expected, words)
	}
	standings.Computed = append(standings.Computed, StandingRow{Franchise: "franchise.03"})
	if _, words := resultsOnly(standings, "no values here"); !strings.Contains(words, "need a game played") {
		t.Fatalf("a franchise with no game gave reduced odds: %q", words)
	}
	out := NativeOdds{PowerWords: "Lineup value blended with results after 2 games",
		GameWords: "Week 3: each side's expected score from its lineup and its points so far"}
	out.reduced("no values here")
	for _, line := range []string{out.PowerWords, out.GameWords} {
		if !strings.HasPrefix(line, "Reduced: no values here. ") || strings.Contains(line, "lineup") {
			t.Fatalf("reduced line: %q", line)
		}
	}
}
