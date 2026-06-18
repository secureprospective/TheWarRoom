package rosters

import (
	"context"
	"encoding/json"
	"testing"
)

func TestRawRoster_Validate(t *testing.T) {
	base := RawRoster{
		FranchiseID:    "0001",
		PlayerID:       "0835",
		Salary:         "7",
		ContractYear:   "2026",
		ContractStatus: "UFA",
		Status:         "ROSTER",
	}

	tests := []struct {
		name    string
		mutate  func(r *RawRoster)
		wantErr bool
	}{
		{"valid", func(*RawRoster) {}, false},
		{"valid decimal salary", func(r *RawRoster) { r.Salary = "17.70" }, false},
		{"valid empty salary kept", func(r *RawRoster) { r.Salary = "" }, false},
		{"valid leading-zero id", func(r *RawRoster) { r.PlayerID = "0099" }, false},
		{"missing franchise", func(r *RawRoster) { r.FranchiseID = "  " }, true},
		{"missing player id", func(r *RawRoster) { r.PlayerID = "" }, true},
		{"non-numeric player id", func(r *RawRoster) { r.PlayerID = "abc" }, true},
		{"missing status", func(r *RawRoster) { r.Status = "" }, true},
		{"non-numeric salary", func(r *RawRoster) { r.Salary = "7m" }, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := base
			tt.mutate(&r)
			err := r.Validate()
			if (err != nil) != tt.wantErr {
				t.Fatalf("Validate() error = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}

func TestFlatten_FiltersAggregatesAndMapsFields(t *testing.T) {
	env := rostersEnvelope{}
	env.Rosters.Franchise = []franchiseBlock{
		{
			ID: "0001",
			Player: []playerBlock{
				{ID: "0835", Salary: "7", ContractYear: "2026", ContractStatus: "UFA ", ContractInfo: "note", Status: "ROSTER"},
				{ID: "0200", Status: "ROSTER"}, // team-aggregate ID (151–782) → filtered
				{ID: "13128", Salary: "1.30", ContractStatus: "FT1 (2026)", Status: "TAXI_SQUAD"},
			},
		},
		{
			ID:     "0032",
			Player: []playerBlock{{ID: "0531", Salary: "", Status: "ROSTER"}}, // 531 is in 151–782 → filtered
		},
	}

	got, err := flatten(context.Background(), env)
	if err != nil {
		t.Fatalf("flatten: %v", err)
	}

	// 0200 and 0531 are both inside the aggregate block and must be dropped,
	// leaving 0835 and 13128.
	if len(got) != 2 {
		t.Fatalf("got %d records, want 2 (aggregates 0200, 0531 filtered)", len(got))
	}

	// Field mapping is verbatim — contractStatus stays dirty ("UFA "), salary raw.
	first := got[0]
	if first.FranchiseID != "0001" || first.PlayerID != "0835" || first.Salary != "7" {
		t.Errorf("first record mapped wrong: %+v", first)
	}
	if first.ContractStatus != "UFA " {
		t.Errorf("contractStatus should be kept dirty, got %q", first.ContractStatus)
	}
	if first.ContractInfo != "note" {
		t.Errorf("contractInfo not mapped, got %q", first.ContractInfo)
	}
}

func TestFlatten_MalformedRecordFailsLoud(t *testing.T) {
	env := rostersEnvelope{}
	env.Rosters.Franchise = []franchiseBlock{
		{ID: "0001", Player: []playerBlock{{ID: "not-a-number", Status: "ROSTER"}}},
	}
	if _, err := flatten(context.Background(), env); err == nil {
		t.Fatal("flatten should error on a malformed (non-aggregate) player id, got nil")
	}
}

// TestDecode_SingleElementCollapse is the BLOCKER regression (Gemini B2 review):
// MFL strips array brackets when a list has exactly one element. A franchise with
// one player must still decode as a one-element slice, not crash the unmarshal.
func TestDecode_SingleElementCollapse(t *testing.T) {
	// franchise[] is a normal array, but its single player is a BARE OBJECT.
	const body = `{"rosters":{"franchise":[{"id":"0001","player":{"id":"13128","salary":"7","status":"ROSTER"}}]}}`

	var env rostersEnvelope
	if err := json.Unmarshal([]byte(body), &env); err != nil {
		t.Fatalf("decode single-player franchise: %v", err)
	}
	got, err := flatten(context.Background(), env)
	if err != nil {
		t.Fatalf("flatten: %v", err)
	}
	if len(got) != 1 || got[0].PlayerID != "13128" {
		t.Fatalf("single-element collapse not handled: %+v", got)
	}
}
