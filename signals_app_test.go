package main

import (
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

func TestSourceViewShowsOnlyAnUnresolvedError(t *testing.T) {
	t0 := time.Date(2026, 10, 4, 6, 0, 0, 0, time.UTC)
	h := history.SourceHealth{Source: "nflverse", State: history.SourceActive,
		LastFailure: t0, LastSuccess: t0.Add(time.Minute), LastError: "timeout"}
	if v := sourceView(h, "nflverse data"); v.LastError != "" || v.Name != "nflverse data" || v.LastSuccess == "" {
		t.Errorf("an error a later load put right is shown: %+v", v)
	}
	h.LastFailure = t0.Add(time.Hour)
	if v := sourceView(h, ""); v.LastError != "timeout" {
		t.Errorf("a current error is hidden: %+v", v)
	}
}

func TestCoverageCountsRosteredPlayersByPosition(t *testing.T) {
	got := coverage(map[string]string{"1": "QB", "2": "QB", "3": "S"}, map[string]bool{"1": true, "3": true, "9": true})
	if len(got) != 2 || got[0] != (PositionShare{"QB", 2, 1}) || got[1] != (PositionShare{"S", 1, 1}) {
		t.Errorf("coverage = %+v", got)
	}
}

// The last and current seasons are scored under today's rules; older ones only exist under
// their own.
func TestScoringYear(t *testing.T) {
	for y, want := range map[int]int{2021: 2021, 2024: 2024, 2025: 2026, 2026: 2026} {
		if got := scoringYear(y, 2026); got != want {
			t.Errorf("scoringYear(%d) = %d, want %d", y, got, want)
		}
	}
}
