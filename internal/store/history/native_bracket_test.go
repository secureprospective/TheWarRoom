package history

import (
	"fmt"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
)

type change = commissioner.RawBracketChange

// rulesBuild builds the bracket from the league's own playoff rules.
func rulesBuild() change { return change{Preset: "league_rules"} }

// setRule records a rule change through the real command and returns its receipt.
func (f *bracketFixture) setRule(t *testing.T, field, value string) commissioner.Receipt {
	t.Helper()
	return f.run(t, ruleCommand(field, value, "now", "Change the playoff rules", f.rev(t)))
}

func TestPlayoffRulesCM016GovernTheWinnerAndReopen(t *testing.T) {
	t.Parallel()
	f := bracketTest(t)
	f.do(t, commissioner.BracketBuild, rulesBuild())
	const game = "AFC/wildcard_3"
	if g := gameIn(f.bracket(t), game); g.Winner != "franchise.11" || g.HomeBonus != 30000 {
		t.Fatalf("baseline: %+v", g)
	}
	receipt := f.setRule(t, "playoffs.homeBonus", "50000")
	for range 2 {
		g := gameIn(f.bracket(t), game)
		if g.Winner != "franchise.08" || g.HomeBonus != 50000 || !strings.Contains(g.Words, "5 home field points") {
			t.Fatalf("five home field points must save the home team: %+v", g)
		}
		f.reopen(t)
	}
	f.undo(t, receipt.Entry)
	for range 2 {
		if g := gameIn(f.bracket(t), game); g.Winner != "franchise.11" || g.HomeBonus != 30000 {
			t.Fatalf("compensation did not restore the bonus: %+v", g)
		}
		f.reopen(t)
	}
}

func TestPlayoffRulesCM016SayWhatIsStillNeededAndReopen(t *testing.T) {
	t.Parallel()
	f := bracketTest(t)
	f.do(t, commissioner.BracketBuild, rulesBuild())
	cases := []struct{ field, value, words string }{
		{"playoffs.reseed", "false", "pairs teams by the seeds left, which is reseeding"},
		{"playoffs.finalWeek", "18", "the playoff weeks are 14 to 18"},
		{"playoffs.wildCards", "2", "4 division winners and 2 wild cards do not make 7 teams"},
		{"playoffs.byeSeed", "7", "cannot be negative or leave no team to play"},
	}
	for _, c := range cases {
		receipt := f.setRule(t, c.field, c.value)
		for range 2 {
			b := f.bracket(t)
			if b.Status != "Needs a ruling" || !strings.HasPrefix(b.Words, "Needs a ruling: ") || !strings.Contains(b.Words, c.words) ||
				len(b.Rounds) != 0 {
				t.Fatalf("%s: %s %q rounds %d", c.field, b.Status, b.Words, len(b.Rounds))
			}
			f.reopen(t)
		}
		f.undo(t, receipt.Entry)
		f.reopen(t)
		if b := f.bracket(t); b.Status != "Built" || len(b.Rounds) != 4 {
			t.Fatalf("%s: compensation did not restore the bracket: %s %q", c.field, b.Status, b.Words)
		}
	}
}

