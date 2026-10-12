package race

import (
	"slices"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
)

type play struct {
	week       int
	home, away string
	hs, as     leaguefeed.Points
}

func weeks(plays ...play) []leaguefeed.WeekResult {
	byWeek := map[int]*leaguefeed.WeekResult{}
	out := []leaguefeed.WeekResult{}
	for _, p := range plays {
		res := func(a, b leaguefeed.Points) leaguefeed.Result {
			switch {
			case a > b:
				return "W"
			case a < b:
				return "L"
			}
			return "T"
		}
		g := leaguefeed.Game{Regular: true,
			Home: leaguefeed.Side{Franchise: p.home, Scored: p.hs >= 0, Points: max(p.hs, 0), Result: res(p.hs, p.as)},
			Away: leaguefeed.Side{Franchise: p.away, Scored: p.as >= 0, Points: max(p.as, 0), Result: res(p.as, p.hs)}}
		if p.hs < 0 {
			g.Home.Result, g.Away.Result = "", ""
		}
		if byWeek[p.week] == nil {
			out = append(out, leaguefeed.WeekResult{Week: p.week})
			byWeek[p.week] = &out[len(out)-1]
		}
		byWeek[p.week].Games = append(byWeek[p.week].Games, g)
	}
	return out
}

func league() []Franchise {
	return []Franchise{
		{"A", "d1", "c1"}, {"B", "d1", "c1"}, {"C", "d1", "c1"}, {"D", "d1", "c1"},
		{"E", "d2", "c1"}, {"F", "d2", "c1"}, {"V", "d3", "c2"},
	}
}

func allKeys(t *testing.T) []Key {
	t.Helper()
	keys, err := ParseSort("PCT,H2H,DIVPCT,CONFPCT,ALL_PLAY_PCT,PTS,PWR,")
	if err != nil {
		t.Fatal(err)
	}
	return keys
}

// A, B and C finish 3-2. A beat both, so head-to-head puts A first; B and C split their two
// games, so they restart: head-to-head is level and division percentage puts C ahead.
func TestThreeWayTieRestartsForTheTeamsStillLevel(t *testing.T) {
	season := weeks(
		play{1, "A", "B", 10, 5}, play{1, "C", "D", 10, 5},
		play{2, "A", "C", 10, 5}, play{2, "B", "D", 10, 5},
		play{3, "B", "C", 10, 5}, play{3, "E", "A", 10, 5},
		play{4, "C", "B", 10, 5}, play{4, "D", "A", 10, 5},
		play{5, "A", "F", 10, 5}, play{5, "B", "E", 10, 5}, play{5, "C", "D", 10, 5},
	)
	table, err := Build(league(), season, allKeys(t), nil)
	if err != nil {
		t.Fatal(err)
	}
	got := table.Division("d1")
	if !slices.Equal(got.Order, []string{"A", "C", "B", "D"}) {
		t.Fatalf("order = %v", got.Order)
	}
	if len(got.Ties) != 1 {
		t.Fatalf("ties = %+v", got.Ties)
	}
	steps := got.Ties[0].Steps
	want := []struct {
		depth   int
		key     Key
		outcome Outcome
		teams   []string
	}{
		{0, KeyH2H, Split, []string{"A", "B", "C"}},
		{1, KeyH2H, Level, []string{"B", "C"}},
		{1, KeyDiv, Split, []string{"B", "C"}},
	}
	if len(steps) != len(want) {
		t.Fatalf("steps = %+v", steps)
	}
	for i, w := range want {
		s := steps[i]
		if s.Depth != w.depth || s.Key != w.key || s.Outcome != w.outcome || !slices.Equal(s.Teams, w.teams) {
			t.Fatalf("step %d = %+v, want %+v", i, s, w)
		}
	}
	if steps[0].Values[0].Record != (Record{W: 2}) || steps[0].Values[1].Record != (Record{W: 1, L: 2}) {
		t.Fatalf("head-to-head values = %+v", steps[0].Values)
	}
	if steps[2].Values[0].Record != (Record{W: 2, L: 2}) || steps[2].Values[1].Record != (Record{W: 3, L: 2}) {
		t.Fatalf("division values = %+v", steps[2].Values)
	}
	if got.Ties[0].Teams[0] != "A" || got.Ties[0].Teams[2] != "B" {
		t.Fatalf("tie order = %v", got.Ties[0].Teams)
	}
}

// Teams that never met, from different divisions, skip head-to-head and division percentage.
func TestCrossDivisionTieSkipsStepsThatDoNotApply(t *testing.T) {
	season := weeks(play{1, "A", "D", 10, 5}, play{1, "E", "V", 10, 5})
	table, err := Build(league(), season, allKeys(t), nil)
	if err != nil {
		t.Fatal(err)
	}
	got := table.Order([]string{"E", "A"})
	if !slices.Equal(got.Order, []string{"A", "E"}) {
		t.Fatalf("order = %v", got.Order)
	}
	outcomes := []Outcome{}
	for _, s := range got.Ties[0].Steps {
		outcomes = append(outcomes, s.Outcome)
	}
	if !slices.Equal(outcomes, []Outcome{Skipped, Skipped, Split}) || got.Ties[0].Steps[2].Key != KeyConf {
		t.Fatalf("steps = %+v", got.Ties[0].Steps)
	}
}

