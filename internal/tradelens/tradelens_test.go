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
	// The league's best benched receiver is the 9, so each player counts for what he scores over 9.
	// You: 11+3 becomes 11+1. They: 1+0 becomes 3+2, and their pick, 6.65 at 70% long term, now has
	// a 2 to beat where it had nothing. Raw sums would call it 23 for 10.
	if !near(got.You.Net, -2) || !near(got.Them.Net, 2) || got.Verdict != "bad" {
		t.Fatalf("you %v them %v verdict %q", got.You.Net, got.Them.Net, got.Verdict)
	}
	if !near(got.You.Out, 3) || !near(got.You.In, 1) || !near(got.You.In-got.You.Out, got.You.Net) {
		t.Fatalf("out %v in %v", got.You.Out, got.You.In)
	}
}

func TestShelfValuesAreWhatEachSideGainsOrLoses(t *testing.T) {
	got, err := Evaluate(league())
	if err != nil {
		t.Fatal(err)
	}
	want := map[string][2]float64{"a1": {9, 10}, "a2": {1, 2}, "a3": {0, 1}, "b1": {0, 1},
		"b2": {0, 0}, "pick:2027:1:b": {3.65, 6.65}}
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

// A pick is priced on its chance: one of the two players taken there would start for either side.
// The later round holds a better player but a worse chance, so the pick keeps its own round.
func TestPickIsWorthWhatItsPlayersWouldAdd(t *testing.T) {
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
		if a.Token == pick.Token && (math.Abs(a.You-4) > 1e-9 || math.Abs(a.Them-5.5) > 1e-9) {
			t.Fatalf("pick adds %v over your second starter and holds %v over theirs, want 4 and 5.5", a.You, a.Them)
		}
	}
	if math.Abs(got.Them.Before-6.5) > 1e-9 {
		t.Fatalf("their count is %v, want 1 in players and 5.5 in the pick", got.Them.Before)
	}
}

// Equal scorers are not equal in a trade: the league's spare quarterback is a 4 and its spare safety
// an 18, so a 20 at quarterback is worth 16 and a 20 at safety is worth 2.
func TestScarcePositionsAreWorthMore(t *testing.T) {
	at := func(token, franchise, position string, v float64) Asset {
		return Asset{Token: token, Franchise: franchise, Position: position, Now: v, Dynasty: v, Valued: true}
	}
	got, err := Evaluate(Input{N: 2, Teams: 2, Slots: []Slot{{"QB", 1, 1}, {"S", 1, 1}},
		You: Side{Franchise: "a", Direction: Retool}, Them: Side{Franchise: "b", Direction: Retool},
		Assets: []Asset{at("qb", "a", "QB", 20), at("s", "a", "S", 20), at("b1", "b", "QB", 10),
			at("b2", "b", "S", 19), at("b3", "b", "QB", 4), at("b4", "b", "S", 18)}})
	if err != nil {
		t.Fatal(err)
	}
	for _, a := range got.Assets {
		if want, ok := map[string]float64{"qb": 16, "s": 2}[a.Token]; ok && !near(a.You, want) {
			t.Errorf("%s holds %v, want %v", a.Token, a.You, want)
		}
	}
}
