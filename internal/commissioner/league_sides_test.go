package commissioner

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

// handMirror is an empty 16-team app league whose commissioner sets per-side pools by hand: four divisions,
// two in each conference, then a player pool per side. Each change passes the store's own check.
func handMirror(t *testing.T) *ledgerBuilder {
	t.Helper()
	b := nativeLedger(t, 16, nil)
	for _, c := range [][2]string{
		{"league.divisions", `4`},
		{"league.divisionConferences", `{"Division 1":"AFC","Division 2":"AFC","Division 3":"NFC","Division 4":"NFC"}`},
		{poolsField, `"per_side"`},
	} {
		if err := b.change(c[0], c[1], 2); err != nil {
			t.Fatalf("%s: %v", c[0], err)
		}
	}
	return b
}

// mirrorLeague is a league created from the 16-Team Mirror preset and started the way creation starts it.
func mirrorLeague(t *testing.T) *ledgerBuilder {
	t.Helper()
	b := &ledgerBuilder{t: t}
	b.add(presetLeague(t, PresetMirror16, 0)[0].Command, seasonMoment(1))
	seed, err := NativeStart("native.test", b.records, seasonMoment(1))
	if err != nil {
		t.Fatal(err)
	}
	b.add(RawCommand{League: "native.test", Command: NativeSeed, Args: RawArgs{Native: &seed}, Request: "seed",
		Effective: "now", Visibility: "league"}, seasonMoment(1))
	return b
}

func sidesOf(t *testing.T, b *ledgerBuilder) map[string]string {
	t.Helper()
	sides, err := LeagueSides(b.records)
	if err != nil {
		t.Fatal(err)
	}
	return sides
}

func countBySide(sides map[string]string) map[string]int {
	out := map[string]int{}
	for _, side := range sides {
		out[side]++
	}
	return out
}

// Each conference is a side, whether the league starts from the 16-Team Mirror or is set by hand.
func TestSidesAreCountedFromTheConferences(t *testing.T) {
	for name, b := range map[string]*ledgerBuilder{"preset": mirrorLeague(t), "by hand": handMirror(t)} {
		sides := sidesOf(t, b)
		if got := countBySide(sides); len(sides) != 16 || got["AFC"] != 8 || got["NFC"] != 8 {
			t.Fatalf("%s: sides %v", name, sides)
		}
		if sides["franchise.01"] != "AFC" || sides["franchise.16"] != "NFC" {
			t.Fatalf("%s: sides %v", name, sides)
		}
		reading, err := RulesReading("native.test", b.records, seasonMoment(3))
		if err != nil {
			t.Fatal(err)
		}
		for _, row := range rows(reading) {
			if row.Field == sidesField && (!strings.HasPrefix(row.Value, "AFC: Franchise 1, Franchise 2") ||
				!strings.Contains(row.Value, "; NFC: Franchise 9,") || row.Citation != sidesCounted) {
				t.Fatalf("%s: sides row %+v", name, row)
			}
			if row.Field == poolsField && (row.Value != "One per side: each conference has its own copy of every player" || !row.NowOnly) {
				t.Fatalf("%s: pools row %+v", name, row)
			}
		}
	}
	single := nativeLedger(t, 4, nil)
	if sides := sidesOf(t, single); sides != nil {
		t.Fatalf("an empty league has sides %v", sides)
	}
	if sides := sidesOf(t, newLedger(t, PresetDefault32)); sides != nil {
		t.Fatalf("the 32-Team Default has sides %v", sides)
	}
}

func TestSidesAndPoolsRefusals(t *testing.T) {
	played := func(b *ledgerBuilder) {
		b.add(RawCommand{League: "native.test", Command: CapAdjust, Request: "played", Effective: "now"}, seasonMoment(3))
	}
	for _, c := range []struct {
		name, field, value, want string
		setup                    func(*ledgerBuilder)
	}{
		{"sides are counted", sidesField, `{"franchise.01":"NFC"}`, "sides are counted from the conferences", nil},
		{"pools after play", poolsField, `"one"`, "player pools can't change once the season has games", played},
		{"a team crosses after play", alignmentField, moved("Division 3"),
			"sides are fixed once the season has games", played},
		{"a side renamed after play", conferencesField,
			`{"Division 1":"East","Division 2":"East","Division 3":"NFC","Division 4":"NFC"}`,
			"would move a team to another side or rename a side", played},
		{"a team without a side", conferencesField, `{"Division 1":"AFC","Division 2":"AFC","Division 3":"NFC"}`,
			"every team needs a side", nil},
		{"one side", conferencesField,
			`{"Division 1":"AFC","Division 2":"AFC","Division 3":"AFC","Division 4":"AFC"}`, "at least two sides", nil},
	} {
		b := handMirror(t)
		if c.setup != nil {
			c.setup(b)
		}
		err := b.change(c.field, c.value, 4)
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: %v, want %q", c.name, err, c.want)
		}
	}
	b := handMirror(t)
	played(b)
	if err := b.change("league.alignment", moved("Division 2"), 4); err != nil {
		t.Fatalf("a move within a side after play: %v", err)
	}
	empty := nativeLedger(t, 16, nil)
	if err := empty.change(poolsField, `"per_side"`, 2); err == nil || !strings.Contains(err.Error(), "every team needs a side") {
		t.Fatalf("per side with no conferences: %v", err)
	}
	if err := empty.change(poolsField, `"per_side"`, 2); err == nil {
		t.Fatal("refused twice")
	}
	imported := newLedger(t, PresetLegacy)
	if err := imported.change(poolsField, `"per_side"`, 2); err == nil || !strings.Contains(err.Error(), "imported league") {
		t.Fatalf("imported: %v", err)
	}
	later := handMirror(t)
	raw := ruleRaw(poolsField, `"one"`, "next_rollover", "Agreed")
	if err := CheckRulesAppend(raw, later.records, seasonMoment(3)); err == nil || !strings.Contains(err.Error(), "takes effect now") {
		t.Fatalf("pools later: %v", err)
	}
	if !leaguerules.PoolsPerSide(rulesAt(t, later)) {
		t.Fatal("the mirror is not per side")
	}
}

// moved is the hand league's alignment with franchise.01 moved to another division.
func moved(division string) string {
	alignment := leaguerules.EvenDivisions(franchiseIDs(16), 4)
	alignment["franchise.01"] = division
	body, _ := json.Marshal(alignment)
	return string(body)
}

func rulesAt(t *testing.T, b *ledgerBuilder) leaguerules.RuleSet {
	t.Helper()
	rules, err := RulesNow(b.records, seasonMoment(3))
	if err != nil {
		t.Fatal(err)
	}
	return rules
}
