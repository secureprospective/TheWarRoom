package leagueweek

import (
	"testing"
	"time"
)

func TestNextLock(t *testing.T) {
	first := time.Date(2026, 10, 9, 0, 15, 0, 0, time.UTC)
	last := first.Add(72 * time.Hour)
	w := Week{Number: 5, Games: []Game{{Kickoff: last}, {Kickoff: first}}}
	for _, tc := range []struct {
		name     string
		at, want time.Time
		ok       bool
	}{
		{"before week", first.Add(-time.Hour), first, true},
		{"at first", first, last, true},
		{"between kickoffs", first.Add(time.Hour), last, true},
		{"at last", last, time.Time{}, false},
		{"after last", last.Add(time.Hour), time.Time{}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := NextLock(tc.at, w)
			if ok != tc.ok || !got.Equal(tc.want) {
				t.Fatalf("NextLock = %v, %t; want %v, %t", got, ok, tc.want, tc.ok)
			}
		})
	}
	if _, ok := NextLock(first, Week{}); ok {
		t.Fatal("empty week has a lock")
	}
}
