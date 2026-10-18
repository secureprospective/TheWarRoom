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
	all := entries(ControlsReading())
	for _, id := range []string{"CM-001", "CM-006", "CM-017", "CM-021", "CM-023", "CM-024", "CM-025",
		"CM-040", "CM-041", "CM-043", "CM-044", "CM-045", "CM-051", "CM-052", "CM-053", "CM-054", "CM-056"} {
		if !all[id].Built {
			t.Errorf("%s is usable today and must say so", id)
		}
	}
	for _, id := range []string{"CM-018", "CM-019", "CM-020", "CM-022", "CM-026", "CM-028", "CM-031", "CM-037",
		"CM-042", "CM-046", "CM-055"} {
		if all[id].Built {
			t.Errorf("%s is not built and must not claim to be", id)
		}
	}
	for id, words := range partlyThere() {
		if all[id].Built || all[id].Note != words || words == "" {
			t.Errorf("%s is only partly there: %+v", id, all[id])
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
