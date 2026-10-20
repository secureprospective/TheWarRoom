package ras

import (
	"context"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/ingestion/crosswalk"
)

// TestLive_RASFetch exercises Fetch against the REAL nflverse combine.csv, resolving
// pfr->gsis from the live dynastyprocess crosswalk. Opt-in: set TWR_LIVE_NFLVERSE=1.
// Makes two outbound network calls, so it stays out of the default suite/CI but
// compiles + lints with everything else.
//
//	TWR_LIVE_NFLVERSE=1 go test -run TestLive_RASFetch -v ./internal/ingestion/ras/...
func TestLive_RASFetch(t *testing.T) {
	if os.Getenv("TWR_LIVE_NFLVERSE") != "1" {
		t.Skip("live ras fetch: set TWR_LIVE_NFLVERSE=1 to run (makes real network calls)")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	client := &http.Client{Timeout: 90 * time.Second}

	// The pfr->gsis bridge comes from the shared crosswalk (one db_playerids fetch,
	// drop-ambiguous dedup), mirroring how the assembly layer supplies it in production.
	cw, err := crosswalk.Fetch(ctx, client, crosswalk.SourceURL)
	if err != nil {
		t.Fatalf("build live crosswalk: %v", err)
	}
	pfrToGSIS := cw.PFRMap()
	t.Logf("crosswalk resolved %d pfr->gsis entries", len(pfrToGSIS))

	m, err := Fetch(ctx, client, SourceURL, pfrToGSIS)
	if err != nil {
		t.Fatalf("Fetch against live source: %v", err)
	}

	// combine.csv carries thousands of NFL players who tested and resolve to a gsis;
	// a low count means a truncated download, a schema change, or a pfr-id format
	// drift between the two sources.
	if len(m) < 2000 {
		t.Errorf("resolved only %d combine records; expected >2000", len(m))
	}
	t.Logf("resolved %d gsis-keyed combine records", len(m))
}
