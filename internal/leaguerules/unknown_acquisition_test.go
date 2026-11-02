package leaguerules

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"
)

func TestUnknownAcquisitionHasNoDefault(t *testing.T) {
	t.Parallel()
	for name, rules := range map[string]RuleSet{"empty": {}, "legacy": LegacyNFL(), "default": Default32(), "mirror": Mirror16()} {
		if rules.Contracts.UnknownAcquisition.State != NeedsRuling || !reflect.ValueOf(rules.Contracts.UnknownAcquisition.Value).IsZero() {
			t.Fatalf("%s invented an unknown-acquisition contract", name)
		}
	}
	field, ok := Field("contracts.unknown_acquisition")
	if !ok || field.Schema.Kind != "record" || field.Section != "Contracts" {
		t.Fatalf("setting is not in the ordinary catalogue: %+v", field)
	}
}

func TestUnknownAcquisitionSetting(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, body string
		bad        bool
	}{
		{"fixed", `{"years":2,"salaryRule":"fixed","salary":33000000}`, false},
		{"explicit zero salary", `{"years":1,"salaryRule":"fixed","salary":0}`, false},
		{"lowest minimum", `{"years":1,"salaryRule":"lowest_minimum"}`, false},
		{"experience minimum", `{"years":3,"salaryRule":"experience_minimum"}`, false},
		{"missing fixed salary", `{"years":1,"salaryRule":"fixed"}`, true},
		{"missing years", `{"salaryRule":"lowest_minimum"}`, true},
		{"no years", `{"years":0,"salaryRule":"lowest_minimum"}`, true},
		{"calendar overflow", `{"years":10000,"salaryRule":"lowest_minimum"}`, true},
		{"unrecognized rule", `{"years":1,"salaryRule":"guess"}`, true},
		{"minimum plus fixed salary", `{"years":1,"salaryRule":"experience_minimum","salary":100}`, true},
		{"negative salary", `{"years":1,"salaryRule":"fixed","salary":-1}`, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			setting, err := DecodeSetting("contracts.unknown_acquisition", json.RawMessage(tc.body))
			if (err != nil) != tc.bad {
				t.Fatalf("got %v, want bad %t", err, tc.bad)
			}
			if tc.bad {
				return
			}
			at := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
			change := Change{LeagueID: "native.any", Field: "contracts.unknown_acquisition", New: setting, Effective: Effective{Kind: Now}, RecordedAt: at, Season: 2026}
			rules, err := Resolve([]Change{change}, 2026, at)
			if err != nil || !reflect.DeepEqual(rules.Contracts.UnknownAcquisition, setting) {
				t.Fatalf("ordinary rule flow lost setting: %+v, %v", rules.Contracts.UnknownAcquisition, err)
			}
			stored, err := EncodeStored(setting)
			if err != nil {
				t.Fatal(err)
			}
			restored, err := DecodeStored(change.Field, stored)
			if err != nil || !reflect.DeepEqual(restored, setting) {
				t.Fatalf("stored setting changed: %v", err)
			}
			before, err := Resolve([]Change{change}, 2026, at.Add(-time.Second))
			if err != nil || before.Contracts.UnknownAcquisition.State != NeedsRuling {
				t.Fatalf("setting applied early: %v", err)
			}
		})
	}
}
