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
	return Asset{Token: token, Franchise: franchise, Pick: true, Dynasty: dynasty, Valued: true, SignedNext: true}
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
	// You: 20+12 becomes 20+10. They: 10+9.5 becomes 12+11. Raw sums would call it 23 for 10.
	if !near(got.You.Net, -2) || !near(got.Them.Net, 3.5) || got.Verdict != "bad" {
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
	want := map[string][2]float64{"a1": {9, 10.5}, "a2": {1, 2.5}, "a3": {0, 1.5}, "b1": {0, 1},
		"b2": {0, 0}, "pick:2027:1:b": {0, 0.5}}
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
	// Contend counts 60% now: the vet is 22 to a contender and 10 to a rebuilder, who would bench him.
	if !near(rebuild.You.Net, 0) || !near(contend.You.Net, 10) {
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
