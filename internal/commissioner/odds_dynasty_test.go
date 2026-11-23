package commissioner

import (
	"fmt"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

// A 32-team league on the default rules: every franchise gets a now rank and a dynasty rank from the one
// blend, and the dynasty order follows each franchise's dynasty value, not its on-field-now value.
func TestOddsRanksThirtyTwoTeamsNowAndDynasty(t *testing.T) {
	rules := leaguerules.LegacyNFLForTeams(32)
	standings := NativeStandings{}
	var players []OddsPlayer
	for i := 1; i <= 32; i++ {
		f := fmt.Sprintf("franchise.%02d", i)
		standings.Computed = append(standings.Computed, StandingRow{Franchise: f, Name: f})
		// Franchise 1 is best now and worst in dynasty; franchise 32 the reverse.
		for _, pos := range []string{"QB", "RB", "WR", "TE", "DT", "DE", "LB", "CB", "S", "K"} {
			players = append(players, OddsPlayer{Franchise: f, Position: pos, Available: true, AgeKnown: true, Age: 26,
				NowPPG: float64(33 - i), DynastyPPG: float64(i)})
		}
	}
	out := NativeOdds{Rows: []OddsRow{}}
	for _, row := range standings.Computed {
		out.Rows = append(out.Rows, OddsRow{Franchise: row.Franchise})
	}
	if err := out.power(standings, rules.Lineup, OddsValues{Players: players}, map[string]float64{}); err != nil {
		t.Fatal(err)
	}
	if len(out.Rows) != 32 {
		t.Fatalf("rows %d", len(out.Rows))
	}
	for i, r := range out.Rows {
		n := i + 1
		if r.Power != n || r.DynastyPower != 33-n {
			t.Fatalf("%s: now %d dynasty %d, want %d and %d", r.Franchise, r.Power, r.DynastyPower, n, 33-n)
		}
	}
}
