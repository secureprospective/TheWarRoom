package nativecheck

import (
	"slices"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

func TestAutoLineupPicksBestHealthyAndBenchesByeAndOut(t *testing.T) {
	rules := leaguerules.LineupRules{
		Positions: leaguerules.Known(map[leaguerules.Position]leaguerules.Bounds{
			leaguerules.QB: {Min: leaguerules.Known(1), Max: leaguerules.Known(1)},
			leaguerules.RB: {Min: leaguerules.Known(1), Max: leaguerules.Known(2)},
			leaguerules.WR: {Min: leaguerules.Known(1), Max: leaguerules.Known(2)},
		}),
		Groups: leaguerules.Known(map[leaguerules.LineupGroup]leaguerules.Bounds{
			leaguerules.Offense: {Min: leaguerules.Known(4), Max: leaguerules.Known(4)},
			leaguerules.Total:   {Min: leaguerules.Known(4), Max: leaguerules.Known(4)},
		}),
	}
	for _, bad := range []leaguerules.NFLStatus{
		leaguerules.Out, leaguerules.InjuredReserve, leaguerules.PUP,
		leaguerules.NFI, leaguerules.Suspended, "Doubtful", "reserve",
	} {
		if !IneligibleDesignation(bad) {
			t.Fatalf("expected %q to be ineligible", bad)
		}
	}
	for _, ok := range []leaguerules.NFLStatus{"", "Questionable", "Probable"} {
		if IneligibleDesignation(ok) {
			t.Fatalf("expected %q to be eligible", ok)
		}
	}
	candidates := []AutoLineupCandidate{
		{ID: "qb-out", Position: leaguerules.QB, Median: 30, Ceiling: 40, Available: true,
			Eligible: !IneligibleDesignation(leaguerules.Out)},
		{ID: "qb-bye", Position: leaguerules.QB, Median: 28, Ceiling: 35, Available: true, Eligible: false},
		{ID: "qb-q", Position: leaguerules.QB, Median: 22, Ceiling: 30, Available: true,
			Eligible: !IneligibleDesignation("Questionable")},
		{ID: "qb-low", Position: leaguerules.QB, Median: 15, Ceiling: 20, Available: true, Eligible: true},
		{ID: "rb-high", Position: leaguerules.RB, Median: 18, Ceiling: 25, Available: true, Eligible: true},
		{ID: "rb-tie-b", Position: leaguerules.RB, Median: 12, Ceiling: 18, Available: true, Eligible: true},
		{ID: "rb-tie-a", Position: leaguerules.RB, Median: 12, Ceiling: 22, Available: true, Eligible: true},
		{ID: "wr-best", Position: leaguerules.WR, Median: 14, Ceiling: 20, Available: true, Eligible: true},
		{ID: "wr-unavail", Position: leaguerules.WR, Median: 99, Ceiling: 99, Available: false, Eligible: true},
	}
	got := SelectAutoLineup(rules, candidates)
	want := []string{"qb-q", "rb-high", "rb-tie-a", "wr-best"}
	if !slices.Equal(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestAutoLineupPassesJudgeLineupAcrossRuleSets(t *testing.T) {
	custom := leaguerules.LineupRules{
		Positions: leaguerules.Known(map[leaguerules.Position]leaguerules.Bounds{
			leaguerules.QB: {Min: leaguerules.Known(1), Max: leaguerules.Known(1)},
			leaguerules.RB: {Min: leaguerules.Known(2), Max: leaguerules.Known(3)},
			leaguerules.WR: {Min: leaguerules.Known(2), Max: leaguerules.Known(3)},
			leaguerules.TE: {Min: leaguerules.Known(1), Max: leaguerules.Known(2)},
			leaguerules.PK: {Min: leaguerules.Known(1), Max: leaguerules.Known(1)},
		}),
		Groups: leaguerules.Known(map[leaguerules.LineupGroup]leaguerules.Bounds{
			leaguerules.Offense: {Min: leaguerules.Known(7), Max: leaguerules.Known(7)},
			leaguerules.Total:   {Min: leaguerules.Known(8), Max: leaguerules.Known(8)},
		}),
	}
	sets := []struct {
		name  string
		rules leaguerules.LineupRules
	}{
		{"Default32", leaguerules.Default32().Lineup},
		{"LegacyNFL", leaguerules.LegacyNFL().Lineup},
		{"CustomFlex8", custom},
	}
	positions := []leaguerules.Position{
		leaguerules.QB, leaguerules.RB, leaguerules.WR, leaguerules.TE, leaguerules.PK, leaguerules.OL,
		leaguerules.DT, leaguerules.DE, leaguerules.LB, leaguerules.CB, leaguerules.S,
	}
	var candidates []AutoLineupCandidate
	posOf := map[string]leaguerules.Position{}
	for pIdx, pos := range positions {
		for k := 1; k <= 5; k++ {
			id := string(pos) + "-" + string(rune('0'+k))
			posOf[id] = pos
			candidates = append(candidates, AutoLineupCandidate{
				ID: id, Position: pos, Median: float64(20 - k + pIdx%3), Ceiling: float64(30 - k),
				Available: true, Eligible: true,
			})
		}
	}
	for _, tc := range sets {
		t.Run(tc.name, func(t *testing.T) {
			ids := SelectAutoLineup(tc.rules, candidates)
			starters := make([]Starter, len(ids))
			for i, id := range ids {
				starters[i] = Starter{ID: id, Position: posOf[id]}
			}
			verdict := JudgeLineup(tc.rules, true, starters)
			if verdict.Status != LineupValid {
				t.Fatalf("%s: verdict=%s words=%s starters=%v", tc.name, verdict.Status, verdict.Words, ids)
			}
		})
	}
}

func TestAutoLineupStartsBenchedPlayerOnlyToMeetAMinimum(t *testing.T) {
	rules := leaguerules.LineupRules{
		Positions: leaguerules.Known(map[leaguerules.Position]leaguerules.Bounds{
			leaguerules.QB: {Min: leaguerules.Known(1), Max: leaguerules.Known(1)},
			leaguerules.OL: {Min: leaguerules.Known(1), Max: leaguerules.Known(1)},
		}),
	}
	got := SelectAutoLineup(rules, []AutoLineupCandidate{
		{ID: "qb-bye", Position: leaguerules.QB, Median: 30, Eligible: false},
		{ID: "qb-playing", Position: leaguerules.QB, Median: 10, Eligible: true},
		{ID: "ol-bye", Position: leaguerules.OL, Median: 8, Eligible: false},
	})
	slices.Sort(got)
	if !slices.Equal(got, []string{"ol-bye", "qb-playing"}) {
		t.Fatalf("lineup %v, want the playing QB and the only O-Line card (on bye)", got)
	}
}
