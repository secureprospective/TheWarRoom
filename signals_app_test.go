package main

import (
	"testing"
)

// The last and current seasons are scored under today's rules; older ones only exist under
// their own.
func TestScoringYear(t *testing.T) {
	t.Parallel()
	for y, want := range map[int]int{2021: 2021, 2024: 2024, 2025: 2026, 2026: 2026} {
		if got := scoringYear(y, 2026); got != want {
			t.Errorf("scoringYear(%d) = %d, want %d", y, got, want)
		}
	}
}
