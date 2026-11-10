package scoring

import (
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/store/history"
)

func TestEngineDueAfterLaunchAndHourly(t *testing.T) {
	t.Parallel()
	now := time.Now()
	for _, tc := range []struct {
		name                  string
		age                   time.Duration
		exists, changed, want bool
	}{
		{"first launch", 0, false, false, true},
		{"changed launch", time.Hour, true, true, true},
		{"stale launch", 25 * time.Hour, true, false, true},
		{"fresh hourly", time.Hour, true, false, false},
		{"boundary", 24 * time.Hour, true, false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := EngineDue(now, history.Run{AsOf: now.Add(-tc.age)}, tc.exists, tc.changed); got != tc.want {
				t.Fatalf("due = %t want %t", got, tc.want)
			}
		})
	}
}
