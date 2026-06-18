package salaryadjustments

import (
	"context"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/mfl"
)

// TestLive_SalaryAdjustmentsFetch exercises Fetch against the REAL MFL
// salaryAdjustments endpoint. Opt-in: set TWR_LIVE_MFL=1. Makes outbound network
// calls, so it stays out of the default suite/CI/pre-commit but still compiles and
// is linted with everything else. Mirrors the rosters live test (league-host path,
// discovers the host first).
//
//	TWR_LIVE_MFL=1 go test -run TestLive_SalaryAdjustmentsFetch -v ./internal/ingestion/salaryadjustments/...
//
// Live cross-check (OQ-005): the Cardinals' franchise 0025 dead-cap ledger summed to
// $5.495 at capture time. We assert the franchise is present and re-print its sum so
// drift surfaces; we do NOT hardcode-assert 5.495 (live drops keep accumulating).
func TestLive_SalaryAdjustmentsFetch(t *testing.T) {
	if os.Getenv("TWR_LIVE_MFL") != "1" {
		t.Skip("live MFL salaryAdjustments fetch: set TWR_LIVE_MFL=1 to run (makes real network calls)")
	}

	c, err := mfl.New("api", 2)
	if err != nil {
		t.Fatalf("mfl.New: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Second)
	defer cancel()

	records, err := Fetch(ctx, c, ingestion.SeasonYear, ingestion.LeagueID)
	if err != nil {
		t.Fatalf("Fetch against live MFL: %v", err)
	}
	if len(records) == 0 {
		t.Fatal("Fetch returned 0 adjustments (expected a populated league ledger)")
	}

	const cardinals = "0025"
	var cardinalsSum float64
	var cardinalsCount int
	for _, r := range records {
		if r.FranchiseID == cardinals {
			amt, perr := strconv.ParseFloat(r.Amount, 64)
			if perr != nil {
				t.Errorf("franchise %s record %s has non-numeric amount %q despite Validate", r.FranchiseID, r.ID, r.Amount)
				continue
			}
			cardinalsSum += amt
			cardinalsCount++
		}
	}
	if cardinalsCount == 0 {
		t.Errorf("franchise %s not present in live ledger", cardinals)
	}
	t.Logf("fetched %d adjustments; franchise %s: %d entries summing $%.3f", len(records), cardinals, cardinalsCount, cardinalsSum)
}
