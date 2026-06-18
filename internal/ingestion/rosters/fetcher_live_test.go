package rosters

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/mfl"
)

// TestLive_RostersFetch exercises Fetch against the REAL MyFantasyLeague API.
// It is opt-in: set TWR_LIVE_MFL=1 to run it. It makes outbound network calls and
// depends on MFL being reachable, so it stays out of the default suite, CI, and
// the pre-commit hook — but it still compiles and is linted/vetted with everything
// else (no build tag), so it cannot rot. Mirrors internal/mfl/client_live_test.go.
//
//	TWR_LIVE_MFL=1 go test -run TestLive_RostersFetch -v ./internal/ingestion/rosters/...
//
// This is the live half of the B2 close gate: Fetch discovers the league host,
// pulls all 32 franchises' rosters, and the ingestion filter has removed every
// team-aggregate ID before the records leave Layer 1.
func TestLive_RostersFetch(t *testing.T) {
	if os.Getenv("TWR_LIVE_MFL") != "1" {
		t.Skip("live MFL rosters fetch: set TWR_LIVE_MFL=1 to run (makes real network calls)")
	}

	c, err := mfl.New("api", 2) // 2 req/s — comfortably under MFL's limits
	if err != nil {
		t.Fatalf("mfl.New: %v", err)
	}

	// Must exceed the client's full 429 backoff cycle (1+2+…+60 ≈ 123s) plus
	// request time, so a single transient 429 self-heals instead of failing
	// (same rationale as internal/mfl/client_live_test.go).
	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Second)
	defer cancel()

	records, err := Fetch(ctx, c, ingestion.SeasonYear, ingestion.LeagueID)
	if err != nil {
		t.Fatalf("Fetch against live MFL: %v", err)
	}
	if len(records) == 0 {
		t.Fatal("Fetch returned 0 roster records")
	}

	franchises := make(map[string]struct{})
	for _, r := range records {
		franchises[r.FranchiseID] = struct{}{}
		if ingestion.IsTeamAggregateID(r.PlayerID) {
			t.Errorf("team-aggregate ID %q survived ingestion filter (franchise %s)", r.PlayerID, r.FranchiseID)
		}
		if r.PlayerID == "" || r.Status == "" {
			t.Errorf("record for franchise %s has empty id/status: %+v", r.FranchiseID, r)
		}
	}

	const wantFranchises = 32 // Legacy NFL is a 32-team league
	if len(franchises) != wantFranchises {
		t.Errorf("got %d distinct franchises, want %d", len(franchises), wantFranchises)
	}
	t.Logf("fetched %d roster records across %d franchises", len(records), len(franchises))
}