func TestBracketBuildCM046GovernsTheRoundsAndReopens(t *testing.T) {
	t.Parallel()
	f := bracketTest(t)
	ready := f.bracket(t)
	if ready.Status != "Ready to build" || len(ready.Rounds) != 0 || ready.Preset != "league_rules" {
		t.Fatalf("before building: %s %q", ready.Status, ready.Words)
	}
	for _, conference := range []string{"AFC", "NFC"} {
		seats := seedsOf(ready, conference)
		if len(seats) != 7 {
			t.Fatalf("%s sends %d teams", conference, len(seats))
		}
		divisions := map[string]bool{}
		for i, s := range seats {
			if (i < 4) != (s.Kind == "Division winner") || s.Bye != (i == 0) || s.Seed != i+1 {
				t.Fatalf("%s seed %d: %+v", conference, i+1, s)
			}
			if i < 4 {
				divisions[s.Division] = true
			}
		}
		if len(divisions) != 4 {
			t.Fatalf("%s: division winners must come from four divisions: %+v", conference, seats[:4])
		}
	}
	receipt := f.do(t, commissioner.BracketBuild, rulesBuild())
	for range 2 {
		b := f.bracket(t)
		assertLegacyShape(t, b)
		f.reopen(t)
	}
	f.undo(t, receipt.Entry)
	for range 2 {
		if b := f.bracket(t); b.Status != "Ready to build" || len(b.Rounds) != 0 {
			t.Fatalf("compensation did not take the bracket back: %s", b.Status)
		}
		f.reopen(t)
	}
}

func assertLegacyShape(t *testing.T, b commissioner.NativeBracket) {
	t.Helper()
	if b.Status != "Built" || len(b.Rounds) != 4 || len(b.Rounds[0].Games) != 6 || len(b.Rounds[1].Games) != 4 ||
		len(b.Rounds[2].Games) != 2 || len(b.Rounds[3].Games) != 1 {
		t.Fatalf("shape: %s %q", b.Status, b.Words)
	}
	for _, g := range b.Rounds[0].Games {
		if g.Home.Seed == 1 || g.Away.Seed == 1 {
			t.Fatalf("the top seed plays in the bye round: %+v", g)
		}
		if g.Home.Seed < 2 || g.Home.Seed > 4 || g.Home.Seed+g.Away.Seed != 9 {
			t.Fatalf("round one pairs 2-7, 3-6 and 4-5: %+v", g)
		}
	}
	for _, conference := range []string{"AFC", "NFC"} {
		top := seedsOf(b, conference)[0].Franchise
		if d := gameIn(b, conference+"/division_1"); d.Home.Franchise != top || d.Away.Seed < 5 {
			t.Fatalf("the top seed hosts the lowest seed left: %+v", d)
		}
	}
}

func TestBracketHandSeedAndEditCM046GovernAndReopen(t *testing.T) {
	t.Parallel()
	f := bracketTest(t)
	f.do(t, commissioner.BracketBuild, rulesBuild())
	base := f.bracket(t)
	if gameIn(base, "AFC/division_1").Home.Franchise != "franchise.16" {
		t.Fatalf("baseline AFC top seed: %+v", seedsOf(base, "AFC")[0])
	}
	seed := f.do(t, commissioner.BracketSeed, change{Slot: 1, Team: "franchise.01"})
	for range 2 {
		b := f.bracket(t)
		top := seedsOf(b, "AFC")[0]
		if top.Franchise != "franchise.01" || !top.ByHand || top.Kind != "Set by the commissioner" || !top.Bye ||
			gameIn(b, "AFC/division_1").Home.Franchise != "franchise.01" || seedsOf(b, "AFC")[1].Franchise != "franchise.16" {
			t.Fatalf("a hand seed must take its slot and the rest follow: %+v", seedsOf(b, "AFC")[:2])
		}
		if seedsOf(b, "NFC")[0].ByHand {
			t.Fatal("a hand seed in one conference marked the other")
		}
		f.reopen(t)
	}
	f.undo(t, seed.Entry)
	f.reopen(t)
	if top := seedsOf(f.bracket(t), "AFC")[0]; top.Franchise != "franchise.16" || top.ByHand {
		t.Fatalf("compensation did not restore the seed: %+v", top)
	}
	edit := f.do(t, commissioner.BracketEdit, change{Node: "AFC/wildcard_2/away", Team: "franchise.02"})
	for range 2 {
		g := gameIn(f.bracket(t), "AFC/wildcard_2")
		if g.Away.Franchise != "franchise.02" || !g.Away.ByHand || g.Home.ByHand {
			t.Fatalf("a hand edit must replace one place and show it: %+v", g)
		}
		f.reopen(t)
	}
	f.undo(t, edit.Entry)
	f.reopen(t)
	if g := gameIn(f.bracket(t), "AFC/wildcard_2"); g.Away.Franchise == "franchise.02" || g.Away.ByHand {
		t.Fatalf("compensation did not restore the place: %+v", g)
	}
}

