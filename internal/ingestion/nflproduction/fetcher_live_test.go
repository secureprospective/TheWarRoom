package nflproduction

import (
	"context"
	"net/http"
	"os"
	"testing"
	"time"
)

// TestLive_NFLProductionFetch exercises Fetch against the REAL nflverse
// player_stats_season file. Opt-in: set TWR_LIVE_NFLVERSE=1. Makes one outbound
// network call, so it stays out of the default suite/CI but compiles + lints with
// everything else.
//
//	TWR_LIVE_NFLVERSE=1 go test -run TestLive_NFLProductionFetch -v ./internal/ingestion/nflproduction/...
func TestLive_NFLProductionFetch(t *testing.T) {
	if os.Getenv("TWR_LIVE_NFLVERSE") != "1" {
		t.Skip("live nflproduction fetch: set TWR_LIVE_NFLVERSE=1 to run (makes a real network call)")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	m, err := Fetch(ctx, &http.Client{Timeout: 60 * time.Second}, SeasonURL("2024"))
	if err != nil {
		t.Fatalf("Fetch against live source: %v", err)
	}

	// A full NFL regular season has several hundred offensive players who recorded
	// stats. A low count means a truncated download or schema change — surface here.
	if len(m) < 300 {
		t.Errorf("resolved only %d REG production records; expected >300 for a full season", len(m))
	}
	t.Logf("resolved %d regular-season production records", len(m))
}
