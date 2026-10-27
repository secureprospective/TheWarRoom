package scoring

import (
	"reflect"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/playerid"
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

func TestFreeAgentIDsPreserveScopeAndNil(t *testing.T) {
	t.Parallel()
	id, err := playerid.New("0042")
	if err != nil {
		t.Fatal(err)
	}
	ids := []playerid.PlayerID{id, id}
	got := FreeAgentIDs(ids)
	if !reflect.DeepEqual(got, []string{"0042", "0042"}) || FreeAgentIDs(nil) != nil || FreeAgentIDs([]playerid.PlayerID{}) != nil {
		t.Fatalf("scope: %v", got)
	}
	got[0] = "changed"
	if ids[0].String() != "0042" {
		t.Fatal("model input changed the held IDs")
	}
}
