package leaguerules

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestAutomaticReserveDefaultsAndOldSeedUpgrade(t *testing.T) {
	legacy, def := LegacyNFL(), Default32()
	if legacy.Taxi.AutoPromote.Value || legacy.IR.AutoReturn.Value || !def.Taxi.AutoPromote.Value || !def.IR.AutoReturn.Value {
		t.Fatal("automatic reserve defaults changed")
	}
	seed := SeedChanges(legacy, "old", 2026, testTime())
	old := []Change{}
	for _, c := range seed {
		if c.Field != "taxi.autoPromote" && c.Field != "ir.autoReturn" {
			old = append(old, c)
		}
	}
	before := append([]Change{}, old...)
	got := UpgradeSeed(old)
	if !reflect.DeepEqual(before, old) {
		t.Fatal("recorded seed mutated")
	}
	count := 0
	for _, c := range got {
		if c.Field == "taxi.autoPromote" || c.Field == "ir.autoReturn" {
			count++
			v, ok := c.New.(Value[bool])
			if !ok || v.State != IsKnown || v.Value {
				t.Fatalf("old league opted in: %+v", c)
			}
		}
	}
	if count != 2 || !reflect.DeepEqual(UpgradeSeed(got), got) {
		t.Fatal("upgrade missing or not idempotent")
	}
}

func TestReserveRuleJSONKeepsLegacyAndExplicitEdits(t *testing.T) {
	for _, r := range []RuleSet{LegacyNFL(), Default32()} {
		body, err := json.Marshal(r)
		if err != nil {
			t.Fatal(err)
		}
		var got RuleSet
		if err := json.Unmarshal(body, &got); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(r, got) {
			t.Fatal("reserve defaults lost on JSON roundtrip")
		}
	}
	r := LegacyNFL()
	r.IR.AutoReturn = AutoMoveValue(cited(false, "Commissioner chose manual returns"))
	body, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	var got RuleSet
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatal(err)
	}
	if got.IR.AutoReturn != r.IR.AutoReturn {
		t.Fatal("explicit off edit lost its citation")
	}
}
