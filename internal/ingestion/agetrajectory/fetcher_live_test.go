package agetrajectory

import (
	"context"
	"net/http"
	"os"
	"testing"
	"time"
)

// TestLive_AgeTrajectoryFetch exercises Fetch against the REAL nflverse players.csv.
// Opt-in: set TWR_LIVE_NFLVERSE=1. Makes one outbound network call, so it stays out
// of the default suite/CI but compiles + lints with everything else.
//
//	TWR_LIVE_NFLVERSE=1 go test -run TestLive_AgeTrajectoryFetch -v ./internal/ingestion/agetrajectory/...
func TestLive_AgeTrajectoryFetch(t *testing.T) {
	if os.Getenv("TWR_LIVE_NFLVERSE") != "1" {
		t.Skip("live agetrajectory fetch: set TWR_LIVE_NFLVERSE=1 to run (makes a real network call)")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	m, err := Fetch(ctx, &http.Client{Timeout: 60 * time.Second}, SourceURL)
	if err != nil {
		t.Fatalf("Fetch against live source: %v", err)
	}

	// players.csv is the full historical player DB — many thousands of players carry
	// a birth date. A low count means a truncated download or a schema change.
	if len(m) < 5000 {
		t.Errorf("resolved only %d birth dates; expected >5000 for the full player DB", len(m))
	}
	t.Logf("resolved %d gsis-keyed birth dates", len(m))
}
