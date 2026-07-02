package playerscores

import (
	"context"
	"encoding/json"
	"testing"
)

func TestRawScore_Validate(t *testing.T) {
	base := RawScore{ID: "13130", Score: "476.75", Week: "YTD"}

	tests := []struct {
		name    string
		mutate  func(s *RawScore)
		wantErr bool
	}{
		{"valid", func(*RawScore) {}, false},
		{"valid integer score", func(s *RawScore) { s.Score = "12" }, false},
		{"valid leading-zero id", func(s *RawScore) { s.ID = "0099" }, false},
		{"valid negative score kept raw", func(s *RawScore) { s.Score = "-1.50" }, false},
		{"missing id", func(s *RawScore) { s.ID = "" }, true},
		{"non-numeric id", func(s *RawScore) { s.ID = "abc" }, true},
		{"missing score", func(s *RawScore) { s.Score = "  " }, true},
		{"non-numeric score", func(s *RawScore) { s.Score = "12pts" }, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := base
			tt.mutate(&s)
			err := s.Validate()
			if (err != nil) != tt.wantErr {
				t.Fatalf("Validate() error = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}

func TestFlatten_FiltersAggregatesAndMapsFields(t *testing.T) {
	env := scoresEnvelope{}
	env.PlayerScores.PlayerScore = []scoreBlock{
		{ID: "13130", Score: "476.75", Week: "YTD"},
		{ID: "0200", Score: "155.00", Week: "YTD"}, // team-aggregate ID (151–782) → filtered
		{ID: "0099", Score: "0.00", Week: "YTD"},
	}

	got, err := flatten(context.Background(), env)
	if err != nil {
		t.Fatalf("flatten: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d records, want 2 (aggregate 0200 filtered)", len(got))
	}
	first := got[0]
	if first.ID != "13130" || first.Score != "476.75" || first.Week != "YTD" {
		t.Errorf("first record mapped wrong: %+v", first)
	}
	// A real 0.00 total (played, scored nothing) survives verbatim — zero is a
	// legitimate score, distinct from absent (MFL omits unscored players entirely).
	if got[1].ID != "0099" || got[1].Score != "0.00" {
		t.Errorf("zero-score record mishandled: %+v", got[1])
	}
}

func TestFlatten_MalformedRecordFailsLoud(t *testing.T) {
	env := scoresEnvelope{}
	env.PlayerScores.PlayerScore = []scoreBlock{{ID: "13130", Score: "n/a", Week: "YTD"}}
	if _, err := flatten(context.Background(), env); err == nil {
		t.Fatal("flatten should error on a non-numeric score, got nil")
	}
}

// TestDecode_SingleElementCollapse locks the MFL single-element quirk: a
// one-record playerScore list arrives as a BARE OBJECT, not a one-element array.
func TestDecode_SingleElementCollapse(t *testing.T) {
	const body = `{"playerScores":{"playerScore":{"id":"13130","score":"476.75","week":"YTD"}}}`

	var env scoresEnvelope
	if err := json.Unmarshal([]byte(body), &env); err != nil {
		t.Fatalf("decode single-score payload: %v", err)
	}
	got, err := flatten(context.Background(), env)
	if err != nil {
		t.Fatalf("flatten: %v", err)
	}
	if len(got) != 1 || got[0].ID != "13130" {
		t.Fatalf("single-element collapse not handled: %+v", got)
	}
}

// TestGuardNonEmpty_AllAggregatePayloadFailsLoud is the planted gate for the GLM
// M1 finding: an envelope of nothing-but-aggregate ids survives decode and
// flatten with zero records and NO error — the post-filter guard must catch it.
func TestGuardNonEmpty_AllAggregatePayloadFailsLoud(t *testing.T) {
	env := scoresEnvelope{}
	env.PlayerScores.PlayerScore = []scoreBlock{
		{ID: "0200", Score: "155.00", Week: "YTD"},
		{ID: "0531", Score: "88.00", Week: "YTD"},
	}
	out, err := flatten(context.Background(), env)
	if err != nil || len(out) != 0 {
		t.Fatalf("flatten precondition: want (0, nil), got (%d, %v)", len(out), err)
	}
	if _, err := guardNonEmpty(out); err == nil {
		t.Fatal("guardNonEmpty should reject a post-filter empty result, got nil")
	}
}

// TestFlatten_WrongWeekEchoFailsLoud is the planted gate for the YTD assert: a
// single-week echo (~17× off as a season total) must never reach a consumer.
func TestFlatten_WrongWeekEchoFailsLoud(t *testing.T) {
	env := scoresEnvelope{}
	env.PlayerScores.PlayerScore = []scoreBlock{{ID: "13130", Score: "28.75", Week: "3"}}
	if _, err := flatten(context.Background(), env); err == nil {
		t.Fatal("flatten should reject a non-YTD week echo, got nil")
	}
}
