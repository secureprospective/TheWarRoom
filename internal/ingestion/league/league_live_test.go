package league

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/mfl"
)

// TestLive_LeagueFetch exercises Fetch against the REAL MyFantasyLeague API. Opt-in:
// set TWR_LIVE_MFL=1. It makes outbound calls and depends on MFL being reachable, so
// it stays out of the default suite, CI, and the pre-commit hook — but it compiles
// and is linted with everything else (no build tag), so it cannot rot. Mirrors
// internal/ingestion/rosters/fetcher_live_test.go.
//
//	TWR_LIVE_MFL=1 go test -run TestLive_LeagueFetch -v ./internal/ingestion/league/...
//
// It is the live half of the B3b close gate: the fetcher discovers the host, pulls
// BOTH the league and rules exports, and assembles a populated rulebook — proving
// the two-endpoint premise correction and the $t/collapse decode against live data.
func TestLive_LeagueFetch(t *testing.T) {
	if os.Getenv("TWR_LIVE_MFL") != "1" {
		t.Skip("live MFL league fetch: set TWR_LIVE_MFL=1 to run (makes real network calls)")
	}

	c, err := mfl.New("api", 2) // 2 req/s — comfortably under MFL's limits
	if err != nil {
		t.Fatalf("mfl.New: %v", err)
	}
	// Two league-specific calls plus a possible 429 backoff cycle (~123s).
	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	cfg, err := Fetch(ctx, c, ingestion.SeasonYear, ingestion.LeagueID)
	if err != nil {
		t.Fatalf("Fetch against live MFL: %v", err)
	}

	if cfg.SalaryCapAmount == "" {
		t.Error("live cap amount empty")
	}
	if len(cfg.ScoringRules) == 0 {
		t.Fatal("live config has zero scoring rule blocks")
	}

	// The decoded scoring config must contain the missed-FG rule (MG, OQ-002) — a
	// concrete proof the rules export decoded, not just the league export.
	var sawMG bool
	for _, set := range cfg.ScoringRules {
		for _, r := range set.Rules {
			if strings.TrimSpace(r.Event) == "MG" {
				sawMG = true
			}
		}
	}
	if !sawMG {
		t.Error("missed-FG (MG) rule not found in live scoring config")
	}
	t.Logf("live rulebook: cap=%s, rosterSize=%s, %d scoring blocks",
		cfg.SalaryCapAmount, cfg.RosterSize, len(cfg.ScoringRules))
}