func TestUnbrokenTieKeepsFranchiseOrderAndSaysSo(t *testing.T) {
	table, err := Build(league(), nil, allKeys(t), map[string]int64{"B": 100})
	if err != nil {
		t.Fatal(err)
	}
	got := table.Order([]string{"F", "E"})
	last := got.Ties[0].Steps[len(got.Ties[0].Steps)-1]
	if !slices.Equal(got.Order, []string{"E", "F"}) || last.Outcome != Unbroken {
		t.Fatalf("order = %v, last step = %+v", got.Order, last)
	}
	if power := got.Ties[0].Steps[len(got.Ties[0].Steps)-2]; power.Key != KeyPower || power.Outcome != Skipped {
		t.Fatalf("power without ratings = %+v", power)
	}
}

func TestStandingsCountRecordsStreaksAllPlayAndRemainingGames(t *testing.T) {
	season := weeks(
		play{1, "A", "B", 10, 5}, play{1, "C", "D", 7, 7}, play{1, "E", "F", 1, 2},
		play{2, "B", "A", 9, 8}, play{2, "D", "C", -1, -1},
	)
	season[0].Others = []leaguefeed.Side{{Franchise: "V", Scored: true, Points: 7}}
	table, err := Build(league(), season, allKeys(t), nil)
	if err != nil {
		t.Fatal(err)
	}
	a, _ := table.Line("A")
	if a.Overall != (Record{W: 1, L: 1}) || a.Streak != (Streak{"L", 1}) || a.For != 18 || a.Against != 14 {
		t.Fatalf("A = %+v", a)
	}
	c, _ := table.Line("C")
	if c.Overall != (Record{T: 1}) || c.InDivision != (Record{T: 1}) || !slices.Equal(c.Remaining, []string{"D"}) {
		t.Fatalf("C = %+v", c)
	}
	// Week 1 scores: A 10, V 7, C 7, D 7, B 5, F 2, E 1. C ties V and D; week 2 adds B and A.
	if c.AllPlay != (Record{W: 3, L: 1, T: 2}) {
		t.Fatalf("C all-play = %+v", c.AllPlay)
	}
	if v, _ := table.Line("V"); v.For != 7 || v.Overall.Games() != 0 {
		t.Fatalf("V = %+v", v)
	}
	if table.RegularWeeks != 1 {
		t.Fatalf("complete weeks = %d", table.RegularWeeks)
	}
	if table.HeadToHead("A", "B") != (Record{W: 1, L: 1}) {
		t.Fatal(table.HeadToHead("A", "B"))
	}
	if _, err := Build(league()[:1], season, allKeys(t), nil); err == nil {
		t.Fatal("accepted a franchise the league does not list")
	}
}

func TestSeedingBubbleAndFirstRound(t *testing.T) {
	season := weeks(
		play{1, "A", "B", 10, 5}, play{1, "E", "F", 10, 5}, play{1, "C", "D", 10, 5},
		play{2, "A", "C", 10, 5}, play{2, "E", "B", 10, 5}, play{2, "D", "F", 10, 5},
		play{3, "A", "D", -1, -1}, play{3, "B", "C", -1, -1}, play{3, "E", "F", -1, -1},
	)
	table, err := Build(league(), season, allKeys(t), nil)
	if err != nil {
		t.Fatal(err)
	}
	field, err := table.Seeding("c1", []string{"d1", "d2"}, 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(field.Seeds) != 3 || field.Seeds[0].Franchise != "A" || field.Seeds[1].Franchise != "E" ||
		!field.Seeds[1].DivisionWinner || field.Seeds[2].DivisionWinner {
		t.Fatalf("seeds = %+v", field.Seeds)
	}
	bubble := table.Bubble(field)
	if len(bubble) == 0 || bubble[0].Seed != 3 {
		t.Fatalf("bubble = %+v", bubble)
	}
	for _, c := range bubble[1:] {
		l, _ := table.Line(c.Franchise)
		if c.Seed != 0 || c.HalfGamesBack > 2*len(l.Remaining) {
			t.Fatalf("contender %+v out of reach", c)
		}
	}
	if _, err := table.Seeding("c1", []string{"d1", "d2"}, 1); err == nil {
		t.Fatal("seeded two leaders into one place")
	}
	byes, games := FirstRound(7)
	if byes != 1 || !slices.Equal(games, [][2]int{{2, 7}, {3, 6}, {4, 5}}) {
		t.Fatal(byes, games)
	}
	if byes, games := FirstRound(8); byes != 0 || len(games) != 4 || games[0] != [2]int{1, 8} {
		t.Fatal(byes, games)
	}
	if HalfGamesBack(Record{W: 9, L: 4}, Record{W: 8, L: 5}) != 2 {
		t.Fatal("one game back is two half games")
	}
}

func TestParseSortRejectsUnknownSteps(t *testing.T) {
	for _, raw := range []string{"", ",", "PCT,VP"} {
		if _, err := ParseSort(raw); err == nil {
			t.Errorf("%q accepted", raw)
		}
	}
}
