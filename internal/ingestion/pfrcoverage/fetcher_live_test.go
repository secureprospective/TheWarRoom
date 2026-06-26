package pfrcoverage

import (
	"context"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/ingestion"
)

// TestLive_CoverageFetch exercises Fetch against the REAL nflverse pfr_advstats
// advstats_season_def.csv.gz, resolving pfr->gsis from the live dynastyprocess
// crosswalk. Opt-in: set TWR_LIVE_NFLVERSE=1. Makes two outbound network calls (one
// gzipped), so it stays out of the default suite/CI but compiles + lints with
// everything else. Also proves the FetchCSVGz gzip seam end-to-end against a real
// .csv.gz source.
//
//	TWR_LIVE_NFLVERSE=1 go test -run TestLive_CoverageFetch -v ./internal/ingestion/pfrcoverage/...
func TestLive_CoverageFetch(t *testing.T) {
	if os.Getenv("TWR_LIVE_NFLVERSE") != "1" {
		t.Skip("live coverage fetch: set TWR_LIVE_NFLVERSE=1 to run (makes real network calls)")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	client := &http.Client{Timeout: 90 * time.Second}

	pfrToGSIS, err := livePfrToGSIS(ctx, client)
	if err != nil {
		t.Fatalf("build live pfr->gsis crosswalk: %v", err)
	}
	t.Logf("crosswalk resolved %d pfr->gsis entries", len(pfrToGSIS))

	const season = "2024"
	m, err := Fetch(ctx, client, SourceURL, season, pfrToGSIS)
	if err != nil {
		t.Fatalf("Fetch against live source: %v", err)
	}

	// A full season carries hundreds of targeted defenders that resolve to a gsis; a low
	// count means a truncated download, a schema change, an absent season, or a pfr-id
	// format drift between the two sources.
	if len(m) < 200 {
		t.Errorf("resolved only %d %s coverage records; expected >200", len(m), season)
	}
	t.Logf("resolved %d gsis-keyed coverage records for %s", len(m), season)

	// Spot-invariant: every emitted record was targeted, and any present passer rating
	// allowed is in the football-legal [0, 158.3] range (sanity on the headline metric).
	for gsis, rc := range m {
		if rc.Targets <= 0 {
			t.Fatalf("%s emitted with Targets=%d (untargeted should be skipped)", gsis, rc.Targets)
		}
		if rc.PasserRatingAllowed != nil {
			if r := *rc.PasserRatingAllowed; r < 0 || r > 158.3 {
				t.Fatalf("%s passer rating allowed %.1f outside [0,158.3]", gsis, r)
			}
		}
	}
}

// livePfrToGSIS builds the pfr_id -> gsis_id map from the dynastyprocess
// db_playerids.csv (the same source the crosswalk fetcher uses), mirroring how the
// assembly layer will supply it in production. (Duplicated from touchshare/ras live
// tests; promoting this bridge into the crosswalk package is a flagged module-close
// item — see the package doc.)
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
