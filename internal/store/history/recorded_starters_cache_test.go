package history

import (
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/commissioner"
)

// A finished week's starters are built once and reused until the TTL; a new key or expiry rebuilds.
func TestRecordedStartersCacheReusesFinishedWeeks(t *testing.T) {
	var c recordedStartersCache
	builds := 0
	at := time.Date(2026, 11, 21, 12, 0, 0, 0, time.UTC)
	read := func(key string, now time.Time) commissioner.NativeResults {
		out := commissioner.NativeResults{Scores: []commissioner.FranchiseWeekScore{
			{Franchise: "0001", RecordedResult: "W"}, {Franchise: "0002"},
		}}
		c.apply(key, now, &out, func() {
			builds++
			out.Scores[0].Starters = []commissioner.StarterScore{{Player: "13604"}}
		})
		return out
	}
	first := read("wk3", at)
	again := read("wk3", at.Add(time.Minute))
	if builds != 1 {
		t.Fatalf("builds = %d, want 1 (second read reuses)", builds)
	}
	if len(again.Scores[0].Starters) != 1 || again.Scores[0].Starters[0].Player != "13604" {
		t.Fatalf("reused starters = %+v", again.Scores[0].Starters)
	}
	if again.Scores[1].Starters != nil {
		t.Fatalf("a franchise without a recorded result got starters")
	}
	again.Scores[0].Starters[0].Player = "changed"
	if read("wk3", at.Add(2*time.Minute)).Scores[0].Starters[0].Player != "13604" || first.Scores[0].Starters[0].Player != "13604" {
		t.Fatalf("callers share starters memory")
	}
	read("wk4", at)
	read("wk3", at.Add(recordedStartersTTL+time.Second))
	if builds != 3 {
		t.Fatalf("builds = %d, want 3 (new key and expiry rebuild)", builds)
	}
}
