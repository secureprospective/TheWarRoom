package ingestion

import (
	"encoding/json"
	"testing"
)

func TestMFLList_UnmarshalJSON(t *testing.T) {
	type elem struct {
		ID string `json:"id"`
	}

	tests := []struct {
		name    string
		json    string
		wantLen int
		wantErr bool
	}{
		{"array of two", `[{"id":"a"},{"id":"b"}]`, 2, false},
		{"single bare object", `{"id":"a"}`, 1, false},
		{"empty array", `[]`, 0, false},
		{"null", `null`, 0, false},
		{"malformed", `{"id":`, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got MFLList[elem]
			err := json.Unmarshal([]byte(tt.json), &got)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Unmarshal(%s) err = %v, wantErr = %v", tt.json, err, tt.wantErr)
			}
			if !tt.wantErr && len(got) != tt.wantLen {
				t.Fatalf("Unmarshal(%s) len = %d, want %d", tt.json, len(got), tt.wantLen)
			}
		})
	}
}

// TestCheckAPIError locks the MFL HTTP-200 error-payload guard: an {"error":{"$t"}}
// body must surface as an error (so a fetcher with no empty sentinel does not read
// it as "no data"), while a real empty/valid payload and malformed JSON must NOT.
func TestCheckAPIError(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		wantErr bool
	}{
		{"mfl error payload", `{"version":"1.0","error":{"$t":"Error - Invalid League!"}}`, true},
		{"empty ledger (valid)", `{"salaryAdjustments":{}}`, false},
		{"empty object", `{}`, false},
		{"populated payload", `{"salaryAdjustments":{"salaryAdjustment":[{"franchise_id":"0025","amount":"0.20"}]}}`, false},
		{"empty error text ignored", `{"error":{"$t":"  "}}`, false},
		{"malformed json not our concern", `{"error":`, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := CheckAPIError([]byte(tt.body)); (err != nil) != tt.wantErr {
				t.Fatalf("CheckAPIError(%s) err = %v, wantErr %v", tt.body, err, tt.wantErr)
			}
		})
	}
}
