package leagueschedule

import (
	"context"
	"encoding/json"
	"testing"
)

func TestRawScheduleWeek_Validate(t *testing.T) {
	base := RawScheduleWeek{
		Week: "1",
		Matchups: []RawMatchup{
			{Franchises: [2]RawMatchupSide{
				{FranchiseID: "0001", IsHome: "1"},
				{FranchiseID: "0002", IsHome: "0"},
			}},
		},
	}

	tests := []struct {
		name    string
		mutate  func(w *RawScheduleWeek)
		wantErr bool
	}{
		{"valid", func(*RawScheduleWeek) {}, false},
		{"missing week", func(w *RawScheduleWeek) { w.Week = "" }, true},
		// A zero-matchup week is NOT an error — a future playoff week whose bracket hasn't
		// been seeded yet legitimately has none (confirmed live 2026-07-27).
		{"zero matchups", func(w *RawScheduleWeek) { w.Matchups = nil }, false},
		{"empty franchise id", func(w *RawScheduleWeek) { w.Matchups[0].Franchises[0].FranchiseID = "" }, true},
		{"bad isHome", func(w *RawScheduleWeek) { w.Matchups[0].Franchises[1].IsHome = "yes" }, true},
		// REGRESSION (DeepSeek blind review): a matchup where both sides claim isHome, or
		// neither does, must fail loud rather than let the App seam silently mislabel which
		// franchise is home — Validate is the one place that invariant is enforced.
		{"both sides home", func(w *RawScheduleWeek) { w.Matchups[0].Franchises[1].IsHome = "1" }, true},
		{"neither side home", func(w *RawScheduleWeek) { w.Matchups[0].Franchises[0].IsHome = "0" }, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := RawScheduleWeek{
				Week:     base.Week,
				Matchups: append([]RawMatchup(nil), base.Matchups...),
			}
			tt.mutate(&w)
			if err := w.Validate(); (err != nil) != tt.wantErr {
				t.Fatalf("Validate() error = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}

// TestFlatten_RealShape decodes a trimmed copy of MFL's documented schedule export
// shape (weeklySchedule -> week -> matchup -> franchise[2]) to lock the field
// mapping and the single-element collapse handling on matchup/franchise.
func TestFlatten_RealShape(t *testing.T) {
	const body = `{"schedule":{"weeklySchedule":[` +
		`{"week":"1","matchup":[` +
		`{"franchise":[{"id":"0001","isHome":"1","score":""},{"id":"0002","isHome":"0","score":""}]}` +
		`]}` +
		`]}}`

	var env scheduleEnvelope
	if err := json.Unmarshal([]byte(body), &env); err != nil {
		t.Fatalf("decode: %v", err)
	}
	got, err := flatten(context.Background(), env)
	if err != nil {
		t.Fatalf("flatten: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d weeks, want 1", len(got))
	}
	w := got[0]
	if w.Week != "1" || len(w.Matchups) != 1 {
		t.Fatalf("week mapped wrong: %+v", w)
	}
	sides := w.Matchups[0].Franchises
	if sides[0].FranchiseID != "0001" || sides[0].IsHome != "1" || sides[1].FranchiseID != "0002" {
		t.Fatalf("franchises mapped wrong: %+v", sides)
	}
}

// TestFlatten_SingleWeekSingleMatchupCollapse locks the MFL array/object-collapse
// quirk: a season with exactly one week and one matchup lets MFL's legacy XML->JSON
// converter emit bare objects instead of one-element arrays for weeklySchedule and
// matchup. ingestion.MFLList must still yield a one-item slice for each.
func TestFlatten_SingleWeekSingleMatchupCollapse(t *testing.T) {
	const body = `{"schedule":{"weeklySchedule":` +
		`{"week":"1","matchup":` +
		`{"franchise":[{"id":"0001","isHome":"1","score":""},{"id":"0002","isHome":"0","score":""}]}` +
		`}}}`

	var env scheduleEnvelope
	if err := json.Unmarshal([]byte(body), &env); err != nil {
		t.Fatalf("decode: %v", err)
	}
	got, err := flatten(context.Background(), env)
	if err != nil {
		t.Fatalf("flatten: %v", err)
	}
	if len(got) != 1 || len(got[0].Matchups) != 1 {
		t.Fatalf("collapse not handled: %+v", got)
	}
}

func TestFlatten_WrongFranchiseCount(t *testing.T) {
	const body = `{"schedule":{"weeklySchedule":[` +
		`{"week":"1","matchup":[{"franchise":[{"id":"0001","isHome":"1","score":""}]}]}` +
		`]}}`

	var env scheduleEnvelope
	if err := json.Unmarshal([]byte(body), &env); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if _, err := flatten(context.Background(), env); err == nil {
		t.Fatal("expected an error for a matchup with 1 franchise, got nil")
	}
}
