package appvalue

import (
	"testing"
	"time"
)

// The producer skips a scope only while revision and params are unchanged, the clock moved forward by
// less than recheck, and the published pass is still in force.
func TestSweepMarkSkipsOnlyWhileNothingButTheClockMoved(t *testing.T) {
	at := time.Date(2026, 11, 2, 20, 0, 0, 0, time.UTC)
	m := sweepMark{revision: 7, params: "p", at: at, ok: true}
	cases := []struct {
		name string
		next sweepMark
		want bool
	}{
		{"same", sweepMark{revision: 7, params: "p", at: at.Add(time.Second), ok: true}, true},
		{"commit", sweepMark{revision: 8, params: "p", at: at.Add(time.Second), ok: true}, false},
		{"params", sweepMark{revision: 7, params: "q", at: at.Add(time.Second), ok: true}, false},
		{"recheck", sweepMark{revision: 7, params: "p", at: at.Add(recheck), ok: true}, false},
		{"clock back", sweepMark{revision: 7, params: "p", at: at.Add(-time.Second), ok: true}, false},
	}
	for _, c := range cases {
		if got := m.unchanged(c.next); got != c.want {
			t.Errorf("%s: unchanged=%v, want %v", c.name, got, c.want)
		}
	}
	if (sweepMark{}).unchanged(m) {
		t.Error("a failed or missing mark must re-read")
	}
	c := &Cache{}
	if c.current(at) {
		t.Error("an empty cache is not current")
	}
	c.publish([32]byte{}, at, at.Add(time.Hour), Reading{}, nil)
	if !c.current(at.Add(time.Minute)) || c.current(at.Add(time.Hour)) || c.current(at.Add(-time.Second)) {
		t.Error("current must follow the pass's window")
	}
}
