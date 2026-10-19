package commissioner

import (
	"slices"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/governance"
)

func entries(c Controls) map[string]ControlEntry {
	out := map[string]ControlEntry{}
	for _, g := range c.Groups {
		for _, e := range g.Controls {
			out[e.ID] = e
		}
	}
	return out
}

func TestEveryCensusRowSitsInExactlyOneLandingGroup(t *testing.T) {
	reading := ControlsReading()
	labels := []string{}
	seen := map[string]int{}
	for _, g := range reading.Groups {
		labels = append(labels, g.Label)
		for _, e := range g.Controls {
			seen[e.ID]++
		}
	}
	want := []string{"Constitution", "People", "Calendar", "Season ops", "Draft", "Playoffs", "League ops", "Ledger"}
	if !slices.Equal(labels, want) {
		t.Fatalf("landing groups: %v", labels)
	}
	for _, row := range governance.CensusControls() {
		if seen[row.ID] != 1 {
			t.Fatalf("%s sits in %d groups", row.ID, seen[row.ID])
		}
	}
	if len(seen) != len(governance.CensusControls()) {
		t.Fatalf("a group holds a row the census does not: %v", seen)
	}
}

func TestOnlyControlsAPersonCanUseAreBuilt(t *testing.T) {

	unbuilt := []string{"CM-048", "CM-049", "CM-050", "CM-055"}
	for id, entry := range entries(ControlsReading()) {
		want := !slices.Contains(unbuilt, id)
		if entry.Built != want {
			t.Errorf("%s built = %t; want %t", id, entry.Built, want)
		}
	}

}

func TestEveryBuiltRuleRowOwnsItsFieldsAndNoOtherRowDoes(t *testing.T) {
	owned := map[string]string{}
	for id, e := range entries(ControlsReading()) {
		for _, f := range e.Fields {
			if prior, dup := owned[f.Field]; dup {
				t.Fatalf("%s is owned by %s and %s", f.Field, prior, id)
			}
			owned[f.Field] = id
			if f.Label == "" {
				t.Fatalf("%s has no plain label", f.Field)
			}
		}
	}
	if owned["calendar.timezone"] != "CM-024" || owned["roster.limits"] != "CM-003" {
		t.Fatalf("fields follow the census owner: %v", owned)
	}
}