func TestBracketPresetsCM046SaveBuildAndReopen(t *testing.T) {
	t.Parallel()
	f := bracketTest(t)
	short := []commissioner.BracketNodeSpec{
		{ID: "semi_1", Week: 14, HomeSeed: 1, AwaySeed: 4}, {ID: "semi_2", Week: 14, HomeSeed: 2, AwaySeed: 3},
		{ID: "conference", Week: 16, HomeFrom: "winner of semi_1", AwayFrom: "winner of semi_2"},
		{ID: "championship", Week: 17, HomeFrom: "conference winner", AwayFrom: "conference winner"}}
	f.do(t, commissioner.BracketBuild, rulesBuild())
	save := f.do(t, commissioner.BracketPresetSave, change{Name: "short", Shape: short})
	f.reopen(t)
	if b := f.bracket(t); len(b.Presets) != 1 || b.Presets[0] != "short" || b.Preset != "league_rules" {
		t.Fatalf("a saved shape is kept and does not change the bracket: %v %s", b.Presets, b.Preset)
	}
	build := f.do(t, commissioner.BracketBuild, change{Preset: "short"})
	for range 2 {
		b := f.bracket(t)
		if b.Status != "Built" || b.Preset != "short" || len(b.Rounds) != 3 || len(b.Rounds[0].Games) != 4 ||
			gameIn(b, "AFC/semi_1").Away.Seed != 4 || gameIn(b, "championship").Home == nil {
			t.Fatalf("the saved shape must drive the games: %s %q", b.Status, b.Words)
		}
		f.reopen(t)
	}
	f.undo(t, build.Entry)
	if b := f.bracket(t); b.Preset != "league_rules" || len(b.Rounds) != 4 {
		t.Fatalf("compensation did not bring the league's own shape back: %s", b.Preset)
	}
	f.undo(t, save.Entry)
	f.reopen(t)
	if b := f.bracket(t); len(b.Presets) != 0 {
		t.Fatalf("a reversed save left a shape: %v", b.Presets)
	}
	f.do(t, commissioner.BracketPresetSave, change{Name: "short", Shape: short})
	f.do(t, commissioner.BracketBuild, change{Preset: "short"})
	f.do(t, commissioner.BracketBuild, change{Preset: "legacy_nfl"})
	if b := f.bracket(t); b.Preset != "legacy_nfl" || len(b.Rounds) != 4 {
		t.Fatalf("the built-in Legacy shape: %s", b.Preset)
	}
}

