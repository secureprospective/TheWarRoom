package archive

import (
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/leaguefeed"
)

func TestOpponentFallback(t *testing.T) {
	t.Parallel()
	weeks := []leaguefeed.WeekResult{{Week: 2, Games: []leaguefeed.Game{{Home: leaguefeed.Side{Franchise: "0001"}, Away: leaguefeed.Side{Franchise: "0002"}}}}}
	for _, tc := range []struct {
		week       int
		team, want string
	}{{2, "0001", "0002"}, {2, "0002", "0001"}, {1, "0001", ""}, {2, "unknown", ""}} {
		if got := OpponentFallback(weeks, tc.week, tc.team); got != tc.want {
			t.Errorf("%+v: %q", tc, got)
		}
	}
	if got := OpponentFallback(nil, 2, "0001"); got != "" {
		t.Fatal(got)
	}
}
