package commissioner

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/governance"
	"github.com/secureprospective/TheWarRoom/internal/leaguerules"
)

func TestEveryRuleFieldHasOneCensusOwner(t *testing.T) {
	census := map[string]bool{}
	for _, row := range governance.CensusControls() {
		census[row.ID] = true
	}
	used := map[string]bool{}
	for _, f := range leaguerules.Catalogue() {
		owners := 0
		for _, owner := range ruleOwners() {
			if owner.owns(f.Path) {
				owners++
				used[owner.prefix] = true
				if !census[owner.control] {
					t.Fatalf("%s maps to %s, which is not a census row", f.Path, owner.control)
				}
			}
		}
		if owners != 1 {
			t.Fatalf("%s has %d census owners", f.Path, owners)
		}
	}
	for _, owner := range ruleOwners() {
		if !used[owner.prefix] {
			t.Fatalf("ownership row %s matches no rule field", owner.prefix)
		}
	}
}

func ruleRaw(field, value, effective, reason string) RawCommand {
	return RawCommand{League: "native.test", Command: RulesSet, Args: RawArgs{Field: field,
		Value: json.RawMessage(value)}, Reason: reason, Expected: 1, Request: "rule",
		Effective: effective, Visibility: "league"}
}

func TestRuleGravityComesFromTheCensus(t *testing.T) {
	for _, c := range []struct{ field, effective, gravity string }{
		{"roster.limits", "next_rollover", "G2"}, {"roster.limits", "now", "G3"},
		{"house.texts", "now", "G1"}, {"calendar.rollover", "next_rollover", "G3"},
	} {
		value := `{"min":20,"max":70}`
		switch c.field {
		case "house.texts":
			value = `{"note":"Be kind"}`
		case "calendar.rollover":
			value = `"2027-03-01T05:00:00Z"`
		}
		d, err := ruleRaw(c.field, value, c.effective, "Agreed at the owners meeting").Definition()
		if err != nil || d.Gravity != c.gravity {
			t.Fatalf("%s %s: %+v %v", c.field, c.effective, d, err)
		}
	}
	raw := ruleRaw("roster.limits", `{"min":20,"max":70}`, "now", " ")
	if err := raw.Validate(); err == nil || !strings.Contains(err.Error(), "reason") {
		t.Fatalf("apply now without a typed reason: %v", err)
	}
	body, err := json.Marshal(ruleRaw("roster.limits", `{"min":20,"max":70}`, "now", "Agreed"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Decode(string(body)); err != nil {
		t.Fatal(err)
	}
	for _, bad := range []string{
		strings.Replace(string(body), `"visibility"`, `"gravity":"G2","visibility"`, 1),
		strings.Replace(string(body), `"visibility"`, `"role":"commissioner","visibility"`, 1),
		strings.Replace(string(body), `"visibility"`, `"control":"CM-017","visibility"`, 1),
		strings.Replace(string(body), `"field"`, `"control":"CM-017","field"`, 1),
		strings.Replace(string(body), `"effective":"now"`, `"effective":"2027-03-01T00:00:00Z"`, 1),
		strings.Replace(string(body), `"effective":"now"`, `"effective":"instant"`, 1),
		strings.Replace(string(body), `"visibility":"league"`, `"visibility":"private"`, 1),
		strings.Replace(string(body), `"field"`, `"preset":"empty","field"`, 1),
	} {
		if _, err := Decode(bad); err == nil {
			t.Fatalf("forged rule command accepted: %s", bad)
		}
	}
}

func TestCreationIsExpandedByGo(t *testing.T) {
	legacy, err := RawCommand{League: "native.test", Command: Create, Args: RawArgs{Name: "Ours",
		Preset: PresetLegacy, Season: 2026}, Reason: "start", Request: "create", Effective: "now", Visibility: "league"}.Prepared()
	if err != nil || legacy.Args.FranchiseCount != 32 || len(legacy.Args.Franchises) != 32 ||
		legacy.Args.Franchises[0] != "franchise.01" || legacy.Args.Franchises[31] != "franchise.32" ||
		len(legacy.Args.Seed) != len(leaguerules.Catalogue()) {
		t.Fatalf("legacy expansion: %+v %v", legacy.Args, err)
	}
	if err := legacy.Validate(); err != nil {
		t.Fatal(err)
	}
	if again, err := legacy.Prepared(); err != nil || !sameSeed(again.Args.Seed, legacy.Args.Seed) {
		t.Fatalf("retry of Go's own expansion: %v", err)
	}
	forged := legacy
	forged.Args.Franchises = append([]string{"mine"}, legacy.Args.Franchises[1:]...)
	if _, err := forged.Prepared(); err == nil {
		t.Fatal("caller-chosen franchise accepted")
	}
	forged = legacy
	forged.Args.Seed = append([]SeedValue{{Field: "cap.amount", Value: json.RawMessage(
		`{"State":1,"Value":1,"Citation":""}`)}}, legacy.Args.Seed[1:]...)
	if _, err := forged.Prepared(); err == nil {
		t.Fatal("caller-chosen preset value accepted")
	}
	for _, args := range []RawArgs{
		{Name: "Ours", Preset: PresetEmpty},
		{Name: "Ours", Preset: PresetEmpty, FranchiseCount: 1}, {Name: "Ours", Preset: PresetEmpty, FranchiseCount: 33},
		{Name: "Ours", Preset: "mfl", FranchiseCount: 12}, {Name: "Ours", FranchiseCount: 12},
	} {
		raw := legacy
		raw.Args = args
		if _, err := raw.Prepared(); err == nil {
			t.Fatalf("invalid creation accepted: %+v", args)
		}
	}
	empty := legacy
	empty.Args = RawArgs{Name: "Ours", Preset: PresetEmpty, FranchiseCount: 12, Franchises: franchiseIDs(12),
		Seed: legacy.Args.Seed}
	if err := empty.Validate(); err == nil {
		t.Fatal("empty league stored with preset values")
	}
}

func TestStartingRuleSourcesArePlainWords(t *testing.T) {
	for _, c := range []struct{ preset, state, citation, want string }{
		{PresetLegacy, "Set", "Census CM-001", "Legacy NFL rules"},
		{PresetLegacy, "Set", "Census CM-002: carried at setup", "Legacy NFL rules"},
		{PresetLegacy, "Set", "fact R", "Legacy NFL rules"},
		{PresetLegacy, "Set", "Rulebook §4; Math §13", "Legacy NFL rules (rulebook section 4)"},
		{PresetLegacy, "Set", "Rulebook §15", "Legacy NFL rules (rulebook section 15)"},
		{PresetEmpty, "Set", "", "Empty league default"},
		{PresetLegacy, "Needs a ruling", "Census CM-001: only for keeper leagues",
			"Needs a ruling: only for keeper leagues"},
		{PresetLegacy, "Needs a ruling", "Census CM-041 Q-E1", "Needs a ruling"},
		{PresetEmpty, "Needs a ruling", "", "Needs a ruling"},
		{PresetEmpty, "Needs a ruling", "Needs a ruling; the commissioner sets the exact trade deadline.",
			"Needs a ruling; the commissioner sets the exact trade deadline."},
		{PresetLegacy, "Needs a ruling", "The commissioner sets the exact trade deadline.",
			"Needs a ruling: The commissioner sets the exact trade deadline."},
		{PresetLegacy, "Needs a ruling", "fact 12c: exact instant chosen by commissioner",
			"Needs a ruling: exact instant chosen by commissioner"},
		{PresetLegacy, "Needs a ruling", "fact 12c", "Needs a ruling"},
		{PresetEmpty, "Needs a ruling", "  Needs a ruling; the commissioner sets it.  ",
			"Needs a ruling; the commissioner sets it."},
		{PresetEmpty, "Needs a ruling", "  ", "Needs a ruling"},
	} {
		if got := startingSource(c.preset, c.state, c.citation); got != c.want {
			t.Errorf("%s %s %q: got %q, want %q", c.preset, c.state, c.citation, got, c.want)
		}
	}
	if rolloverWords != "The next rollover has no date yet; the commissioner sets it." {
		t.Fatal("rollover wording must not depend on implementation status")
	}
}
