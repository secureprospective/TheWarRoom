package leagueschedule

import (
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

func TestFlatten_WrongFranchiseCount(t *testing.T) {
	const body = `{"schedule":{"weeklySchedule":[` +
		`{"week":"1","matchup":[{"franchise":[{"id":"0001","isHome":"1","score":""}]}]}` +
		`]}}`

	var env scheduleEnvelope
	if err := json.Unmarshal([]byte(body), &env); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if _, err := flatten(env); err == nil {
		t.Fatal("expected an error for a matchup with 1 franchise, got nil")
	}
}
