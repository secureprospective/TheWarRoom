package leaguerules

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestCurrentRoundTripsThroughDecode(t *testing.T) {
	fields, current := Catalogue(), Current(LegacyNFL())
	if len(current) != len(fields) {
		t.Fatalf("%d current values for %d fields", len(current), len(fields))
	}
	preset := LegacyNFL()
	for i, f := range fields {
		var want Setting
		for _, candidate := range ruleFields() {
			if candidate.pathName() == f.Path {
				want = candidate.read(preset)
			}
		}
		if string(current[i]) == "null" {
			if DescribeSetting(want).State != "Needs a ruling" {
				t.Fatalf("%s is set but sent as null", f.Path)
			}
			continue
		}
		got, err := DecodeSetting(f.Path, current[i])
		if err != nil {
			t.Fatalf("%s: current value is not accepted back: %v", f.Path, err)
		}
		again, _ := naturalValue(reflect.ValueOf(got))
		original, _ := naturalValue(reflect.ValueOf(want))
		if DescribeField(f.Path, got).Words != DescribeField(f.Path, want).Words ||
			!reflect.DeepEqual(again, original) {
			t.Fatalf("%s: round trip changed the value: %s", f.Path, current[i])
		}
	}
}

func TestCurrentIsNullWhereARuleNeedsARuling(t *testing.T) {
	for i, body := range Current(RuleSet{}) {
		if !json.Valid(body) || string(body) != "null" {
			t.Fatalf("field %d of an empty rule set sent %s", i, body)
		}
	}
}

func TestPercentFieldsAreTheShares(t *testing.T) {
	if !IsPercent("cap.irShare") || !IsPercent("contracts.deadCap") || IsPercent("cap.amount") {
		t.Fatal("percent fields are wrong")
	}
}

func TestScheduleFollowSettingValidation(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		body  string
		valid bool
	}{
		{`{"Source":"mfl","League":"00123","Season":2026}`, true},
		{`{"Source":"","League":"","Season":0}`, true},
		{`{"Source":"mfl","League":"bad","Season":2026}`, false},
		{`{"Source":"other","League":"123","Season":2026}`, false},
		{`{"Source":"mfl","League":"123","Season":0}`, false},
		{`{"Source":"","League":"123","Season":2026}`, false},
	} {
		if _, err := DecodeSetting("schedule.follow", json.RawMessage(tc.body)); (err == nil) != tc.valid {
			t.Errorf("%s: %v", tc.body, err)
		}
	}
}
