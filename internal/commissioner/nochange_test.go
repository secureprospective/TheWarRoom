package commissioner

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/governance"
)

func noChangeTime() time.Time { return time.Date(2026, 10, 14, 12, 0, 0, 0, time.UTC) }

func legacyLeague(t *testing.T) []Record {
	t.Helper()
	raw, err := (RawCommand{League: "native.test", Command: Create,
		Args:   RawArgs{Name: "Ours", Preset: PresetLegacy, FranchiseCount: 32},
		Reason: "Start", Request: "create", Effective: "now", Visibility: "league"}).Prepared()
	if err != nil {
		t.Fatal(err)
	}
	return []Record{{Command: raw, Principal: "creator", Role: "commissioner",
		Receipt: Receipt{Entry: "create", Revision: 1, RecordedAt: noChangeTime().Add(-time.Hour)}}}
}

func inForce(t *testing.T, records []Record, field string) string {
	t.Helper()
	value, err := ruleValue(records, field)
	if err != nil || value == "null" {
		t.Fatalf("%s is not set in the preset: %q %v", field, value, err)
	}
	return value
}

func TestARuleChangeThatChangesNothingIsRefused(t *testing.T) {
	records := legacyLeague(t)
	same := inForce(t, records, "roster.limits")
	other := `{"min":20,"max":70}`
	if same == other {
		t.Fatal("test value must differ from the preset")
	}
	waiting := append(append([]Record(nil), records...), Record{Command: ruleRaw("roster.limits", other, "next_rollover", ""),
		Receipt: Receipt{Entry: "waiting", Revision: 2, RecordedAt: noChangeTime().Add(-time.Minute)}})
	intervening := append(append([]Record(nil), waiting...), Record{
		Command: ruleRaw("roster.limits", `{"min":21,"max":71}`, "now", "Agreed"),
		Receipt: Receipt{Entry: "intervening", Revision: 3, RecordedAt: noChangeTime()}})
	for _, c := range []struct {
		name             string
		records          []Record
		value, effective string
		want             string
	}{
		{"now, the value in force", records, same, "now", "This is already the rule. Nothing would change"},
		{"next rollover, the value in force", records, same, "next_rollover",
			"This is already the rule. Nothing would change"},
		{"next rollover, the value already waiting", waiting, other, "next_rollover",
			"This change is already waiting for the next rollover. Nothing would change"},
		{"waiting value after an intervening immediate change", intervening, other, "next_rollover",
			"This change is already waiting for the next rollover. Nothing would change"},
		{"now, the value in force while another waits", waiting, same, "now",
			"This is already the rule. Nothing would change"},
		{"a different value now", records, other, "now", ""},
		{"a different value for the next rollover", records, other, "next_rollover", ""},
		{"the value in force, replacing one that waits", waiting, same, "next_rollover", ""},
	} {
		t.Run(c.name, func(t *testing.T) {
			raw := ruleRaw("roster.limits", c.value, c.effective, "Agreed")
			err := CheckRulesAppend(raw, c.records, noChangeTime())
			if c.want == "" {
				if err != nil {
					t.Fatalf("a real change was refused: %v", err)
				}
				return
			}
			var refusal *Refusal
			if !errors.As(err, &refusal) || refusal.Reason() != c.want {
				t.Fatalf("want refusal %q, got %v", c.want, err)
			}
		})
	}
}

func TestARoleChangeThatChangesNothingIsRefused(t *testing.T) {
	records := legacyLeague(t)
	role := func(command, person string, held governance.Role) RawCommand {
		return RawCommand{League: "native.test", Command: command, Reason: "Agreed", Expected: 1,
			Request: "role", Effective: "now", Visibility: "private",
			Args: RawArgs{People: &RawPeople{Principal: person, Role: held}}}
	}
	given := append(append([]Record(nil), records...), Record{Command: role(roleGrant, "deputy", governance.DOT),
		Principal: "creator", Receipt: Receipt{Entry: "dot", Revision: 2, RecordedAt: noChangeTime()}})
	for _, c := range []struct {
		name    string
		records []Record
		raw     RawCommand
		want    string
	}{
		{"give the role the person holds", given, role(roleGrant, "deputy", governance.DOT),
			"This person already holds that role. Nothing would change"},
		{"give a founder the role they founded", records, role(roleGrant, "creator", governance.Commissioner),
			"This person already holds that role. Nothing would change"},
		{"take a role they do not hold", records, role(roleRevoke, "deputy", governance.DOT),
			"This person does not hold that role. Nothing would change"},
		{"give a new role", records, role(roleGrant, "deputy", governance.DOT), ""},
		{"take a role they hold", given, role(roleRevoke, "deputy", governance.DOT), ""},
	} {
		t.Run(c.name, func(t *testing.T) {
			err := CheckGovernanceAppend(c.raw, "creator", c.records, noChangeTime())
			if c.want == "" {
				if err != nil {
					t.Fatalf("a real change was refused: %v", err)
				}
				return
			}
			var refusal *Refusal
			if !errors.As(err, &refusal) || refusal.Reason() != c.want {
				t.Fatalf("want refusal %q, got %v", c.want, err)
			}
		})
	}
}

func TestEveryCommandSaysWhichConfirmItNeeds(t *testing.T) {
	seen := map[string]string{}
	for _, row := range CommandSafeguards() {
		if row.Safeguard != "confirm" && row.Safeguard != "hold" && row.Safeguard != "reason" {
			t.Fatalf("unknown confirm: %+v", row)
		}
		seen[row.Command+"/"+row.Effective+"/"+row.Safeguard] = row.Field
	}
	for _, want := range []string{Create + "/now/reason", Revert + "/now/reason",
		"commish.franchise.edit/now/hold", roleGrant + "/now/reason", "commish.vote.cast/now/hold",
		"commish.succession.activity/now/confirm", RulesSet + "/now/reason",
		RulesSet + "/next_rollover/hold", RulesSet + "/next_rollover/confirm"} {
		if _, ok := seen[want]; !ok {
			t.Errorf("the list of commands lacks %s", want)
		}
	}
	for _, d := range peopleDefinitions() {
		if _, ok := seen[d.command+"/now/"+safeguardFor(d.gravity)]; !ok {
			t.Errorf("%s is not listed with its own gravity", d.command)
		}
	}
	for command := range successionDefinitions() {
		if !strings.Contains(strings.Join(mapKeys(seen), " "), command+"/") {
			t.Errorf("%s is not listed", command)
		}
	}
}

func mapKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	return keys
}

func TestAPreviewDecodeLeavesOutOnlyTheReason(t *testing.T) {
	raw := ruleRaw("roster.limits", `{"min":20,"max":70}`, "now", "")
	body, err := json.Marshal(raw)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Decode(string(body)); err == nil {
		t.Fatal("a change needing a reason was accepted without one")
	}
	read, err := DecodePreview(string(body))
	if err != nil || read.MissingReason() == nil {
		t.Fatalf("a dry run must read it and say the reason is owed: %v", err)
	}
	bad := strings.Replace(string(body), `"roster.limits"`, `"nothing.here"`, 1)
	if _, err := DecodePreview(bad); err == nil {
		t.Fatal("a dry run skipped a check other than the reason")
	}
}
