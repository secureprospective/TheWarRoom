package nativecheck

import (
	"fmt"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

// lineupOf builds starters from "position:count" pairs, scoring each one a point higher than the last.
func lineupOf(pairs ...string) []Starter {
	var out []Starter
	for _, pair := range pairs {
		var position string
		var n int
		parts := strings.Split(pair, ":")
		position = parts[0]
		if _, err := fmt.Sscanf(parts[1], "%d", &n); err != nil {
			panic(err)
		}
		for i := 0; i < n; i++ {
			out = append(out, Starter{ID: fmt.Sprintf("%s%d", position, i), Position: leaguerules.Position(position),
				Points: leaguerules.Decimal((len(out) + 1) * 10000)})
		}
	}
	return out
}

func legalLineup() []Starter {
	return lineupOf("QB:1", "RB:2", "WR:3", "TE:2", "PK:1", "DT:2", "DE:2", "LB:4", "CB:2", "S:2")
}

func TestLegalLineupIsValid(t *testing.T) {
	v := JudgeLineup(leaguerules.LegacyNFL().Lineup, true, legalLineup())
	if v.Status != LineupValid || len(v.Problems) != 0 || v.Penalty != "" || v.Words != "Valid lineup" {
		t.Fatalf("%+v", v)
	}
}

func TestLineupBreaksEachKindOfLimit(t *testing.T) {
	rules := leaguerules.LegacyNFL().Lineup
	for _, c := range []struct {
		name  string
		lines []Starter
		want  string
	}{
		{"position over", lineupOf("QB:2", "RB:2", "WR:3", "TE:1", "PK:1", "DT:2", "DE:2", "LB:4", "CB:2", "S:2"),
			"QB: 2 starting, needs exactly 1"},
		{"position under", lineupOf("QB:1", "RB:2", "WR:3", "TE:2", "PK:1", "DT:1", "DE:2", "LB:5", "CB:2", "S:2"),
			"DT: 1 starting, needs at least 2"},
		{"offense group over", lineupOf("QB:1", "RB:3", "WR:3", "TE:2", "PK:1", "DT:2", "DE:2", "LB:3", "CB:2", "S:2"),
			"Offense: 9 starting, at most 8"},
		{"defense group under", lineupOf("QB:1", "RB:3", "WR:3", "TE:1", "PK:1", "DT:2", "DE:2", "LB:2", "CB:2", "S:2", "XX:1"),
			"Defense: 10 starting, needs exactly 12"},
		{"total under", lineupOf("QB:1", "RB:2", "WR:3", "TE:2", "PK:1", "DT:2", "DE:2", "LB:3", "CB:2", "S:2"),
			"Total: 20 starting, needs exactly 21"},
		{"not a lineup position", append(legalLineup(), Starter{ID: "x", Position: "XX"}),
			"XX: 1 starting, not a lineup position"},
		{"position not held", append(legalLineup(), Starter{ID: "x"}), "Position unknown: 1 starting"},
	} {
		v := JudgeLineup(rules, true, c.lines)
		found := false
		for _, p := range v.Problems {
			found = found || p.Words == c.want
		}
		if v.Status != LineupInvalid || !found {
			t.Errorf("%s: want %q, got %+v", c.name, c.want, v)
		}
		if v.Penalty != "The highest-scoring starter's points are set to zero" {
			t.Errorf("%s: penalty %q", c.name, v.Penalty)
		}
	}
}

func TestGroupLimitAloneInvalidatesALineup(t *testing.T) {
	rules := leaguerules.LegacyNFL().Lineup
	// Every position is inside its own range and the total is not limited; only the offense group
	// (at most 8) is broken by nine offensive starters.
	delete(rules.Groups.Value, leaguerules.Total)
	lines := lineupOf("QB:1", "RB:3", "WR:3", "TE:2", "PK:1", "DT:2", "DE:2", "LB:4", "CB:2", "S:2")
	if v := JudgeLineup(rules, true, lines); v.Status != LineupInvalid || len(v.Problems) != 1 || v.Problems[0].Subject != "Offense" {
		t.Fatalf("%+v", v)
	}
	delete(rules.Groups.Value, leaguerules.Offense)
	if v := JudgeLineup(rules, true, lines); v.Status != LineupValid {
		t.Fatalf("the removed group limit still applied: %+v", v)
	}
}

func TestKickerCountsTowardTheTotalButNeitherSide(t *testing.T) {
	rules := leaguerules.LegacyNFL().Lineup
	rules.Groups = leaguerules.Known(map[leaguerules.LineupGroup]leaguerules.Bounds{
		leaguerules.Offense: {Max: leaguerules.Known(1)}, leaguerules.Defense: {Max: leaguerules.Known(1)}})
	rules.Positions = leaguerules.Known(map[leaguerules.Position]leaguerules.Bounds{
		leaguerules.PK: {Max: leaguerules.Known(3)}, leaguerules.QB: {Max: leaguerules.Known(3)}})
	if v := JudgeLineup(rules, true, lineupOf("PK:3", "QB:1")); v.Status != LineupValid {
		t.Fatalf("kickers were counted as a side: %+v", v)
	}
}

func TestMissedLineupIsFlaggedNotPenalised(t *testing.T) {
	rules := leaguerules.LegacyNFL().Lineup
	v := JudgeLineup(rules, false, nil)
	if v.Status != LineupMissed || !v.Flagged || v.Penalty != "" || len(v.Problems) != 0 ||
		!strings.Contains(v.Words, "flagged to the commissioner; nothing is zeroed") {
		t.Fatalf("%+v", v)
	}
	rules.MissedFlag = leaguerules.Known(false)
	if v := JudgeLineup(rules, false, nil); v.Flagged || v.Status != LineupMissed {
		t.Fatalf("%+v", v)
	}
	rules.MissedFlag = leaguerules.Needs[bool]("")
	if v := JudgeLineup(rules, false, nil); v.Flagged || !strings.Contains(v.Words, "Needs a ruling") {
		t.Fatalf("%+v", v)
	}
}

func TestUnsetLimitsAreNotJudged(t *testing.T) {
	rules := leaguerules.LegacyNFL().Lineup
	rules.Positions = leaguerules.Needs[map[leaguerules.Position]leaguerules.Bounds]("")
	rules.Groups = leaguerules.Needs[map[leaguerules.LineupGroup]leaguerules.Bounds]("")
	if v := JudgeLineup(rules, true, legalLineup()); v.Status != LineupUnjudged {
		t.Fatalf("%+v", v)
	}
}

func TestPenaltyZeroesTheHighestScoringStarter(t *testing.T) {
	starters := []Starter{{ID: "a", Points: 50000}, {ID: "b", Points: 190000}, {ID: "c", Points: 120000}}
	got := ApplyPenalty(leaguerules.Known(leaguerules.ZeroTopStarter), starters)
	if got.Zeroed != "b" || got.Starters[1].Points != 0 || got.Starters[0].Points != 50000 ||
		got.Starters[2].Points != 120000 || Total(got.Starters) != 170000 {
		t.Fatalf("%+v", got)
	}
	if starters[1].Points != 190000 {
		t.Fatal("the caller's starters were changed")
	}
	tied := ApplyPenalty(leaguerules.Known(leaguerules.ZeroTopStarter),
		[]Starter{{ID: "z", Points: 90000}, {ID: "m", Points: 90000}, {ID: "q", Points: 10000}})
	if tied.Zeroed != "m" {
		t.Fatalf("a tie did not go to the lower id: %+v", tied)
	}
}

func TestOtherPenaltiesDoNotZeroAnyone(t *testing.T) {
	starters := []Starter{{ID: "a", Points: 50000}, {ID: "b", Points: 190000}}
	for _, p := range []leaguerules.Value[leaguerules.Penalty]{leaguerules.Known(leaguerules.NoPenalty),
		leaguerules.Known(leaguerules.ZeroSlot), leaguerules.Needs[leaguerules.Penalty]("")} {
		got := ApplyPenalty(p, starters)
		if got.Zeroed != "" || Total(got.Starters) != 240000 {
			t.Errorf("%+v zeroed someone: %+v", p, got)
		}
	}
	if w := ApplyPenalty(leaguerules.Known(leaguerules.ZeroSlot), starters).Words; !strings.HasPrefix(w, "Needs a ruling") {
		t.Fatalf("a penalty with no ruling on its slot was described as settled: %q", w)
	}
	if got := ApplyPenalty(leaguerules.Known(leaguerules.ZeroTopStarter), nil); got.Zeroed != "" {
		t.Fatalf("%+v", got)
	}
}

// Total adds up the points of a lineup.
func Total(starters []Starter) leaguerules.Decimal {
	var sum leaguerules.Decimal
	for _, s := range starters {
		sum += s.Points
	}
	return sum
}

func TestLineupNeedsPosition(t *testing.T) {
	t.Parallel()
	d32 := leaguerules.Default32().Lineup
	allDraftable := []leaguerules.Position{
		leaguerules.QB, leaguerules.RB, leaguerules.WR, leaguerules.TE, leaguerules.PK,
		leaguerules.DT, leaguerules.DE, leaguerules.LB, leaguerules.CB, leaguerules.S,
	}

	t.Run("default32 unmet position minimum", func(t *testing.T) {
		t.Parallel()
		held := map[leaguerules.Position]int{leaguerules.CB: 1}
		if !LineupNeedsPosition(d32, held, leaguerules.CB) {
			t.Fatal("CB with 1/2 held should be needed")
		}
		if !LineupNeedsPosition(d32, held, leaguerules.PK) {
			t.Fatal("PK with 0/1 held should be needed")
		}
		if !LineupNeedsPosition(d32, held, leaguerules.TE) {
			t.Fatal("TE with 0 held should be needed for unmet Offense group min")
		}
	})

	t.Run("default32 unmet group minimum respects position max and reserved mins", func(t *testing.T) {
		t.Parallel()
		held := map[leaguerules.Position]int{
			leaguerules.QB: 1, leaguerules.RB: 1, leaguerules.WR: 2,
		}
		if !LineupNeedsPosition(d32, held, leaguerules.WR) {
			t.Fatal("WR under max 4 should be needed while Offense is 4/6")
		}
		if LineupNeedsPosition(d32, held, leaguerules.QB) {
			t.Fatal("QB at max 1 should not be needed")
		}
		// 4 WRs + 1 reserved QB + 1 reserved RB already fill all 6 Offense slots.
		fourWR := map[leaguerules.Position]int{leaguerules.WR: 4}
		if LineupNeedsPosition(d32, fourWR, leaguerules.WR) {
			t.Fatal("WR at max 4 should not be needed")
		}
		if LineupNeedsPosition(d32, fourWR, leaguerules.TE) {
			t.Fatal("TE should not be needed when 4 WR + reserved QB/RB fill Offense")
		}
		if !LineupNeedsPosition(d32, fourWR, leaguerules.QB) {
			t.Fatal("QB with 0/1 held should still be needed")
		}
	})

	t.Run("default32 all draftable minimums met leaves nothing draftable needed", func(t *testing.T) {
		t.Parallel()
		full := map[leaguerules.Position]int{
			leaguerules.QB: 1, leaguerules.RB: 1, leaguerules.WR: 4, leaguerules.PK: 1,
			leaguerules.DT: 1, leaguerules.DE: 1, leaguerules.LB: 6, leaguerules.CB: 2,
			leaguerules.S: 1,
		}
		for _, pos := range allDraftable {
			if LineupNeedsPosition(d32, full, pos) {
				t.Errorf("%s should not be needed once all draftable minimums are met", pos)
			}
		}
		if LineupNeedsPosition(d32, full, "XX") || LineupNeedsPosition(d32, full, "") {
			t.Error("unknown or empty position should never be needed")
		}
	})

	t.Run("mirror16 and legacy rules", func(t *testing.T) {
		t.Parallel()
		m16 := leaguerules.Mirror16().Lineup
		held := map[leaguerules.Position]int{
			leaguerules.QB: 1, leaguerules.RB: 1, leaguerules.WR: 2,
		}
		if !LineupNeedsPosition(m16, held, leaguerules.WR) {
			t.Fatal("Mirror16 WR should be needed at Offense 4/5")
		}
		if LineupNeedsPosition(m16, held, leaguerules.OL) {
			t.Fatal("Mirror16 has no OL slot so OL must not be needed")
		}
		held[leaguerules.WR] = 3
		if LineupNeedsPosition(m16, held, leaguerules.WR) {
			t.Fatal("Mirror16 WR should not be needed once Offense reaches 5/5")
		}

		// LegacyNFL has no Offense min (only max 8) and Total 21..21.
		leg := leaguerules.LegacyNFL().Lineup
		legHeld := map[leaguerules.Position]int{
			leaguerules.QB: 1, leaguerules.RB: 1, leaguerules.WR: 2,
			leaguerules.TE: 1, leaguerules.PK: 1,
		}
		if !LineupNeedsPosition(leg, legHeld, leaguerules.WR) {
			t.Fatal("LegacyNFL WR should be needed to reach Total min 21")
		}
		legHeld[leaguerules.WR] = 5
		if LineupNeedsPosition(leg, legHeld, leaguerules.RB) {
			t.Fatal("LegacyNFL RB should not be needed once Offense hits max 8")
		}

		unset := leaguerules.LineupRules{
			Positions: leaguerules.Needs[map[leaguerules.Position]leaguerules.Bounds](""),
			Groups:    leaguerules.Needs[map[leaguerules.LineupGroup]leaguerules.Bounds](""),
		}
		if LineupNeedsPosition(unset, nil, leaguerules.QB) {
			t.Fatal("unjudged lineup rules should never report a need")
		}
	})
}
