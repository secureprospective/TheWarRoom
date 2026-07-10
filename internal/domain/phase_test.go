package domain

import "testing"

func TestPhaseValid(t *testing.T) {
	valid := []Phase{PhaseOffseason, PhaseRegularSeason, PhasePlayoffs}
	for _, p := range valid {
		if !p.Valid() {
			t.Errorf("Phase(%q).Valid() = false, want true", p)
		}
	}
	invalid := []Phase{"", "offseason", "PRESEASON", "OFF_SEASON", "REGULAR"}
	for _, p := range invalid {
		if p.Valid() {
			t.Errorf("Phase(%q).Valid() = true, want false", p)
		}
	}
}
