package madden

import (
	"context"
	"net/http"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/secureprospective/TheWarRoom/internal/ingestion"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/crosswalk"
)

// TestLive_MaddenFetch exercises Fetch against the REAL EA ratings-api (m24 — the only
// populated slug) and wires a REAL name+birthdate->gsis resolver built from
// db_playerids, proving the pagination, the dynamic "_rating" parse, and the brittle
// keying end to end. It demonstrates how a caller assembles the injected resolver.
// Opt-in: set TWR_LIVE_EA=1 (it also fetches db_playerids).
//
//	TWR_LIVE_EA=1 go test -run TestLive_MaddenFetch -v ./internal/ingestion/madden/...
func TestLive_MaddenFetch(t *testing.T) {
	if os.Getenv("TWR_LIVE_EA") != "1" {
		t.Skip("live madden fetch: set TWR_LIVE_EA=1 to run (hits EA + db_playerids)")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	resolve := buildNameBirthdateResolver(ctx, t)

	out, err := Fetch(ctx, &http.Client{Timeout: 60 * time.Second}, RatingsURL, resolve)
	if err != nil {
		t.Fatalf("Fetch against live EA m24: %v", err)
	}

	// ~77% of m24's ~1992 players match by name+birthdate -> expect well over 1000.
	if len(out) < 1000 {
		t.Errorf("resolved only %d gsis-keyed Madden records; expected >1000", len(out))
	}

	// Every overall rating is a sane Madden integer; every dynamic attribute too.
	for gsis, r := range out {
		if r.Overall < 0 || r.Overall > 99 {
			t.Errorf("%s (%s) Overall = %d, out of [0,99]", r.FullName, gsis, r.Overall)
		}
		for name, v := range r.Attributes {
			if v < 0 || v > 99 {
				t.Errorf("%s %s = %d, out of [0,99]", r.FullName, name, v)
			}
		}
	}
	t.Logf("resolved %d gsis-keyed Madden records from m24", len(out))
}

// buildNameBirthdateResolver fetches db_playerids and indexes (normalized name, ISO
// birthdate) -> gsis, returning a GSISResolver. This is the brittle match the fetcher
// deliberately keeps injected; a real caller (assembly) would build it the same way.
func buildNameBirthdateResolver(ctx context.Context, t *testing.T) GSISResolver {
	t.Helper()
	records, err := ingestion.FetchCSV(ctx, &http.Client{Timeout: 60 * time.Second}, crosswalk.SourceURL, ingestion.DefaultMaxCSVBytes)
	if err != nil {
		t.Fatalf("fetch db_playerids for resolver: %v", err)
	}
	cols, err := ingestion.CSVColumns(records[0], "name", "birthdate", "gsis_id")
	if err != nil {
		t.Fatalf("db_playerids columns: %v", err)
	}

	index := map[string]string{}
	for _, rec := range records[1:] {
		gsis := strings.TrimSpace(rec[cols["gsis_id"]])
		birth := isoBirth(strings.TrimSpace(rec[cols["birthdate"]]))
		if ingestion.IsMissing(gsis) || birth == "" {
			continue
		}
		index[normName(rec[cols["name"]])+"|"+birth] = gsis
	}

	return func(fullName, birthdate string) (string, bool) {
		g, ok := index[normName(fullName)+"|"+isoBirth(birthdate)]
		return g, ok
	}
}

var nonAlpha = regexp.MustCompile(`[^a-z]`)

func normName(s string) string { return nonAlpha.ReplaceAllString(strings.ToLower(s), "") }

// isoBirth normalizes both EA's M/D/YYYY and db_playerids' ISO YYYY-MM-DD to YYYY-MM-DD,
// or "" if unparseable (an unparseable birthdate just yields a non-match).
func isoBirth(b string) string {
	for _, layout := range []string{"2006-01-02", "1/2/2006"} {
		if d, err := time.Parse(layout, strings.TrimSpace(b)); err == nil {
			return d.Format("2006-01-02")
		}
	}
	return ""
}
