package schedule

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/mfl"
)

// TestLive_ScheduleFetch exercises Fetch against the REAL MFL nflSchedule endpoint.
// Opt-in: set TWR_LIVE_MFL=1. It makes outbound network calls, so it stays out of
// the default suite/CI/pre-commit but still compiles and is linted with everything
// else. Mirrors the rosters live test; proves the non-league `api`-host path.
//
//	TWR_LIVE_MFL=1 go test -run TestLive_ScheduleFetch -v ./internal/ingestion/schedule/...
func TestLive_ScheduleFetch(t *testing.T) {
	if os.Getenv("TWR_LIVE_MFL") != "1" {
		t.Skip("live MFL schedule fetch: set TWR_LIVE_MFL=1 to run (makes real network calls)")
	}

	c, err := mfl.New("api", 2)
	if err != nil {
		t.Fatalf("mfl.New: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Second)
	defer cancel()

	matchups, err := Fetch(ctx, c, ingestion.SeasonYear)
	if err != nil {
		t.Fatalf("Fetch against live MFL: %v", err)
	}
	if len(matchups) == 0 {
		t.Fatal("Fetch returned 0 matchups")
	}

	for _, m := range matchups {
		homes := 0
		for _, tm := range m.Teams {
			if tm.IsHome == "1" {
				homes++
			}
		}
		if homes != 1 {
			t.Errorf("matchup (kickoff %s) has %d home teams, want exactly 1: %+v", m.Kickoff, homes, m.Teams)
		}
	}
	t.Logf("fetched %d scheduled matchups", len(matchups))
}
