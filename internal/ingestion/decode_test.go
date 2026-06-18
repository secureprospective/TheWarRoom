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
