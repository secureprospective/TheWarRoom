package pfrcoverage

import (
	"bytes"
	"compress/gzip"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

const header = "season,player,pfr_id,tm,pos,g,gs,int,tgt,cmp,cmp_percent,yds,yds_cmp,yds_tgt,td,rat,dadot,air,yac,bltz,hrry,qbkd,sk,prss,comb,m_tkl,m_tkl_percent,loaded,bats"

// serveGzCSV serves the given CSV body gzip-compressed (the real source is .csv.gz).
func serveGzCSV(t *testing.T, body string) (*http.Client, string) {
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

// row builds a defense CSV line matching header. Only the columns the fetcher reads are
// meaningful; the rest are padded.
func row(season, player, pfr, tm, pos, intc, tgt, cmp, cmpPct, yds, ydsCmp, ydsTgt, td, rat, dadot string) string {
	// header order: season,player,pfr_id,tm,pos,g,gs,int,tgt,cmp,cmp_percent,yds,
	// yds_cmp,yds_tgt,td,rat,dadot,air,yac,bltz,hrry,qbkd,sk,prss,comb,m_tkl,
	// m_tkl_percent,loaded,bats  (29 cols)
	c := []string{season, player, pfr, tm, pos, "16", "16", intc, tgt, cmp, cmpPct, yds,
		ydsCmp, ydsTgt, td, rat, dadot, "", "", "", "", "", "", "", "", "", "", "", ""}
	out := c[0]
	for _, v := range c[1:] {
		out += "," + v
	}
	return out
}

func fetch(t *testing.T, body, season string, m map[string]string) (map[string]RawCoverage, error) {
	t.Helper()
	client, url := serveGzCSV(t, body)
	return Fetch(context.Background(), client, url, season, m)
}

func TestFetch_HappyPath(t *testing.T) {
	t.Parallel()
	body := header + "\n" +
		row("2024", "Budda Baker", "BakeBu00", "ARI", "FS", "0", "74", "55", "0.743", "608", "11.1", "8.2", "3", "129.8", "8.4") + "\n"
	m := map[string]string{"BakeBu00": "00-0033107"}

	out, err := fetch(t, body, "2024", m)
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	rc, ok := out["00-0033107"]
	if !ok {
		t.Fatalf("expected gsis 00-0033107 in output, got %v", out)
	}
	if rc.Position != "FS" || rc.Targets != 74 || rc.Completions != 55 || rc.YardsAllowed != 608 || rc.TDsAllowed != 3 {
		t.Errorf("counting fields wrong: %+v", rc)
	}
	if rc.PasserRatingAllowed == nil || *rc.PasserRatingAllowed != 129.8 {
		t.Errorf("PasserRatingAllowed = %v, want 129.8", rc.PasserRatingAllowed)
	}
	if rc.CompletionPct == nil || *rc.CompletionPct != 0.743 {
		t.Errorf("CompletionPct = %v, want 0.743", rc.CompletionPct)
	}
}

func TestFetch_UntargetedDefenderSkipped(t *testing.T) {
	t.Parallel()
	// A pure pass-rusher: 0 targets, blank rates -> no coverage signal -> skipped.
	body := header + "\n" +
		row("2024", "Aaron Donald", "DonaAa00", "LA", "DT", "0", "0", "0", "", "0", "", "", "0", "", "") + "\n"
	m := map[string]string{"DonaAa00": "00-0031544"}

	_, err := fetch(t, body, "2024", m)
	if err == nil {
		t.Fatal("expected errEmpty: the only defender was untargeted")
	}
}

func TestFetch_TradedPlayerPrefersAggregate(t *testing.T) {
	t.Parallel()
	// Traded player: a 2TM aggregate plus two per-team split rows. The aggregate must
	// win and the splits must be dropped (one row out, with the aggregate's numbers).
	body := header + "\n" +
		row("2024", "Trade Guy", "TradGu00", "2TM", "CB", "1", "60", "35", "0.583", "400", "11.4", "6.7", "2", "78.0", "9.0") + "\n" +
		row("2024", "Trade Guy", "TradGu00", "DAL", "CB", "0", "40", "24", "0.600", "280", "11.7", "7.0", "1", "82.0", "9.2") + "\n" +
		row("2024", "Trade Guy", "TradGu00", "NYG", "CB", "1", "20", "11", "0.550", "120", "10.9", "6.0", "1", "70.0", "8.6") + "\n"
	m := map[string]string{"TradGu00": "00-0011111"}

	out, err := fetch(t, body, "2024", m)
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if len(out) != 1 {
		t.Fatalf("want exactly 1 record (the aggregate), got %d: %v", len(out), out)
	}
	rc := out["00-0011111"]
	if rc.Targets != 60 || rc.PasserRatingAllowed == nil || *rc.PasserRatingAllowed != 78.0 {
		t.Errorf("did not pick the 2TM aggregate row: %+v", rc)
	}
}

func TestFetch_TradedPlayerNoAggregateDropped(t *testing.T) {
	t.Parallel()
	// Splits with NO combined aggregate row -> ambiguous -> dropped entirely.
	body := header + "\n" +
		row("2024", "Split Guy", "SpltGu00", "DAL", "CB", "0", "40", "24", "0.600", "280", "11.7", "7.0", "1", "82.0", "9.2") + "\n" +
		row("2024", "Split Guy", "SpltGu00", "NYG", "CB", "1", "20", "11", "0.550", "120", "10.9", "6.0", "1", "70.0", "8.6") + "\n"
	m := map[string]string{"SpltGu00": "00-0022222"}

	_, err := fetch(t, body, "2024", m)
	if err == nil {
		t.Fatal("expected errEmpty: the only player had ambiguous splits and is dropped")
	}
}

func TestFetch_UnresolvablePfrSkipped(t *testing.T) {
	t.Parallel()
	body := header + "\n" +
		row("2024", "Known", "KnowGu00", "ARI", "FS", "0", "50", "30", "0.600", "350", "11.6", "7.0", "2", "90.0", "8.0") + "\n" +
		row("2024", "Unknown", "UnknGu00", "ARI", "CB", "0", "40", "20", "0.500", "200", "10.0", "5.0", "1", "75.0", "7.0") + "\n"
	m := map[string]string{"KnowGu00": "00-0033333"} // UnknGu00 absent from crosswalk

	out, err := fetch(t, body, "2024", m)
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if len(out) != 1 || out["00-0033333"].Targets != 50 {
		t.Fatalf("want only the resolvable player, got %v", out)
	}
}

func TestFetch_PoisonGsisDropped(t *testing.T) {
	t.Parallel()
	// Two DISTINCT pfr ids resolving to one gsis -> drop both (ras precedent).
	body := header + "\n" +
		row("2024", "Player A", "PlayA000", "ARI", "CB", "0", "50", "30", "0.600", "350", "11.6", "7.0", "2", "90.0", "8.0") + "\n" +
		row("2024", "Player B", "PlayB000", "SEA", "CB", "0", "40", "20", "0.500", "200", "10.0", "5.0", "1", "75.0", "7.0") + "\n"
	m := map[string]string{"PlayA000": "00-0044444", "PlayB000": "00-0044444"}

	_, err := fetch(t, body, "2024", m)
	if err == nil {
		t.Fatal("expected errEmpty: the only gsis was poisoned by two pfr ids")
	}
}

func TestFetch_SeasonFilter(t *testing.T) {
	t.Parallel()
	body := header + "\n" +
		row("2023", "Old Year", "YearGu00", "ARI", "FS", "0", "99", "60", "0.606", "700", "11.7", "7.1", "5", "95.0", "8.0") + "\n" +
		row("2024", "New Year", "YearGu00", "ARI", "FS", "0", "50", "30", "0.600", "350", "11.6", "7.0", "2", "90.0", "8.0") + "\n"
	m := map[string]string{"YearGu00": "00-0055555"}

	out, err := fetch(t, body, "2024", m)
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if out["00-0055555"].Targets != 50 {
		t.Fatalf("season filter failed: got %+v, want the 2024 row (tgt=50)", out["00-0055555"])
	}
}

func TestFetch_AbsentRateIsNilNotZero(t *testing.T) {
	t.Parallel()
	// dadot blank -> nil (absent); rat present as 0.0 -> &0.0 (a real perfect-coverage
	// value, which must NOT be confused with absent). Both on a targeted defender.
	body := header + "\n" +
		row("2024", "Edge Case", "EdgeGu00", "ARI", "CB", "0", "10", "0", "0.000", "0", "", "0.0", "0", "0.0", "") + "\n"
	m := map[string]string{"EdgeGu00": "00-0066666"}

	out, err := fetch(t, body, "2024", m)
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	rc := out["00-0066666"]
	if rc.AvgDepthOfTarget != nil {
		t.Errorf("AvgDepthOfTarget = %v, want nil (blank dadot is absent)", rc.AvgDepthOfTarget)
	}
	if rc.PasserRatingAllowed == nil || *rc.PasserRatingAllowed != 0.0 {
		t.Errorf("PasserRatingAllowed = %v, want &0.0 (a real value, not absent)", rc.PasserRatingAllowed)
	}
}

func TestFetch_MalformedCountFailsLoud(t *testing.T) {
	t.Parallel()
	// A present-but-unparseable counting cell (tgt="oops") is corruption, not zero.
	body := header + "\n" +
		row("2024", "Bad Row", "BadrGu00", "ARI", "CB", "0", "oops", "30", "0.600", "350", "11.6", "7.0", "2", "90.0", "8.0") + "\n"
	m := map[string]string{"BadrGu00": "00-0077777"}

	_, err := fetch(t, body, "2024", m)
	if err == nil {
		t.Fatal("expected a malformed-cell error on tgt=oops")
	}
}

func TestIsTeamAggregate(t *testing.T) {
	t.Parallel()
	for _, s := range []string{"2TM", "3TM", "10TM"} {
		if !isTeamAggregate(s) {
			t.Errorf("isTeamAggregate(%q) = false, want true", s)
		}
	}
	for _, s := range []string{"TB", "LA", "TM", "2T", "2TMX", ""} {
		if isTeamAggregate(s) {
			t.Errorf("isTeamAggregate(%q) = true, want false", s)
		}
	}
}
