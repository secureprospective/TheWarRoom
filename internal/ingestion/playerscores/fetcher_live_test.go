package playerscores

import (
	"context"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/mfl"
)

// TestLive_PlayerScoresFetch exercises Fetch against the REAL MFL playerScores
// endpoint. Opt-in: set TWR_LIVE_MFL=1. Makes outbound network calls, so it stays
// out of the default suite/CI/pre-commit but still compiles and is linted with
// everything else. Mirrors the rosters live test (league-host path, discovers the
// host first).
//
//	TWR_LIVE_MFL=1 go test -run TestLive_PlayerScoresFetch -v ./internal/ingestion/playerscores/...
//
// Live cross-check (M1 recon, 2026-07-02): the 2025 YTD export carried 1650
// records with the top score 476.75 (id 13130). We assert a populated, sane
// export — every score parses, every week echoes "YTD", and the top score is in a
// plausible fantasy-points band — without hardcoding exact values (a re-scoring or
// stat correction may shift totals slightly).
func TestLive_PlayerScoresFetch(t *testing.T) {
	if os.Getenv("TWR_LIVE_MFL") != "1" {
		t.Skip("live MFL playerScores fetch: set TWR_LIVE_MFL=1 to run (makes real network calls)")
	}

	c, err := mfl.New("api", 2)
	if err != nil {
		t.Fatalf("mfl.New: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Second)
	defer cancel()

	records, err := Fetch(ctx, c, ingestion.SeasonYear, ingestion.LeagueID, "2025")
	if err != nil {
		t.Fatalf("Fetch against live MFL: %v", err)
	}
	if len(records) < 500 {
		t.Fatalf("got %d records, want a populated completed-season export (recon saw 1650)", len(records))
	}

	var top float64
	for _, r := range records {
		if r.Week != "YTD" {
			t.Fatalf("record %s has week %q, want the requested YTD aggregate", r.ID, r.Week)
		}
		v, perr := strconv.ParseFloat(r.Score, 64)
		if perr != nil {
			t.Fatalf("record %s has non-numeric score %q despite Validate", r.ID, r.Score)
		}
		if v > top {
			top = v
		}
	}
	if top < 100 || top > 1500 {
		t.Fatalf("top YTD score %v outside plausible fantasy band [100,1500]", top)
	}
	t.Logf("live playerScores: %d records, top YTD %v", len(records), top)
}
