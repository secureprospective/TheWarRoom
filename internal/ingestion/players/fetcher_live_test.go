package players

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/mfl"
)

// TestLive_PlayersFetch exercises Fetch against the REAL MFL players endpoint.
// Opt-in: set TWR_LIVE_MFL=1. It makes outbound network calls, so it stays out of
// the default suite/CI/pre-commit but still compiles and is linted with everything
// else. Mirrors the schedule live test; proves the non-league `api`-host path on
// the full ~2,500-record database. NOTE: MFL caps this endpoint at once per day —
// do not loop it.
//
//	TWR_LIVE_MFL=1 go test -run TestLive_PlayersFetch -v ./internal/ingestion/players/...
func TestLive_PlayersFetch(t *testing.T) {
	if os.Getenv("TWR_LIVE_MFL") != "1" {
		t.Skip("live MFL players fetch: set TWR_LIVE_MFL=1 to run (makes real network calls)")
	}

	c, err := mfl.New("api", 2)
	if err != nil {
		t.Fatalf("mfl.New: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Second)
	defer cancel()

	all, err := Fetch(ctx, c, ingestion.SeasonYear, ingestion.LeagueID)
	if err != nil {
		t.Fatalf("Fetch against live MFL: %v", err)
	}
	if len(all) == 0 {
		t.Fatal("Fetch returned 0 players")
	}

	// Sanity: the real DB has thousands of records and every one carries a
	// position (Validate already enforced it). Spot-confirm a plausible scale so
	// a truncated payload surfaces here rather than at the downstream join.
	if len(all) < 1000 {
		t.Errorf("fetched only %d players; expected >1000 for the full database", len(all))
	}
	t.Logf("fetched %d player records", len(all))
}
