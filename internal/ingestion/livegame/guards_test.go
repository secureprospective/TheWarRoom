package livegame

import (
	"context"
	"testing"
	"time"
)

func TestWindowBoundaries(t *testing.T) {
	at := time.Date(2026, 10, 13, 0, 15, 0, 0, time.UTC)
	for _, tc := range []struct {
		name              string
		now               time.Time
		final, open, want bool
	}{
		{"before kickoff", at.Add(-time.Nanosecond), false, true, false},
		{"kickoff", at, false, true, true},
		{"after final", at.Add(time.Hour), true, true, false},
		{"never marked final, last minute", at.Add(MaxGame - time.Nanosecond), false, true, true},
		{"never marked final, six hours on", at.Add(MaxGame), false, true, false},
		{"Now closed", at, false, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			a := Adapter{Fetch: func(context.Context, string, string) ([]byte, error) {
				calls++
				return fixture(t, "espn-scoreboard"), nil
			}}
			w := []Window{{Teams: [2]string{"BUF", "LAR"}, Kickoff: at, Final: tc.final}}
			a.Refresh(context.Background(), tc.now, tc.open, w, "")
			if (calls > 0) != tc.want {
				t.Fatalf("calls %d", calls)
			}
		})
	}
}