func TestAdvancementCM047GovernsTheNextRoundAndReopens(t *testing.T) {
	t.Parallel()
	f := bracketTest(t)
	f.do(t, commissioner.BracketBuild, rulesBuild())
	base := f.bracket(t)
	if d := gameIn(base, "AFC/division_1"); d.Away.Franchise != "franchise.11" {
		t.Fatalf("baseline: %+v", d)
	}
	raw := f.bracketCommand(t, commissioner.BracketAdvance, change{Game: "AFC/wildcard_3", Team: "franchise.08"}, "")
	if msg := f.refused(t, raw, "creator"); !strings.Contains(msg, "reason") {
		t.Fatalf("an advancement needs a reason: %s", msg)
	}
	raw.Reason = "The scorer missed a touchdown"
	raw.Gravity = "G2"
	f.refused(t, raw, "creator")
	raw.Gravity = ""
	receipt := f.run(t, raw)
	for range 2 {
		b := f.bracket(t)
		g := gameIn(b, "AFC/wildcard_3")
		if g.Winner != "franchise.08" || !g.ByHand || g.Status != "Set by the commissioner" ||
			!strings.Contains(g.Words, "despite the computed result") {
			t.Fatalf("the named team advances and the game says so: %+v", g)
		}
		if d := gameIn(b, "AFC/division_1"); d.Away.Franchise != "franchise.15" || d.Home.Franchise != "franchise.16" {
			t.Fatalf("the next round must re-pair from the advanced team: %+v", d)
		}
		if d := gameIn(b, "AFC/division_2"); d.Home.Franchise != "franchise.12" || d.Away.Franchise != "franchise.08" {
			t.Fatalf("the other game takes the rest: %+v", d)
		}
		f.reopen(t)
	}
	f.undo(t, receipt.Entry)
	for range 2 {
		b := f.bracket(t)
		if g := gameIn(b, "AFC/wildcard_3"); g.Winner != "franchise.11" || g.ByHand {
			t.Fatalf("compensation did not restore the computed winner: %+v", g)
		}
		if d := gameIn(b, "AFC/division_1"); d.Away.Franchise != "franchise.11" {
			t.Fatalf("compensation did not re-pair: %+v", d)
		}
		f.reopen(t)
	}
	other := f.bracketCommand(t, commissioner.BracketAdvance, change{Game: "AFC/wildcard_3", Team: "franchise.20"}, "Wrong team")
	f.run(t, other)
	if g := gameIn(f.bracket(t), "AFC/wildcard_3"); g.Winner != "franchise.11" || !strings.Contains(g.Words, "set aside") {
		t.Fatalf("a team outside the game cannot advance from it: %+v", g)
	}
}

func TestTheTitleCM048IsStampedWithTheSeasonAndReopens(t *testing.T) {
	t.Parallel()
	f := bracketTest(t)
	f.do(t, commissioner.BracketBuild, rulesBuild())
	title := func(label string) commissioner.RawCommand {
		return commissioner.RawCommand{League: bracketLeague, Command: commissioner.TitleSet, Reason: "", Expected: f.rev(t),
			Args:    commissioner.RawArgs{Title: &commissioner.RawTitleChange{Season: 2026, Label: label}},
			Request: "title." + label + fmt.Sprint(f.rev(t)), Effective: "now", Visibility: "league"}
	}
	b := f.bracket(t)
	if b.Champion == nil || b.Champion.Franchise != "franchise.32" || b.Champion.Label != "2026 League Champion" ||
		b.Title.Label != "2026 League Champion" || !b.Title.Default {
		t.Fatalf("default title: %+v %+v", b.Champion, b.Title)
	}
	receipt := f.run(t, title("{year} Legacy Bowl Champions"))
	for range 2 {
		b := f.bracket(t)
		if b.Champion.Label != "2026 Legacy Bowl Champions" || b.Title.Default || b.Champion.Franchise != "franchise.32" {
			t.Fatalf("the title must carry the season's own year: %+v", b.Champion)
		}
		f.reopen(t)
	}
	if msg := f.refused(t, title("2025 Super Bowl Champion"), "creator"); !strings.Contains(msg, "names 2025") {
		t.Fatalf("a stale year: %s", msg)
	}
	f.undo(t, receipt.Entry)
	for range 2 {
		if b := f.bracket(t); b.Champion.Label != "2026 League Champion" || !b.Title.Default {
			t.Fatalf("compensation did not restore the default: %+v", b.Champion)
		}
		f.reopen(t)
	}
	future := title("Early")
	future.Args.Title.Season = 2027
	if msg := f.refused(t, future, "creator"); !strings.Contains(msg, "has not been played") {
		t.Fatalf("a title for a season not yet played: %s", msg)
	}
	f.run(t, title("Legacy Champions"))
	f.run(t, title(""))
	if b := f.bracket(t); !b.Title.Default || b.Champion.Label != "2026 League Champion" {
		t.Fatalf("an empty label puts the default back: %+v", b.Title)
	}
}
