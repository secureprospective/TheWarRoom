package assembly

import (
	"bytes"
	"compress/gzip"
	"context"
	"math"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/secureprospective/TheWarRoom/internal/domain"
	"github.com/secureprospective/TheWarRoom/internal/ingestion/crosswalk"
	"github.com/secureprospective/TheWarRoom/internal/playerid"
)

// covHeader mirrors the real pfr_advstats season-defense header (the fetcher binds by name,
// so only the columns it reads must carry values; the rest are padded).
const covHeader = "season,player,pfr_id,tm,pos,g,gs,int,tgt,cmp,cmp_percent,yds,yds_cmp,yds_tgt,td,rat,dadot,air,yac,bltz,hrry,qbkd,sk,prss,comb,m_tkl,m_tkl_percent,loaded,bats"

// covRow builds one defense CSV line. Only season/player/pfr/pos/tgt/rat matter to the
// coverage leaf; everything else is padded to the 29-column width.
func covRow(season, player, pfr, pos, tgt, rat string) string {
	c := []string{season, player, pfr, "TM", pos, "16", "16", "0", tgt, "10", "0.5", "100",
		"10.0", "8.0", "1", rat, "8.0", "", "", "", "", "", "", "", "", "", "", "", ""}
	out := c[0]
	for _, v := range c[1:] {
		out += "," + v
	}
	return out
}

// serveCoverageGz serves body gzip-compressed (the real source is .csv.gz).
func serveCoverageGz(t *testing.T, body string) (*http.Client, string) {
	t.Helper()
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	if _, err := zw.Write([]byte(body)); err != nil {
		t.Fatalf("gzip write: %v", err)
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("gzip close: %v", err)
	}
	raw := buf.Bytes()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(raw)
	}))
	t.Cleanup(srv.Close)
	return srv.Client(), srv.URL
}

// covCrosswalkCSV maps six fixture defenders end to end via the pfr_id bridge (the join
// pfrcoverage uses): mfl 1001..1006 → gsis G-1..G-6, pfr P-1..P-6 → the SAME gsis.
const covCrosswalkCSV = `mfl_id,gsis_id,espn_id,pfr_id
1001,G-1,,P-1
1002,G-2,,P-2
1003,G-3,,P-3
1004,G-4,,P-4
1005,G-5,,P-5
1006,G-6,,P-6
`

func covClose(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

// TestBuildCoverage_InvertNormalizeBoundary proves the coverage leaf end to end: the
// passer-rating-allowed is inverted+clamped to [0,1] (best→1, worst→0, median→~0.5), the
// CB/S boundary drops a non-CB/S defender, the target floor drops a thin sample, and an
// absent rating is a clean miss.
func TestBuildCoverage_InvertNormalizeBoundary(t *testing.T) {
	body := covHeader + "\n" +
		covRow("2024", "CB Elite", "P-1", "CB", "50", "40.0") + "\n" + // → 1.0 (best anchor)
		covRow("2024", "S Poor", "P-2", "FS", "30", "145.0") + "\n" + // → 0.0 (worst anchor)
		covRow("2024", "CB Mid", "P-3", "CB", "40", "92.5") + "\n" + // → 0.5 (median)
		covRow("2024", "LB Cover", "P-4", "ILB", "40", "90.0") + "\n" + // non-CB/S → dropped
		covRow("2024", "CB Thin", "P-5", "CB", "5", "50.0") + "\n" + // < floor → dropped
		covRow("2024", "CB NoRat", "P-6", "CB", "20", "") + "\n" // absent rating → dropped
	client, url := serveCoverageGz(t, body)
	cw := crosswalkFixture(t, covCrosswalkCSV)

	roster := []string{"1001", "1002", "1003", "1004", "1005", "1006"}
	pos := fakePosLookup{
		"1001": domain.PosCB,
		"1002": domain.PosS,
		"1003": domain.PosCB,
		"1004": domain.PosLB,
		"1005": domain.PosCB,
		"1006": domain.PosCB,
	}

	out, err := BuildCoverage(context.Background(), client, url, cw, "2024", roster, pos)
	if err != nil {
		t.Fatalf("BuildCoverage: %v", err)
	}
	if len(out) != 3 {
		t.Fatalf("want 3 coverage anchors (CB/S above floor with a rating), got %d: %+v", len(out), out)
	}

	id1, _ := playerid.New("1001")
	id2, _ := playerid.New("1002")
	id3, _ := playerid.New("1003")
	if !covClose(out[id1], 1.0) {
		t.Errorf("CB Elite (rat 40): got %v, want 1.0", out[id1])
	}
	if !covClose(out[id2], 0.0) {
		t.Errorf("S Poor (rat 145): got %v, want 0.0", out[id2])
	}
	wantMid := (coverageWorst - 92.5) / (coverageWorst - coverageBest)
	if !covClose(out[id3], wantMid) {
		t.Errorf("CB Mid (rat 92.5): got %v, want %v (~0.5)", out[id3], wantMid)
	}
}

// TestBuildCoverage_ClampsBeyondAnchors: a rating better than coverageBest or worse than
// coverageWorst clamps to 1.0 / 0.0 rather than exceeding the unit range (the film
// composite's [0,1] invariant).
func TestBuildCoverage_ClampsBeyondAnchors(t *testing.T) {
	body := covHeader + "\n" +
		covRow("2024", "CB Perfect", "P-1", "CB", "50", "0.0") + "\n" + // below best → clamp 1.0
		covRow("2024", "CB Torched", "P-2", "CB", "50", "158.3") + "\n" // above worst → clamp 0.0
	client, url := serveCoverageGz(t, body)
	cw := crosswalkFixture(t, covCrosswalkCSV)

	roster := []string{"1001", "1002"}
	pos := fakePosLookup{"1001": domain.PosCB, "1002": domain.PosCB}

	out, err := BuildCoverage(context.Background(), client, url, cw, "2024", roster, pos)
	if err != nil {
		t.Fatalf("BuildCoverage: %v", err)
	}
	id1, _ := playerid.New("1001")
	id2, _ := playerid.New("1002")
	if !covClose(out[id1], 1.0) {
		t.Errorf("CB Perfect (rat 0): got %v, want clamp 1.0", out[id1])
	}
	if !covClose(out[id2], 0.0) {
		t.Errorf("CB Torched (rat 158.3): got %v, want clamp 0.0", out[id2])
	}
}

// TestBuildCoverage_FetchFailureLoud: a source that resolves zero defenders surfaces loudly
// (a coverage-less league must be visible), matching BuildRAS/BuildCollegeDefense.
func TestBuildCoverage_FetchFailureLoud(t *testing.T) {
	client, url := serveCoverageGz(t, covHeader+"\n") // header only, no rows
	cw := crosswalkFixture(t, covCrosswalkCSV)
	if _, err := BuildCoverage(context.Background(), client, url, cw, "2024",
		[]string{"1001"}, fakePosLookup{"1001": domain.PosCB}); err == nil {
		t.Fatal("want a loud error on a zero-defender source, got nil")
	}
}

// TestBuildCoverage_GuardsNilDeps: nil client / nil lookup fail loud before any fetch.
func TestBuildCoverage_GuardsNilDeps(t *testing.T) {
	if _, err := BuildCoverage(context.Background(), nil, "u", crosswalk.Map{}, "2024",
		nil, fakePosLookup{}); err == nil {
		t.Fatal("want error on nil client, got nil")
	}
	if _, err := BuildCoverage(context.Background(), &http.Client{}, "u", crosswalk.Map{}, "2024",
		nil, nil); err == nil {
		t.Fatal("want error on nil PositionLookup, got nil")
	}
}
