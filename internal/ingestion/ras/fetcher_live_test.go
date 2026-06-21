package ras

import (
	"context"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/ingestion"
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

	pfrToGSIS, err := livePfrToGSIS(ctx, client)
	if err != nil {
		t.Fatalf("build live pfr->gsis crosswalk: %v", err)
	}
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

// livePfrToGSIS builds the pfr_id -> gsis_id map from the dynastyprocess
// db_playerids.csv (the same source the crosswalk fetcher uses), mirroring how the
// assembly layer will supply it in production.
func livePfrToGSIS(ctx context.Context, client *http.Client) (map[string]string, error) {
	const src = "https://raw.githubusercontent.com/dynastyprocess/data/master/files/db_playerids.csv"
	records, err := ingestion.FetchCSV(ctx, client, src, ingestion.DefaultMaxCSVBytes)
	if err != nil {
		return nil, err
	}
	cols, err := ingestion.CSVColumns(records[0], "pfr_id", "gsis_id")
	if err != nil {
		return nil, err
	}
	out := make(map[string]string)
	for _, rec := range records[1:] {
		pfr := strings.TrimSpace(rec[cols["pfr_id"]])
		gsis := strings.TrimSpace(rec[cols["gsis_id"]])
		if ingestion.IsMissing(pfr) || ingestion.IsMissing(gsis) {
			continue
		}
		out[pfr] = gsis
	}
	return out, nil
}
