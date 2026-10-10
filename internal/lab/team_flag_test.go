package lab

import "testing"

// Pins the against-standing season flag to the Lab's rule (tradelog.js:169):
// at least two against-standing trades AND at least 30% of the season's trades.
// Real data has no season on these edges, so the goldens alone cannot catch a drift.
func TestTeamSeasonCellFlaggedBoundaries(t *testing.T) {
	cases := []struct {
		against, n int
		want       bool
	}{
		{2, 6, true},  // two trades, a third of the season
		{3, 10, true}, // exactly 30%
		{2, 7, false}, // two trades, under 30%
		{1, 2, false}, // half the season, but only one trade
		{1, 3, false}, // one trade, a third of the season
		{0, 4, false}, // none against standing
		{5, 5, true},  // every trade against standing
		{29, 100, false},
	}
	for _, c := range cases {
		got := TeamSeasonCell{N: c.n, Against: c.against}.Flagged()
		if got != c.want {
			t.Errorf("against %d of %d: Flagged() = %v, want %v", c.against, c.n, got, c.want)
		}
	}
}
