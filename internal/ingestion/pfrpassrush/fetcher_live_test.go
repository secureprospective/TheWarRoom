package pfrpassrush

import (
	"context"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/ingestion/crosswalk"
)

// TestLive_PassRushFetch exercises Fetch against the REAL nflverse pfr_advstats
// advstats_season_def.csv.gz, resolving pfr->gsis from the live dynastyprocess crosswalk.
// Opt-in: set TWR_LIVE_NFLVERSE=1. Makes two outbound network calls (one gzipped), so it stays
// out of the default suite/CI but compiles + lints with everything else.
//
//	TWR_LIVE_NFLVERSE=1 go test -run TestLive_PassRushFetch -v ./internal/ingestion/pfrpassrush/...
func TestLive_PassRushFetch(t *testing.T) {
	if os.Getenv("TWR_LIVE_NFLVERSE") != "1" {
		t.Skip("live pass-rush fetch: set TWR_LIVE_NFLVERSE=1 to run (makes real network calls)")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	client := &http.Client{Timeout: 90 * time.Second}

	cw, err := crosswalk.Fetch(ctx, client, crosswalk.SourceURL)
	if err != nil {
		t.Fatalf("build live crosswalk: %v", err)
	}
	pfrToGSIS := cw.PFRMap()
	t.Logf("crosswalk resolved %d pfr->gsis entries", len(pfrToGSIS))

	const season = "2023"
	m, err := Fetch(ctx, client, SourceURL, season, pfrToGSIS)
	if err != nil {
		t.Fatalf("Fetch against live source: %v", err)
	}

	// A full season carries hundreds of pass-rushing defenders that resolve to a gsis; a low
	// count means a truncated download, a schema change, an absent season, or a pfr-id format
	// drift between the two sources.
	if len(m) < 200 {
		t.Errorf("resolved only %d %s pass-rush records; expected >200", len(m), season)
	}
	t.Logf("resolved %d gsis-keyed pass-rush records for %s", len(m), season)

	// Spot-invariant: every emitted record has real pass-rush production and no negative/absurd
	// counts (a sanity guard on the raw counting stats).
	for gsis, pr := range m {
		if !(pr.Pressures > 0 || pr.Sacks > 0 || pr.QBKnockdowns > 0 || pr.Hurries > 0) {
			t.Fatalf("%s emitted with no pass-rush production: %+v", gsis, pr)
		}
		if pr.Sacks < 0 || pr.Pressures < 0 || pr.Hurries < 0 || pr.QBKnockdowns < 0 ||
			pr.Blitzes < 0 || pr.Games < 0 {
			t.Fatalf("%s has a negative count: %+v", gsis, pr)
		}
		if pr.Sacks > 40 || pr.Pressures > 200 {
			t.Fatalf("%s pass-rush count implausibly high: %+v", gsis, pr)
		}
	}
}
