package tradelens

import (
	"math"
	"testing"
)

func near(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

func player(token, franchise string, now, dynasty float64) Asset {
	return Asset{Token: token, Franchise: franchise, Position: "WR", Now: now, Dynasty: dynasty, Valued: true,
		SignedNext: true}
}

func heldPick(token, franchise string, dynasty float64) Asset {
	return Asset{Token: token, Franchise: franchise, Pick: true, Dynasty: dynasty, Valued: true, SignedNext: true,
		Comps: [][]Comp{{{Dynasty: dynasty}}}, Weight: 1}
}

// A league that starts two: the contender holds 20, 12, 11 and the rebuilder 10, 9, 2 plus a pick.
func league() Input {
	return Input{N: 2, Teams: 2, Slots: []Slot{{Position: "WR", Min: 1, Max: 2}},
		You:  Side{Franchise: "a", Direction: Rebuild, Set: true},
		Them: Side{Franchise: "b", Direction: Rebuild},
		Assets: []Asset{player("a1", "a", 20, 20), player("a2", "a", 12, 12), player("a3", "a", 11, 11),
			player("b1", "b", 10, 10), player("b2", "b", 9, 9), player("b3", "b", 2, 2),
			heldPick("pick:2027:1:b", "b", 9.5)}}
}

func TestTwoForOneCountsOnlyTheBestN(t *testing.T) {
	in := league()
	in.Give, in.Get = []string{"a2", "a3"}, []string{"b1"}
	got, err := Evaluate(in)
	if err != nil {
		t.Fatal(err)
	}
	// Every receiver is rostered, so nobody is free to set a replacement level and each counts in
	// full. You: 20+12 becomes 20+10. They: 10+9 becomes 12+11, and their pick is the same 6.65
	// either way. Raw sums would call it 23 for 10.
	if !near(got.You.Net, -2) || !near(got.Them.Net, 4) || got.Verdict != "bad" {
		t.Fatalf("you %v them %v verdict %q", got.You.Net, got.Them.Net, got.Verdict)
	}
	if !near(got.You.Out, 12) || !near(got.You.In, 10) || !near(got.You.In-got.You.Out, got.You.Net) {
		t.Fatalf("out %v in %v", got.You.Out, got.You.In)
	}
}

func TestShelfValuesAreWhatEachSideGainsOrLoses(t *testing.T) {
	got, err := Evaluate(league())
	if err != nil {
		t.Fatal(err)
	}
	want := map[string][2]float64{"a1": {9, 11}, "a2": {1, 3}, "a3": {0, 2}, "b1": {0, 8},
		"b2": {0, 7}, "pick:2027:1:b": {6.65, 6.65}}
	for _, a := range got.Assets {
		if w, ok := want[a.Token]; ok && (!near(a.You, w[0]) || !near(a.Them, w[1])) {
			t.Errorf("%s: you %v them %v, want %v", a.Token, a.You, a.Them, w)
		}
	}
	if got.Verdict != "empty" || len(got.Assets) != 7 {
		t.Fatalf("verdict %q, %d assets", got.Verdict, len(got.Assets))
	}
}

func TestDirectionMovesTheLens(t *testing.T) {
	in := league()
	in.Assets = append(in.Assets, Asset{Token: "vet", Franchise: "b", Position: "WR", Now: 30, Dynasty: 10,
		Valued: true})
	in.Get = []string{"vet"}
	rebuild, _ := Evaluate(in)
	in.You.Direction = Contend
	contend, _ := Evaluate(in)
	// The vet is 23 to a contender (65% now) and 16 to a rebuilder (30% now), beside starters of 20 and 12.
	if !near(rebuild.You.Net, 4) || !near(contend.You.Net, 11) {
		t.Fatalf("rebuild %v contend %v", rebuild.You.Net, contend.You.Net)
	}
}

func TestTuneKeepsTheOtherSideWhole(t *testing.T) {
	in := league()
	in.Assets = append(in.Assets, player("a4", "a", 1, 1))
	in.Give, in.Get = []string{"a4"}, []string{"b1"}
	got, err := Evaluate(in)
	if err != nil {
		t.Fatal(err)
	}
	// They lose a starter for a bench piece; adding your third receiver makes them whole at no cost to you.
	if got.Verdict != "balk" || len(got.Tune) == 0 || got.Tune[0].Kind != "sweeten" || got.Tune[0].Token != "a3" {
		t.Fatalf("verdict %q tune %v", got.Verdict, got.Tune)
	}
	for _, s := range got.Tune {
		if s.Them <= -even {
			t.Errorf("%s %s leaves them at %v", s.Kind, s.Token, s.Them)
		}
	}
}

func TestNeedsAndRefusals(t *testing.T) {
	got, _ := Evaluate(league())
	if len(got.You.Needs) != 1 || got.You.Needs[0].State != "ok" || got.Them.Needs[0].State != "hole" {
		t.Fatalf("needs %v %v", got.You.Needs, got.Them.Needs)
	}
	in := league()
	in.Give = []string{"b1"}
	if _, err := Evaluate(in); err == nil {
		t.Fatal("sending their player was allowed")
	}
	in = league()
	in.Them = Side{}
	if got, err := Evaluate(in); err != nil || len(got.Assets) != 3 {
		t.Fatalf("no partner: %v, %d assets", err, len(got.Assets))
	}
}

func TestOurRead(t *testing.T) {
	for _, c := range []struct {
		s    Standing
		want Direction
	}{{Standing{Wins: 7, Losses: 2, Age: 25}, Contend}, {Standing{Wins: 2, Losses: 7, Age: 29}, Rebuild},
		{Standing{Wins: 5, Losses: 4, Age: 29}, Contend}, {Standing{Wins: 5, Losses: 4, Age: 25}, Retool},
		{Standing{Age: 29}, Retool}} {
		if got := Read(c.s, 27); got != c.want {
			t.Errorf("%+v: %s, want %s", c.s, got, c.want)
		}
	}
}

// A pick is worth the players taken where it sits, on average, to whoever holds it: 20 and
// nothing make 10. The later round holds the same star among more misses, so the pick keeps its own.
func TestPickIsWorthItsPlayersOnAverage(t *testing.T) {
	in := league()
	in.Assets = in.Assets[:len(in.Assets)-1]
	pick := heldPick("pick:2027:1:b", "b", 10)
	pick.Comps, pick.Weight = [][]Comp{{{Position: "WR", Now: 20, Dynasty: 20}, {}},
		{{Position: "WR", Now: 20, Dynasty: 20}, {}, {}, {}}}, 1
	in.Assets = append(in.Assets, pick)
	got, err := Evaluate(in)
	if err != nil {
		t.Fatal(err)
	}
	for _, a := range got.Assets {
		if a.Token == pick.Token && (!near(a.You, 10) || !near(a.Them, 10)) {
			t.Fatalf("pick is worth %v to you and %v to them, want 10 to both", a.You, a.Them)
		}
	}
	if !near(got.Them.Before, 29) {
		t.Fatalf("their count is %v, want 19 in players and 10 in the pick", got.Them.Before)
	}
}

// Equal scorers are not equal in a trade. The best quarterback nobody rosters is a 3 and the best
// safety a 17, and a safety's edge only half lasts a season: a 20 at quarterback is worth 17 and
// a 20 at safety 1.5.
func TestScarcePositionsAreWorthMore(t *testing.T) {
	at := func(token, franchise, position string, v float64) Asset {
		return Asset{Token: token, Franchise: franchise, Position: position, Now: v, Dynasty: v, Valued: true}
	}
	got, err := Evaluate(Input{N: 2, Teams: 2, Slots: []Slot{{"QB", 1, 1}, {"S", 1, 1}},
		Keep: map[string]float64{"S": 0.5},
		You:  Side{Franchise: "a", Direction: Retool}, Them: Side{Franchise: "b", Direction: Retool},
		Assets: []Asset{at("qb", "a", "QB", 20), at("s", "a", "S", 20), at("b1", "b", "QB", 10),
			at("b2", "b", "S", 19), at("b3", "b", "QB", 4), at("b4", "b", "S", 18),
			at("f1", "", "QB", 3), at("f2", "", "S", 17)}})
	if err != nil {
		t.Fatal(err)
	}
	for _, a := range got.Assets {
		if want, ok := map[string]float64{"qb": 17, "s": 1.5}[a.Token]; ok && !near(a.You, want) {
			t.Errorf("%s holds %v, want %v", a.Token, a.You, want)
		}
	}
}

// A position the draft restocks is cheap to trade for. This league's drafted safeties lead by 4
// on average and its drafted quarterbacks by 1, so a 10 lead at each is worth 6 and 9; the pick
// that could take either keeps its full 2.5.
func TestDraftSupplyMakesAPositionCheap(t *testing.T) {
	at := func(token, franchise, position string, v float64) Asset {
		return Asset{Token: token, Franchise: franchise, Position: position, Now: v, Dynasty: v, Valued: true}
	}
	taken := []Comp{{Position: "S", Now: 9, Dynasty: 9}, {Position: "S", Now: 5, Dynasty: 5},
		{Position: "QB", Now: 7, Dynasty: 7}, {Position: "QB", Now: 0, Dynasty: 0}, {}}
	pick := heldPick("pick:2027:1:a", "a", 0)
	pick.Comps = [][]Comp{taken[:4]}
	got, err := Evaluate(Input{N: 2, Teams: 2, Slots: []Slot{{"QB", 1, 1}, {"S", 1, 1}}, Drafted: taken,
		You: Side{Franchise: "a", Direction: Retool}, Them: Side{Franchise: "b", Direction: Retool},
		Assets: []Asset{at("qb", "a", "QB", 15), at("s", "a", "S", 13), pick, at("b1", "b", "QB", 8),
			at("b2", "b", "S", 7), at("f1", "", "QB", 5), at("f2", "", "S", 3)}})
	if err != nil {
		t.Fatal(err)
	}
	for _, a := range got.Assets {
		if want, ok := map[string]float64{"qb": 9, "s": 6, pick.Token: 2.5}[a.Token]; ok && !near(a.You, want) {
			t.Errorf("%s holds %v, want %v", a.Token, a.You, want)
		}
	}
}
