package salaryadjustments

import (
	"context"
	"encoding/json"
	"testing"
)

func TestRawSalaryAdjustment_Validate(t *testing.T) {
	base := RawSalaryAdjustment{
		FranchiseID: "0025",
		Amount:      "0.203",
		Description: "Dropped Kennedy, Tom DET WR (Salary: $0.58, Contract Year : 2026, Contract Status: UFA)",
		ID:          "2617",
		Timestamp:   "1777671128",
	}

	tests := []struct {
		name    string
		mutate  func(a *RawSalaryAdjustment)
		wantErr bool
	}{
		{"valid", func(*RawSalaryAdjustment) {}, false},
		{"missing franchise", func(a *RawSalaryAdjustment) { a.FranchiseID = "" }, true},
		{"non-numeric franchise", func(a *RawSalaryAdjustment) { a.FranchiseID = "WAIVERS" }, true},
		{"missing amount", func(a *RawSalaryAdjustment) { a.Amount = "" }, true},
		{"non-numeric amount", func(a *RawSalaryAdjustment) { a.Amount = "lots" }, true},
		// Negative amounts are VALID — commissioners issue negative cap credits.
		{"negative amount (credit)", func(a *RawSalaryAdjustment) { a.Amount = "-0.50" }, false},
		{"non-numeric timestamp", func(a *RawSalaryAdjustment) { a.Timestamp = "yesterday" }, true},
		// Description is free text and intentionally unchecked; empty is fine.
		{"empty description tolerated", func(a *RawSalaryAdjustment) { a.Description = "" }, false},
		// Timestamp is optional at the boundary (validated only when present).
		{"empty timestamp tolerated", func(a *RawSalaryAdjustment) { a.Timestamp = "" }, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := base
			tt.mutate(&a)
			if err := a.Validate(); (err != nil) != tt.wantErr {
				t.Fatalf("Validate() error = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}

// TestDecode_RealShape decodes a trimmed copy of the live salaryAdjustments payload
// (the real Cardinals 0025 entries) to lock the snake_case field mapping and the
// raw passthrough of amount/description.
func TestDecode_RealShape(t *testing.T) {
	const body = `{"version":"1.0","salaryAdjustments":{"salaryAdjustment":[` +
		`{"franchise_id":"0025","amount":"0.203","description":"Dropped Kennedy, Tom DET WR (Salary: $0.58, Contract Year : 2026, Contract Status: UFA)","id":"2617","timestamp":"1777671128"},` +
		`{"franchise_id":"0025","amount":"1.05","description":"Dropped Williams, Trayveon CIN RB (Salary: $3.00, Contract Year : 2026, Contract Status: UFA)","id":"2074","timestamp":"1712347847"}` +
		`]},"encoding":"utf-8"}`

	var env salaryAdjustmentsEnvelope
	if err := json.Unmarshal([]byte(body), &env); err != nil {
		t.Fatalf("decode: %v", err)
	}
	got, err := flatten(context.Background(), env)
	if err != nil {
		t.Fatalf("flatten: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d adjustments, want 2", len(got))
	}
	if got[0].FranchiseID != "0025" || got[0].Amount != "0.203" || got[0].ID != "2617" {
		t.Fatalf("adjustment 0 mapped wrong: %+v", got[0])
	}
}
